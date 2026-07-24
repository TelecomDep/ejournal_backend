package httpserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "github.com/TelecomDep/ejournal_backend/docs"
	"github.com/TelecomDep/ejournal_backend/internal/app"
	"github.com/TelecomDep/ejournal_backend/internal/config"
	"github.com/ansrivas/fiberprometheus/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	swagger "github.com/gofiber/swagger"
)

type Server struct {
	cfg            config.AppConfig
	svc            *app.Service
	requestTimeout time.Duration
	metrics        *httpMetrics
}

func New(cfg config.AppConfig, svc *app.Service) *Server {
	return &Server{
		cfg:            cfg,
		svc:            svc,
		requestTimeout: 3 * time.Second,
		metrics:        newHTTPMetrics(),
	}
}

func (s *Server) Start() {
	fiberApp := fiber.New()
	fiberApp.Use(s.metricsMiddleware)

	prometheus := fiberprometheus.NewWithDefaultRegistry("ejournal-backend")
	prometheus.RegisterAt(fiberApp, "/metrics")
	fiberApp.Use(prometheus.Middleware)

	fiberApp.Use(cors.New(cors.Config{
		AllowOrigins: s.cfg.CORSAllowOrigins,
		AllowMethods: "GET,POST,DELETE,PATCH,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	fiberApp.Post("/register", s.registerHandler)
	fiberApp.Get("/healthz", s.healthHandler)
	fiberApp.Get("/internal/metrics", s.internalMetricsHandler)
	fiberApp.Post("/register/by-invite", s.registerByInviteHandler)
	fiberApp.Post("/login", s.loginHandler)
	fiberApp.Get("/profile", s.profileHandler)
	fiberApp.Post("/lessons/create", s.androidLessonCreateHandler)
	fiberApp.Post("/api/auth/forgot-password", s.forgotPasswordHandler)
	fiberApp.Post("/api/auth/reset-password", s.resetPasswordHandler)
	fiberApp.Post("/api/user/email", s.updateEmailHandler)
	fiberApp.Post("/api/user/email/bind/request", s.requestEmailBindHandler)
	fiberApp.Post("/api/user/email/bind/confirm", s.confirmEmailBindHandler)
	fiberApp.Get("/api/user/2fa/generate", s.generate2faHandler)
	fiberApp.Post("/api/user/2fa/verify", s.verify2faHandler)
	fiberApp.Post("/api/user/2fa/disable", s.disable2faHandler)
	fiberApp.Get("/api/semesters", s.semestersListHandler)
	fiberApp.Get("/api/semesters/current", s.currentSemesterHandler)
	fiberApp.Post("/api/admin/semesters", s.createSemesterHandler)
	fiberApp.Patch("/api/admin/semesters/:semester_id/activate", s.activateSemesterHandler)

	fiberApp.Post("/api/teacher/attendance-link", s.teacherAttendanceLinkHandler)
	fiberApp.Post("/api/teacher/attendance/session", s.teacherAttendanceLinkHandler)
	fiberApp.Get("/api/teacher/attendance/session/marked-count", s.teacherAttendanceMarkedCountHandler)
	fiberApp.Get("/api/teacher/attendance/session/timer", s.teacherAttendanceSessionTimerHandler)
	fiberApp.Get("/api/teacher/attendance/session/active", s.teacherActiveAttendanceSessionHandler)
	fiberApp.Post("/api/teacher/attendance/mark", s.teacherAttendanceMarkHandler)
	fiberApp.Get("/api/teacher/subjects", s.teacherSubjectsHandler)
	fiberApp.Post("/api/teacher/attendance/group", s.teacherAttendanceByGroupHandler)
	fiberApp.Post("/api/teacher/group/performance", s.teacherGroupPerformanceHandler)
	fiberApp.Post("/api/teacher/attendance/student/history", s.teacherAttendanceStudentHistoryHandler)
	fiberApp.Post("/api/student/attendance/confirm", s.studentAttendanceConfirmHandler)
	fiberApp.Post("/api/student/mark-attendance", s.androidStudentAttendanceMarkHandler)
	fiberApp.Get("/api/student/attendance/history", s.studentAttendanceHistoryHandler)
	fiberApp.Get("/api/staff/overview", s.staffOverviewHandler)
	fiberApp.Get("/api/staff/overview/students", s.staffStudentsPageHandler)
	fiberApp.Get("/api/staff/reports/performance.xlsx", s.staffPerformanceReportHandler)
	fiberApp.Get("/api/staff/reports/performance.pdf", s.staffPerformanceReportPDFHandler)
	fiberApp.Get("/api/student/schedule/day", s.studentScheduleDayHandler)
	fiberApp.Post("/api/user/upload-avatar", s.uploadAvatarHandler)
	fiberApp.Post("/api/teacher/grades/items", s.teacherCreateGradeItemHandler)
	fiberApp.Post("/api/teacher/grades/items/list", s.teacherGradeItemsBySubjectHandler)
	fiberApp.Delete("/api/teacher/grades/items/:item_id", s.teacherDeleteGradeItemHandler)
	fiberApp.Post("/api/teacher/grades/items/:item_id/restore", s.teacherRestoreGradeItemHandler)
	fiberApp.Post("/api/teacher/grades", s.teacherUpsertGradeHandler)
	fiberApp.Delete("/api/teacher/grades/:grade_id", s.teacherDeleteGradeHandler)
	fiberApp.Post("/api/teacher/grades/:grade_id/restore", s.teacherRestoreGradeHandler)
	fiberApp.Post("/api/teacher/grades/student", s.teacherStudentGradesBySubjectHandler)
	fiberApp.Post("/api/student/grades", s.studentGradesBySubjectHandler)
	fiberApp.Get("/api/student/performance/radar", s.studentPerformanceRadarHandler)
	fiberApp.Get("/api/student/grades/all", s.studentAllGradesHandler)
	fiberApp.Post("/api/teacher/student/performance/radar", s.teacherStudentPerformanceRadarHandler)
	fiberApp.Static("/uploads", s.cfg.UploadDir)
	fiberApp.Get("/swagger/*", swagger.HandlerDefault)
	fiberApp.Get("/healthz", s.healthzHandler)

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

// semestersListHandler godoc
// @Summary List semesters
// @Description Returns all known semesters ordered by start date.
// @Tags semesters
// @Produce json
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Router /api/semesters [get]
func (s *Server) semestersListHandler(c *fiber.Ctx) error {
	req := app.Request{ID: "http-semesters-list", Action: "semesters_list"}
	raw, err := json.Marshal(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling envelope"})
	}

	resp, err := s.svc.DispatchRequest(string(raw), s.requestTimeout)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(app.Response{OK: false, Error: err.Error()})
	}
	if !resp.OK {
		return c.Status(semesterHTTPStatus(resp)).JSON(resp)
	}
	return c.JSON(resp)
}

// currentSemesterHandler godoc
// @Summary Get current semester
// @Description Returns the active semester used by grade and attendance calculations.
// @Tags semesters
// @Produce json
// @Success 200 {object} app.Response
// @Failure 404 {object} app.Response
// @Router /api/semesters/current [get]
func (s *Server) currentSemesterHandler(c *fiber.Ctx) error {
	req := app.Request{ID: "http-current-semester", Action: "current_semester"}
	raw, err := json.Marshal(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling envelope"})
	}

	resp, err := s.svc.DispatchRequest(string(raw), s.requestTimeout)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(app.Response{OK: false, Error: err.Error()})
	}
	if !resp.OK {
		return c.Status(semesterHTTPStatus(resp)).JSON(resp)
	}
	return c.JSON(resp)
}

// createSemesterHandler godoc
// @Summary Create semester
// @Description Admin creates a semester record and may mark it current.
// @Tags semesters
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body app.SemesterCreateData true "Semester payload"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Router /api/admin/semesters [post]
func (s *Server) createSemesterHandler(c *fiber.Ctx) error {
	var body app.SemesterCreateData
	return s.semesterActionHandler(c, "http-create-semester", "create_semester", &body)
}

// activateSemesterHandler godoc
// @Summary Activate semester
// @Description Admin marks one semester as current and clears the previous one.
// @Tags semesters
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body app.SemesterIDData true "Semester payload"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 404 {object} app.Response
// @Router /api/admin/semesters/{semester_id}/activate [patch]
func (s *Server) activateSemesterHandler(c *fiber.Ctx) error {
	semesterID, err := c.ParamsInt("semester_id")
	if err != nil || semesterID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "invalid semester_id"})
	}
	body := app.SemesterIDData{SemesterID: int32(semesterID)}
	return s.semesterActionHandler(c, "http-activate-semester", "activate_semester", &body)
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

