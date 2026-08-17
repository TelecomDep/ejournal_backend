package httpserver

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/TelecomDep/ejournal_backend/internal/app"
	"github.com/gofiber/fiber/v2"
)

// notificationsListHandler godoc
// @Summary List current user notifications
// @Tags notifications
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page"
// @Param page_size query int false "Page size"
// @Param category query string false "grades, schedule, attendance or system"
// @Param unread_only query bool false "Only unread notifications"
// @Success 200 {object} app.Response
// @Router /api/user/notifications [get]
func (s *Server) notificationsListHandler(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	page_size, _ := strconv.Atoi(c.Query("page_size", "20"))

	unread_only := strings.EqualFold(
		strings.TrimSpace(c.Query("unread_only")),
		"true",
	)

	data := app.NotificationsListData{
		Page:       int32(page),
		PageSize:   int32(page_size),
		Category:   strings.TrimSpace(c.Query("category")),
		UnreadOnly: unread_only,
	}

	return s.notificationActionHandler(
		c,
		"http-notifications-list",
		"notifications_list",
		data,
		fiber.StatusOK,
	)
}

// notificationsUnreadCountHandler godoc
// @Summary Get unread notifications count
// @Tags notifications
// @Produce json
// @Security BearerAuth
// @Success 200 {object} app.Response
// @Router /api/user/notifications/unread-count [get]
func (s *Server) notificationsUnreadCountHandler(c *fiber.Ctx) error {
	return s.notificationActionHandler(
		c,
		"http-notifications-unread-count",
		"notifications_unread_count",
		map[string]any{},
		fiber.StatusOK,
	)
}

// notificationMarkReadHandler godoc
// @Summary Mark notification as read
// @Tags notifications
// @Produce json
// @Security BearerAuth
// @Param notification_id path int true "Notification ID"
// @Success 200 {object} app.Response
// @Router /api/user/notifications/{notification_id}/read [patch]
func (s *Server) notificationMarkReadHandler(c *fiber.Ctx) error {
	notification_id, err := strconv.ParseInt(
		c.Params("notification_id"),
		10,
		64,
	)

	if err != nil || notification_id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(
			app.Response{OK: false, Error: "invalid notification_id"},
		)
	}

	data := app.NotificationIDData{
		NotificationID: notification_id,
	}

	return s.notificationActionHandler(
		c,
		"http-notification-mark-read",
		"notification_mark_read",
		data,
		fiber.StatusOK,
	)
}

func (s *Server) notificationDeleteHandler(c *fiber.Ctx) error {
	notification_id, err := strconv.ParseInt(
		c.Params("notification_id"),
		10,
		64,
	)

	if err != nil || notification_id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(
			app.Response{OK: false, Error: "invalid notification_id"},
		)
	}

	data := app.NotificationIDData{
		NotificationID: notification_id,
	}

	return s.notificationActionHandler(
		c,
		"http-notification-delete",
		"notification_delete",
		data,
		fiber.StatusOK,
	)
}

// notificationsMarkAllReadHandler godoc
// @Summary Mark all notifications as read
// @Tags notifications
// @Produce json
// @Security BearerAuth
// @Success 200 {object} app.Response
// @Router /api/user/notifications/read-all [patch]
func (s *Server) notificationsMarkAllReadHandler(c *fiber.Ctx) error {
	return s.notificationActionHandler(
		c,
		"http-notifications-mark-all-read",
		"notifications_mark_all_read",
		map[string]any{},
		fiber.StatusOK,
	)
}

// notificationSettingsGetHandler godoc
// @Summary Get notification settings
// @Tags notifications
// @Produce json
// @Security BearerAuth
// @Success 200 {object} app.Response
// @Router /api/user/notification-settings [get]
func (s *Server) notificationSettingsGetHandler(c *fiber.Ctx) error {
	return s.notificationActionHandler(
		c,
		"http-notification-settings-get",
		"notification_settings_get",
		map[string]any{},
		fiber.StatusOK,
	)
}

// notificationSettingsUpdateHandler godoc
// @Summary Update notification settings
// @Tags notifications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body app.NotificationSettingsData true "Notification settings"
// @Success 200 {object} app.Response
// @Router /api/user/notification-settings [put]
func (s *Server) notificationSettingsUpdateHandler(c *fiber.Ctx) error {
	var data app.NotificationSettingsData

	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			app.Response{OK: false, Error: "invalid request body"},
		)
	}

	return s.notificationActionHandler(
		c,
		"http-notification-settings-update",
		"notification_settings_update",
		data,
		fiber.StatusOK,
	)
}

