package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"mime"
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
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/recover"
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
	trustedProxies := strings.FieldsFunc(s.cfg.TrustedProxies, func(r rune) bool { return r == ',' || r == ' ' })
	fiberApp := fiber.New(fiber.Config{
		BodyLimit:               52 * 1024 * 1024,
		ReadTimeout:             15 * time.Second,
		WriteTimeout:            60 * time.Second,
		IdleTimeout:             60 * time.Second,
		ProxyHeader:             fiber.HeaderXForwardedFor,
		EnableTrustedProxyCheck: true,
		TrustedProxies:          trustedProxies,
	})
	fiberApp.Use(recover.New())
	fiberApp.Use(helmet.New())
	fiberApp.Use(s.metricsMiddleware)
	fiberApp.Use(s.metricsAccessMiddleware)

	prometheus := fiberprometheus.NewWithDefaultRegistry("ejournal-backend")
	prometheus.RegisterAt(fiberApp, "/metrics")
	fiberApp.Use(prometheus.Middleware)

	fiberApp.Use(cors.New(cors.Config{
		AllowOrigins: s.cfg.CORSAllowOrigins,
		AllowMethods: "GET,POST,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))
	fiberApp.Use(s.maintenanceMiddleware)

	loginLimit := limiter.New(limiter.Config{Max: 10, Expiration: 5 * time.Minute})
	registrationLimit := limiter.New(limiter.Config{Max: 5, Expiration: 5 * time.Minute})
	recoveryLimit := limiter.New(limiter.Config{Max: 5, Expiration: 15 * time.Minute})
	verificationLimit := limiter.New(limiter.Config{Max: 10, Expiration: 5 * time.Minute})

	fiberApp.Post("/register", registrationLimit, s.registerHandler)
	fiberApp.Get("/healthz", s.healthHandler)
	fiberApp.Get("/internal/metrics", s.internalMetricsHandler)
	fiberApp.Post("/register/by-invite", registrationLimit, s.registerByInviteHandler)
	fiberApp.Post("/login", loginLimit, s.loginHandler)
	fiberApp.Post("/api/auth/refresh", s.refreshTokenHandler)
	fiberApp.Post("/auth/refresh", s.refreshTokenHandler)
	fiberApp.Get("/api/auth/refresh", s.refreshTokenHandler)
	fiberApp.Get("/profile", s.profileHandler)
	fiberApp.Post("/lessons/create", s.androidLessonCreateHandler)
	fiberApp.Post("/api/auth/forgot-password", recoveryLimit, s.forgotPasswordHandler)
	fiberApp.Post("/api/auth/reset-password", recoveryLimit, s.resetPasswordHandler)
	fiberApp.Post("/api/user/email", s.updateEmailHandler)
	fiberApp.Post("/api/user/email/bind/request", verificationLimit, s.requestEmailBindHandler)
	fiberApp.Post("/api/user/email/bind/confirm", verificationLimit, s.confirmEmailBindHandler)
	fiberApp.Get("/api/user/notifications", s.notificationsListHandler)
	fiberApp.Get("/api/user/notifications/unread-count", s.notificationsUnreadCountHandler)
	fiberApp.Patch("/api/user/notifications/read-all", s.notificationsMarkAllReadHandler)
	fiberApp.Patch("/api/user/notifications/:notification_id/read", s.notificationMarkReadHandler)
	fiberApp.Delete("/api/user/notifications/:notification_id", s.notificationDeleteHandler)
	fiberApp.Get("/api/user/notification-settings", s.notificationSettingsGetHandler)
	fiberApp.Put("/api/user/notification-settings", s.notificationSettingsUpdateHandler)
	fiberApp.Post("/api/user/2fa/request-enable", verificationLimit, s.request2FAEnableHandler)
	fiberApp.Get("/api/user/2fa/generate", s.generate2faHandler)
	fiberApp.Post("/api/user/2fa/generate", s.generate2faHandler)
	fiberApp.Post("/api/user/2fa/verify", verificationLimit, s.verify2faHandler)
	fiberApp.Post("/api/user/2fa/disable", verificationLimit, s.disable2faHandler)
	fiberApp.Get("/api/semesters", s.semestersListHandler)
	fiberApp.Get("/api/semesters/current", s.currentSemesterHandler)
	fiberApp.Post("/api/admin/semesters", s.createSemesterHandler)
	fiberApp.Patch("/api/admin/semesters/:semester_id/activate", s.activateSemesterHandler)
	fiberApp.Patch("/api/admin/semesters/:semester_id/close", s.closeSemesterHandler)
	fiberApp.Patch("/api/admin/semesters/:semester_id/archive", s.archiveSemesterHandler)
	fiberApp.Delete("/api/admin/semesters/:semester_id", s.deleteSemesterHandler)
	fiberApp.Get("/api/admin/users", s.adminUsersListHandler)
	fiberApp.Get("/api/admin/users/:user_id", s.adminUserGetHandler)
	fiberApp.Post("/api/admin/users", s.adminUserCreateHandler)
	fiberApp.Patch("/api/admin/users/:user_id", s.adminUserUpdateHandler)
	fiberApp.Delete("/api/admin/users/:user_id", s.adminUserDeleteHandler)
	fiberApp.Get("/api/admin/invites", s.adminInvitesListHandler)
	fiberApp.Post("/api/admin/invites/teacher", s.adminGenerateTeacherInviteHandler)
	fiberApp.Post("/api/admin/invites/student", s.adminGenerateStudentInviteHandler)
	fiberApp.Delete("/api/admin/invites/:invite_id", s.adminRevokeInviteHandler)
	fiberApp.Get("/api/admin/catalogs", s.adminCatalogsHandler)
	fiberApp.Get("/api/admin/notifications", s.adminNotificationsListHandler)
	fiberApp.Post("/api/admin/notifications", s.adminNotificationsCreateHandler)
	fiberApp.Patch("/api/admin/notifications/:notification_id", s.adminNotificationsUpdateHandler)
	fiberApp.Delete("/api/admin/notifications/:notification_id", s.adminNotificationsDeleteHandler)
	fiberApp.Get("/api/admin/stats", s.adminSystemStatsHandler)
	fiberApp.Get("/api/admin/org-structure", s.adminOrgStructureHandler)
	fiberApp.Get("/api/admin/roles", s.adminRolesListHandler)
	fiberApp.Patch("/api/admin/roles/:role", s.adminRoleUpdateHandler)
	fiberApp.Get("/api/admin/antifraud/logs", s.adminAntifraudLogsHandler)
	fiberApp.Get("/api/admin/antifraud/top-cheaters", s.adminAntifraudTopCheatersHandler)
	fiberApp.Get("/api/staff/antifraud/logs", s.adminAntifraudLogsHandler)
	fiberApp.Get("/api/staff/antifraud/top-cheaters", s.adminAntifraudTopCheatersHandler)
	fiberApp.Get("/api/admin/services", s.adminServicesListHandler)
	fiberApp.Get("/api/admin/audit-logs", s.adminAuditLogsHandler)
	fiberApp.Get("/api/admin/system/maintenance", s.adminSystemMaintenanceGetHandler)
	fiberApp.Post("/api/admin/system/maintenance", s.adminSystemMaintenanceSetHandler)

	fiberApp.Post("/api/teacher/attendance-link", s.teacherAttendanceLinkHandler)
	fiberApp.Post("/api/teacher/attendance/session", s.teacherAttendanceLinkHandler)
	fiberApp.Get("/api/teacher/attendance/session/marked-count", s.teacherAttendanceMarkedCountHandler)
	fiberApp.Get("/api/teacher/attendance/session/roster", s.teacherAttendanceSessionRosterHandler)
	fiberApp.Get("/api/teacher/attendance/session/timer", s.teacherAttendanceSessionTimerHandler)
	fiberApp.Get("/api/teacher/attendance/session/active", s.teacherActiveAttendanceSessionHandler)
	fiberApp.Post("/api/teacher/attendance/session/finish", s.teacherFinishAttendanceSessionHandler)
	fiberApp.Post("/api/teacher/lesson/finish", s.teacherFinishAttendanceSessionHandler)
	fiberApp.Post("/api/teacher/attendance/mark", s.teacherAttendanceMarkHandler)
	fiberApp.Get("/api/teacher/subjects", s.teacherSubjectsHandler)
	fiberApp.Post("/api/teacher/attendance/group", s.teacherAttendanceByGroupHandler)
	fiberApp.Post("/api/teacher/group/performance", s.teacherGroupPerformanceHandler)
	fiberApp.Post("/api/teacher/attendance/student/history", s.teacherAttendanceStudentHistoryHandler)
	fiberApp.Post("/api/student/attendance/confirm", s.studentAttendanceConfirmHandler)
	fiberApp.Post("/api/student/mark-attendance", s.androidStudentAttendanceMarkHandler)
	fiberApp.Get("/api/student/attendance/active-session", s.studentActiveAttendanceSessionHandler)
	fiberApp.Get("/api/student/active-session", s.studentActiveAttendanceSessionHandler)
	fiberApp.Get("/api/student/attendance/history", s.studentAttendanceHistoryHandler)
	fiberApp.Get("/api/student/attendance/summary", s.studentAttendanceSummaryHandler)
	fiberApp.Get("/api/student/schedule/day", s.studentScheduleDayHandler)
	fiberApp.Get("/api/student/ratings/group", s.generalRatingHandler)
	fiberApp.Get("/api/teacher/schedule/day", s.teacherScheduleDayHandler)
	fiberApp.Get("/api/staff/overview", s.staffOverviewHandler)
	fiberApp.Get("/api/staff/overview/students", s.staffStudentsPageHandler)
	fiberApp.Get("/api/staff/ratings/general", s.generalRatingHandler)
	fiberApp.Get("/api/staff/reports/performance.xlsx", s.staffPerformanceReportHandler)
	fiberApp.Get("/api/staff/reports/performance.pdf", s.staffPerformanceReportPDFHandler)
	fiberApp.Get("/api/user/avatar/:user_id", s.getUserAvatarHandler)
	fiberApp.Post("/api/user/upload-avatar", s.uploadAvatarHandler)
	fiberApp.Post("/api/user/device-token", s.registerDeviceTokenHandler)
	fiberApp.Get("/api/user/device-tokens", s.listDeviceTokensHandler)
	fiberApp.Delete("/api/user/device-token", s.deleteDeviceTokenHandler)
	fiberApp.Post("/api/attachments/upload", s.uploadAttachmentHandler)
	fiberApp.Get("/api/attachments/:id", s.getAttachmentHandler)
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
	fiberApp.Post("/api/user/agreements/decision", s.userAgreementDecisionHandler)
	fiberApp.Get("/api/user/agreements/current", s.currentUserAgreementHandler)
	fiberApp.Get("/swagger/*", swagger.HandlerDefault)
	fiberApp.Get("/healthz", s.healthzHandler)
	fiberApp.Delete("/api/user/email", s.deleteEmailHandler)

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

	var data []byte = []byte("{}")
	var err error
	if body != nil && len(c.Body()) > 0 {
		if err := c.BodyParser(body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "Error parsing body"})
		}
		data, err = json.Marshal(body)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling request"})
		}
	}
	raw, err := json.Marshal(app.Request{
		ID:     requestID,
		Action: action,
		Token:  token,
		Data:   data,
		Meta: &app.RequestMeta{
			IP:        c.IP(),
			UserAgent: c.Get("User-Agent"),
		},
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling envelope"})
	}

	resp, err := s.svc.DispatchRequest(string(raw), s.requestTimeout)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(app.Response{OK: false, Error: err.Error()})
	}
	if !resp.OK {
		switch resp.Error {
		case "invalid token", "session not found", "session revoked", "missing token", "account is not active":
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

// getUserAvatarHandler godoc
// @Summary Get user avatar
// @Description Returns the avatar image for the selected user.
// @Tags profile
// @Produce octet-stream
// @Param user_id path int true "User ID"
// @Success 200 {file} binary
// @Failure 400 {object} app.Response
// @Failure 404 {object} app.Response
// @Failure 500 {object} app.Response
// @Router /api/user/avatar/{user_id} [get]
func (s *Server) getUserAvatarHandler(c *fiber.Ctx) error {
	userID, err := strconv.ParseInt(c.Params("user_id"), 10, 32)
	if err != nil || userID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "invalid user_id"})
	}

	avatar, found, err := s.svc.GetUserAvatar(int32(userID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "failed to retrieve avatar"})
	}
	if !found || len(avatar.ImageData) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(app.Response{OK: false, Error: "avatar not found"})
	}

	clientETag := strings.Trim(c.Get("If-None-Match"), `"`)
	if clientETag != "" && (clientETag == avatar.Hash || (len(avatar.Hash) >= 8 && clientETag == avatar.Hash[:8])) {
		return c.SendStatus(fiber.StatusNotModified)
	}

	c.Set("Content-Type", avatar.ContentType)
	c.Set("ETag", fmt.Sprintf(`"%s"`, avatar.Hash))
	c.Set("Cache-Control", "public, max-age=31536000, immutable")
	return c.Send(avatar.ImageData)
}

