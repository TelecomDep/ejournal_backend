package httpserver

import (
	"testing"

	"github.com/TelecomDep/ejournal_backend/internal/app"
	"github.com/gofiber/fiber/v2"
)

func TestAdminUserHTTPStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		errorText string
		want      int
	}{
		{errorText: "invalid token", want: fiber.StatusUnauthorized},
		{errorText: "forbidden: admin role required", want: fiber.StatusForbidden},
		{errorText: "user not found", want: fiber.StatusNotFound},
		{errorText: "cannot disable the last active admin", want: fiber.StatusConflict},
		{errorText: "invalid user role", want: fiber.StatusBadRequest},
	}

	for _, test := range tests {
		test := test
		t.Run(test.errorText, func(t *testing.T) {
			t.Parallel()
			got := adminUserHTTPStatus(app.Response{OK: false, Error: test.errorText})
			if got != test.want {
				t.Fatalf("adminUserHTTPStatus() = %d, want %d", got, test.want)
			}
		})
	}
}
