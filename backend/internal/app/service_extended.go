package app

import (
	"bytes"
	"encoding/base64"
	"image/png"
	"log"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pquerna/otp/totp"
)

var (
	hexConfirmationCodePattern = regexp.MustCompile(`^[0-9a-f]{6}$`)
	totpCodePattern            = regexp.MustCompile(`^[0-9]{6}$`)
)

type TwoFaCodeData struct {
	Code string `json:"code"`
}

type Generate2FAData struct {
	EmailCode string `json:"email_code"`
}

type RequestEmailData struct {
	Email string `json:"email"`
}

type ConfirmEmailData struct {
	Code string `json:"code"`
}

func (s *Service) request2FAEnable(token string) Response {
	user, err := s.userBySessionToken(token)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	if user.Email == "" {
		return Response{OK: false, Error: "email_required"}
	}

	if s.mailer == nil || !s.mailer.Available() {
		return Response{OK: false, Error: "email delivery is unavailable"}
	}
	code, err := randomHexCode(3)
	if err != nil {
		return Response{OK: false, Error: "failed to generate confirmation code"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	if err := s.createAuthChallenge(ctx, user.ID, "2fa_enable", "", code, 15*time.Minute); err != nil {
		return Response{OK: false, Error: "failed to generate confirmation code"}
	}

	go func() {
		if err := s.mailer.Send2FAEnableConfirmation(user.Email, code); err != nil {
			log.Printf("failed to send 2FA enable confirmation: %v", err)
		}
	}()

	return Response{OK: true, Result: "confirmation code sent to email"}
}

func (s *Service) generate2fa(token string, data Generate2FAData) Response {
	user, err := s.userBySessionToken(token)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	if user.Email == "" {
		return Response{OK: false, Error: "email_required"}
	}

	emailCode := strings.ToLower(strings.TrimSpace(data.EmailCode))
	if !hexConfirmationCodePattern.MatchString(emailCode) {
		return Response{OK: false, Error: "email confirmation code is required"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "TelecomDep E-Journal",
		AccountName: user.Login,
	})
	if err != nil {
		return Response{OK: false, Error: "failed to generate TOTP secret"}
	}

	err = s.consumeAuthChallenge(ctx, user.ID, "2fa_enable", emailCode, func(tx pgx.Tx, _ string) error {
		_, err := tx.Exec(ctx, "UPDATE users SET totp_secret = $1 WHERE id = $2", key.Secret(), user.ID)
		return err
	})
	if err != nil {
		if err == errInvalidAuthChallenge {
			return Response{OK: false, Error: "invalid or expired email confirmation code"}
		}
		return Response{OK: false, Error: "failed to save TOTP secret"}
	}

	var buf bytes.Buffer
	img, err := key.Image(200, 200)
	if err != nil {
		return Response{OK: false, Error: "failed to generate QR code image"}
	}
	if err := png.Encode(&buf, img); err != nil {
		return Response{OK: false, Error: "failed to encode QR code image"}
	}

	qrBase64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	return Response{
		OK: true,
		Result: map[string]any{
			"qr_code": "data:image/png;base64," + qrBase64,
			"secret":  key.Secret(),
		},
	}
}

func (s *Service) verify2fa(token string, data TwoFaCodeData) Response {
	user, err := s.userBySessionToken(token)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	var secret string
	err = s.store.Pool().QueryRow(ctx, "SELECT totp_secret FROM users WHERE id = $1", user.ID).Scan(&secret)
	if err != nil || secret == "" {
		return Response{OK: false, Error: "2FA setup not initiated"}
	}

	code := strings.TrimSpace(data.Code)
	valid := totpCodePattern.MatchString(code) && totp.Validate(code, secret)
	if !valid {
		return Response{OK: false, Error: "invalid TOTP code"}
	}

	_, err = s.store.Pool().Exec(ctx, `
		UPDATE users
		SET is_2fa_enabled = true, token_version = token_version + 1
		WHERE id = $1
	`, user.ID)
	if err != nil {
		return Response{OK: false, Error: "failed to enable 2FA"}
	}

	return Response{OK: true, Result: "2FA enabled; sign in again"}
}

func (s *Service) disable2fa(token string, data TwoFaCodeData) Response {
	user, err := s.userBySessionToken(token)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	code := strings.TrimSpace(data.Code)
	if !totpCodePattern.MatchString(code) {
		return Response{OK: false, Error: "valid TOTP code is required"}
	}

	var secret string
	if err := s.store.Pool().QueryRow(ctx, `
		SELECT COALESCE(totp_secret, '') FROM users WHERE id = $1 AND is_2fa_enabled = true
	`, user.ID).Scan(&secret); err != nil || secret == "" || !totp.Validate(code, secret) {
		return Response{OK: false, Error: "invalid TOTP code"}
	}

	_, err = s.store.Pool().Exec(ctx, `
		UPDATE users
		SET is_2fa_enabled = false,
		    totp_secret = NULL,
		    token_version = token_version + 1
		WHERE id = $1
	`, user.ID)
	if err != nil {
		return Response{OK: false, Error: "failed to disable 2FA"}
	}

	return Response{OK: true, Result: "2FA disabled; sign in again"}
}

func (s *Service) requestEmailBind(token string, data RequestEmailData) Response {
	user, err := s.userBySessionToken(token)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	emailAddress := strings.ToLower(strings.TrimSpace(data.Email))
	parsedAddress, parseErr := mail.ParseAddress(emailAddress)
	if parseErr != nil || !strings.EqualFold(parsedAddress.Address, emailAddress) || len(emailAddress) > 254 {
		return Response{OK: false, Error: "email is required"}
	}
	if s.mailer == nil || !s.mailer.Available() {
		return Response{OK: false, Error: "email delivery is unavailable"}
	}

	code, err := randomHexCode(3)
	if err != nil {
		return Response{OK: false, Error: "failed to generate confirmation code"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	var existingUserID int32
	err = s.store.Pool().QueryRow(ctx, `SELECT id FROM users WHERE LOWER(email) = LOWER($1)`, emailAddress).Scan(&existingUserID)
	if err == nil && existingUserID != user.ID {
		return Response{OK: false, Error: "email is already in use"}
	}
	if err != nil && err != pgx.ErrNoRows {
		return Response{OK: false, Error: "failed to validate email"}
	}

	if err := s.createAuthChallenge(ctx, user.ID, "email_bind", emailAddress, code, 15*time.Minute); err != nil {
		return Response{OK: false, Error: "failed to generate confirmation code"}
	}

	go func() {
		if err := s.mailer.SendEmailConfirmation(emailAddress, code); err != nil {
			log.Printf("failed to send email confirmation: %v", err)
		}
	}()

	return Response{OK: true, Result: "confirmation code sent"}
}

func (s *Service) confirmEmailBind(token string, data ConfirmEmailData) Response {
	user, err := s.userBySessionToken(token)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	code := strings.ToLower(strings.TrimSpace(data.Code))
	if !hexConfirmationCodePattern.MatchString(code) {
		return Response{OK: false, Error: "invalid or expired confirmation code"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	err = s.consumeAuthChallenge(ctx, user.ID, "email_bind", code, func(tx pgx.Tx, email string) error {
		_, err := tx.Exec(ctx, `UPDATE users SET email = $1 WHERE id = $2`, email, user.ID)
		return err
	})
	if err != nil {
		if err == errInvalidAuthChallenge {
			return Response{OK: false, Error: "invalid or expired confirmation code"}
		}
		return Response{OK: false, Error: "failed to update email, maybe it is already in use"}
	}

	return Response{OK: true, Result: "email successfully bound"}
}