// uploadAvatarHandler godoc
// @Summary Upload current user avatar
// @Description Resizes avatar image to 256x256, stores in PostgreSQL BYTEA, and returns avatar URL.
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

	fileHeader, err := c.FormFile("avatar")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "avatar file is required"})
	}
	if fileHeader.Size <= 0 || fileHeader.Size > 5*1024*1024 {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "avatar file must be between 1 byte and 5 MiB"})
	}

	extension := strings.ToLower(filepath.Ext(fileHeader.Filename))
	allowedExtensions := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}
	if !allowedExtensions[extension] {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "avatar file type is not supported"})
	}

	file, err := fileHeader.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "failed to open avatar file"})
	}
	defer file.Close()

	resp := s.svc.SaveAvatarData(token, file)
	if !resp.OK {
		switch resp.Error {
		case "invalid token", "session not found", "session revoked", "missing token", "account is not active":
			return c.Status(fiber.StatusUnauthorized).JSON(resp)
		default:
			return c.Status(fiber.StatusBadRequest).JSON(resp)
		}
	}
	resp.ID = "http-upload-avatar"
	return c.JSON(resp)
}

func (s *Server) uploadAttachmentHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(app.Response{OK: false, Error: "missing Authorization header"})
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "attachment file is required"})
	}

	file, err := fileHeader.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "failed to open attachment file"})
	}
	defer file.Close()

	resp := s.svc.UploadAttachment(token, fileHeader.Filename, file, fileHeader.Size, fileHeader.Header.Get("Content-Type"))
	if !resp.OK {
		return c.Status(fiber.StatusBadRequest).JSON(resp)
	}
	return c.JSON(resp)
}

