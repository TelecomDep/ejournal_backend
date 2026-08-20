package httpserver

import (
	"encoding/json"

	"github.com/TelecomDep/ejournal_backend/internal/app"
	"github.com/gofiber/fiber/v2"
)

// adminCatalogsHandler godoc
// @Summary List entity catalogs for administrative pickers (groups, lecterns, faculties, teachers, students)
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Success 200 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Router /api/admin/catalogs [get]
func (s *Server) adminCatalogsHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(app.Response{OK: false, Error: "missing Authorization header"})
	}

	req := app.Request{ID: "http-admin-catalogs", Action: "admin_list_catalogs", Token: token}
	raw, err := json.Marshal(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling envelope"})
	}

	resp, err := s.svc.DispatchRequest(string(raw), s.requestTimeout)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(app.Response{OK: false, Error: err.Error()})
	}

	if !resp.OK {
		status := fiber.StatusUnauthorized
		if resp.Error == "forbidden" {
			status = fiber.StatusForbidden
		}
		return c.Status(status).JSON(resp)
	}

	return c.JSON(resp)
}
