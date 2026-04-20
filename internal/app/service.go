package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/TelecomDep/ejournal_backend/internal/db"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Request struct {
	ID     string          `json:"id"`
	Action string          `json:"action"`
	Token  string          `json:"token,omitempty"`
	Data   json.RawMessage `json:"data"`
}

type Response struct {
	ID     string `json:"id"`
	OK     bool   `json:"ok"`
	Result any    `json:"result,omitempty"`
	Error  string `json:"error"`
}

type LoginData struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	Role     string `json:"role,omitempty"`
}

type User struct {
	ID     int32
	UserID string
	Login  string
	Pass   string
	Role   string
}

type AttendanceCreateData struct {
	SubjectID      int32   `json:"subject_id"`
	GroupIDs       []int32 `json:"group_ids"`
	LessonName     string  `json:"lesson_name,omitempty"`
	ExpiresMinutes int     `json:"expires_minutes"`
}

type AttendanceConfirmData struct {
	InviteToken string `json:"invite_token"`
}

type AttendanceGroupStatsData struct {
	GroupID   int32  `json:"group_id"`
	SubjectID *int32 `json:"subject_id,omitempty"`
}

type AttendanceInviteClaims struct {
	Type      string `json:"type"`
	LessonID  string `json:"lesson_id"`
	TeacherID string `json:"teacher_id"`
	jwt.RegisteredClaims
}

type requestJob struct {
	rawRequest string
	resultCh   chan Response
}

type Service struct {
	jwtSecret    []byte
	siteBaseURL  string
	store        *db.Store
	requestQueue chan requestJob
}

func normalizeInviteTTL(expiresMinutes int) int {
	if expiresMinutes <= 0 {
		return 15
	}
	if expiresMinutes > 180 {
		return 180
	}
	return expiresMinutes
}

func normalizeGroupIDs(groupIDs []int32) []int32 {
	if len(groupIDs) == 0 {
		return nil
	}

	seen := make(map[int32]struct{}, len(groupIDs))
	result := make([]int32, 0, len(groupIDs))
	for _, groupID := range groupIDs {
		if groupID <= 0 {
			continue
		}
		if _, ok := seen[groupID]; ok {
			continue
		}
		seen[groupID] = struct{}{}
		result = append(result, groupID)
	}
	return result
}

func NewService(jwtSecret, siteBaseURL string, store *db.Store) *Service {
	return &Service{
		jwtSecret:   []byte(strings.TrimSpace(jwtSecret)),
		siteBaseURL: strings.TrimSpace(siteBaseURL),
		store:       store,
	}
}

func (s *Service) StartWorkerPool(workersCount int) {
	s.requestQueue = make(chan requestJob, 1024)
	for i := 0; i < workersCount; i++ {
		go func() {
			for job := range s.requestQueue {
				job.resultCh <- s.handleRequest(job.rawRequest)
			}
		}()
	}
}

func (s *Service) DispatchRequest(raw string, timeout time.Duration) (Response, error) {
	job := requestJob{
		rawRequest: raw,
		resultCh:   make(chan Response, 1),
	}

	select {
	case s.requestQueue <- job:
	case <-time.After(timeout):
		return Response{}, errors.New("server is busy")
	}

	select {
	case resp := <-job.resultCh:
		return resp, nil
	case <-time.After(timeout):
		return Response{}, errors.New("request timeout")
	}
}

func normalizeRole(role string) string {
	role = strings.ToLower(strings.TrimSpace(role))
	switch role {
	case "teacher", "admin":
		return role
	default:
		return "student"
	}
}

func normalizeAuthHeader(token string) string {
	token = strings.TrimSpace(token)
	token = strings.TrimPrefix(token, "Bearer ")
	return strings.TrimSpace(token)
}

func (s *Service) dbContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

