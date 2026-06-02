package httpserver

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/TelecomDep/ejournal_backend/docs"
	"github.com/TelecomDep/ejournal_backend/internal/app"
	"github.com/TelecomDep/ejournal_backend/internal/config"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	swagger "github.com/gofiber/swagger"
)

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
	fiberApp.Post("/lessons/create", s.androidLessonCreateHandler)

	fiberApp.Post("/api/teacher/attendance-link", s.teacherAttendanceLinkHandler)
	fiberApp.Post("/api/teacher/attendance/session", s.teacherAttendanceLinkHandler)
	fiberApp.Get("/api/teacher/subjects", s.teacherSubjectsHandler)
	fiberApp.Post("/api/teacher/attendance/group", s.teacherAttendanceByGroupHandler)
	fiberApp.Post("/api/student/attendance/confirm", s.studentAttendanceConfirmHandler)
	fiberApp.Post("/api/student/mark-attendance", s.androidStudentAttendanceMarkHandler)
	fiberApp.Get("/api/student/attendance/history", s.studentAttendanceHistoryHandler)
	fiberApp.Post("/api/user/upload-avatar", s.uploadAvatarHandler)
	fiberApp.Post("/api/teacher/grades/items", s.teacherCreateGradeItemHandler)
	fiberApp.Post("/api/teacher/grades/items/list", s.teacherGradeItemsBySubjectHandler)
	fiberApp.Post("/api/teacher/grades", s.teacherUpsertGradeHandler)
	fiberApp.Post("/api/teacher/grades/student", s.teacherStudentGradesBySubjectHandler)
	fiberApp.Post("/api/student/grades", s.studentGradesBySubjectHandler)
	fiberApp.Static("/uploads", s.cfg.UploadDir)
	fiberApp.Get("/swagger/*", swagger.HandlerDefault)

	addr := fmt.Sprintf(":%s", s.cfg.AppPort)
	log.Printf("Starting HTTP server on %s", addr)
	if err := fiberApp.Listen(addr); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}

// androidLessonCreateHandler godoc
// @Summary Create lesson for Android client
// @Description Teacher creates an attendance lesson using subject/group names or IDs and current location.
// @Tags attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body app.AndroidLessonCreateData true "Android lesson payload"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Router /lessons/create [post]
func (s *Server) androidLessonCreateHandler(c *fiber.Ctx) error {
	var body app.AndroidLessonCreateData
	return s.androidJSONActionHandler(c, "http-android-lesson-create", "create_android_lesson", &body)
}

// androidStudentAttendanceMarkHandler godoc
// @Summary Mark attendance for Android client
// @Description Student marks attendance with lesson ID, device ID, and current location.
// @Tags attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body app.AndroidAttendanceMarkData true "Android attendance payload"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 409 {object} app.Response
// @Router /api/student/mark-attendance [post]
func (s *Server) androidStudentAttendanceMarkHandler(c *fiber.Ctx) error {
	var body app.AndroidAttendanceMarkData
	return s.androidJSONActionHandler(c, "http-android-mark-attendance", "mark_android_attendance", &body)
}

func (s *Server) androidJSONActionHandler(c *fiber.Ctx, requestID, action string, body any) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(app.Response{OK: false, Error: "missing Authorization header"})
	}
	if err := c.BodyParser(body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "Error parsing body"})
	}

	data, err := json.Marshal(body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling request"})
	}
	raw, err := json.Marshal(app.Request{ID: requestID, Action: action, Token: token, Data: data})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling envelope"})
	}

	resp, err := s.svc.DispatchRequest(string(raw), s.requestTimeout)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(app.Response{OK: false, Error: err.Error()})
	}
	if !resp.OK {
		switch resp.Error {
		case "invalid token", "session not found", "missing token":
			return c.Status(fiber.StatusUnauthorized).JSON(resp)
		case "forbidden: teacher role required",
			"forbidden: student role required",
			"forbidden: student is not in session roster",
			"forbidden: teacher_id does not match current user",
			"forbidden: teacher is not assigned to subject",
			"forbidden: lesson belongs to another teacher":
			return c.Status(fiber.StatusForbidden).JSON(resp)
		case "attendance already confirmed":
			return c.Status(fiber.StatusConflict).JSON(resp)
		default:
			return c.Status(fiber.StatusBadRequest).JSON(resp)
		}
	}
	return c.JSON(resp)
}