func (s *Server) getAttachmentHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(app.Response{OK: false, Error: "missing Authorization header"})
	}
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "invalid attachment id"})
	}

	att, found, err := s.svc.GetAttachmentByID(token, id)
	if err != nil {
		if strings.Contains(err.Error(), "token") || strings.Contains(err.Error(), "session") {
			return c.Status(fiber.StatusUnauthorized).JSON(app.Response{OK: false, Error: "invalid token"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "failed to load attachment"})
	}
	if !found {
		return c.Status(fiber.StatusNotFound).JSON(app.Response{OK: false, Error: "attachment not found"})
	}

	c.Set("Content-Type", att.MimeType)
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": att.Filename})
	if disposition == "" {
		disposition = "attachment"
	}
	c.Set("Content-Disposition", disposition)
	c.Set("X-Content-Type-Options", "nosniff")
	return c.Send(att.Data)
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
// @Description Admin creates a planned semester or opens it immediately with status=open.
// @Tags semesters
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body app.SemesterCreateData true "Semester payload"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 409 {object} app.Response
// @Router /api/admin/semesters [post]
func (s *Server) createSemesterHandler(c *fiber.Ctx) error {
	var body app.SemesterCreateData
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "Error parsing body"})
	}
	return s.semesterActionHandler(c, "http-create-semester", "create_semester", &body)
}

