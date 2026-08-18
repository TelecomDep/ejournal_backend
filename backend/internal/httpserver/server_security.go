package httpserver

import (
	"context"
	"crypto/subtle"
	"strings"
	"time"

	"github.com/TelecomDep/ejournal_backend/internal/app"
	"github.com/gofiber/fiber/v2"
)

func (s *Server) metricsAccessMiddleware(c *fiber.Ctx) error {
	if c.Path() != "/metrics" {
		return c.Next()
	}
	expected := strings.TrimSpace(s.cfg.MetricsToken)
	provided := c.Get("X-Metrics-Token")
	if expected == "" || subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) != 1 {
		return c.SendStatus(fiber.StatusNotFound)
	}
	return c.Next()
}

func maintenanceExempt(path string) bool {
	switch path {
	case "/healthz", "/login", "/metrics", "/internal/metrics",
		"/api/admin/system/maintenance":
		return true
	default:
		return false
	}
}

func (s *Server) maintenanceMiddleware(c *fiber.Ctx) error {
	if maintenanceExempt(c.Path()) || c.Method() == fiber.MethodOptions {
		return c.Next()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	status, err := s.svc.GetMaintenanceStatus(ctx)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(app.Response{
			OK: false, Error: "service status is temporarily unavailable",
		})
	}
	if !status.Enabled {
		return c.Next()
	}

	c.Set("Retry-After", "60")
	return c.Status(fiber.StatusServiceUnavailable).JSON(app.Response{
		OK: false,
		Error: func() string {
			if strings.TrimSpace(status.Message) == "" {
				return "service is under maintenance"
			}
			return status.Message
		}(),
	})
}
