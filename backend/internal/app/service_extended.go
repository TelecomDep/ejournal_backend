package app

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"image/png"
	"time"

	"github.com/pquerna/otp/totp"
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

	b := make([]byte, 3)
	rand.Read(b)
	code := hex.EncodeToString(b) // 6 chars hex

	ctx, cancel := s.dbContext()
	defer cancel()

	_, err = s.store.Pool().Exec(ctx, `
		INSERT INTO password_reset_tokens (user_id, token, expires_at) 
		VALUES ($1, $2, $3)
	`, user.ID, "2FA_ENABLE_"+code, time.Now().Add(15*time.Minute))

	if err != nil {
		return Response{OK: false, Error: "failed to generate confirmation code"}
	}

	if s.mailer != nil {
		go s.mailer.Send2FAEnableConfirmation(user.Email, code)
	}

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

	if data.EmailCode == "" {
		return Response{OK: false, Error: "email confirmation code is required"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	var dbToken string
	err = s.store.Pool().QueryRow(ctx, `
		SELECT token FROM password_reset_tokens 
		WHERE user_id = $1 AND token = $2 AND expires_at > NOW()
	`, user.ID, "2FA_ENABLE_"+data.EmailCode).Scan(&dbToken)

	if err != nil {
		return Response{OK: false, Error: "invalid or expired email confirmation code"}
	}

	// Code is valid, remove used token
	_, _ = s.store.Pool().Exec(ctx, "DELETE FROM password_reset_tokens WHERE user_id = $1 AND token = $2", user.ID, "2FA_ENABLE_"+data.EmailCode)

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "TelecomDep E-Journal",
		AccountName: user.Login,
	})
	if err != nil {
		return Response{OK: false, Error: "failed to generate TOTP secret"}
	}

	_, err = s.store.Pool().Exec(ctx, "UPDATE users SET totp_secret = $1 WHERE id = $2", key.Secret(), user.ID)
	if err != nil {
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

	valid := totp.Validate(data.Code, secret)
	if !valid {
		return Response{OK: false, Error: "invalid TOTP code"}
	}

	_, err = s.store.Pool().Exec(ctx, "UPDATE users SET is_2fa_enabled = true WHERE id = $1", user.ID)
	if err != nil {
		return Response{OK: false, Error: "failed to enable 2FA"}
	}

	return Response{OK: true, Result: "2FA enabled"}
}

func (s *Service) disable2fa(token string) Response {
	user, err := s.userBySessionToken(token)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	_, err = s.store.Pool().Exec(ctx, "UPDATE users SET is_2fa_enabled = false, totp_secret = NULL WHERE id = $1", user.ID)
	if err != nil {
		return Response{OK: false, Error: "failed to disable 2FA"}
	}

	return Response{OK: true, Result: "2FA disabled"}
}

func (s *Service) requestEmailBind(token string, data RequestEmailData) Response {
	user, err := s.userBySessionToken(token)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	if data.Email == "" {
		return Response{OK: false, Error: "email is required"}
	}

	b := make([]byte, 3)
	rand.Read(b)
	code := hex.EncodeToString(b) // 6 chars hex

	ctx, cancel := s.dbContext()
	defer cancel()

	_, err = s.store.Pool().Exec(ctx, `
		INSERT INTO password_reset_tokens (user_id, token, expires_at) 
		VALUES ($1, $2, $3)
	`, user.ID, "EMAIL_BIND_"+code+"_"+data.Email, time.Now().Add(15*time.Minute))
	
	if err != nil {
		return Response{OK: false, Error: "failed to generate confirmation code"}
	}

	if s.mailer != nil {
		go s.mailer.SendEmailConfirmation(data.Email, code)
	}

	return Response{OK: true, Result: "confirmation code sent"}
}

func (s *Service) confirmEmailBind(token string, data ConfirmEmailData) Response {
	user, err := s.userBySessionToken(token)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	var dbToken string
	err = s.store.Pool().QueryRow(ctx, `
		SELECT token FROM password_reset_tokens 
		WHERE user_id = $1 AND token LIKE $2 AND expires_at > NOW() 
		ORDER BY created_at DESC LIMIT 1
	`, user.ID, "EMAIL_BIND_"+data.Code+"_%").Scan(&dbToken)

	if err != nil {
		return Response{OK: false, Error: "invalid or expired confirmation code"}
	}

	email := dbToken[len("EMAIL_BIND_"+data.Code+"_"):]

	_, err = s.store.Pool().Exec(ctx, "UPDATE users SET email = $1 WHERE id = $2", email, user.ID)
	if err != nil {
		return Response{OK: false, Error: "failed to update email, maybe it is already in use"}
	}

	_, _ = s.store.Pool().Exec(ctx, "DELETE FROM password_reset_tokens WHERE user_id = $1 AND token LIKE 'EMAIL_BIND_%'", user.ID)

	return Response{OK: true, Result: "email successfully bound"}
}