func (s *Service) userBySessionToken(token string) (User, error) {
	token = normalizeAuthHeader(token)
	if token == "" {
		return User{}, errors.New("missing token")
	}

	userID, err := s.validateJWT(token)
	if err != nil {
		return User{}, errors.New("invalid token")
	}

	id64, err := strconv.ParseInt(userID, 10, 32)
	if err != nil {
		return User{}, errors.New("invalid token")
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	dbUser, ok, err := s.store.Users.GetByID(ctx, int32(id64))
	if err != nil {
		return User{}, errors.New("session not found")
	}
	if !ok {
		return User{}, errors.New("session not found")
	}

	return User{
		ID:     dbUser.ID,
		UserID: strconv.FormatInt(int64(dbUser.ID), 10),
		Login:  dbUser.Login,
		Pass:   dbUser.PasswordHash,
		Role:   dbUser.Role,
	}, nil
}

func (s *Service) profileByToken(token string) Response {
	user, err := s.userBySessionToken(token)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"user_id": user.UserID,
			"login":   user.Login,
			"role":    user.Role,
		},
	}
}

func (s *Service) generateAttendanceInviteToken(lessonID, teacherID string, expiresMinutes int) (string, time.Time, error) {
	expiresMinutes = normalizeInviteTTL(expiresMinutes)

	exp := time.Now().Add(time.Duration(expiresMinutes) * time.Minute)
	claims := AttendanceInviteClaims{
		Type:      "attendance_invite",
		LessonID:  lessonID,
		TeacherID: teacherID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", time.Time{}, err
	}

	return signed, exp, nil
}