// activateSemesterHandler godoc
// @Summary Activate semester
// @Description Admin opens a planned semester and closes the previously open semester.
// @Tags semesters
// @Produce json
// @Security BearerAuth
// @Param semester_id path int true "Semester ID"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 404 {object} app.Response
// @Failure 409 {object} app.Response
// @Router /api/admin/semesters/{semester_id}/activate [patch]
func (s *Server) activateSemesterHandler(c *fiber.Ctx) error {
	semesterID, err := c.ParamsInt("semester_id")
	if err != nil || semesterID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "invalid semester_id"})
	}
	body := app.SemesterIDData{SemesterID: int32(semesterID)}
	return s.semesterActionHandler(c, "http-activate-semester", "activate_semester", &body)
}

// closeSemesterHandler godoc
// @Summary Close semester
// @Description Admin closes the open semester. Closed semester data becomes read-only.
// @Tags semesters
// @Produce json
// @Security BearerAuth
// @Param semester_id path int true "Semester ID"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 404 {object} app.Response
// @Failure 409 {object} app.Response
// @Router /api/admin/semesters/{semester_id}/close [patch]
func (s *Server) closeSemesterHandler(c *fiber.Ctx) error {
	return s.semesterTransitionHandler(c, "http-close-semester", "close_semester")
}

// archiveSemesterHandler godoc
// @Summary Archive semester
// @Description Admin archives a closed semester. Archived data remains available for historical reads.
// @Tags semesters
// @Produce json
// @Security BearerAuth
// @Param semester_id path int true "Semester ID"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 404 {object} app.Response
// @Failure 409 {object} app.Response
// @Router /api/admin/semesters/{semester_id}/archive [patch]
func (s *Server) archiveSemesterHandler(c *fiber.Ctx) error {
	return s.semesterTransitionHandler(c, "http-archive-semester", "archive_semester")
}

// deleteSemesterHandler godoc
// @Summary Delete semester
// @Description Admin deletes a non-open semester.
// @Tags semesters
// @Produce json
// @Security BearerAuth
// @Param semester_id path int true "Semester ID"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 404 {object} app.Response
// @Router /api/admin/semesters/{semester_id} [delete]
func (s *Server) deleteSemesterHandler(c *fiber.Ctx) error {
	semesterID, err := c.ParamsInt("semester_id")
	if err != nil || semesterID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "invalid semester_id"})
	}
	body := app.SemesterIDData{SemesterID: int32(semesterID)}
	return s.semesterActionHandler(c, "http-delete-semester", "delete_semester", &body)
}

func (s *Server) semesterTransitionHandler(c *fiber.Ctx, requestID, action string) error {
	semesterID, err := c.ParamsInt("semester_id")
	if err != nil || semesterID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "invalid semester_id"})
	}
	body := app.SemesterIDData{SemesterID: int32(semesterID)}
	return s.semesterActionHandler(c, requestID, action, &body)
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
func (s *Server) refreshTokenHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		type refreshReq struct {
			Token string `json:"token"`
		}
		var r refreshReq
		_ = c.BodyParser(&r)
		token = r.Token
	}
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(app.Response{OK: false, Error: "missing token"})
	}

	req := app.Request{ID: "http-auth-refresh", Action: "auth_refresh", Token: token, Data: []byte("{}")}
	raw, _ := json.Marshal(req)

	resp, err := s.svc.DispatchRequest(string(raw), s.requestTimeout)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(app.Response{OK: false, Error: err.Error()})
	}
	if !resp.OK {
		return c.Status(fiber.StatusUnauthorized).JSON(resp)
	}
	return c.JSON(resp)
}

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

