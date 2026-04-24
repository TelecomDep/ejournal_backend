package httpserver

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	_ "github.com/TelecomDep/ejournal_backend/docs"
	"github.com/TelecomDep/ejournal_backend/internal/app"
	"github.com/TelecomDep/ejournal_backend/internal/config"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	swagger "github.com/gofiber/swagger"
)

type authBody struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	Role     string `json:"role,omitempty"`
	RoleHash string `json:"role_hash,omitempty"`
}

type Server struct {
	cfg            config.AppConfig
	svc            *app.Service
	requestTimeout time.Duration
}

func New(cfg config.AppConfig, svc *app.Service) *Server {
	return &Server{
		cfg:            cfg,
		svc:            svc,
		requestTimeout: 3 * time.Second,
	}
}

func (s *Server) Start() {
	fiberApp := fiber.New()

	fiberApp.Use(cors.New(cors.Config{
		AllowOrigins: s.cfg.CORSAllowOrigins,
		AllowMethods: "GET,POST,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	fiberApp.Post("/register", s.registerHandler)
	fiberApp.Post("/register/by-invite", s.registerByInviteHandler)
	fiberApp.Post("/login", s.loginHandler)
	fiberApp.Get("/profile", s.profileHandler)

	fiberApp.Post("/api/teacher/attendance-link", s.teacherAttendanceLinkHandler)
	fiberApp.Post("/api/teacher/attendance/session", s.teacherAttendanceLinkHandler)
	fiberApp.Post("/api/teacher/attendance/group", s.teacherAttendanceByGroupHandler)
	fiberApp.Post("/api/student/attendance/confirm", s.studentAttendanceConfirmHandler)
	fiberApp.Get("/swagger/*", swagger.HandlerDefault)

	addr := fmt.Sprintf(":%s", s.cfg.AppPort)
	log.Printf("Starting HTTP server on %s", addr)
	if err := fiberApp.Listen(addr); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}

// registerHandler godoc
// @Summary Register user
// @Description Registers a user by login/password and role.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body app.LoginData true "Register payload"
// @Success 200 {object} registerResponse
// @Failure 400 {object} app.Response
// @Failure 500 {object} app.Response
// @Router /register [post]
func (s *Server) registerHandler(c *fiber.Ctx) error {
	var body authBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "Error parsing body"})
	}

	data, err := json.Marshal(body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling request"})
	}

	req := app.Request{ID: "http-register", Action: "register", Data: data}
	raw, err := json.Marshal(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling envelope"})
	}

	resp, err := s.svc.DispatchRequest(string(raw), s.requestTimeout)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(app.Response{OK: false, Error: err.Error()})
	}

	if !resp.OK {
		return c.Status(fiber.StatusBadRequest).JSON(resp)
	}

	return c.JSON(resp)
}

// loginHandler godoc
// @Summary Login user
// @Description Authenticates a user and returns JWT token.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body app.LoginData true "Login payload"
// @Success 200 {object} loginResponse
// @Failure 401 {object} app.Response
// @Failure 500 {object} app.Response
// @Router /login [post]
func (s *Server) loginHandler(c *fiber.Ctx) error {
	var body authBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "Error parsing body"})
	}

	data, err := json.Marshal(body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling request"})
	}

	req := app.Request{ID: "http-login", Action: "login", Data: data}
	raw, err := json.Marshal(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling envelope"})
	}

	resp, err := s.svc.DispatchRequest(string(raw), s.requestTimeout)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(app.Response{OK: false, Error: err.Error()})
	}

	if !resp.OK {
		return c.Status(fiber.StatusUnauthorized).JSON(resp)
	}

	return c.JSON(resp)
}

// registerByInviteHandler godoc
// @Summary Register student by invite code
// @Description Creates student account by one-time invite code from database.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body app.RegisterByInviteData true "Register by invite payload"
// @Success 200 {object} registerByInviteResponse
// @Failure 400 {object} app.Response
// @Failure 409 {object} app.Response
// @Failure 500 {object} app.Response
// @Router /register/by-invite [post]
func (s *Server) registerByInviteHandler(c *fiber.Ctx) error {
	var body app.RegisterByInviteData
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "Error parsing body"})
	}

	data, err := json.Marshal(body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling request"})
	}

	req := app.Request{ID: "http-register-by-invite", Action: "register_by_invite", Data: data}
	raw, err := json.Marshal(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling envelope"})
	}

	resp, err := s.svc.DispatchRequest(string(raw), s.requestTimeout)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(app.Response{OK: false, Error: err.Error()})
	}

	if !resp.OK {
		if resp.Error == "user exist" {
			return c.Status(fiber.StatusConflict).JSON(resp)
		}
		return c.Status(fiber.StatusBadRequest).JSON(resp)
	}

	return c.JSON(resp)
}