// uploadAvatarHandler godoc
// @Summary Upload current user avatar
// @Description Saves a JPEG, PNG, or WebP profile picture up to 5 MiB and returns its public URL.
// @Tags profile
// @Accept mpfd
// @Produce json
// @Security BearerAuth
// @Param avatar formData file true "Avatar image"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Router /api/user/upload-avatar [post]
func (s *Server) uploadAvatarHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(app.Response{OK: false, Error: "missing Authorization header"})
	}

	file, err := c.FormFile("avatar")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "avatar file is required"})
	}
	if file.Size <= 0 || file.Size > 5*1024*1024 {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "avatar file must be between 1 byte and 5 MiB"})
	}

	extension := strings.ToLower(filepath.Ext(file.Filename))
	allowedExtensions := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}
	if !allowedExtensions[extension] {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "avatar file type is not supported"})
	}

	randomBytes := make([]byte, 16)
	if _, err = rand.Read(randomBytes); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "failed to generate avatar filename"})
	}
	filename := hex.EncodeToString(randomBytes) + extension
	avatarDir := filepath.Join(s.cfg.UploadDir, "avatars")
	if err = os.MkdirAll(avatarDir, 0o755); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "failed to prepare avatar storage"})
	}
	avatarPath := filepath.Join(avatarDir, filename)
	if err = c.SaveFile(file, avatarPath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "failed to save avatar"})
	}

	avatarURL := fmt.Sprintf("%s/uploads/avatars/%s", strings.TrimRight(s.cfg.SiteBaseURL, "/"), filename)
	resp := s.svc.UpdateAvatarByToken(token, avatarURL)
	if !resp.OK {
		_ = os.Remove(avatarPath)
		switch resp.Error {
		case "invalid token", "session not found", "missing token":
			return c.Status(fiber.StatusUnauthorized).JSON(resp)
		default:
			return c.Status(fiber.StatusBadRequest).JSON(resp)
		}
	}
	resp.ID = "http-upload-avatar"
	return c.JSON(resp)
}

// registerHandler godoc
// @Summary Register user
// @Description Registers a user by one-time invite_code. Legacy role_hash registration remains supported for existing clients.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body app.RegisterData true "Register payload"
// @Success 200 {object} registerResponse
// @Failure 400 {object} app.Response
// @Failure 500 {object} app.Response
// @Router /register [post]
func (s *Server) registerHandler(c *fiber.Ctx) error {
	var body app.RegisterData
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
	var body app.LoginData
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
// @Summary Register user by invite code
// @Description Creates student, teacher, or admin account by one-time invite code from database.
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

// teacherSubjectsHandler godoc
// @Summary Get teacher subjects
// @Description Returns subjects (and groups) assigned to current teacher from schedule.
// @Tags attendance
// @Produce json
// @Security BearerAuth
// @Success 200 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 500 {object} app.Response
// @Router /api/teacher/subjects [get]
func (s *Server) teacherSubjectsHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(app.Response{OK: false, Error: "missing Authorization header"})
	}

	req := app.Request{ID: "http-teacher-subjects", Action: "teacher_subjects", Token: token}
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
		return c.Status(fiber.StatusBadRequest).JSON(resp)
	}

	return c.JSON(resp)
}