// staffStudentsPageHandler godoc
// @Summary List students with pagination
// @Description Returns a role-scoped, paginated student list for staff users.
// @Tags staff
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Rows per page" default(100)
// @Param group_id query int false "Filter by group ID"
// @Param search query string false "Search by student name or related fields"
// @Param sort query string false "Sort field"
// @Param order query string false "Sort order: asc or desc"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Router /api/staff/overview/students [get]
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

	semesterID, parseErr := optionalSemesterIDFromQuery(c)
	if parseErr != nil {
		return nil, c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: parseErr.Error()})
	}

	report, resp := s.svc.StaffPerformanceReport(token, semesterID)
	if report == nil {
		switch resp.Error {
		case "forbidden: head role or higher required":
			return nil, c.Status(fiber.StatusForbidden).JSON(resp)
		case "invalid token", "session not found", "session revoked", "missing token", "account is not active":
			return nil, c.Status(fiber.StatusUnauthorized).JSON(resp)
		case "semester not found", "open semester not found":
			return nil, c.Status(fiber.StatusNotFound).JSON(resp)
		default:
			return nil, c.Status(fiber.StatusInternalServerError).JSON(resp)
		}
	}
	return report, nil
}

func optionalSemesterIDFromQuery(c *fiber.Ctx) (*int32, error) {
	rawSemesterID := strings.TrimSpace(c.Query("semester_id"))
	if rawSemesterID == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseInt(rawSemesterID, 10, 32)
	if err != nil || parsed <= 0 {
		return nil, fmt.Errorf("invalid semester_id")
	}
	value := int32(parsed)
	return &value, nil
}

func semesterSelectionRequest(c *fiber.Ctx, requestID, action, token string) (app.Request, error) {
	semesterID, err := optionalSemesterIDFromQuery(c)
	if err != nil {
		return app.Request{}, err
	}
	data, err := json.Marshal(app.SemesterSelectionData{SemesterID: semesterID})
	if err != nil {
		return app.Request{}, err
	}
	return app.Request{ID: requestID, Action: action, Token: token, Data: data}, nil
}