// staffOverviewHandler godoc
// @Summary Supervisory overview (teacher/head/dean/admin)
// @Description Returns groups, teachers and students scoped to the caller's role: teacher -> own groups, head -> own lectern, dean -> own faculty, admin -> everything.
// @Tags staff
// @Produce json
// @Security BearerAuth
// @Success 200 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Router /api/staff/overview [get]
func (s *Server) staffOverviewHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(app.Response{OK: false, Error: "missing Authorization header"})
	}

	req := app.Request{ID: "http-staff-overview", Action: "staff_overview", Token: token}
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

func (s *Server) staffStudentsPageHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(app.Response{OK: false, Error: "missing Authorization header"})
	}
	data := app.StaffStudentsPageData{
		Page:     int32(c.QueryInt("page", 1)),
		PageSize: int32(c.QueryInt("page_size", 100)),
		Search:   c.Query("search"),
		Sort:     c.Query("sort"),
		Order:    c.Query("order"),
	}
	if rawGroupID := strings.TrimSpace(c.Query("group_id")); rawGroupID != "" {
		groupID, err := strconv.ParseInt(rawGroupID, 10, 32)
		if err != nil || groupID <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "invalid group_id"})
		}
		parsed := int32(groupID)
		data.GroupID = &parsed
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling query"})
	}
	raw, err := json.Marshal(app.Request{ID: "http-staff-students-page", Action: "staff_students_page", Token: token, Data: payload})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling envelope"})
	}

	resp, err := s.svc.DispatchRequest(string(raw), s.requestTimeout)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(app.Response{OK: false, Error: err.Error()})
	}
	if !resp.OK {
		status := fiber.StatusBadRequest
		if resp.Error == "unauthorized" {
			status = fiber.StatusUnauthorized
		} else if resp.Error == "forbidden" {
			status = fiber.StatusForbidden
		}
		return c.Status(status).JSON(resp)
	}
	return c.JSON(resp)
}

