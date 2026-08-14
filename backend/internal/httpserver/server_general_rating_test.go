package httpserver

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/TelecomDep/ejournal_backend/internal/app"
	"github.com/gofiber/fiber/v2"
)

func TestGeneralRatingHTTPStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		errorText string
		want      int
	}{
		{errorText: "invalid token", want: fiber.StatusUnauthorized},
		{errorText: "forbidden: staff role required", want: fiber.StatusForbidden},
		{errorText: "semester not found", want: fiber.StatusNotFound},
		{errorText: "failed to load rating grades", want: fiber.StatusInternalServerError},
	}

	for _, test := range tests {
		test := test
		t.Run(test.errorText, func(t *testing.T) {
			t.Parallel()
			got := generalRatingHTTPStatus(app.Response{OK: false, Error: test.errorText})
			if got != test.want {
				t.Fatalf("generalRatingHTTPStatus() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestGeneralRatingUsesStandardResponseEnvelope(t *testing.T) {
	t.Parallel()

	response := app.Response{
		ID: generalRatingRequestID,
		OK: true,
		Result: &app.GeneralRatingPayload{
			SchemaVersion: "1.0",
			Departments:   make([]app.GeneralRatingDepartment, 0),
			Subjects:      make([]app.GeneralRatingSubject, 0),
			Groups:        make([]app.GeneralRatingGroup, 0),
		},
	}
	raw, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	got := string(raw)
	for _, fragment := range []string{
		`"id":"http-general-rating"`,
		`"ok":true`,
		`"result":{"schema_version":"1.0"`,
		`"error":""`,
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("marshalled response does not contain %s: %s", fragment, got)
		}
	}
}
