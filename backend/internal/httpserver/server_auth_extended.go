package httpserver

import (
	"github.com/gofiber/fiber/v2"
	"github.com/TelecomDep/ejournal_backend/internal/app"
)

func (s *Server) generate2faHandler(c *fiber.Ctx) error {
	var body any
	return s.androidJSONActionHandler(c, "http-generate-2fa", "generate_2fa", &body)
}

func (s *Server) verify2faHandler(c *fiber.Ctx) error {
	var body app.TwoFaCodeData
	return s.androidJSONActionHandler(c, "http-verify-2fa", "verify_2fa", &body)
}

func (s *Server) disable2faHandler(c *fiber.Ctx) error {
	var body any
	return s.androidJSONActionHandler(c, "http-disable-2fa", "disable_2fa", &body)
}

func (s *Server) requestEmailBindHandler(c *fiber.Ctx) error {
	var body app.RequestEmailData
	return s.androidJSONActionHandler(c, "http-request-email-bind", "request_email_bind", &body)
}

func (s *Server) confirmEmailBindHandler(c *fiber.Ctx) error {
	var body app.ConfirmEmailData
	return s.androidJSONActionHandler(c, "http-confirm-email-bind", "confirm_email_bind", &body)
}