func (s *Server) loadStaffPerformanceReport(c *fiber.Ctx) (*app.PerformanceReport, error) {
	token := c.Get("Authorization")
	if token == "" {
		return nil, c.Status(fiber.StatusUnauthorized).JSON(app.Response{OK: false, Error: "missing Authorization header"})
	}

	report, resp := s.svc.StaffPerformanceReport(token)
	if report == nil {
		switch resp.Error {
		case "forbidden: head role or higher required":
			return nil, c.Status(fiber.StatusForbidden).JSON(resp)
		case "invalid token", "session not found", "missing token":
			return nil, c.Status(fiber.StatusUnauthorized).JSON(resp)
		default:
			return nil, c.Status(fiber.StatusInternalServerError).JSON(resp)
		}
	}
	return report, nil
}

// staffPerformanceReportHandler godoc
// @Summary Download performance report as Excel
// @Description Head, dean, or admin downloads an xlsx performance rating report: one sheet for the whole scope plus one sheet per group. Rows are students ranked by overall rating with per-subject percents and attendance.
// @Tags staff
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Security BearerAuth
// @Success 200 {file} file "Excel workbook"
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 500 {object} app.Response
// @Router /api/staff/reports/performance.xlsx [get]
func (s *Server) staffPerformanceReportHandler(c *fiber.Ctx) error {
	report, err := s.loadStaffPerformanceReport(c)
	if report == nil {
		return err
	}

	buf, buildErr := app.BuildPerformanceReportXLSX(report)
	if buildErr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "failed to build report file"})
	}

	filename := "performance_" + report.GeneratedAt.Format("2006-01-02") + ".xlsx"
	c.Set(fiber.HeaderContentType, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="`+filename+`"`)
	return c.Send(buf.Bytes())
}

// staffPerformanceReportPDFHandler godoc
// @Summary Download performance report as PDF
// @Description Head, dean, or admin downloads a PDF performance rating report: one page (or more) for the whole scope plus one section per group, colour-coded by percent, matching the xlsx report.
// @Tags staff
// @Produce application/pdf
// @Security BearerAuth
// @Success 200 {file} file "PDF document"
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 500 {object} app.Response
// @Router /api/staff/reports/performance.pdf [get]
func (s *Server) staffPerformanceReportPDFHandler(c *fiber.Ctx) error {
	report, err := s.loadStaffPerformanceReport(c)
	if report == nil {
		return err
	}

	buf, buildErr := app.BuildPerformanceReportPDF(report)
	if buildErr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "failed to build report file"})
	}

	filename := "performance_" + report.GeneratedAt.Format("2006-01-02") + ".pdf"
	c.Set(fiber.HeaderContentType, "application/pdf")
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="`+filename+`"`)
	return c.Send(buf.Bytes())
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

