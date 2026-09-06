package app

import "testing"

func TestRegistrationAgreement(t *testing.T) {
	tests := []struct {
		name      string
		agreement *UserAgreementDecisionData
		provided  bool
		wantError string
	}{
		{
			name:     "omitted for legacy clients",
			provided: false,
		},
		{
			name: "accepts current version",
			agreement: &UserAgreementDecisionData{
				Version:  currentAgreementVersion,
				Decision: "accepted",
			},
			provided: true,
		},
		{
			name: "rejects missing version",
			agreement: &UserAgreementDecisionData{
				Decision: "accepted",
			},
			provided:  true,
			wantError: "agreement version is required",
		},
		{
			name: "rejects stale version",
			agreement: &UserAgreementDecisionData{
				Version:  "2026-08-01",
				Decision: "accepted",
			},
			provided:  true,
			wantError: "agreement version is not current",
		},
		{
			name: "rejects declined decision",
			agreement: &UserAgreementDecisionData{
				Version:  currentAgreementVersion,
				Decision: "declined",
			},
			provided:  true,
			wantError: "agreement must be accepted for registration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, provided, gotError := registrationAgreement(tt.agreement)
			if provided != tt.provided {
				t.Fatalf("provided = %v, want %v", provided, tt.provided)
			}
			if gotError != tt.wantError {
				t.Fatalf("error = %q, want %q", gotError, tt.wantError)
			}
			if gotError == "" && tt.provided && (got.Version != currentAgreementVersion || got.Decision != "accepted") {
				t.Fatalf("normalized agreement = %#v", got)
			}
		})
	}
}
