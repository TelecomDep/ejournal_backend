package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Agreement struct {
	pool *pgxpool.Pool
}

func NewAgreement(pool *pgxpool.Pool) *Agreement {
	return &Agreement{pool: pool}
}

func (r *Agreement) RecordDecision(
	ctx context.Context,
	userID int32,
	agreementKey string,
	version string,
	decision string,
	documentHash string,
	ip string,
	userAgent string,
) (int64, time.Time, error) {
	var decisionID int64
	var decidedAt time.Time

	err := r.pool.QueryRow(ctx, `
		INSERT INTO user_agreement_decisions (
			user_id,
			agreement_key,
			version,
			decision,
			document_hash,
			ip,
			user_agent
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			NULLIF($5, ''),
			NULLIF($6, '')::inet,
			$7
		)
		RETURNING decision_id, decided_at
	`,
		userID,
		agreementKey,
		version,
		decision,
		documentHash,
		ip,
		userAgent,
	).Scan(&decisionID, &decidedAt)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("record agreement decision: %w", err)
	}

	return decisionID, decidedAt, nil
}

func (r *Agreement) LatestDecision(
	ctx context.Context,
	userID int32,
	agreementKey string,
	version string,
) (string, time.Time, bool, error) {
	var decision string
	var decidedAt time.Time

	err := r.pool.QueryRow(ctx, `
		SELECT decision, decided_at
		FROM user_agreement_decisions
		WHERE user_id = $1
		  AND agreement_key = $2
		  AND version = $3
		ORDER BY decided_at DESC, decision_id DESC
		LIMIT 1
	`, userID, agreementKey, version).Scan(&decision, &decidedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", time.Time{}, false, nil
	}
	if err != nil {
		return "", time.Time{}, false, fmt.Errorf("get latest agreement decision: %w", err)
	}

	return decision, decidedAt, true, nil
}

func (r *Agreement) IsAccepted(
	ctx context.Context,
	userID int32,
	agreementKey string,
	version string,
) (bool, error) {
	decision, _, found, err := r.LatestDecision(ctx, userID, agreementKey, version)
	if err != nil {
		return false, err
	}

	return found && decision == "accepted", nil
}