// teacherAttendanceStudentHistoryHandler godoc
// @Summary Get a student's attendance history for a subject
// @Description Teacher gets the detailed attendance history of a specific student for a specific subject.
// @Tags attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body app.TeacherAttendanceStudentHistoryData true "Student and subject payload"
// @Success 200 {object} teacherAttendanceStudentHistoryResponse
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 404 {object} app.Response
// @Router /api/teacher/attendance/student/history [post]
func (s *Server) teacherAttendanceStudentHistoryHandler(c *fiber.Ctx) error {
	var body app.TeacherAttendanceStudentHistoryData
	return s.gradeActionHandler(c, "http-teacher-attendance-student-history", "teacher_attendance_student_history", &body)
}

// teacherGroupPerformanceHandler godoc
// @Summary Get a group performance overview for a subject
// @Description Teacher gets a combined per-student overview (attendance and grade totals) for a group on a subject.
// @Tags attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body app.GroupPerformanceData true "Group and subject payload"
// @Success 200 {object} teacherGroupPerformanceResponse
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 404 {object} app.Response
// @Router /api/teacher/group/performance [post]
func (s *Server) teacherGroupPerformanceHandler(c *fiber.Ctx) error {
	var body app.GroupPerformanceData
	return s.gradeActionHandler(c, "http-teacher-group-performance", "teacher_group_performance", &body)
}

// teacherAttendanceMarkedCountHandler godoc
// @Summary Get attendance marked count
// @Description Returns how many students have marked attendance in a teacher-owned session.
// @Tags attendance
// @Produce json
// @Security BearerAuth
// @Param lesson_id query int true "Attendance session ID"
// @Success 200 {object} teacherAttendanceMarkedCountResponse
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Router /api/teacher/attendance/session/marked-count [get]
func (s *Server) teacherAttendanceMarkedCountHandler(c *fiber.Ctx) error {
	body := app.AttendanceSessionData{LessonID: int32(c.QueryInt("lesson_id", 0))}
	return s.teacherAttendanceReadHandler(c, "http-teacher-attendance-marked-count", "teacher_attendance_marked_count", body)
}

// teacherAttendanceSessionTimerHandler godoc
// @Summary Get attendance session timer
// @Description Returns remaining seconds for a teacher-owned attendance session.
// @Tags attendance
// @Produce json
// @Security BearerAuth
// @Param lesson_id query int true "Attendance session ID"
// @Success 200 {object} teacherAttendanceSessionTimerResponse
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Router /api/teacher/attendance/session/timer [get]
func (s *Server) teacherAttendanceSessionTimerHandler(c *fiber.Ctx) error {
	body := app.AttendanceSessionData{LessonID: int32(c.QueryInt("lesson_id", 0))}
	return s.teacherAttendanceReadHandler(c, "http-teacher-attendance-session-timer", "teacher_attendance_session_timer", body)
}

// teacherActiveAttendanceSessionHandler godoc
// @Summary Get active attendance session
// @Description Returns current teacher attendance session that has not expired yet.
// @Tags attendance
// @Produce json
// @Security BearerAuth
// @Success 200 {object} teacherActiveAttendanceSessionResponse
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Router /api/teacher/attendance/session/active [get]
func (s *Server) teacherActiveAttendanceSessionHandler(c *fiber.Ctx) error {
	return s.teacherAttendanceReadHandler(c, "http-teacher-active-attendance-session", "teacher_active_attendance_session", nil)
}

func (s *Server) teacherAttendanceReadHandler(c *fiber.Ctx, requestID, action string, body any) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(app.Response{OK: false, Error: "missing Authorization header"})
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
		if resp.Error == "forbidden: teacher role required" || resp.Error == "forbidden: lesson belongs to another teacher" {
			return c.Status(fiber.StatusForbidden).JSON(resp)
		}
		if resp.Error == "invalid token" || resp.Error == "session not found" || resp.Error == "missing token" {
			return c.Status(fiber.StatusUnauthorized).JSON(resp)
		}
		return c.Status(fiber.StatusBadRequest).JSON(resp)
	}

	return c.JSON(resp)
}

