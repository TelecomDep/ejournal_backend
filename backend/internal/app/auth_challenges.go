package app

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const authChallengeMaxAttempts = 5

var errInvalidAuthChallenge = errors.New("invalid or expired confirmation code")

func randomHexCode(byteCount int) (string, error) {
	if byteCount <= 0 {
		return "", errors.New("invalid random code size")
	}
	b := make([]byte, byteCount)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *Service) authChallengeDigest(userID int32, purpose, code string) []byte {
	mac := hmac.New(sha256.New, s.jwtSecret)
	_, _ = fmt.Fprintf(mac, "%d\x00%s\x00%s", userID, purpose, strings.ToLower(strings.TrimSpace(code)))
	return mac.Sum(nil)
}

func (s *Service) createAuthChallenge(
	ctx context.Context,
	userID int32,
	purpose string,
	target string,
	code string,
	ttl time.Duration,
) error {
	tx, err := s.store.Pool().BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()


	if _, err := tx.Exec(ctx, `
		UPDATE auth_challenges
		SET consumed_at = NOW()
		WHERE user_id = $1 AND purpose = $2 AND consumed_at IS NULL
	`, userID, purpose); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO auth_challenges (user_id, purpose, target, code_hash, expires_at)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, purpose, target, s.authChallengeDigest(userID, purpose, code), time.Now().Add(ttl)); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *Service) consumeAuthChallenge(
	ctx context.Context,
	userID int32,
	purpose string,
	code string,
	apply func(pgx.Tx, string) error,
) error {
	tx, err := s.store.Pool().BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var (
		challengeID int64
		target      string
		expected    []byte
		attempts    int16
	)
	err = tx.QueryRow(ctx, `
		SELECT challenge_id, target, code_hash, attempts
		FROM auth_challenges
		WHERE user_id = $1
		  AND purpose = $2
		  AND consumed_at IS NULL
		  AND expires_at > NOW()
		ORDER BY created_at DESC, challenge_id DESC
		LIMIT 1
		FOR UPDATE
	`, userID, purpose).Scan(&challengeID, &target, &expected, &attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return errInvalidAuthChallenge
	}
	if err != nil {
		return err
	}

	actual := s.authChallengeDigest(userID, purpose, code)
	if len(actual) != len(expected) || subtle.ConstantTimeCompare(actual, expected) != 1 {
		_, updateErr := tx.Exec(ctx, `
			UPDATE auth_challenges
			SET attempts = attempts + 1,
			    consumed_at = CASE WHEN attempts + 1 >= $2 THEN NOW() ELSE consumed_at END
			WHERE challenge_id = $1
		`, challengeID, authChallengeMaxAttempts)
		if updateErr != nil {
			return updateErr
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		return errInvalidAuthChallenge
	}

	if apply != nil {
		if err := apply(tx, target); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE auth_challenges SET consumed_at = NOW() WHERE challenge_id = $1
	`, challengeID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
