package app

import (
	"strings"
	"time"
)

const (
	userAgreementKey        = "user_agreement"
	currentAgreementVersion = "2026-08-01"
)

type UserAgreementDecisionData struct {
	Version  string `json:"version"`
	Decision string `json:"decision"`
}

type UserAgreementStatus struct {
	Agreement string     `json:"agreement"`
	Version   string     `json:"version"`
	Decision  string     `json:"decision,omitempty"`
	Accepted  bool       `json:"accepted"`
	DecidedAt *time.Time `json:"decided_at,omitempty"`
}

func (s *Service) recordUserAgreementDecision(
	token string,
	data UserAgreementDecisionData,
	meta RequestMeta,
) Response {
	user, err := s.userBySessionToken(token)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	version := strings.TrimSpace(data.Version)
	decision := strings.ToLower(strings.TrimSpace(data.Decision))

	if version == "" {
		return Response{OK: false, Error: "agreement version is required"}
	}
	if version != currentAgreementVersion {
		return Response{OK: false, Error: "agreement version is not current"}
	}
	if decision != "accepted" && decision != "declined" {
		return Response{OK: false, Error: "decision must be accepted or declined"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	decisionID, decidedAt, err := s.store.Agreement.RecordDecision(
		ctx,
		user.ID,
		userAgreementKey,
		version,
		decision,
		"",
		meta.IP,
		meta.UserAgent,
	)
	if err != nil {
		return Response{OK: false, Error: "failed to save agreement decision"}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"decision_id": decisionID,
			"agreement":   userAgreementKey,
			"version":     version,
			"decision":    decision,
			"accepted":    decision == "accepted",
			"decided_at":  decidedAt,
		},
	}
}

func (s *Service) currentUserAgreement(token string) Response {
	user, err := s.userBySessionToken(token)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	decision, decidedAt, found, err := s.store.Agreement.LatestDecision(
		ctx,
		user.ID,
		userAgreementKey,
		currentAgreementVersion,
	)
	if err != nil {
		return Response{OK: false, Error: "failed to check agreement decision"}
	}

	status := UserAgreementStatus{
		Agreement: userAgreementKey,
		Version:   currentAgreementVersion,
		Accepted:  found && decision == "accepted",
	}
	if found {
		status.Decision = decision
		status.DecidedAt = &decidedAt
	}

	return Response{OK: true, Result: status}
}