// teacherAttendanceMarkHandler godoc
// @Summary Manually correct a student's attendance status
// @Description Teacher sets a student's attendance status (present/absent/late/excused) for a session they own, overriding self/device check-in.
// @Tags attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body app.TeacherAttendanceMarkData true "Lesson, student and new status"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 404 {object} app.Response
// @Router /api/teacher/attendance/mark [post]
func (s *Server) teacherAttendanceMarkHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(app.Response{OK: false, Error: "missing Authorization header"})
	}

	var body app.TeacherAttendanceMarkData
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "Error parsing body"})
	}

	data, err := json.Marshal(body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling request"})
	}

	req := app.Request{ID: "http-teacher-attendance-mark", Action: "teacher_attendance_mark", Token: token, Data: data}
	raw, err := json.Marshal(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling envelope"})
	}

	resp, err := s.svc.DispatchRequest(string(raw), s.requestTimeout)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(app.Response{OK: false, Error: err.Error()})
	}
	if !resp.OK {
		if resp.Error == "forbidden: teacher role required" || resp.Error == "forbidden: lesson belongs to another teacher" {
			return c.Status(fiber.StatusForbidden).JSON(resp)
		}
		if resp.Error == "invalid token" || resp.Error == "session not found" || resp.Error == "missing token" {
			return c.Status(fiber.StatusUnauthorized).JSON(resp)
		}
		if resp.Error == "student not found in this session's roster" {
			return c.Status(fiber.StatusNotFound).JSON(resp)
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

// studentScheduleDayHandler godoc
// @Summary Get student's schedule for a day
// @Description Returns the current student's lessons for a given date, restricted to the current or next calendar week. Defaults to today if date is omitted.
// @Tags schedule
// @Produce json
// @Security BearerAuth
// @Param date query string false "Date in YYYY-MM-DD format (defaults to today)"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Router /api/student/schedule/day [get]
func (s *Server) studentScheduleDayHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(app.Response{OK: false, Error: "missing Authorization header"})
	}

	body := app.StudentScheduleDayData{Date: c.Query("date", "")}
	data, err := json.Marshal(body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling request"})
	}

	req := app.Request{ID: "http-student-schedule-day", Action: "student_schedule_day", Token: token, Data: data}
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

func (s *Server) teacherDeleteGradeHandler(c *fiber.Ctx) error {
	gradeID, err := c.ParamsInt("grade_id")
	if err != nil || gradeID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "invalid grade_id"})
	}
	body := app.GradeDeleteData{GradeID: int32(gradeID)}
	return s.gradeDeleteActionHandler(c, "http-teacher-delete-grade", "teacher_delete_grade", &body, int32(gradeID))
}

func (s *Server) teacherRestoreGradeHandler(c *fiber.Ctx) error {
	gradeID, err := c.ParamsInt("grade_id")
	if err != nil || gradeID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "invalid grade_id"})
	}
	body := app.GradeRestoreData{GradeID: int32(gradeID)}
	return s.gradeDeleteActionHandler(c, "http-teacher-restore-grade", "teacher_restore_grade", &body, int32(gradeID))
}

func (s *Server) teacherDeleteGradeItemHandler(c *fiber.Ctx) error {
	itemID, err := c.ParamsInt("item_id")
	if err != nil || itemID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "invalid item_id"})
	}
	body := app.GradeItemDeleteData{ItemID: int32(itemID)}
	return s.gradeDeleteActionHandler(c, "http-teacher-delete-grade-item", "teacher_delete_grade_item", &body, int32(itemID))
}

func (s *Server) teacherRestoreGradeItemHandler(c *fiber.Ctx) error {
	itemID, err := c.ParamsInt("item_id")
	if err != nil || itemID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "invalid item_id"})
	}
	body := app.GradeItemRestoreData{ItemID: int32(itemID)}
	return s.gradeDeleteActionHandler(c, "http-teacher-restore-grade-item", "teacher_restore_grade_item", &body, int32(itemID))
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

// studentPerformanceRadarHandler godoc
// @Summary Get current student performance radar
// @Description Returns one point per subject (from the student's group schedule) with the score/percent the student earned on graded-so-far work this semester.
// @Tags grades
// @Produce json
// @Security BearerAuth
// @Success 200 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 500 {object} app.Response
// @Router /api/student/performance/radar [get]
func (s *Server) studentPerformanceRadarHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(app.Response{OK: false, Error: "missing Authorization header"})
	}

	req := app.Request{ID: "http-student-performance-radar", Action: "student_performance_radar", Token: token}
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

// studentAllGradesHandler godoc
// @Summary Get all grades for the current student
// @Description Returns every plan subject with its grade items and per-subject totals, plus an aggregate summary, in one request.
// @Tags grades
// @Produce json
// @Security BearerAuth
// @Success 200 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Router /api/student/grades/all [get]
func (s *Server) studentAllGradesHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(app.Response{OK: false, Error: "missing Authorization header"})
	}

	req := app.Request{ID: "http-student-all-grades", Action: "student_all_grades", Token: token}
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

