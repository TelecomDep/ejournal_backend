package httpserver

import (
	"github.com/TelecomDep/ejournal_backend/internal/app"
	"github.com/gofiber/fiber/v2"
)

// request2FAEnableHandler godoc
// @Summary Request 2FA enablement code
// @Description Sends a confirmation code to user's bound email before allowing 2FA QR code generation.
// @Tags profile
// @Produce json
// @Security BearerAuth
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Router /api/user/2fa/request-enable [post]
func (s *Server) request2FAEnableHandler(c *fiber.Ctx) error {
	var body any
	return s.androidJSONActionHandler(c, "http-request-2fa-enable", "request_2fa_enable", &body)
}

// generate2faHandler godoc
// @Summary Generate 2FA setup
// @Description Generates a TOTP secret and QR code for the current user after validating email confirmation code.
// @Tags profile
// @Produce json
// @Security BearerAuth
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Router /api/user/2fa/generate [post]
func (s *Server) generate2faHandler(c *fiber.Ctx) error {
	var body app.Generate2FAData
	return s.androidJSONActionHandler(c, "http-generate-2fa", "generate_2fa", &body)
}

// verify2faHandler godoc
// @Summary Verify and enable 2FA
// @Description Verifies a TOTP code and enables 2FA for the current user.
// @Tags profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body app.TwoFaCodeData true "TOTP code"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Router /api/user/2fa/verify [post]
func (s *Server) verify2faHandler(c *fiber.Ctx) error {
	var body app.TwoFaCodeData
	return s.androidJSONActionHandler(c, "http-verify-2fa", "verify_2fa", &body)
}

// disable2faHandler godoc
// @Summary Disable 2FA
// @Description Disables 2FA and removes the stored TOTP secret for the current user.
// @Tags profile
// @Produce json
// @Security BearerAuth
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Router /api/user/2fa/disable [post]
func (s *Server) disable2faHandler(c *fiber.Ctx) error {
	var body any
	return s.androidJSONActionHandler(c, "http-disable-2fa", "disable_2fa", &body)
}

// requestEmailBindHandler godoc
// @Summary Request email binding
// @Description Sends a confirmation code to the requested email address.
// @Tags profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body app.RequestEmailData true "Email address"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Router /api/user/email/bind/request [post]
func (s *Server) requestEmailBindHandler(c *fiber.Ctx) error {
	var body app.RequestEmailData
	return s.androidJSONActionHandler(c, "http-request-email-bind", "request_email_bind", &body)
}

// confirmEmailBindHandler godoc
// @Summary Confirm email binding
// @Description Confirms the email address using the code sent to the user.
// @Tags profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body app.ConfirmEmailData true "Confirmation code"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Router /api/user/email/bind/confirm [post]
func (s *Server) confirmEmailBindHandler(c *fiber.Ctx) error {
	var body app.ConfirmEmailData
	return s.androidJSONActionHandler(c, "http-confirm-email-bind", "confirm_email_bind", &body)
}