func (s *Service) parseAttendanceInviteToken(inviteToken string) (*AttendanceInviteClaims, error) {
	inviteToken = strings.TrimSpace(inviteToken)
	if inviteToken == "" {
		return nil, errors.New("missing invite token")
	}

	parsed, err := jwt.ParseWithClaims(inviteToken, &AttendanceInviteClaims{}, func(token *jwt.Token) (any, error) {
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, errors.New("invalid invite token")
	}
	if !parsed.Valid {
		return nil, errors.New("invite token is not valid")
	}

	claims, ok := parsed.Claims.(*AttendanceInviteClaims)
	if !ok {
		return nil, errors.New("invalid invite claims")
	}
	if claims.Type != "attendance_invite" {
		return nil, errors.New("wrong invite token type")
	}
	if claims.LessonID == "" || claims.TeacherID == "" {
		return nil, errors.New("invite token payload is incomplete")
	}

	return claims, nil
}

func (s *Service) register(data LoginData) Response {
	login := strings.TrimSpace(data.Login)
	password := strings.TrimSpace(data.Password)
	if login == "" || password == "" {
		return Response{OK: false, Error: "login and password are required"}
	}

	role := normalizeRole(data.Role)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return Response{OK: false, Error: "failed to hash password"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	created, err := s.store.Users.Create(ctx, login, string(hashedPassword), role)
	if err != nil {
		if errors.Is(err, db.ErrUserLoginTaken) {
			return Response{OK: false, Error: "user exist"}
		}
		return Response{OK: false, Error: "failed to create user"}
	}

	switch role {
	case "teacher":
		_, err = s.store.Teachers.Create(ctx, db.Teacher{ID: created.ID, Name: login})
	case "student":
		_, err = s.store.Students.Create(ctx, db.Student{ID: created.ID, StudentName: login})
	}
	if err != nil {
		_ = s.store.Users.DeleteByID(ctx, created.ID)
		return Response{OK: false, Error: "failed to create role profile"}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"user_id": strconv.FormatInt(int64(created.ID), 10),
			"login":   created.Login,
			"role":    created.Role,
		},
	}
}

func (s *Service) login(data LoginData) Response {
	login := strings.TrimSpace(data.Login)
	password := strings.TrimSpace(data.Password)
	if login == "" || password == "" {
		return Response{OK: false, Error: "login and password are required"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	storedUser, ok, err := s.store.Users.GetByLogin(ctx, login)
	if err != nil {
		return Response{OK: false, Error: "failed to read user"}
	}
	if !ok {
		return Response{OK: false, Error: "user does not exist"}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(storedUser.PasswordHash), []byte(password)); err != nil {
		return Response{OK: false, Error: "wrong password"}
	}

	userID := strconv.FormatInt(int64(storedUser.ID), 10)
	token, err := s.generateJWT(userID)
	if err != nil {
		return Response{OK: false, Error: "EROR_generateJWT: " + err.Error()}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"token":   token,
			"user_ID": userID,
			"login":   storedUser.Login,
			"role":    storedUser.Role,
		},
	}
}

func (s *Service) createAttendanceLinkByTeacher(sessionToken string, data AttendanceCreateData) Response {
	teacher, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if teacher.Role != "teacher" {
		return Response{OK: false, Error: "forbidden: teacher role required"}
	}
	if data.SubjectID <= 0 {
		return Response{OK: false, Error: "subject_id is required"}
	}
	groupIDs := normalizeGroupIDs(data.GroupIDs)
	if len(groupIDs) == 0 {
		return Response{OK: false, Error: "group_ids is required"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	subject, found, err := s.store.Subjects.GetByID(ctx, data.SubjectID)
	if err != nil {
		return Response{OK: false, Error: "failed to load subject"}
	}
	if !found {
		return Response{OK: false, Error: "subject not found"}
	}
	for _, groupID := range groupIDs {
		_, found, err = s.store.Groups.GetByID(ctx, groupID)
		if err != nil {
			return Response{OK: false, Error: "failed to load group"}
		}
		if !found {
			return Response{OK: false, Error: "group not found"}
		}
	}

	effectiveTTL := normalizeInviteTTL(data.ExpiresMinutes)
	expiresAt := time.Now().Add(time.Duration(effectiveTTL) * time.Minute)
	session, rosterSize, err := s.store.Attendance.CreateSessionWithGroups(ctx, teacher.ID, subject.ID, groupIDs, expiresAt)
	if err != nil {
		return Response{OK: false, Error: "failed to create attendance session"}
	}

	lessonID := strconv.FormatInt(int64(session.ID), 10)
	inviteToken, signedExpiresAt, err := s.generateAttendanceInviteToken(lessonID, teacher.UserID, effectiveTTL)
	if err != nil {
		return Response{OK: false, Error: "failed to generate invite token"}
	}

	joinURL := fmt.Sprintf("%s/attendance/join?token=%s", strings.TrimRight(s.siteBaseURL, "/"), inviteToken)
	lessonName := strings.TrimSpace(data.LessonName)
	if lessonName == "" {
		lessonName = subject.Name
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"lesson_id":       lessonID,
			"subject_id":      subject.ID,
			"lesson_name":     lessonName,
			"invite_token":    inviteToken,
			"url":             joinURL,
			"join_url":        joinURL,
			"qr_payload":      joinURL,
			"group_ids":       groupIDs,
			"roster_size":     rosterSize,
			"teacher_id":      teacher.UserID,
			"expires_at":      signedExpiresAt.UTC().Format(time.RFC3339),
			"expires_minutes": effectiveTTL,
		},
	}
}

func (s *Service) confirmAttendanceByStudent(sessionToken string, data AttendanceConfirmData) Response {
	student, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if student.Role != "student" {
		return Response{OK: false, Error: "forbidden: student role required"}
	}

	claims, err := s.parseAttendanceInviteToken(data.InviteToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	sessionID64, err := strconv.ParseInt(claims.LessonID, 10, 32)
	if err != nil {
		return Response{OK: false, Error: "invalid invite token"}
	}
	teacherID64, err := strconv.ParseInt(claims.TeacherID, 10, 32)
	if err != nil {
		return Response{OK: false, Error: "invalid invite token"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	session, found, err := s.store.Attendance.GetSessionByID(ctx, int32(sessionID64))
	if err != nil {
		return Response{OK: false, Error: "failed to load attendance session"}
	}
	if !found {
		return Response{OK: false, Error: "attendance session not found"}
	}
	if session.TeacherID != int32(teacherID64) {
		return Response{OK: false, Error: "invite token is not valid"}
	}
	if time.Now().UTC().After(session.ExpiresAt.UTC()) {
		return Response{OK: false, Error: "invite token expired"}
	}

	markedAt := time.Now().UTC()
	markResult, err := s.store.Attendance.MarkStudentPresent(ctx, session.ID, student.ID, markedAt)
	if err != nil {
		return Response{OK: false, Error: "failed to confirm attendance"}
	}
	if markResult == "not_found" {
		return Response{OK: false, Error: "forbidden: student is not in session roster"}
	}
	if markResult == "already" {
		return Response{OK: false, Error: "attendance already confirmed"}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"lesson_id":  claims.LessonID,
			"student_id": student.UserID,
			"teacher_id": claims.TeacherID,
			"subject_id": session.SubjectID,
			"marked_at":  markedAt.Format(time.RFC3339),
			"attendance": "confirmed",
		},
	}
}

func (s *Service) attendanceByGroupForTeacher(sessionToken string, data AttendanceGroupStatsData) Response {
	teacher, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if teacher.Role != "teacher" {
		return Response{OK: false, Error: "forbidden: teacher role required"}
	}
	if data.GroupID <= 0 {
		return Response{OK: false, Error: "group_id is required"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	_, found, err := s.store.Groups.GetByID(ctx, data.GroupID)
	if err != nil {
		return Response{OK: false, Error: "failed to load group"}
	}
	if !found {
		return Response{OK: false, Error: "group not found"}
	}

	stats, err := s.store.Attendance.GetTeacherGroupAttendanceStats(ctx, teacher.ID, data.GroupID, data.SubjectID)
	if err != nil {
		return Response{OK: false, Error: "failed to load attendance stats"}
	}

	students := make([]map[string]any, 0, len(stats))
	var sessionsCount int32
	for _, row := range stats {
		var lastMarkedAt any
		if row.LastMarkedAt != nil {
			lastMarkedAt = row.LastMarkedAt.UTC().Format(time.RFC3339)
		}
		attendancePercent := 0.0
		if row.TotalSessions > 0 {
			attendancePercent = float64(row.AttendedSessions) * 100 / float64(row.TotalSessions)
		}
		if row.TotalSessions > sessionsCount {
			sessionsCount = row.TotalSessions
		}

		students = append(students, map[string]any{
			"student_id":         row.StudentID,
			"student_name":       row.StudentName,
			"total_sessions":     row.TotalSessions,
			"attended_sessions":  row.AttendedSessions,
			"attendance_percent": attendancePercent,
			"last_marked_at":     lastMarkedAt,
		})
	}

	result := map[string]any{
		"group_id": data.GroupID,
		"students": students,
		"summary": map[string]any{
			"students_count": len(students),
			"sessions_count": sessionsCount,
		},
	}
	if data.SubjectID != nil {
		result["subject_id"] = *data.SubjectID
	}

	return Response{
		OK:     true,
		Result: result,
	}
}

func (s *Service) handleRequest(raw string) Response {
	var req Request
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		return Response{OK: false, Error: "EROR: " + err.Error()}
	}

	switch req.Action {
	case "ping":
		return Response{
			ID:     req.ID,
			OK:     true,
			Result: map[string]any{"pong": true},
		}
	case "register":
		var data LoginData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "EROR reg: " + err.Error()}
		}
		resp := s.register(data)
		resp.ID = req.ID
		return resp
	case "login":
		var data LoginData
		err := json.Unmarshal(req.Data, &data)
		if err != nil {
			return Response{ID: req.ID, OK: false, Error: "EROR_login: " + err.Error()}
		}
		resp := s.login(data)
		resp.ID = req.ID
		return resp
	case "profile":
		resp := s.profileByToken(req.Token)
		resp.ID = req.ID
		return resp
	case "create_attendance_link":
		var data AttendanceCreateData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid create_attendance_link payload"}
		}
		resp := s.createAttendanceLinkByTeacher(req.Token, data)
		resp.ID = req.ID
		return resp
	case "confirm_attendance":
		var data AttendanceConfirmData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid confirm_attendance payload"}
		}
		resp := s.confirmAttendanceByStudent(req.Token, data)
		resp.ID = req.ID
		return resp
	case "teacher_attendance_by_group":
		var data AttendanceGroupStatsData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid teacher_attendance_by_group payload"}
		}
		resp := s.attendanceByGroupForTeacher(req.Token, data)
		resp.ID = req.ID
		return resp
	default:
		return Response{ID: req.ID, OK: false, Error: "unknown_action: " + req.Action}
	}
}

func (s *Service) generateJWT(userID string) (string, error) {
	cl := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 12).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, cl)
	return token.SignedString(s.jwtSecret)
}

func (s *Service) validateJWT(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		return s.jwtSecret, nil
	})
	if err != nil {
		return "", err
	}

	if !token.Valid {
		return "", errors.New("token is not valid")
	}

	cl, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("claims type is invalid")
	}

	if userID, ok := cl["user_id"].(string); ok {
		return userID, nil
	}
	if userID, ok := cl["user_id"].(float64); ok {
		return strconv.FormatInt(int64(userID), 10), nil
	}

	return "", fmt.Errorf("no user id found in claims")
}