// studentAttendanceHistoryHandler godoc
// @Summary Get current student attendance history
// @Description Returns attendance marks grouped by date for current student and selected year.
// @Tags attendance
// @Produce json
// @Security BearerAuth
// @Param year query int false "History year" default(2026)
// @Success 200 {object} studentAttendanceHistoryResponse
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 500 {object} app.Response
// @Router /api/student/attendance/history [get]
func (s *Server) studentAttendanceHistoryHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(app.Response{OK: false, Error: "missing Authorization header"})
	}

	body := app.AttendanceHistoryData{Year: time.Now().Year()}
	if year := c.QueryInt("year", 0); year > 0 {
		body.Year = year
	}

	data, err := json.Marshal(body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling request"})
	}

	req := app.Request{ID: "http-student-attendance-history", Action: "student_attendance_history", Token: token, Data: data}
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
		if resp.Error == "invalid token" || resp.Error == "session not found" || resp.Error == "missing token" {
			return c.Status(fiber.StatusUnauthorized).JSON(resp)
		}
		return c.Status(fiber.StatusBadRequest).JSON(resp)
	}

	return c.JSON(resp)
}

// teacherCreateGradeItemHandler godoc
// @Summary Create grade item
// @Description Teacher creates a subject grade item/control point.
// @Tags grades
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body app.GradeItemCreateData true "Grade item payload"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 404 {object} app.Response
// @Router /api/teacher/grades/items [post]
func (s *Server) teacherCreateGradeItemHandler(c *fiber.Ctx) error {
	var body app.GradeItemCreateData
	return s.gradeActionHandler(c, "http-teacher-create-grade-item", "teacher_create_grade_item", &body)
}

// teacherGradeItemsBySubjectHandler godoc
// @Summary List grade items by subject
// @Description Teacher lists grade items/control points for assigned subject.
// @Tags grades
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body app.GradeSubjectData true "Subject payload"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 404 {object} app.Response
// @Router /api/teacher/grades/items/list [post]
func (s *Server) teacherGradeItemsBySubjectHandler(c *fiber.Ctx) error {
	var body app.GradeSubjectData
	return s.gradeActionHandler(c, "http-teacher-grade-items", "teacher_grade_items_by_subject", &body)
}

// teacherUpsertGradeHandler godoc
// @Summary Create or update student grade
// @Description Teacher creates or updates a student's score for a grade item.
// @Tags grades
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body app.GradeUpsertData true "Grade payload"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 404 {object} app.Response
// @Router /api/teacher/grades [post]
func (s *Server) teacherUpsertGradeHandler(c *fiber.Ctx) error {
	var body app.GradeUpsertData
	return s.gradeActionHandler(c, "http-teacher-upsert-grade", "teacher_upsert_grade", &body)
}

// teacherStudentGradesBySubjectHandler godoc
// @Summary Get student grades by subject
// @Description Teacher gets a student's grade sheet for an assigned subject.
// @Tags grades
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body app.TeacherStudentGradesData true "Student subject payload"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 404 {object} app.Response
// @Router /api/teacher/grades/student [post]
func (s *Server) teacherStudentGradesBySubjectHandler(c *fiber.Ctx) error {
	var body app.TeacherStudentGradesData
	return s.gradeActionHandler(c, "http-teacher-student-grades", "teacher_student_grades_by_subject", &body)
}

// studentGradesBySubjectHandler godoc
// @Summary Get current student grades by subject
// @Description Student gets own grade sheet for a subject.
// @Tags grades
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body app.GradeSubjectData true "Subject payload"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 404 {object} app.Response
// @Router /api/student/grades [post]
func (s *Server) studentGradesBySubjectHandler(c *fiber.Ctx) error {
	var body app.GradeSubjectData
	return s.gradeActionHandler(c, "http-student-grades", "student_grades_by_subject", &body)
}

func (s *Server) gradeActionHandler(c *fiber.Ctx, requestID, action string, body any) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(app.Response{OK: false, Error: "missing Authorization header"})
	}

	if err := c.BodyParser(body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "Error parsing body"})
	}

	data, err := json.Marshal(body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling request"})
	}

	req := app.Request{ID: requestID, Action: action, Token: token, Data: data}
	raw, err := json.Marshal(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling envelope"})
	}

	resp, err := s.svc.DispatchRequest(string(raw), s.requestTimeout)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(app.Response{OK: false, Error: err.Error()})
	}
	if !resp.OK {
		return c.Status(app.GradeHTTPStatus(resp)).JSON(resp)
	}

	return c.JSON(resp)
}
