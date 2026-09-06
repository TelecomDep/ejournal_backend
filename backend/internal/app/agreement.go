package app

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

const (
	userAgreementKey        = "user_agreement"
	currentAgreementVersion = "2026-09-01"
	currentAgreementText    = "Я даю согласие на обработку персональных данных"
)

func currentAgreementDocumentHash() string {
	sum := sha256.Sum256([]byte(userAgreementKey + "\n" + currentAgreementVersion + "\n" + currentAgreementText))
	return hex.EncodeToString(sum[:])
}

type UserAgreementDecisionData struct {
	Version  string `json:"version"`
	Decision string `json:"decision"`
}

func requestMeta(metas []RequestMeta) RequestMeta {
	if len(metas) == 0 {
		return RequestMeta{}
	}
	return metas[0]
}

// registrationAgreement validates the optional consent accepted by
// registration endpoints. Existing mobile clients do not send consent yet,
// so an entirely omitted agreement remains a legacy-compatible request. Once
// it is present, however, registration requires an explicit acceptance of the
// current document version.
func registrationAgreement(
	agreement *UserAgreementDecisionData,
) (UserAgreementDecisionData, bool, string) {
	if agreement == nil {
		return UserAgreementDecisionData{}, false, ""
	}
	version := strings.TrimSpace(agreement.Version)
	decision := strings.ToLower(strings.TrimSpace(agreement.Decision))
	if version == "" {
		return UserAgreementDecisionData{}, true, "agreement version is required"
	}
	if version != currentAgreementVersion {
		return UserAgreementDecisionData{}, true, "agreement version is not current"
	}
	if decision == "" {
		return UserAgreementDecisionData{}, true, "agreement decision is required"
	}
	if decision != "accepted" && decision != "declined" {
		return UserAgreementDecisionData{}, true, "decision must be accepted or declined"
	}
	if decision != "accepted" {
		return UserAgreementDecisionData{}, true, "agreement must be accepted for registration"
	}

	return UserAgreementDecisionData{Version: version, Decision: decision}, true, ""
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
		currentAgreementDocumentHash(),
		user.Login,
		user.Role,
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