// adminNotificationsCreateHandler godoc
// @Summary Create admin update notification
// @Tags admin notifications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body app.AdminNotificationCreateData true "Notification payload"
// @Success 201 {object} app.Response
// @Router /api/admin/notifications [post]
func (s *Server) adminNotificationsCreateHandler(c *fiber.Ctx) error {
	var data app.AdminNotificationCreateData

	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			app.Response{OK: false, Error: "invalid request body"},
		)
	}

	return s.notificationActionHandler(
		c,
		"http-admin-notifications-create",
		"admin_notifications_create",
		data,
		fiber.StatusCreated,
	)
}

// adminNotificationsListHandler godoc
// @Summary List notifications for admin
// @Tags admin notifications
// @Produce json
// @Security BearerAuth
// @Success 200 {object} app.Response
// @Router /api/admin/notifications [get]
func (s *Server) adminNotificationsListHandler(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	page_size, _ := strconv.Atoi(c.Query("page_size", "20"))

	data := app.NotificationsListData{
		Page:     int32(page),
		PageSize: int32(page_size),
		Category: strings.TrimSpace(c.Query("category")),
	}

	return s.notificationActionHandler(
		c,
		"http-admin-notifications-list",
		"admin_notifications_list",
		data,
		fiber.StatusOK,
	)
}

// adminNotificationsUpdateHandler godoc
// @Summary Update admin notification
// @Tags admin notifications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param notification_id path int true "Notification ID"
// @Param request body app.AdminNotificationUpdateData true "Notification payload"
// @Success 200 {object} app.Response
// @Router /api/admin/notifications/{notification_id} [patch]
func (s *Server) adminNotificationsUpdateHandler(c *fiber.Ctx) error {
	notification_id, err := strconv.ParseInt(
		c.Params("notification_id"),
		10,
		64,
	)

	if err != nil || notification_id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(
			app.Response{OK: false, Error: "invalid notification_id"},
		)
	}

	var data app.AdminNotificationUpdateData

	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(
			app.Response{OK: false, Error: "invalid request body"},
		)
	}

	data.NotificationID = notification_id

	return s.notificationActionHandler(
		c,
		"http-admin-notifications-update",
		"admin_notifications_update",
		data,
		fiber.StatusOK,
	)
}

// adminNotificationsDeleteHandler godoc
// @Summary Delete admin notification
// @Tags admin notifications
// @Produce json
// @Security BearerAuth
// @Param notification_id path int true "Notification ID"
// @Success 200 {object} app.Response
// @Router /api/admin/notifications/{notification_id} [delete]
func (s *Server) adminNotificationsDeleteHandler(c *fiber.Ctx) error {
	notification_id, err := strconv.ParseInt(
		c.Params("notification_id"),
		10,
		64,
	)

	if err != nil || notification_id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(
			app.Response{OK: false, Error: "invalid notification_id"},
		)
	}

	data := app.NotificationIDData{
		NotificationID: notification_id,
	}

	return s.notificationActionHandler(
		c,
		"http-admin-notifications-delete",
		"admin_notifications_delete",
		data,
		fiber.StatusOK,
	)
}

func (s *Server) notificationActionHandler(
	c *fiber.Ctx,
	request_id string,
	action string,
	data any,
	success_status int,
) error {
	token := c.Get("Authorization")

	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(
			app.Response{OK: false, Error: "missing Authorization header"},
		)
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			app.Response{OK: false, Error: "failed to prepare request"},
		)
	}

	req := app.Request{
		ID:     request_id,
		Action: action,
		Token:  token,
		Data:   payload,
	}

	raw, err := json.Marshal(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(
			app.Response{OK: false, Error: "failed to prepare request"},
		)
	}

	resp, err := s.svc.DispatchRequest(string(raw), s.requestTimeout)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(
			app.Response{OK: false, Error: err.Error()},
		)
	}

	if !resp.OK {
		return c.Status(notificationHTTPStatus(resp)).JSON(resp)
	}

	return c.Status(success_status).JSON(resp)
}

func notificationHTTPStatus(resp app.Response) int {
	switch resp.Error {
	case "missing token",
		"invalid token",
		"session not found",
		"account is not active":
		return fiber.StatusUnauthorized

	case "forbidden: admin role required":
		return fiber.StatusForbidden

	case "notification not found":
		return fiber.StatusNotFound

	default:
		return fiber.StatusBadRequest
	}
}
