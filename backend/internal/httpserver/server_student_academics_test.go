package httpserver

import (
	"testing"

	"github.com/TelecomDep/ejournal_backend/internal/app"
	"github.com/gofiber/fiber/v2"
)

func TestStudentAcademicReadHTTPStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		response   app.Response
		wantStatus int
	}{
		{name: "success", response: app.Response{OK: true}, wantStatus: fiber.StatusOK},
		{name: "invalid token", response: app.Response{Error: "invalid token"}, wantStatus: fiber.StatusUnauthorized},
		{name: "wrong role", response: app.Response{Error: "forbidden: teacher role required"}, wantStatus: fiber.StatusForbidden},
		{name: "semester not found", response: app.Response{Error: "semester not found"}, wantStatus: fiber.StatusNotFound},
		{name: "database failure", response: app.Response{Error: "failed to load attendance summary"}, wantStatus: fiber.StatusInternalServerError},
		{name: "invalid date", response: app.Response{Error: "invalid date format, expected YYYY-MM-DD"}, wantStatus: fiber.StatusBadRequest},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := studentAcademicReadHTTPStatus(test.response); got != test.wantStatus {
				t.Fatalf("studentAcademicReadHTTPStatus() = %d, want %d", got, test.wantStatus)
			}
		})
	}
}
