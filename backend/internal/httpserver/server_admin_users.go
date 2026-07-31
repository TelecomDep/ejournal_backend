package httpserver

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/TelecomDep/ejournal_backend/internal/app"
	"github.com/gofiber/fiber/v2"
)

// adminUsersListHandler godoc
// @Summary List users for admin
// @Description Returns users with pagination, search and filters. Admin only.
// @Tags admin users
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page"
// @Param page_size query int false "Page size"
// @Param search query string false "Search by login or email"
// @Param role query string false "Role"
// @Param status query string false "Status"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Router /api/admin/users [get]
func (s *Server) adminUsersListHandler(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	page_size, _ := strconv.Atoi(c.Query("page_size", "20"))
	data := app.AdminUsersListData{
		Page:     int32(page),
		PageSize: int32(page_size),
		Search:   strings.TrimSpace(c.Query("search")),
		Role:     strings.TrimSpace(c.Query("role")),
		Status:   strings.TrimSpace(c.Query("status")),
	}
	return s.adminUserActionHandler(c, "http-admin-users-list", "admin_users_list", data, fiber.StatusOK)
}

// adminUserGetHandler godoc
// @Summary Get user for admin
// @Description Returns one user. Admin only.
// @Tags admin users
// @Produce json
// @Security BearerAuth
// @Param user_id path int true "User ID"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 404 {object} app.Response
// @Router /api/admin/users/{user_id} [get]
func (s *Server) adminUserGetHandler(c *fiber.Ctx) error {
	user_id, err := c.ParamsInt("user_id")
	if err != nil || user_id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "invalid user_id"})
	}
	data := app.AdminUserIDData{UserID: int32(user_id)}
	return s.adminUserActionHandler(c, "http-admin-user-get", "admin_user_get", data, fiber.StatusOK)
}

// adminUserCreateHandler godoc
// @Summary Create user
// @Description Creates a user and the required role profile in one transaction. Admin only.
// @Tags admin users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body app.AdminUserCreateData true "User payload"
// @Success 201 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 409 {object} app.Response
// @Router /api/admin/users [post]
func (s *Server) adminUserCreateHandler(c *fiber.Ctx) error {
	var data app.AdminUserCreateData
	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "invalid request body"})
	}
	return s.adminUserActionHandler(c, "http-admin-user-create", "admin_user_create", data, fiber.StatusCreated)
}

// adminUserUpdateHandler godoc
// @Summary Update user
// @Description Updates user fields, status or role. Admin only.
// @Tags admin users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param user_id path int true "User ID"
// @Param request body app.AdminUserUpdateData true "User payload"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 404 {object} app.Response
// @Failure 409 {object} app.Response
// @Router /api/admin/users/{user_id} [patch]
func (s *Server) adminUserUpdateHandler(c *fiber.Ctx) error {
	user_id, err := c.ParamsInt("user_id")
	if err != nil || user_id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "invalid user_id"})
	}

	var data app.AdminUserUpdateData
	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "invalid request body"})
	}
	data.UserID = int32(user_id)
	return s.adminUserActionHandler(c, "http-admin-user-update", "admin_user_update", data, fiber.StatusOK)
}

// adminUserDeleteHandler godoc
// @Summary Archive user
// @Description Soft deletes a user by setting status=archived. Admin only.
// @Tags admin users
// @Produce json
// @Security BearerAuth
// @Param user_id path int true "User ID"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 404 {object} app.Response
// @Failure 409 {object} app.Response
// @Router /api/admin/users/{user_id} [delete]
func (s *Server) adminUserDeleteHandler(c *fiber.Ctx) error {
	user_id, err := c.ParamsInt("user_id")
	if err != nil || user_id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "invalid user_id"})
	}
	data := app.AdminUserIDData{UserID: int32(user_id)}
	return s.adminUserActionHandler(c, "http-admin-user-delete", "admin_user_delete", data, fiber.StatusOK)
}

func (s *Server) adminUserActionHandler(c *fiber.Ctx, request_id, action string, data any, success_status int) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(app.Response{OK: false, Error: "missing Authorization header"})
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "failed to prepare request"})
	}
	req := app.Request{ID: request_id, Action: action, Token: token, Data: payload}
	raw, err := json.Marshal(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "failed to prepare request"})
	}

	resp, err := s.svc.DispatchRequest(string(raw), s.requestTimeout)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(app.Response{OK: false, Error: err.Error()})
	}
	if !resp.OK {
		return c.Status(adminUserHTTPStatus(resp)).JSON(resp)
	}
	return c.Status(success_status).JSON(resp)
}

func adminUserHTTPStatus(resp app.Response) int {
	if resp.OK {
		return fiber.StatusOK
	}

	switch resp.Error {
	case "missing token", "invalid token", "session not found", "account is not active":
		return fiber.StatusUnauthorized
	case "forbidden: admin role required",
		"admin cannot change own role or status",
		"admin cannot archive own account":
		return fiber.StatusForbidden
	case "user not found":
		return fiber.StatusNotFound
	case "login, email or profile is already used",
		"student profile not found or already used",
		"teacher profile not found or already used",
		"cannot disable the last active admin",
		"user is already archived":
		return fiber.StatusConflict
	default:
		return fiber.StatusBadRequest
	}
}
