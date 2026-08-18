package httpserver

import (
	"github.com/TelecomDep/ejournal_backend/internal/app"
	"github.com/gofiber/fiber/v2"
)

const generalRatingRequestID = "http-general-rating"

// generalRatingResponse documents the standard response envelope while keeping
// the concrete result schema visible in Swagger.
type generalRatingResponse struct {
	ID     string                   `json:"id" example:"http-general-rating"`
	OK     bool                     `json:"ok" example:"true"`
	Result app.GeneralRatingPayload `json:"result"`
	Error  string                   `json:"error" example:""`
}

// generalRatingHandler godoc
// @Summary Get source data for the common student rating
// @Description Returns semester metadata, role-scoped departments, subjects, groups, student consent status, attendance, laboratory/practice grades, and per-subject summaries in the standard response envelope.
// @Tags staff
// @Produce json
// @Security BearerAuth
// @Param semester_id query int false "Semester ID; defaults to the open semester"
// @Param page query int false "Group page; defaults to 1"
// @Param page_size query int false "Groups per page; defaults to 20, maximum 50"
// @Success 200 {object} generalRatingResponse
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 404 {object} app.Response
// @Failure 500 {object} app.Response
// @Router /api/staff/ratings/general [get]
func (s *Server) generalRatingHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(app.Response{ID: generalRatingRequestID, OK: false, Error: "missing Authorization header"})
	}

	semesterID, err := optionalSemesterIDFromQuery(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{ID: generalRatingRequestID, OK: false, Error: err.Error()})
	}

	payload, resp := s.svc.GeneralRating(token, semesterID, int32(c.QueryInt("page", 1)), int32(c.QueryInt("page_size", 20)))
	resp.ID = generalRatingRequestID
	if payload == nil {
		return c.Status(generalRatingHTTPStatus(resp)).JSON(resp)
	}

	resp.OK = true
	resp.Result = payload
	resp.Error = ""
	return c.JSON(resp)
}

func generalRatingHTTPStatus(resp app.Response) int {
	switch resp.Error {
	case "invalid token", "session not found", "session revoked", "missing token", "account is not active":
		return fiber.StatusUnauthorized
	case "forbidden: staff role required":
		return fiber.StatusForbidden
	case "semester not found", "open semester not found":
		return fiber.StatusNotFound
	default:
		return fiber.StatusInternalServerError
	}
}