// profileHandler godoc
// @Summary Get user profile
// @Description Returns current user profile from Authorization token.
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} profileResponse
// @Failure 401 {object} app.Response
// @Failure 500 {object} app.Response
// @Router /profile [get]
func (s *Server) profileHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(app.Response{OK: false, Error: "missing Authorization header"})
	}

	req := app.Request{ID: "http-profile", Action: "profile", Token: token}
	raw, err := json.Marshal(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling envelope"})
	}

	resp, err := s.svc.DispatchRequest(string(raw), s.requestTimeout)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(app.Response{OK: false, Error: err.Error()})
	}

	if !resp.OK {
		return c.Status(fiber.StatusUnauthorized).JSON(resp)
	}

	return c.JSON(resp)
}

// teacherAttendanceLinkHandler godoc
// @Summary Create attendance session link
// @Description Teacher creates attendance session and gets invite/join URL. If subject_id/group_ids are omitted, they are taken from nearest scheduled lesson.
// @Tags attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body teacherAttendanceLinkRequest true "Attendance session payload (subject_id/group_ids are optional)"
// @Success 200 {object} teacherAttendanceLinkResponse
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 500 {object} app.Response
// @Router /api/teacher/attendance-link [post]
// @Router /api/teacher/attendance/session [post]
func (s *Server) teacherAttendanceLinkHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(app.Response{OK: false, Error: "missing Authorization header"})
	}

	var body app.AttendanceCreateData
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "Error parsing body"})
	}

	data, err := json.Marshal(body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling request"})
	}

	req := app.Request{ID: "http-attendance-link", Action: "create_attendance_link", Token: token, Data: data}
	raw, err := json.Marshal(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling envelope"})
	}

	resp, err := s.svc.DispatchRequest(string(raw), s.requestTimeout)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(app.Response{OK: false, Error: err.Error()})
	}

	if !resp.OK {
		if resp.Error == "forbidden: teacher role required" {
			return c.Status(fiber.StatusForbidden).JSON(resp)
		}
		if resp.Error == "invalid token" || resp.Error == "session not found" || resp.Error == "missing token" {
			return c.Status(fiber.StatusUnauthorized).JSON(resp)
		}
		return c.Status(fiber.StatusBadRequest).JSON(resp)
	}

	return c.JSON(resp)
}

// studentAttendanceConfirmHandler godoc
// @Summary Confirm attendance by invite token
// @Description Student confirms attendance for active session.
// @Tags attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body app.AttendanceConfirmData true "Attendance confirm payload"
// @Success 200 {object} studentAttendanceConfirmResponse
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 409 {object} app.Response
// @Failure 500 {object} app.Response
// @Router /api/student/attendance/confirm [post]
func (s *Server) studentAttendanceConfirmHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(app.Response{OK: false, Error: "missing Authorization header"})
	}

	var body app.AttendanceConfirmData
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "Error parsing body"})
	}

	data, err := json.Marshal(body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling request"})
	}

	req := app.Request{ID: "http-attendance-confirm", Action: "confirm_attendance", Token: token, Data: data}
	raw, err := json.Marshal(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling envelope"})
	}

	resp, err := s.svc.DispatchRequest(string(raw), s.requestTimeout)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(app.Response{OK: false, Error: err.Error()})
	}

	if !resp.OK {
		if resp.Error == "forbidden: student role required" {
			return c.Status(fiber.StatusForbidden).JSON(resp)
		}
		if resp.Error == "forbidden: student is not in session roster" {
			return c.Status(fiber.StatusForbidden).JSON(resp)
		}
		if resp.Error == "invalid token" || resp.Error == "session not found" || resp.Error == "missing token" {
			return c.Status(fiber.StatusUnauthorized).JSON(resp)
		}
		if resp.Error == "attendance already confirmed" {
			return c.Status(fiber.StatusConflict).JSON(resp)
		}
		return c.Status(fiber.StatusBadRequest).JSON(resp)
	}

	return c.JSON(resp)
}

// teacherAttendanceByGroupHandler godoc
// @Summary Get attendance stats by group
// @Description Returns per-student attendance stats for selected group and optional subject.
// @Tags attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body app.AttendanceGroupStatsData true "Group stats payload"
// @Success 200 {object} teacherAttendanceGroupResponse
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 500 {object} app.Response
// @Router /api/teacher/attendance/group [post]
func (s *Server) teacherAttendanceByGroupHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(app.Response{OK: false, Error: "missing Authorization header"})
	}

	var body app.AttendanceGroupStatsData
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "Error parsing body"})
	}

	data, err := json.Marshal(body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling request"})
	}

	req := app.Request{ID: "http-teacher-attendance-group", Action: "teacher_attendance_by_group", Token: token, Data: data}
	raw, err := json.Marshal(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling envelope"})
	}

	resp, err := s.svc.DispatchRequest(string(raw), s.requestTimeout)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(app.Response{OK: false, Error: err.Error()})
	}

	if !resp.OK {
		if resp.Error == "forbidden: teacher role required" {
			return c.Status(fiber.StatusForbidden).JSON(resp)
		}
		if resp.Error == "invalid token" || resp.Error == "session not found" || resp.Error == "missing token" {
			return c.Status(fiber.StatusUnauthorized).JSON(resp)
		}
		return c.Status(fiber.StatusBadRequest).JSON(resp)
	}

	return c.JSON(resp)
}