// staffPerformanceReportHandler godoc
// @Summary Download performance report as Excel
// @Description Head, dean, or admin downloads an xlsx performance rating report: one sheet for the whole scope plus one sheet per group. Rows are students ranked by overall rating with per-subject percents and attendance.
// @Tags staff
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Security BearerAuth
// @Param semester_id query int false "Semester ID; defaults to the open semester"
// @Param department_id query int false "Department ID for head/dean/admin report; required when several departments are visible"
// @Param subject_id query int false "Subject ID; required for teacher report and optional for department report"
// @Param group_ids query string false "Comma-separated group IDs"
// @Success 200 {file} file "Excel workbook"
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 500 {object} app.Response
// @Router /api/staff/reports/performance.xlsx [get]
func (s *Server) staffPerformanceReportHandler(c *fiber.Ctx) error {
	content, filename, err := s.buildScriptedPerformanceReport(c)
	if content == nil {
		return err
	}

	c.Set(fiber.HeaderContentType, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="`+filename+`"`)
	return c.Send(content)
}

// staffPerformanceReportPDFHandler godoc
// @Summary Download performance report as PDF
// @Description Head, dean, or admin downloads a PDF performance rating report: one page (or more) for the whole scope plus one section per group, colour-coded by percent, matching the xlsx report.
// @Tags staff
// @Produce application/pdf
// @Security BearerAuth
// @Param semester_id query int false "Semester ID; defaults to the open semester"
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

	filename := fmt.Sprintf("performance_semester_%d_%s.pdf", report.SemesterID, report.GeneratedAt.Format("2006-01-02"))
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
		if resp.Error == "invalid token" || resp.Error == "session not found" || resp.Error == "session revoked" || resp.Error == "missing token" || resp.Error == "account is not active" {
			return c.Status(fiber.StatusUnauthorized).JSON(resp)
		}
		if resp.Error == "semester is not open for changes" ||
			resp.Error == "semester has not started" ||
			resp.Error == "semester has ended" {
			return c.Status(fiber.StatusConflict).JSON(resp)
		}
		if resp.Error == "semester not found" || resp.Error == "open semester not found" {
			return c.Status(fiber.StatusNotFound).JSON(resp)
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
		if resp.Error == "invalid token" || resp.Error == "session not found" || resp.Error == "session revoked" || resp.Error == "missing token" || resp.Error == "account is not active" {
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
		if resp.Error == "invalid token" || resp.Error == "session not found" || resp.Error == "session revoked" || resp.Error == "missing token" || resp.Error == "account is not active" {
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

// teacherAttendanceSessionRosterHandler godoc
// @Summary Get the live attendance roster
// @Description Returns every student in a teacher-owned attendance session with the current status and mark time.
// @Tags attendance
// @Produce json
// @Security BearerAuth
// @Param lesson_id query int true "Attendance session ID"
// @Success 200 {object} teacherAttendanceSessionRosterResponse
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Router /api/teacher/attendance/session/roster [get]
func (s *Server) teacherAttendanceSessionRosterHandler(c *fiber.Ctx) error {
	body := app.AttendanceSessionData{LessonID: int32(c.QueryInt("lesson_id", 0))}
	return s.teacherAttendanceReadHandler(c, "http-teacher-attendance-session-roster", "teacher_attendance_session_roster", body)
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

func (s *Server) teacherFinishAttendanceSessionHandler(c *fiber.Ctx) error {
	var body app.AttendanceSessionData
	if len(c.Body()) > 0 {
		_ = c.BodyParser(&body)
	}
	return s.teacherAttendanceReadHandler(c, "http-teacher-finish-attendance-session", "teacher_finish_attendance_session", body)
}

func (s *Server) studentActiveAttendanceSessionHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(app.Response{OK: false, Error: "missing Authorization header"})
	}

	req := app.Request{ID: "http-student-active-attendance-session", Action: "student_active_attendance_session", Token: token, Data: []byte("{}")}
	raw, _ := json.Marshal(req)

	resp, err := s.svc.DispatchRequest(string(raw), s.requestTimeout)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(app.Response{OK: false, Error: err.Error()})
	}
	return c.JSON(resp)
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
		if resp.Error == "invalid token" || resp.Error == "session not found" || resp.Error == "session revoked" || resp.Error == "missing token" || resp.Error == "account is not active" {
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
		if resp.Error == "invalid token" || resp.Error == "session not found" || resp.Error == "session revoked" || resp.Error == "missing token" || resp.Error == "account is not active" {
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
		if resp.Error == "invalid token" || resp.Error == "session not found" || resp.Error == "session revoked" || resp.Error == "missing token" || resp.Error == "account is not active" {
			return c.Status(fiber.StatusUnauthorized).JSON(resp)
		}
		return c.Status(fiber.StatusBadRequest).JSON(resp)
	}

	return c.JSON(resp)
}

// studentAttendanceSummaryHandler godoc
// @Summary Get student attendance summary
// @Description Returns the current student's overall and per-subject attendance percentage for the selected semester. Excused sessions are excluded from the percentage denominator.
// @Tags attendance
// @Produce json
// @Security BearerAuth
// @Param semester_id query int false "Semester ID; defaults to the open semester"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 404 {object} app.Response
// @Failure 500 {object} app.Response
// @Router /api/student/attendance/summary [get]
func (s *Server) studentAttendanceSummaryHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(app.Response{OK: false, Error: "missing Authorization header"})
	}

	req, err := semesterSelectionRequest(c, "http-student-attendance-summary", "student_attendance_summary", token)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: err.Error()})
	}
	raw, err := json.Marshal(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling envelope"})
	}

	resp, err := s.svc.DispatchRequest(string(raw), s.requestTimeout)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(app.Response{OK: false, Error: err.Error()})
	}
	if !resp.OK {
		return c.Status(studentAcademicReadHTTPStatus(resp)).JSON(resp)
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
// @Failure 500 {object} app.Response
// @Router /api/student/schedule/day [get]
func (s *Server) studentScheduleDayHandler(c *fiber.Ctx) error {
	return s.scheduleDayHandler(c, "http-student-schedule-day", "student_schedule_day")
}

// teacherScheduleDayHandler godoc
// @Summary Get teacher's schedule for a day
// @Description Returns the current teacher's lessons and groups for a given date, restricted to the current or next calendar week. Defaults to today if date is omitted.
// @Tags schedule
// @Produce json
// @Security BearerAuth
// @Param date query string false "Date in YYYY-MM-DD format (defaults to today)"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 500 {object} app.Response
// @Router /api/teacher/schedule/day [get]
func (s *Server) teacherScheduleDayHandler(c *fiber.Ctx) error {
	return s.scheduleDayHandler(c, "http-teacher-schedule-day", "teacher_schedule_day")
}

func (s *Server) scheduleDayHandler(c *fiber.Ctx, requestID, action string) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(app.Response{OK: false, Error: "missing Authorization header"})
	}

	body := app.StudentScheduleDayData{Date: c.Query("date", "")}
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
		return c.Status(studentAcademicReadHTTPStatus(resp)).JSON(resp)
	}

	return c.JSON(resp)
}

func studentAcademicReadHTTPStatus(resp app.Response) int {
	if resp.OK {
		return fiber.StatusOK
	}
	switch resp.Error {
	case "invalid token", "session not found", "session revoked", "missing token", "account is not active":
		return fiber.StatusUnauthorized
	case "forbidden: student role required", "forbidden: teacher role required":
		return fiber.StatusForbidden
	case "semester not found", "open semester not found":
		return fiber.StatusNotFound
	case "failed to load attendance summary", "failed to load subjects", "failed to load schedule", "failed to read schedule row", "failed to iterate schedule rows":
		return fiber.StatusInternalServerError
	default:
		return fiber.StatusBadRequest
	}
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

// teacherDeleteGradeHandler godoc
// @Summary Delete student grade
// @Description Soft-deletes a grade in the open semester. An optional reason can be sent in the request body.
// @Tags grades
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param grade_id path int true "Grade ID"
// @Param request body app.GradeDeleteData false "Optional deletion reason; grade_id from the path takes precedence"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 404 {object} app.Response
// @Failure 409 {object} app.Response
// @Router /api/teacher/grades/{grade_id} [delete]
func (s *Server) teacherDeleteGradeHandler(c *fiber.Ctx) error {
	gradeID, err := c.ParamsInt("grade_id")
	if err != nil || gradeID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "invalid grade_id"})
	}
	body := app.GradeDeleteData{GradeID: int32(gradeID)}
	return s.gradeDeleteActionHandler(c, "http-teacher-delete-grade", "teacher_delete_grade", &body, int32(gradeID))
}

// teacherRestoreGradeHandler godoc
// @Summary Restore student grade
// @Description Restores a soft-deleted grade in the open semester.
// @Tags grades
// @Produce json
// @Security BearerAuth
// @Param grade_id path int true "Grade ID"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 404 {object} app.Response
// @Failure 409 {object} app.Response
// @Router /api/teacher/grades/{grade_id}/restore [post]
func (s *Server) teacherRestoreGradeHandler(c *fiber.Ctx) error {
	gradeID, err := c.ParamsInt("grade_id")
	if err != nil || gradeID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "invalid grade_id"})
	}
	body := app.GradeRestoreData{GradeID: int32(gradeID)}
	return s.gradeDeleteActionHandler(c, "http-teacher-restore-grade", "teacher_restore_grade", &body, int32(gradeID))
}

// teacherDeleteGradeItemHandler godoc
// @Summary Delete grade item
// @Description Soft-deletes a grade item in the open semester. Set cascade=true to also delete its active grades.
// @Tags grades
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param item_id path int true "Grade item ID"
// @Param request body app.GradeItemDeleteData false "Deletion options; item_id from the path takes precedence"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 404 {object} app.Response
// @Failure 409 {object} app.Response
// @Router /api/teacher/grades/items/{item_id} [delete]
func (s *Server) teacherDeleteGradeItemHandler(c *fiber.Ctx) error {
	itemID, err := c.ParamsInt("item_id")
	if err != nil || itemID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "invalid item_id"})
	}
	body := app.GradeItemDeleteData{ItemID: int32(itemID)}
	return s.gradeDeleteActionHandler(c, "http-teacher-delete-grade-item", "teacher_delete_grade_item", &body, int32(itemID))
}

// teacherRestoreGradeItemHandler godoc
// @Summary Restore grade item
// @Description Restores a soft-deleted grade item and its cascade-deleted grades in the open semester.
// @Tags grades
// @Produce json
// @Security BearerAuth
// @Param item_id path int true "Grade item ID"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 404 {object} app.Response
// @Failure 409 {object} app.Response
// @Router /api/teacher/grades/items/{item_id}/restore [post]
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
// @Summary Get student performance radar
// @Description Returns one point per subject for the selected semester. Defaults to the open semester.
// @Tags grades
// @Produce json
// @Security BearerAuth
// @Param semester_id query int false "Semester ID; defaults to the open semester"
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

	req, err := semesterSelectionRequest(c, "http-student-performance-radar", "student_performance_radar", token)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: err.Error()})
	}
	raw, err := json.Marshal(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling envelope"})
	}

	resp, err := s.svc.DispatchRequest(string(raw), s.requestTimeout)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(app.Response{OK: false, Error: err.Error()})
	}
	if !resp.OK {
		return c.Status(studentAcademicReadHTTPStatus(resp)).JSON(resp)
	}

	return c.JSON(resp)
}

// studentAllGradesHandler godoc
// @Summary Get all grades for the current student
// @Description Returns every plan subject with its grade items, totals, and attendance percentage for the selected semester. Defaults to the open semester.
// @Tags grades
// @Produce json
// @Security BearerAuth
// @Param semester_id query int false "Semester ID; defaults to the open semester"
// @Success 200 {object} app.Response
// @Failure 400 {object} app.Response
// @Failure 401 {object} app.Response
// @Failure 403 {object} app.Response
// @Failure 404 {object} app.Response
// @Failure 500 {object} app.Response
// @Router /api/student/grades/all [get]
func (s *Server) studentAllGradesHandler(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(app.Response{OK: false, Error: "missing Authorization header"})
	}

	req, err := semesterSelectionRequest(c, "http-student-all-grades", "student_all_grades", token)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: err.Error()})
	}
	raw, err := json.Marshal(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(app.Response{OK: false, Error: "Error marshalling envelope"})
	}

	resp, err := s.svc.DispatchRequest(string(raw), s.requestTimeout)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(app.Response{OK: false, Error: err.Error()})
	}
	if !resp.OK {
		return c.Status(studentAcademicReadHTTPStatus(resp)).JSON(resp)
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
	case "missing token", "invalid token", "session not found", "session revoked", "account is not active":
		return fiber.StatusUnauthorized
	case "forbidden: admin role required":
		return fiber.StatusForbidden
	case "semester not found", "open semester not found":
		return fiber.StatusNotFound
	case "semester already exists",
		"semester date range overlaps an existing semester",
		"invalid semester status transition",
		"semester has active attendance sessions",
		"semester is not open for changes",
		"semester has not started",
		"semester has ended",
		"ended semester cannot be opened":
		return fiber.StatusConflict
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
		if resp.Error == "invalid token" || resp.Error == "session not found" || resp.Error == "session revoked" || resp.Error == "missing token" || resp.Error == "account is not active" {
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

func (s *Server) adminSystemStatsHandler(c *fiber.Ctx) error {
	return s.androidJSONActionHandler(c, "http-admin-system-stats", "admin_system_stats", nil)
}

func (s *Server) adminOrgStructureHandler(c *fiber.Ctx) error {
	return s.androidJSONActionHandler(c, "http-admin-org-structure", "admin_org_structure", nil)
}

func (s *Server) adminRolesListHandler(c *fiber.Ctx) error {
	return s.androidJSONActionHandler(c, "http-admin-roles-list", "admin_roles_list", nil)
}

func (s *Server) adminRoleUpdateHandler(c *fiber.Ctx) error {
	role := c.Params("role")
	var body app.RolePermissions
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "Error parsing body"})
	}
	body.Role = role
	return s.androidJSONActionHandler(c, "http-admin-role-update-"+role, "admin_role_update", &body)
}

func (s *Server) adminAntifraudLogsHandler(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))
	query := app.FraudLogsQuery{
		Page:      int32(page),
		PageSize:  int32(pageSize),
		Search:    c.Query("search"),
		GroupID:   int32(c.QueryInt("group_id", 0)),
		TeacherID: int32(c.QueryInt("teacher_id", 0)),
		Reason:    c.Query("reason"),
		DateFrom:  c.Query("date_from"),
		DateTo:    c.Query("date_to"),
	}
	return s.androidJSONActionHandler(c, "http-admin-antifraud-logs", "admin_antifraud_logs", &query)
}

func (s *Server) adminAntifraudTopCheatersHandler(c *fiber.Ctx) error {
	query := app.FraudLogsQuery{
		Search:    c.Query("search"),
		GroupID:   int32(c.QueryInt("group_id", 0)),
		TeacherID: int32(c.QueryInt("teacher_id", 0)),
		Reason:    c.Query("reason"),
		DateFrom:  c.Query("date_from"),
		DateTo:    c.Query("date_to"),
	}
	return s.androidJSONActionHandler(c, "http-admin-antifraud-top-cheaters", "admin_antifraud_top_cheaters", &query)
}

func (s *Server) adminServicesListHandler(c *fiber.Ctx) error {
	return s.androidJSONActionHandler(c, "http-admin-services-list", "admin_services_list", nil)
}

func (s *Server) adminAuditLogsHandler(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))
	action := c.Query("action", "")
	query := app.AuditLogsQuery{
		Page:     int32(page),
		PageSize: int32(pageSize),
		Action:   action,
	}
	return s.androidJSONActionHandler(c, "http-admin-audit-logs", "admin_audit_logs", &query)
}

func (s *Server) adminSystemMaintenanceGetHandler(c *fiber.Ctx) error {
	return s.androidJSONActionHandler(c, "http-admin-system-maintenance-get", "admin_system_maintenance_get", nil)
}

func (s *Server) adminSystemMaintenanceSetHandler(c *fiber.Ctx) error {
	var body app.MaintenanceStatus
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "Error parsing body"})
	}
	return s.androidJSONActionHandler(c, "http-admin-system-maintenance-set", "admin_system_maintenance_set", &body)
}

func (s *Server) registerDeviceTokenHandler(c *fiber.Ctx) error {
	var body app.DeviceTokenRegistration
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(app.Response{OK: false, Error: "Error parsing body"})
	}
	return s.androidJSONActionHandler(c, "http-register-device-token", "register_device_token", &body)
}

func (s *Server) listDeviceTokensHandler(c *fiber.Ctx) error {
	return s.androidJSONActionHandler(c, "http-list-device-tokens", "list_device_tokens", nil)
}

func (s *Server) deleteDeviceTokenHandler(c *fiber.Ctx) error {
	deviceToken := c.Query("device_token", "")
	if deviceToken == "" {
		var body map[string]string
		_ = c.BodyParser(&body)
		deviceToken = body["device_token"]
	}
	payload := map[string]string{"device_token": deviceToken}
	return s.androidJSONActionHandler(c, "http-delete-device-token", "delete_device_token", &payload)
}