// teacherStudentPerformanceRadarHandler godoc
// @Summary Get a student's performance radar (teacher)
// @Description Teacher gets the per-subject performance radar for a student they teach.
// @Tags grades
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body app.TeacherStudentRadarData true "Student payload"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 404 {object} app.Response
// @Router /api/teacher/student/performance/radar [post]
func (s *Server) teacherStudentPerformanceRadarHandler(c *fiber.Ctx) error {
	var body app.TeacherStudentRadarData
	return s.gradeActionHandler(c, "http-teacher-student-performance-radar", "teacher_student_performance_radar", &body)
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

func (s *Server) semesterActionHandler(c *fiber.Ctx, requestID, action string, body any) error {
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
		return c.Status(semesterHTTPStatus(resp)).JSON(resp)
	}

	return c.JSON(resp)
}

func semesterHTTPStatus(resp app.Response) int {
	if resp.OK {
		return fiber.StatusOK
	}
	switch resp.Error {
	case "missing token", "invalid token", "session not found":
		return fiber.StatusUnauthorized
	case "forbidden: admin role required":
		return fiber.StatusForbidden
	case "semester not found", "current semester not found":
		return fiber.StatusNotFound
	default:
		return fiber.StatusBadRequest
	}
}

func (s *Server) gradeDeleteActionHandler(c *fiber.Ctx, requestID, action string, body any, pathID int32) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(app.Response{OK: false, Error: "missing Authorization header"})
	}
	if len(c.Body()) > 0 {
		if err := c.BodyParser(body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "Error parsing body"})
		}
	}
	switch target := body.(type) {
	case *app.GradeDeleteData:
		target.GradeID = pathID
	case *app.GradeRestoreData:
		target.GradeID = pathID
	case *app.GradeItemDeleteData:
		target.ItemID = pathID
	case *app.GradeItemRestoreData:
		target.ItemID = pathID
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
		return c.Status(app.GradeHTTPStatus(resp)).JSON(resp)
	}
	return c.JSON(resp)
}

// forgotPasswordHandler godoc
// @Summary Forgot password
// @Description Generates a password reset token and sends it to the user's email.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body app.ForgotPasswordData true "Forgot password payload"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 500 {object} app.Response
// @Router /api/auth/forgot-password [post]
func (s *Server) forgotPasswordHandler(c *fiber.Ctx) error {
	var body app.ForgotPasswordData
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "Error parsing body"})
	}

	data, err := json.Marshal(body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling request"})
	}

	req := app.Request{ID: "http-forgot-password", Action: "forgot_password", Data: data}
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

// resetPasswordHandler godoc
// @Summary Reset password
// @Description Resets user password using the provided reset token.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body app.ResetPasswordData true "Reset password payload"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 500 {object} app.Response
// @Router /api/auth/reset-password [post]
func (s *Server) resetPasswordHandler(c *fiber.Ctx) error {
	var body app.ResetPasswordData
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "Error parsing body"})
	}

	data, err := json.Marshal(body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling request"})
	}

	req := app.Request{ID: "http-reset-password", Action: "reset_password", Data: data}
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

// updateEmailHandler godoc
// @Summary Update user email
// @Description Links/updates email for current user.
// @Tags profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body app.UpdateEmailData true "Update email payload"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 500 {object} app.Response
// @Router /api/user/email [post]
func (s *Server) updateEmailHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(app.Response{OK: false, Error: "missing Authorization header"})
	}

	var body app.UpdateEmailData
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "Error parsing body"})
	}

	data, err := json.Marshal(body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling request"})
	}

	req := app.Request{ID: "http-update-email", Action: "update_email", Token: token, Data: data}
	raw, err := json.Marshal(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling envelope"})
	}

	resp, err := s.svc.DispatchRequest(string(raw), s.requestTimeout)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(app.Response{OK: false, Error: err.Error()})
	}

	if !resp.OK {
		if resp.Error == "invalid token" || resp.Error == "session not found" || resp.Error == "missing token" {
			return c.Status(fiber.StatusUnauthorized).JSON(resp)
		}
		return c.Status(fiber.StatusBadRequest).JSON(resp)
	}

	return c.JSON(resp)
}

func (s *Server) healthzHandler(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := s.svc.Ping(ctx); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(app.Response{
			OK:    false,
			Error: "database connection issues: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(app.Response{
		OK: true,
	})
}
