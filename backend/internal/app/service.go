package app

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/TelecomDep/ejournal_backend/internal/db"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

type Request struct {
	ID     string          `json:"id"`
	Action string          `json:"action"`
	Token  string          `json:"token,omitempty"`
	Data   json.RawMessage `json:"data"`
	Meta   *RequestMeta    `json:"meta,omitempty"`
}

type RequestMeta struct {
	IP        string `json:"ip,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
}

type Response struct {
	ID     string `json:"id"`
	OK     bool   `json:"ok"`
	Result any    `json:"result,omitempty"`
	Error  string `json:"error"`
}

type LoginData struct {
	Login     string `json:"login"`
	Password  string `json:"password"`
	TwoFaCode string `json:"two_fa_code,omitempty"`
}

type RegisterData struct {
	Login      string `json:"login"`
	Password   string `json:"password"`
	InviteCode string `json:"invite_code,omitempty"`
	RoleHash   string `json:"role_hash,omitempty"`
	Role       string `json:"role,omitempty"`
}

type RegisterByInviteData struct {
	InviteCode string `json:"invite_code"`
	Login      string `json:"login"`
	Password   string `json:"password"`
}

type ForgotPasswordData struct {
	Identity string `json:"identity"`
}

type ResetPasswordData struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

type UpdateEmailData struct {
	Email string `json:"email"`
}

type User struct {
	ID     int32
	UserID string
	Login  string
	Pass   string
	Role   string
	Email  string
}

type AttendanceCreateData struct {
	LessonID       int32   `json:"lesson_id,omitempty"`
	SubjectID      int32   `json:"subject_id,omitempty"`
	SemesterID     *int32  `json:"semester_id,omitempty"`
	GroupIDs       []int32 `json:"group_ids,omitempty"`
	LessonName     string  `json:"lesson_name,omitempty"`
	ExpiresMinutes int     `json:"expires_minutes,omitempty"`
}

type AttendanceConfirmData struct {
	InviteToken string `json:"invite_token"`
}

type AttendanceGroupStatsData struct {
	GroupID    int32  `json:"group_id"`
	SubjectID  *int32 `json:"subject_id,omitempty"`
	SemesterID *int32 `json:"semester_id,omitempty"`
}

type GroupPerformanceData struct {
	GroupID    int32  `json:"group_id"`
	SubjectID  int32  `json:"subject_id"`
	SemesterID *int32 `json:"semester_id,omitempty"`
}

type AttendanceSessionData struct {
	LessonID int32 `json:"lesson_id"`
}

type TeacherAttendanceMarkData struct {
	LessonID  int32  `json:"lesson_id"`
	StudentID int32  `json:"student_id"`
	Status    string `json:"status"`
}

type AttendanceHistoryData struct {
	Year int `json:"year"`
}

type TeacherAttendanceStudentHistoryData struct {
	StudentID  int32  `json:"student_id"`
	SubjectID  int32  `json:"subject_id"`
	SemesterID *int32 `json:"semester_id,omitempty"`
}

type TeacherSubjectsResultItem struct {
	SubjectID   int32                 `json:"subject_id"`
	SubjectName string                `json:"subject_name"`
	GroupIDs    []int32               `json:"group_ids"`
	Groups      []TeacherSubjectGroup `json:"groups"`
}

type TeacherSubjectGroup struct {
	ID   int32  `json:"id"`
	Name string `json:"name"`
}

type AttendanceInviteClaims struct {
	Type      string `json:"type"`
	LessonID  string `json:"lesson_id"`
	TeacherID string `json:"teacher_id"`
	jwt.RegisteredClaims
}

type SessionClaims struct {
	UserID       int32 `json:"user_id"`
	TokenVersion int64 `json:"token_version"`
	jwt.RegisteredClaims
}

type TeacherNearestLesson struct {
	SubjectID int32
	LessonNum int32
	GroupIDs  []int32
	StartAt   time.Time
	EndAt     time.Time
}

type requestJob struct {
	rawRequest string
	resultCh   chan Response
}

type Service struct {
	jwtSecret            []byte
	siteBaseURL          string
	roleHashTeacher      string
	roleHashStudent      string
	defaultGroupID       int32
	allowEarlyAttendance bool
	store                *db.Store
	mailer               *Mailer
	requestQueue         chan requestJob
}

type RuntimeStats struct {
	QueueLength        int                 `json:"queue_length"`
	QueueCapacity      int                 `json:"queue_capacity"`
	DBMaxConnections   int32               `json:"db_max_connections"`
	DBTotalConnections int32               `json:"db_total_connections"`
	DBAcquired         int32               `json:"db_acquired_connections"`
	DBIdle             int32               `json:"db_idle_connections"`
	DBEmptyAcquires    int64               `json:"db_empty_acquires"`
	DBQueries          db.QueryTimingStats `json:"db_queries"`
}

var appTimeLocation = loadAppTimeLocation()

var dummyPasswordHash = func() []byte {
	hash, _ := bcrypt.GenerateFromPassword([]byte("invalid-password-placeholder"), bcrypt.DefaultCost)
	return hash
}()

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

func normalizeInviteCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func normalizeRoleHash(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func loadAppTimeLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Novosibirsk")
	if err != nil {
		return time.FixedZone("Asia/Novosibirsk", 7*60*60)
	}
	return loc
}

func formatAPITime(ts time.Time) string {
	return ts.In(appTimeLocation).Format(time.RFC3339)
}

func parseAPITime(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, fmt.Errorf("empty time value")
	}
	ts, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, err
	}
	return ts, nil
}

func weekdayToDayIdx(weekday time.Weekday) int32 {
	if weekday == time.Sunday {
		return 7
	}
	return int32(weekday)
}

func weekTypeByISOParity(ts time.Time) int32 {
	_, week := ts.ISOWeek()
	if week%2 == 0 {
		return 2
	}
	return 1
}

func containsAllGroupIDs(fromSchedule, requested []int32) bool {
	if len(requested) == 0 {
		return true
	}

	index := make(map[int32]struct{}, len(fromSchedule))
	for _, groupID := range fromSchedule {
		index[groupID] = struct{}{}
	}

	for _, groupID := range requested {
		if _, ok := index[groupID]; !ok {
			return false
		}
	}
	return true
}

func NewService(jwtSecret, siteBaseURL, roleHashTeacher, roleHashStudent string, defaultGroupID int32, allowEarlyAttendance bool, store *db.Store, mailer *Mailer) *Service {
	return &Service{
		jwtSecret:            []byte(strings.TrimSpace(jwtSecret)),
		siteBaseURL:          strings.TrimSpace(siteBaseURL),
		roleHashTeacher:      normalizeRoleHash(roleHashTeacher),
		roleHashStudent:      normalizeRoleHash(roleHashStudent),
		defaultGroupID:       defaultGroupID,
		allowEarlyAttendance: allowEarlyAttendance,
		store:                store,
		mailer:               mailer,
	}
}

func (s *Service) resolveRoleByHash(roleHash string) (string, bool) {
	roleHash = normalizeRoleHash(roleHash)
	switch roleHash {
	case s.roleHashTeacher:
		return "teacher", true
	case s.roleHashStudent:
		return "student", true
	default:
		return "", false
	}
}

func (s *Service) StartWorkerPool(workersCount int) {
	s.requestQueue = make(chan requestJob, 1024)
	for i := 0; i < workersCount; i++ {
		go func() {
			for job := range s.requestQueue {
				job.resultCh <- s.safeHandleRequest(job.rawRequest)
			}
		}()
	}
}

func (s *Service) safeHandleRequest(raw string) (resp Response) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("recovered request worker panic: %v", recovered)
			resp = Response{OK: false, Error: "internal server error"}
		}
	}()
	return s.handleRequest(raw)
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

func (s *Service) Ping(ctx context.Context) error {
	return s.store.Ping(ctx)
}

func (s *Service) RuntimeStats() RuntimeStats {
	stats := RuntimeStats{}
	if s.requestQueue != nil {
		stats.QueueLength = len(s.requestQueue)
		stats.QueueCapacity = cap(s.requestQueue)
	}
	if s.store == nil || s.store.Pool() == nil {
		return stats
	}
	poolStats := s.store.Pool().Stat()
	stats.DBMaxConnections = poolStats.MaxConns()
	stats.DBTotalConnections = poolStats.TotalConns()
	stats.DBAcquired = poolStats.AcquiredConns()
	stats.DBIdle = poolStats.IdleConns()
	stats.DBEmptyAcquires = poolStats.EmptyAcquireCount()
	stats.DBQueries = s.store.QueryTimingStats()
	return stats
}

func normalizeRole(role string) string {
	role = strings.ToLower(strings.TrimSpace(role))
	if IsValidRole(role) {
		return role
	}
	return RoleStudent
}

func normalizeAuthHeader(token string) string {
	token = strings.TrimSpace(token)
	parts := strings.Fields(token)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return token
}

func (s *Service) dbContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

func (s *Service) userBySessionToken(token string) (User, error) {
	token = normalizeAuthHeader(token)
	if token == "" {
		return User{}, errors.New("missing token")
	}

	userID, tokenVersion, err := s.validateJWT(token)
	if err != nil {
		return User{}, errors.New("invalid token")
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	dbUser, ok, err := s.store.Users.GetByID(ctx, userID)
	if err != nil {
		return User{}, errors.New("session not found")
	}
	if !ok {
		return User{}, errors.New("session not found")
	}
	if dbUser.Status != "active" {
		return User{}, errors.New("account is not active")
	}
	var currentTokenVersion int64
	if err := s.store.Pool().QueryRow(ctx, `SELECT token_version FROM users WHERE id = $1`, userID).Scan(&currentTokenVersion); err != nil {
		return User{}, errors.New("session not found")
	}
	if tokenVersion != currentTokenVersion {
		return User{}, errors.New("session revoked")
	}

	var emailStr string
	if dbUser.Email != nil {
		emailStr = *dbUser.Email
	}

	return User{
		ID:     dbUser.ID,
		UserID: strconv.FormatInt(int64(dbUser.ID), 10),
		Login:  dbUser.Login,
		Pass:   dbUser.PasswordHash,
		Role:   dbUser.Role,
		Email:  emailStr,
	}, nil
}

func (s *Service) teacherProfileByUser(user User) (db.Teacher, error) {
	ctx, cancel := s.dbContext()
	defer cancel()

	teacher, found, err := s.store.Teachers.GetByUserID(ctx, user.ID)
	if err != nil {
		return db.Teacher{}, errors.New("failed to load teacher profile")
	}
	if found {
		return teacher, nil
	}

	// Backward compatibility for legacy rows where teacher_id == users.id.
	teacher, found, err = s.store.Teachers.GetByID(ctx, user.ID)
	if err != nil {
		return db.Teacher{}, errors.New("failed to load teacher profile")
	}
	if found {
		return teacher, nil
	}

	return db.Teacher{}, errors.New("teacher profile not found")
}

func (s *Service) nearestLessonForTeacher(ctx context.Context, teacherID int32, nowLocal time.Time) (TeacherNearestLesson, bool, error) {
	for dayOffset := 0; dayOffset < 14; dayOffset++ {
		lessonDate := nowLocal.AddDate(0, 0, dayOffset)
		dayIdx := weekdayToDayIdx(lessonDate.Weekday())
		weekType := weekTypeByISOParity(lessonDate)

		var fromTime any
		if dayOffset == 0 {
			fromTime = nowLocal.Format("15:04:05")
		}

		var nearest TeacherNearestLesson
		var startClock time.Time
		var endClock time.Time
		err := s.store.Pool().QueryRow(
			ctx,
			`SELECT s.subject_id,
			        s.lesson_num,
			        COALESCE(ARRAY_REMOVE(ARRAY_AGG(DISTINCT s.group_id), NULL), '{}')::INTEGER[] AS group_ids,
			        lt.start_time,
			        lt.end_time
			 FROM schedules s
			 JOIN lesson_times lt ON lt.lesson_num = s.lesson_num
			 WHERE s.teacher_id = $1
			   AND s.day_idx = $2
			   AND COALESCE(s.week_type, $3) = $3
			   AND ($4::time IS NULL OR lt.end_time >= $4::time)
			   AND s.semester_id = (
			       SELECT semester_id FROM semesters WHERE status = 'open' LIMIT 1
			   )
			 GROUP BY s.subject_id, s.lesson_num, lt.start_time, lt.end_time
			 ORDER BY
			     CASE
			         WHEN $4::time IS NOT NULL
			              AND lt.start_time <= $4::time
			              AND lt.end_time >= $4::time THEN 0
			         ELSE 1
			     END,
			     lt.start_time
			 LIMIT 1`,
			teacherID,
			dayIdx,
			weekType,
			fromTime,
		).Scan(
			&nearest.SubjectID,
			&nearest.LessonNum,
			&nearest.GroupIDs,
			&startClock,
			&endClock,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return TeacherNearestLesson{}, false, fmt.Errorf("load nearest lesson: %w", err)
		}

		nearest.StartAt = time.Date(
			lessonDate.Year(),
			lessonDate.Month(),
			lessonDate.Day(),
			startClock.Hour(),
			startClock.Minute(),
			startClock.Second(),
			0,
			appTimeLocation,
		)
		nearest.EndAt = time.Date(
			lessonDate.Year(),
			lessonDate.Month(),
			lessonDate.Day(),
			endClock.Hour(),
			endClock.Minute(),
			endClock.Second(),
			0,
			appTimeLocation,
		)

		return nearest, true, nil
	}

	return TeacherNearestLesson{}, false, nil
}

func (s *Service) profileByToken(token string) Response {
	user, err := s.userBySessionToken(token)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	result := map[string]any{
		"user_id": user.UserID,
		"user_ID": user.UserID,
		"login":   user.Login,
		"role":    user.Role,
		"email":   user.Email,
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	var avatarURL sql.NullString
	if err = s.store.Pool().QueryRow(ctx, `SELECT avatar_url FROM users WHERE id = $1`, user.ID).Scan(&avatarURL); err == nil && avatarURL.Valid {
		result["avatar"] = avatarURL.String
	}

	switch user.Role {
	case "student":
		var studentID int32
		var studentName sql.NullString
		var groupID sql.NullInt32
		var groupName sql.NullString
		var nfcID sql.NullString
		var totalCheatAttempts int32
		err = s.store.Pool().QueryRow(
			ctx,
			`SELECT s.student_id, s.student_name, s.group_id, g.group_name, s.nfc_id, s.total_cheat_attempts
			 FROM students s
			 LEFT JOIN groups g ON g.group_id = s.group_id
			 WHERE s.user_id = $1 OR s.student_id = $1
			 ORDER BY CASE WHEN s.user_id = $1 THEN 0 ELSE 1 END
			 LIMIT 1`,
			user.ID,
		).Scan(&studentID, &studentName, &groupID, &groupName, &nfcID, &totalCheatAttempts)
		if err == nil {
			result["student_id"] = studentID
			result["total_cheat_attempts"] = totalCheatAttempts
			if studentName.Valid {
				result["name"] = studentName.String
				result["student_name"] = studentName.String
			}
			if groupID.Valid {
				result["group_id"] = groupID.Int32
			}
			if groupName.Valid {
				result["group_name"] = groupName.String
				result["group"] = groupName.String
			}
			if nfcID.Valid {
				result["nfc_tag"] = nfcID.String
			}
		}
	case "teacher":
		var teacherID int32
		var teacherName sql.NullString
		var lecternID sql.NullInt32
		var jobTitle sql.NullString
		err = s.store.Pool().QueryRow(
			ctx,
			`SELECT teacher_id, name, lectern_id, job_title
			 FROM teachers
			 WHERE user_id = $1 OR teacher_id = $1
			 ORDER BY CASE WHEN user_id = $1 THEN 0 ELSE 1 END
			 LIMIT 1`,
			user.ID,
		).Scan(&teacherID, &teacherName, &lecternID, &jobTitle)
		if err == nil {
			result["teacher_id"] = teacherID
			if teacherName.Valid {
				result["name"] = teacherName.String
				result["teacher_name"] = teacherName.String
			}
			if lecternID.Valid {
				result["lectern_id"] = lecternID.Int32
			}
			if jobTitle.Valid {
				result["job_title"] = jobTitle.String
			}
		}
	}

	return Response{OK: true, Result: result}
}

func (s *Service) generateAttendanceInviteToken(lessonID, teacherID string, expiresMinutes int) (string, time.Time, error) {
	expiresMinutes = normalizeInviteTTL(expiresMinutes)

	return s.generateAttendanceInviteTokenUntil(lessonID, teacherID, time.Now().Add(time.Duration(expiresMinutes)*time.Minute))
}

func (s *Service) generateAttendanceInviteTokenUntil(lessonID, teacherID string, exp time.Time) (string, time.Time, error) {
	claims := AttendanceInviteClaims{
		Type:      "attendance_invite",
		LessonID:  lessonID,
		TeacherID: teacherID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "ejournal",
			Audience:  jwt.ClaimStrings{"ejournal-attendance"},
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-5 * time.Second)),
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

	parsed, err := jwt.ParseWithClaims(
		inviteToken,
		&AttendanceInviteClaims{},
		func(token *jwt.Token) (any, error) { return s.jwtSecret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer("ejournal"),
		jwt.WithAudience("ejournal-attendance"),
		jwt.WithExpirationRequired(),
	)
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

func (s *Service) register(data RegisterData) Response {
	if strings.TrimSpace(data.InviteCode) != "" {
		return s.registerByInvite(RegisterByInviteData{
			InviteCode: data.InviteCode,
			Login:      data.Login,
			Password:   data.Password,
		})
	}

	login := strings.TrimSpace(data.Login)
	password := strings.TrimSpace(data.Password)
	if login == "" || password == "" {
		return Response{OK: false, Error: "login and password are required"}
	}
	if len(password) < 8 {
		return Response{OK: false, Error: "password must be at least 8 characters long"}
	}

	role, ok := s.resolveRoleByHash(data.RoleHash)
	if !ok {
		return Response{OK: false, Error: "invalid role_hash"}
	}

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
		userID := created.ID
		_, err = s.store.Teachers.Create(ctx, db.Teacher{UserID: &userID, Name: login})
	case "student":
		var groupID *int32
		if s.defaultGroupID > 0 {
			_, foundGroup, groupErr := s.store.Groups.GetByID(ctx, s.defaultGroupID)
			if groupErr == nil && foundGroup {
				groupID = &s.defaultGroupID
			}
		}
		_, err = s.store.Students.Create(ctx, db.Student{ID: created.ID, StudentName: login, GroupID: groupID})
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

func (s *Service) registerByInvite(data RegisterByInviteData) Response {
	inviteCode := normalizeInviteCode(data.InviteCode)
	login := strings.TrimSpace(data.Login)
	password := strings.TrimSpace(data.Password)

	if inviteCode == "" {
		return Response{OK: false, Error: "invite_code is required"}
	}
	if login == "" || password == "" {
		return Response{OK: false, Error: "login and password are required"}
	}
	if len(password) < 8 {
		return Response{OK: false, Error: "password must be at least 8 characters long"}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return Response{OK: false, Error: "failed to hash password"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	tx, err := s.store.Pool().BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Response{OK: false, Error: "failed to start registration transaction"}
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var inviteID int32
	var inviteRole string
	var studentID sql.NullInt32
	var teacherID sql.NullInt32
	err = tx.QueryRow(
		ctx,
		`SELECT invite_id, role::text, student_id, teacher_id
		 FROM registration_invites
		 WHERE used_at IS NULL
		   AND invite_code_hash = crypt($1, invite_code_hash)
		 FOR UPDATE
		 LIMIT 1`,
		inviteCode,
	).Scan(&inviteID, &inviteRole, &studentID, &teacherID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Response{OK: false, Error: "invalid or used invite_code"}
	}
	if err != nil {
		return Response{OK: false, Error: "failed to validate invite_code"}
	}
	switch inviteRole {
	case "student", "teacher", "admin":
	default:
		return Response{OK: false, Error: "invalid invite role"}
	}

	var created db.User
	err = tx.QueryRow(
		ctx,
		`INSERT INTO users (login, password_hash, role)
		 VALUES ($1, $2, $3)
		 RETURNING id, login, password_hash, role, created_at`,
		login,
		string(hashedPassword),
		inviteRole,
	).Scan(&created.ID, &created.Login, &created.PasswordHash, &created.Role, &created.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Response{OK: false, Error: "user exist"}
		}
		return Response{OK: false, Error: "failed to create user"}
	}

	switch inviteRole {
	case "student":
		if !studentID.Valid {
			return Response{OK: false, Error: "invalid invite configuration"}
		}
		cmd, err := tx.Exec(
			ctx,
			`UPDATE students
			 SET user_id = $2
			 WHERE student_id = $1
			   AND user_id IS NULL`,
			studentID.Int32,
			created.ID,
		)
		if err != nil {
			return Response{OK: false, Error: "failed to bind student profile"}
		}
		if cmd.RowsAffected() == 0 {
			return Response{OK: false, Error: "failed to bind student profile"}
		}
	case "teacher":
		if !teacherID.Valid {
			return Response{OK: false, Error: "invalid invite configuration"}
		}
		cmd, err := tx.Exec(
			ctx,
			`UPDATE teachers
			 SET user_id = $2
			 WHERE teacher_id = $1
			   AND user_id IS NULL`,
			teacherID.Int32,
			created.ID,
		)
		if err != nil {
			return Response{OK: false, Error: "failed to bind teacher profile"}
		}
		if cmd.RowsAffected() == 0 {
			return Response{OK: false, Error: "failed to bind teacher profile"}
		}
	case "admin":
		// Admin invites do not bind to a profile table.
	default:
		return Response{OK: false, Error: "invalid invite role"}
	}

	cmd, err := tx.Exec(
		ctx,
		`UPDATE registration_invites
		 SET used_at = NOW()
		 WHERE invite_id = $1
		   AND used_at IS NULL`,
		inviteID,
	)
	if err != nil {
		return Response{OK: false, Error: "failed to mark invite as used"}
	}
	if cmd.RowsAffected() == 0 {
		return Response{OK: false, Error: "invalid or used invite_code"}
	}

	if err = tx.Commit(ctx); err != nil {
		return Response{OK: false, Error: "failed to commit registration"}
	}

	userID := strconv.FormatInt(int64(created.ID), 10)
	token, err := s.generateJWT(created.ID, 1)
	if err != nil {
		return Response{OK: false, Error: "EROR_generateJWT: " + err.Error()}
	}

	resultResp := s.profileByToken(token)
	if !resultResp.OK {
		return Response{OK: false, Error: resultResp.Error}
	}

	result := map[string]any{
		"token":   token,
		"user_ID": userID,
	}
	if profileResult, ok := resultResp.Result.(map[string]any); ok {
		for key, value := range profileResult {
			result[key] = value
		}
	}
	if _, ok := result["user_id"]; !ok {
		result["user_id"] = userID
	}
	if _, ok := result["login"]; !ok {
		result["login"] = created.Login
	}
	if _, ok := result["role"]; !ok {
		result["role"] = created.Role
	}

	return Response{OK: true, Result: result}
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
		_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(password))
		return Response{OK: false, Error: "invalid credentials"}
	}

	if !s.passwordMatches(ctx, storedUser.PasswordHash, password) || storedUser.Status != "active" {
		return Response{OK: false, Error: "invalid credentials"}
	}

	var is2faEnabled bool
	var totpSecret string
	var tokenVersion int64
	if err := s.store.Pool().QueryRow(ctx, `
		SELECT is_2fa_enabled, COALESCE(totp_secret, ''), token_version
		FROM users WHERE id = $1
	`, storedUser.ID).Scan(&is2faEnabled, &totpSecret, &tokenVersion); err != nil {
		return Response{OK: false, Error: "failed to read user security settings"}
	}

	if is2faEnabled {
		if data.TwoFaCode == "" {
			pushSent, _ := s.DispatchTOTPPushNotification(ctx, storedUser.ID, totpSecret, "")
			return Response{
				OK:    false,
				Error: "requires_2fa",
				Result: map[string]any{
					"push_sent": pushSent,
					"message":   "Enter the current code from your authenticator app",
				},
			}
		}
		if !totp.Validate(data.TwoFaCode, totpSecret) {
			return Response{OK: false, Error: "invalid 2fa code"}
		}
	}

	userID := strconv.FormatInt(int64(storedUser.ID), 10)
	token, err := s.generateJWT(storedUser.ID, tokenVersion)
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

func (s *Service) passwordMatches(ctx context.Context, storedHash, password string) bool {
	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)); err == nil {
		return true
	}

	var matches bool
	err := s.store.Pool().QueryRow(
		ctx,
		`SELECT crypt($1, $2) = $2`,
		password,
		storedHash,
	).Scan(&matches)
	return err == nil && matches
}

func buildAttendanceJoinURL(siteBaseURL, inviteToken string) string {
	return fmt.Sprintf("%s/#/attendance/join?token=%s", strings.TrimRight(siteBaseURL, "/"), inviteToken)
}

func (s *Service) createAttendanceLinkByTeacher(sessionToken string, data AttendanceCreateData) Response {
	if data.LessonID > 0 {
		return s.attendanceLinkForExistingLessonByTeacher(sessionToken, data.LessonID)
	}

	teacherUser, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if teacherUser.Role != "teacher" {
		return Response{OK: false, Error: "forbidden: teacher role required"}
	}
	teacherProfile, err := s.teacherProfileByUser(teacherUser)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	nowLocal := time.Now().In(appTimeLocation)
	nearestLesson, found, err := s.nearestLessonForTeacher(ctx, teacherProfile.ID, nowLocal)
	if err != nil {
		return Response{OK: false, Error: "failed to load nearest lesson"}
	}
	if !found {
		return Response{OK: false, Error: "no scheduled lessons found for teacher"}
	}
	if !s.allowEarlyAttendance && nowLocal.Before(nearestLesson.StartAt.Add(-15*time.Minute)) {
		return Response{
			OK: false,
			Error: fmt.Sprintf(
				"attendance can be started no earlier than 15 minutes before class start (%s)",
				formatAPITime(nearestLesson.StartAt),
			),
		}
	}

	requestedSubjectID := data.SubjectID
	if requestedSubjectID <= 0 && found {
		requestedSubjectID = nearestLesson.SubjectID
	}
	if requestedSubjectID <= 0 {
		return Response{OK: false, Error: "subject_id is required"}
	}
	if requestedSubjectID != nearestLesson.SubjectID {
		return Response{OK: false, Error: "subject_id does not match nearest scheduled lesson"}
	}

	groupIDs := normalizeGroupIDs(data.GroupIDs)
	if len(groupIDs) == 0 && found {
		groupIDs = normalizeGroupIDs(nearestLesson.GroupIDs)
	}
	if len(groupIDs) == 0 {
		return Response{OK: false, Error: "group_ids are required"}
	}
	if !containsAllGroupIDs(nearestLesson.GroupIDs, groupIDs) {
		return Response{OK: false, Error: "group_ids do not match nearest scheduled lesson"}
	}
	sort.Slice(groupIDs, func(i, j int) bool { return groupIDs[i] < groupIDs[j] })

	subject, found, err := s.store.Subjects.GetByID(ctx, requestedSubjectID)
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

	semester, err := s.semesterForWrite(ctx, data.SemesterID)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	effectiveTTL := normalizeInviteTTL(data.ExpiresMinutes)
	expiresAt := time.Now().Add(time.Duration(effectiveTTL) * time.Minute)
	session, rosterSize, err := s.store.Attendance.CreateSessionWithGroups(ctx, teacherProfile.ID, subject.ID, semester.ID, groupIDs, expiresAt)
	if err != nil {
		return Response{OK: false, Error: "failed to create attendance session"}
	}

	lessonID := strconv.FormatInt(int64(session.ID), 10)
	teacherID := strconv.FormatInt(int64(teacherProfile.ID), 10)
	inviteToken, signedExpiresAt, err := s.generateAttendanceInviteToken(lessonID, teacherID, effectiveTTL)
	if err != nil {
		return Response{OK: false, Error: "failed to generate invite token"}
	}

	_ = s.create_attendance_opened_notification(
		ctx,
		session,
		groupIDs,
	)

	joinURL := buildAttendanceJoinURL(s.siteBaseURL, inviteToken)
	lessonName := strings.TrimSpace(data.LessonName)
	if lessonName == "" {
		lessonName = subject.Name
	}
	scheduleStart := nowLocal
	scheduleEnd := nowLocal.Add(95 * time.Minute)
	if found && requestedSubjectID == nearestLesson.SubjectID {
		scheduleStart = nearestLesson.StartAt
		scheduleEnd = nearestLesson.EndAt
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"lesson_id":       lessonID,
			"subject_id":      subject.ID,
			"semester_id":     semester.ID,
			"semester":        semesterToMap(semester),
			"lesson_name":     lessonName,
			"invite_token":    inviteToken,
			"url":             joinURL,
			"join_url":        joinURL,
			"qr_payload":      joinURL,
			"group_ids":       groupIDs,
			"roster_size":     rosterSize,
			"teacher_id":      teacherID,
			"schedule_start":  formatAPITime(scheduleStart),
			"schedule_end":    formatAPITime(scheduleEnd),
			"timezone":        "Asia/Novosibirsk",
			"expires_at":      formatAPITime(signedExpiresAt),
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
	studentProfile, err := s.studentProfileByUser(student)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
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
	markResult, err := s.store.Attendance.MarkStudentPresent(ctx, session.ID, studentProfile.ID, markedAt)
	if err != nil {
		return Response{OK: false, Error: "failed to confirm attendance"}
	}
	if markResult == "not_found" {
		return Response{OK: false, Error: "forbidden: student is not in session roster"}
	}
	if markResult == "already" {
		return Response{OK: false, Error: "attendance already confirmed"}
	}

	// Recalculate auto attendance grades
	_ = s.updateAutoAttendanceGrades(ctx, session.SubjectID, session.SemesterID, &studentProfile.ID, session.TeacherID)

	_ = s.create_attendance_result_notification(
		ctx,
		studentProfile.ID,
		session,
		true,
	)

	return Response{
		OK: true,
		Result: map[string]any{
			"lesson_id":  claims.LessonID,
			"student_id": student.UserID,
			"teacher_id": claims.TeacherID,
			"subject_id": session.SubjectID,
			"marked_at":  formatAPITime(markedAt),
			"attendance": "confirmed",
		},
	}
}

func (s *Service) attendanceByGroupForTeacher(sessionToken string, data AttendanceGroupStatsData) Response {
	teacherUser, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if teacherUser.Role != "teacher" {
		return Response{OK: false, Error: "forbidden: teacher role required"}
	}
	teacherProfile, err := s.teacherProfileByUser(teacherUser)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
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

	semester, err := s.semesterForOptionalID(ctx, data.SemesterID)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	allowed, err := s.teacherCanAccessGroup(ctx, teacherProfile.ID, data.GroupID, semester.ID, data.SubjectID)
	if err != nil {
		return Response{OK: false, Error: "failed to check teacher group access"}
	}
	if !allowed {
		return Response{OK: false, Error: "forbidden: group is not assigned to teacher"}
	}

	stats, err := s.store.Attendance.GetTeacherGroupAttendanceStats(ctx, teacherProfile.ID, data.GroupID, data.SubjectID, semester.ID)
	if err != nil {
		return Response{OK: false, Error: "failed to load attendance stats"}
	}

	students := make([]map[string]any, 0, len(stats))
	var sessionsCount int32
	for _, row := range stats {
		var lastMarkedAt any
		if row.LastMarkedAt != nil {
			lastMarkedAt = formatAPITime(*row.LastMarkedAt)
		}
		countedSessions := row.TotalSessions - row.ExcusedSessions
		attendancePercent := 0.0
		if countedSessions > 0 {
			attendancePercent = float64(row.AttendedSessions) * 100 / float64(countedSessions)
		}
		if row.TotalSessions > sessionsCount {
			sessionsCount = row.TotalSessions
		}

		students = append(students, map[string]any{
			"student_id":         row.StudentID,
			"student_name":       row.StudentName,
			"total_sessions":     row.TotalSessions,
			"attended_sessions":  row.AttendedSessions,
			"excused_sessions":   row.ExcusedSessions,
			"attendance_percent": attendancePercent,
			"last_marked_at":     lastMarkedAt,
		})
	}

	result := map[string]any{
		"group_id":    data.GroupID,
		"semester_id": semester.ID,
		"semester":    semesterToMap(semester),
		"timezone":    "Asia/Novosibirsk",
		"students":    students,
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

// groupPerformanceForTeacher returns a combined overview for a group on a
// subject: per-student attendance and grade totals plus group averages.
func (s *Service) groupPerformanceForTeacher(sessionToken string, data GroupPerformanceData) Response {
	teacherUser, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if teacherUser.Role != "teacher" {
		return Response{OK: false, Error: "forbidden: teacher role required"}
	}
	teacherProfile, err := s.teacherProfileByUser(teacherUser)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if data.GroupID <= 0 {
		return Response{OK: false, Error: "group_id is required"}
	}
	if data.SubjectID <= 0 {
		return Response{OK: false, Error: "subject_id is required"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	group, found, err := s.store.Groups.GetByID(ctx, data.GroupID)
	if err != nil {
		return Response{OK: false, Error: "failed to load group"}
	}
	if !found {
		return Response{OK: false, Error: "group not found"}
	}
	subject, found, err := s.store.Subjects.GetByID(ctx, data.SubjectID)
	if err != nil {
		return Response{OK: false, Error: "failed to load subject"}
	}
	if !found {
		return Response{OK: false, Error: "subject not found"}
	}

	semester, err := s.semesterForOptionalID(ctx, data.SemesterID)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if resp := s.ensureTeacherSubjectAccess(ctx, teacherProfile.ID, data.SubjectID, semester.ID); !resp.OK {
		return resp
	}
	allowed, err := s.teacherCanAccessGroup(ctx, teacherProfile.ID, data.GroupID, semester.ID, &data.SubjectID)
	if err != nil {
		return Response{OK: false, Error: "failed to check teacher group access"}
	}
	if !allowed {
		return Response{OK: false, Error: "forbidden: group is not assigned to teacher for this subject"}
	}

	rows, err := s.store.Attendance.GetGroupSubjectPerformance(ctx, teacherProfile.ID, data.GroupID, data.SubjectID, semester.ID)
	if err != nil {
		return Response{OK: false, Error: "failed to load group performance"}
	}

	students := make([]map[string]any, 0, len(rows))
	var (
		sessionsCount      int32
		attendancePctSum   float64
		gradePctSum        float64
		gradedStudentCount int
	)
	for _, row := range rows {
		countedSessions := row.TotalSessions - row.ExcusedSessions
		attendancePercent := 0.0
		if countedSessions > 0 {
			attendancePercent = float64(row.AttendedSessions) * 100 / float64(countedSessions)
		}
		gradePercent := 0.0
		if row.TotalMax > 0 {
			gradePercent = float64(row.CurrentScore) * 100 / float64(row.TotalMax)
		}
		if row.TotalSessions > sessionsCount {
			sessionsCount = row.TotalSessions
		}
		attendancePctSum += attendancePercent
		gradePctSum += gradePercent
		gradedStudentCount++

		students = append(students, map[string]any{
			"student_id":         row.StudentID,
			"student_name":       row.StudentName,
			"total_sessions":     row.TotalSessions,
			"attended_sessions":  row.AttendedSessions,
			"excused_sessions":   row.ExcusedSessions,
			"attendance_percent": attendancePercent,
			"current_score":      row.CurrentScore,
			"total_max":          row.TotalMax,
			"passed_max":         row.PassedMax,
			"grade_percent":      gradePercent,
		})
	}

	avgAttendance := 0.0
	avgGrade := 0.0
	if gradedStudentCount > 0 {
		avgAttendance = attendancePctSum / float64(gradedStudentCount)
		avgGrade = gradePctSum / float64(gradedStudentCount)
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"group_id":     data.GroupID,
			"group_name":   group.GroupName,
			"subject_id":   subject.ID,
			"subject_name": subject.Name,
			"semester_id":  semester.ID,
			"semester":     semesterToMap(semester),
			"timezone":     "Asia/Novosibirsk",
			"students":     students,
			"summary": map[string]any{
				"students_count":         len(students),
				"sessions_count":         sessionsCount,
				"avg_attendance_percent": avgAttendance,
				"avg_grade_percent":      avgGrade,
			},
		},
	}
}

func (s *Service) attendanceSessionByTeacher(ctx context.Context, sessionToken string, lessonID int32) (db.AttendanceSession, bool, Response) {
	teacherUser, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return db.AttendanceSession{}, false, Response{OK: false, Error: err.Error()}
	}
	if teacherUser.Role != "teacher" {
		return db.AttendanceSession{}, false, Response{OK: false, Error: "forbidden: teacher role required"}
	}
	teacherProfile, err := s.teacherProfileByUser(teacherUser)
	if err != nil {
		return db.AttendanceSession{}, false, Response{OK: false, Error: err.Error()}
	}
	if lessonID <= 0 {
		return db.AttendanceSession{}, false, Response{OK: false, Error: "lesson_id is required"}
	}

	session, found, err := s.store.Attendance.GetSessionByID(ctx, lessonID)
	if err != nil {
		return db.AttendanceSession{}, false, Response{OK: false, Error: "failed to load attendance session"}
	}
	if !found {
		return db.AttendanceSession{}, false, Response{OK: false, Error: "attendance session not found"}
	}
	if session.TeacherID != teacherProfile.ID {
		return db.AttendanceSession{}, false, Response{OK: false, Error: "forbidden: lesson belongs to another teacher"}
	}

	return session, true, Response{}
}

func (s *Service) attendanceProgressResult(ctx context.Context, session db.AttendanceSession, now time.Time) (map[string]any, error) {
	progress, err := s.store.Attendance.GetSessionProgress(ctx, session.ID)
	if err != nil {
		return nil, err
	}

	secondsRemaining := int64(session.ExpiresAt.UTC().Sub(now.UTC()).Seconds())
	if secondsRemaining < 0 {
		secondsRemaining = 0
	}
	attendancePercent := 0.0
	if progress.RosterSize > 0 {
		attendancePercent = float64(progress.MarkedCount) * 100 / float64(progress.RosterSize)
	}

	return map[string]any{
		"id":                 session.ID,
		"lesson_id":          session.ID,
		"teacher_id":         session.TeacherID,
		"subject_id":         session.SubjectID,
		"semester_id":        session.SemesterID,
		"lesson_name":        session.LessonName,
		"created_at":         formatAPITime(session.CreatedAt),
		"expires_at":         formatAPITime(session.ExpiresAt),
		"server_time":        formatAPITime(now),
		"timezone":           "Asia/Novosibirsk",
		"is_active":          secondsRemaining > 0,
		"seconds_remaining":  secondsRemaining,
		"remaining_seconds":  secondsRemaining,
		"marked_count":       progress.MarkedCount,
		"roster_size":        progress.RosterSize,
		"attendance_percent": attendancePercent,
	}, nil
}

func (s *Service) attendanceMarkedCountForTeacher(sessionToken string, data AttendanceSessionData) Response {
	ctx, cancel := s.dbContext()
	defer cancel()

	session, ok, resp := s.attendanceSessionByTeacher(ctx, sessionToken, data.LessonID)
	if !ok {
		return resp
	}

	result, err := s.attendanceProgressResult(ctx, session, time.Now().UTC())
	if err != nil {
		return Response{OK: false, Error: "failed to load attendance progress"}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"lesson_id":          result["lesson_id"],
			"marked_count":       result["marked_count"],
			"roster_size":        result["roster_size"],
			"attendance_percent": result["attendance_percent"],
		},
	}
}

func (s *Service) attendanceSessionTimerForTeacher(sessionToken string, data AttendanceSessionData) Response {
	ctx, cancel := s.dbContext()
	defer cancel()

	session, ok, resp := s.attendanceSessionByTeacher(ctx, sessionToken, data.LessonID)
	if !ok {
		return resp
	}

	result, err := s.attendanceProgressResult(ctx, session, time.Now().UTC())
	if err != nil {
		return Response{OK: false, Error: "failed to load attendance progress"}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"lesson_id":         result["lesson_id"],
			"expires_at":        result["expires_at"],
			"server_time":       result["server_time"],
			"timezone":          result["timezone"],
			"is_active":         result["is_active"],
			"seconds_remaining": result["seconds_remaining"],
			"remaining_seconds": result["remaining_seconds"],
		},
	}
}

var validAttendanceStatuses = map[string]bool{
	"present": true,
	"absent":  true,
	"late":    true,
	"excused": true,
}

func (s *Service) attendanceManualMarkByTeacher(sessionToken string, data TeacherAttendanceMarkData) Response {
	ctx, cancel := s.dbContext()
	defer cancel()

	session, ok, resp := s.attendanceSessionByTeacher(ctx, sessionToken, data.LessonID)
	if !ok {
		return resp
	}

	if data.StudentID <= 0 {
		return Response{OK: false, Error: "student_id is required"}
	}
	status := strings.ToLower(strings.TrimSpace(data.Status))
	if !validAttendanceStatuses[status] {
		return Response{OK: false, Error: "status must be one of: present, absent, late, excused"}
	}

	result, err := s.store.Attendance.SetStudentAttendanceStatus(ctx, session.ID, data.StudentID, status, time.Now().UTC())
	if err != nil {
		return Response{OK: false, Error: "failed to update attendance status"}
	}
	if result == "not_found" {
		return Response{OK: false, Error: "student not found in this session's roster"}
	}

	// Recalculate auto attendance grades
	_ = s.updateAutoAttendanceGrades(ctx, session.SubjectID, session.SemesterID, &data.StudentID, session.TeacherID)

	return Response{
		OK: true,
		Result: map[string]any{
			"lesson_id":  session.ID,
			"student_id": data.StudentID,
			"status":     status,
		},
	}
}

func (s *Service) activeAttendanceSessionForTeacher(sessionToken string) Response {
	teacherUser, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if teacherUser.Role != "teacher" {
		return Response{OK: false, Error: "forbidden: teacher role required"}
	}
	teacherProfile, err := s.teacherProfileByUser(teacherUser)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	now := time.Now().UTC()
	session, found, err := s.store.Attendance.GetActiveSessionByTeacherID(ctx, teacherProfile.ID, now)
	if err != nil {
		return Response{OK: false, Error: "failed to load active attendance session"}
	}
	if !found {
		return Response{
			OK: true,
			Result: map[string]any{
				"active":            false,
				"session":           nil,
				"seconds_remaining": int64(0),
				"remaining_seconds": int64(0),
				"server_time":       formatAPITime(now),
				"timezone":          "Asia/Novosibirsk",
			},
		}
	}

	sessionResult, err := s.attendanceProgressResult(ctx, session, now)
	if err != nil {
		return Response{OK: false, Error: "failed to load attendance progress"}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"active":            true,
			"session":           sessionResult,
			"seconds_remaining": sessionResult["seconds_remaining"],
			"remaining_seconds": sessionResult["remaining_seconds"],
			"server_time":       sessionResult["server_time"],
			"timezone":          sessionResult["timezone"],
		},
	}
}

func (s *Service) attendanceHistoryForStudent(sessionToken string, data AttendanceHistoryData) Response {
	studentUser, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if studentUser.Role != "student" {
		return Response{OK: false, Error: "forbidden: student role required"}
	}

	studentProfile, err := s.studentProfileByUser(studentUser)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	year := data.Year
	if year <= 0 {
		year = time.Now().Year()
	}
	start := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(1, 0, 0)

	ctx, cancel := s.dbContext()
	defer cancel()

	rows, err := s.store.Pool().Query(
		ctx,
		`SELECT (ass.marked_at AT TIME ZONE 'Asia/Novosibirsk')::date AS day,
		        COUNT(*)::int AS count
		 FROM attendance_session_students ass
		 WHERE ass.student_id = $1
		   AND ass.status IN ('present', 'late')
		   AND ass.marked_at IS NOT NULL
		   AND ass.marked_at >= $2
		   AND ass.marked_at < $3
		 GROUP BY day
		 ORDER BY day`,
		studentProfile.ID,
		start,
		end,
	)
	if err != nil {
		return Response{OK: false, Error: "failed to load attendance history"}
	}
	defer rows.Close()

	items := make([]map[string]any, 0)
	for rows.Next() {
		var day time.Time
		var count int32
		if err := rows.Scan(&day, &count); err != nil {
			return Response{OK: false, Error: "failed to scan attendance history"}
		}
		items = append(items, map[string]any{
			"date":  day.Format("2006-01-02"),
			"count": count,
		})
	}
	if err := rows.Err(); err != nil {
		return Response{OK: false, Error: "failed to iterate attendance history"}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"year":  year,
			"items": items,
		},
	}
}

func attendanceStatusLabel(status string) string {
	if status == "present" {
		return "attended"
	}
	return status
}

func (s *Service) attendanceHistoryForTeacherStudent(sessionToken string, data TeacherAttendanceStudentHistoryData) Response {
	teacherUser, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if teacherUser.Role != "teacher" {
		return Response{OK: false, Error: "forbidden: teacher role required"}
	}
	teacherProfile, err := s.teacherProfileByUser(teacherUser)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if data.StudentID <= 0 {
		return Response{OK: false, Error: "student_id is required"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	_, found, err := s.store.Students.GetByID(ctx, data.StudentID)
	if err != nil {
		return Response{OK: false, Error: "failed to load student"}
	}
	if !found {
		return Response{OK: false, Error: "student not found"}
	}

	semester, err := s.semesterForOptionalID(ctx, data.SemesterID)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if resp := s.ensureTeacherSubjectAccess(ctx, teacherProfile.ID, data.SubjectID, semester.ID); !resp.OK {
		return resp
	}
	if resp := s.ensureTeacherStudentSubjectAccess(ctx, teacherProfile.ID, data.StudentID, data.SubjectID, semester.ID); !resp.OK {
		return resp
	}

	rows, err := s.store.Attendance.GetStudentSubjectAttendanceHistory(ctx, data.StudentID, data.SubjectID, semester.ID)
	if err != nil {
		return Response{OK: false, Error: "failed to load attendance history"}
	}

	items := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		items = append(items, map[string]any{
			"date":        row.Date.Format("2006-01-02"),
			"lesson_name": row.LessonName,
			"status":      attendanceStatusLabel(row.Status),
		})
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"semester_id": semester.ID,
			"semester":    semesterToMap(semester),
			"items":       items,
		},
	}
}

func (s *Service) teacherSubjects(sessionToken string) Response {
	teacherUser, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if teacherUser.Role != "teacher" {
		return Response{OK: false, Error: "forbidden: teacher role required"}
	}

	teacherProfile, err := s.teacherProfileByUser(teacherUser)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	items := make([]TeacherSubjectsResultItem, 0)
	rows, err := s.store.Pool().Query(
		ctx,
		`SELECT sch.subject_id,
			        sub.name,
			        COALESCE(ARRAY_REMOVE(ARRAY_AGG(DISTINCT sch.group_id), NULL), '{}')::INTEGER[] AS group_ids
			 FROM schedules sch
			 JOIN subjects sub ON sub.subject_id = sch.subject_id
			 WHERE sch.teacher_id = $1
			   AND sch.semester_id = (
			       SELECT semester_id FROM semesters WHERE status = 'open' LIMIT 1
			   )
			 GROUP BY sch.subject_id, sub.name
			 ORDER BY sub.name, sch.subject_id`,
		teacherProfile.ID,
	)
	if err != nil {
		return Response{OK: false, Error: "failed to load teacher subjects"}
	}
	defer rows.Close()

	for rows.Next() {
		var item TeacherSubjectsResultItem
		if err := rows.Scan(&item.SubjectID, &item.SubjectName, &item.GroupIDs); err != nil {
			return Response{OK: false, Error: "failed to scan teacher subjects"}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return Response{OK: false, Error: "failed to iterate teacher subjects"}
	}

	if err := s.populateSubjectGroupNames(ctx, items); err != nil {
		return Response{OK: false, Error: "failed to load group names"}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"subjects": items,
		},
	}
}

// populateSubjectGroupNames fills the Groups field ({id, name}) of each subject
// item using the group IDs already collected, with a single lookup query.
func (s *Service) populateSubjectGroupNames(ctx context.Context, items []TeacherSubjectsResultItem) error {
	idSet := make(map[int32]struct{})
	for i := range items {
		for _, id := range items[i].GroupIDs {
			idSet[id] = struct{}{}
		}
	}
	if len(idSet) == 0 {
		for i := range items {
			items[i].Groups = make([]TeacherSubjectGroup, 0)
		}
		return nil
	}

	ids := make([]int32, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	rows, err := s.store.Pool().Query(
		ctx,
		`SELECT group_id, group_name FROM groups WHERE group_id = ANY($1)`,
		ids,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	names := make(map[int32]string, len(ids))
	for rows.Next() {
		var (
			id   int32
			name string
		)
		if err := rows.Scan(&id, &name); err != nil {
			return err
		}
		names[id] = name
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for i := range items {
		groups := make([]TeacherSubjectGroup, 0, len(items[i].GroupIDs))
		for _, id := range items[i].GroupIDs {
			name := names[id]
			if name == "" {
				name = fmt.Sprintf("Группа %d", id)
			}
			groups = append(groups, TeacherSubjectGroup{ID: id, Name: name})
		}
		items[i].Groups = groups
	}
	return nil
}

func (s *Service) handleRequest(raw string) Response {
	var req Request
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		return Response{OK: false, Error: "invalid request payload"}
	}

	switch req.Action {
	case "ping":
		return Response{
			ID:     req.ID,
			OK:     true,
			Result: map[string]any{"pong": true},
		}
	case "register":
		var data RegisterData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid registration payload"}
		}
		resp := s.register(data)
		resp.ID = req.ID
		return resp
	case "register_by_invite":
		var data RegisterByInviteData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid register_by_invite payload"}
		}
		resp := s.registerByInvite(data)
		resp.ID = req.ID
		return resp
	case "login":
		var data LoginData
		err := json.Unmarshal(req.Data, &data)
		if err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid login payload"}
		}
		resp := s.login(data)
		resp.ID = req.ID
		return resp
	case "profile":
		resp := s.profileByToken(req.Token)
		resp.ID = req.ID
		return resp
	case "user_agreement_decision":
		var data UserAgreementDecisionData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid user agreement decision payload"}
		}

		var meta RequestMeta
		if req.Meta != nil {
			meta = *req.Meta
		}

		resp := s.recordUserAgreementDecision(req.Token, data, meta)
		resp.ID = req.ID
		return resp
	case "user_agreement_current":
		resp := s.currentUserAgreement(req.Token)
		resp.ID = req.ID
		return resp
	case "admin_users_list":
		var data AdminUsersListData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid admin_users_list payload"}
		}
		resp := s.admin_users_list(req.Token, data)
		resp.ID = req.ID
		return resp
	case "admin_user_get":
		var data AdminUserIDData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid admin_user_get payload"}
		}
		resp := s.admin_user_get(req.Token, data.UserID)
		resp.ID = req.ID
		return resp
	case "admin_user_create":
		var data AdminUserCreateData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid admin_user_create payload"}
		}
		resp := s.admin_user_create(req.Token, data)
		resp.ID = req.ID
		return resp
	case "admin_user_update":
		var data AdminUserUpdateData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid admin_user_update payload"}
		}
		resp := s.admin_user_update(req.Token, data)
		resp.ID = req.ID
		return resp
	case "admin_user_delete":
		var data AdminUserIDData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid admin_user_delete payload"}
		}
		resp := s.admin_user_delete(req.Token, data.UserID)
		resp.ID = req.ID
		return resp
	case "admin_generate_teacher_invite":
		var data AdminGenerateTeacherInviteData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid admin_generate_teacher_invite payload"}
		}
		resp := s.admin_generate_teacher_invite(req.Token, data)
		resp.ID = req.ID
		return resp
	case "admin_generate_student_invite":
		var data AdminGenerateStudentInviteData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid admin_generate_student_invite payload"}
		}
		resp := s.admin_generate_student_invite(req.Token, data)
		resp.ID = req.ID
		return resp
	case "admin_list_catalogs":
		resp := s.adminListCatalogs(req.Token)
		resp.ID = req.ID
		return resp
	case "admin_list_invites":
		var data AdminInvitesListData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid admin_list_invites payload"}
		}
		resp := s.admin_list_invites(req.Token, data)
		resp.ID = req.ID
		return resp
	case "admin_revoke_invite":
		var data AdminRevokeInviteData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid admin_revoke_invite payload"}
		}
		resp := s.admin_revoke_invite(req.Token, data)
		resp.ID = req.ID
		return resp
	case "admin_notifications_create":
		var data AdminNotificationCreateData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid admin_notifications_create payload"}
		}
		resp := s.admin_notifications_create(req.Token, data)
		resp.ID = req.ID
		return resp
	case "admin_notifications_list":
		var data NotificationsListData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid admin_notifications_list payload"}
		}
		resp := s.admin_notifications_list(req.Token, data)
		resp.ID = req.ID
		return resp
	case "admin_notifications_update":
		var data AdminNotificationUpdateData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid admin_notifications_update payload"}
		}
		resp := s.admin_notifications_update(req.Token, data)
		resp.ID = req.ID
		return resp
	case "admin_notifications_delete":
		var data NotificationIDData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid admin_notifications_delete payload"}
		}
		resp := s.admin_notifications_delete(req.Token, data.NotificationID)
		resp.ID = req.ID
		return resp
	case "admin_system_stats":
		resp := s.admin_system_stats(req.Token)
		resp.ID = req.ID
		return resp
	case "admin_org_structure":
		resp := s.admin_org_structure(req.Token)
		resp.ID = req.ID
		return resp
	case "admin_roles_list":
		resp := s.admin_roles_list(req.Token)
		resp.ID = req.ID
		return resp
	case "admin_role_update":
		var data RolePermissions
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid admin_role_update payload"}
		}
		resp := s.admin_role_update(req.Token, data.Role, data)
		resp.ID = req.ID
		return resp
	case "admin_antifraud_logs":
		var query FraudLogsQuery
		_ = json.Unmarshal(req.Data, &query)
		resp := s.admin_antifraud_logs(req.Token, query)
		resp.ID = req.ID
		return resp
	case "admin_antifraud_top_cheaters":
		resp := s.admin_antifraud_top_cheaters(req.Token)
		resp.ID = req.ID
		return resp
	case "admin_services_list":
		resp := s.admin_services_list(req.Token)
		resp.ID = req.ID
		return resp
	case "admin_audit_logs":
		var query AuditLogsQuery
		_ = json.Unmarshal(req.Data, &query)
		resp := s.admin_audit_logs(req.Token, query)
		resp.ID = req.ID
		return resp
	case "admin_system_maintenance_get":
		resp := s.admin_system_maintenance_get(req.Token)
		resp.ID = req.ID
		return resp
	case "admin_system_maintenance_set":
		var reqData MaintenanceStatus
		if err := json.Unmarshal(req.Data, &reqData); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid maintenance payload"}
		}
		resp := s.admin_system_maintenance_set(req.Token, reqData)
		resp.ID = req.ID
		return resp
	case "register_device_token":
		var data DeviceTokenRegistration
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid device token payload"}
		}
		resp := s.RegisterDeviceToken(req.Token, data)
		resp.ID = req.ID
		return resp
	case "list_device_tokens":
		resp := s.ListDeviceTokens(req.Token)
		resp.ID = req.ID
		return resp
	case "delete_device_token":
		var data map[string]string
		_ = json.Unmarshal(req.Data, &data)
		deviceToken := data["device_token"]
		resp := s.DeleteDeviceToken(req.Token, deviceToken)
		resp.ID = req.ID
		return resp
	case "forgot_password":
		var data ForgotPasswordData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid forgot_password payload"}
		}
		resp := s.forgotPassword(data)
		resp.ID = req.ID
		return resp
	case "reset_password":
		var data ResetPasswordData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid reset_password payload"}
		}
		resp := s.resetPassword(data)
		resp.ID = req.ID
		return resp
	case "update_email":
		var data UpdateEmailData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid update_email payload"}
		}
		resp := s.updateEmail(req.Token, data)
		resp.ID = req.ID
		return resp
	case "notifications_list":
		var data NotificationsListData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid notifications_list payload"}
		}
		resp := s.notifications_list(req.Token, data)
		resp.ID = req.ID
		return resp
	case "notifications_unread_count":
		resp := s.notifications_unread_count(req.Token)
		resp.ID = req.ID
		return resp
	case "notification_mark_read":
		var data NotificationIDData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid notification_mark_read payload"}
		}
		resp := s.notification_mark_read(req.Token, data.NotificationID)
		resp.ID = req.ID
		return resp
	case "notification_delete":
		var data NotificationIDData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid notification_delete payload"}
		}
		resp := s.notification_delete(req.Token, data.NotificationID)
		resp.ID = req.ID
		return resp
	case "notifications_mark_all_read":
		resp := s.notifications_mark_all_read(req.Token)
		resp.ID = req.ID
		return resp
	case "notification_settings_get":
		resp := s.notification_settings_get(req.Token)
		resp.ID = req.ID
		return resp
	case "notification_settings_update":
		var data NotificationSettingsData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid notification_settings_update payload"}
		}
		resp := s.notification_settings_update(req.Token, data)
		resp.ID = req.ID
		return resp
	case "request_2fa_enable":
		resp := s.request2FAEnable(req.Token)
		resp.ID = req.ID
		return resp
	case "generate_2fa":
		var data Generate2FAData
		if len(req.Data) > 0 && string(req.Data) != "{}" {
			_ = json.Unmarshal(req.Data, &data)
		}
		resp := s.generate2fa(req.Token, data)
		resp.ID = req.ID
		return resp
	case "verify_2fa":
		var data TwoFaCodeData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid verify_2fa payload"}
		}
		resp := s.verify2fa(req.Token, data)
		resp.ID = req.ID
		return resp
	case "disable_2fa":
		var data TwoFaCodeData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid disable_2fa payload"}
		}
		resp := s.disable2fa(req.Token, data)
		resp.ID = req.ID
		return resp
	case "request_email_bind":
		var data RequestEmailData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid request_email_bind payload"}
		}
		resp := s.requestEmailBind(req.Token, data)
		resp.ID = req.ID
		return resp
	case "confirm_email_bind":
		var data ConfirmEmailData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid confirm_email_bind payload"}
		}
		resp := s.confirmEmailBind(req.Token, data)
		resp.ID = req.ID
		return resp
	case "semesters_list":
		resp := s.semestersList()
		resp.ID = req.ID
		return resp
	case "current_semester":
		resp := s.currentSemesterInfo()
		resp.ID = req.ID
		return resp
	case "create_semester":
		var data SemesterCreateData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid create_semester payload"}
		}
		resp := s.createSemester(req.Token, data)
		resp.ID = req.ID
		return resp
	case "activate_semester":
		var data SemesterIDData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid activate_semester payload"}
		}
		resp := s.activateSemester(req.Token, data)
		resp.ID = req.ID
		return resp
	case "close_semester":
		var data SemesterIDData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid close_semester payload"}
		}
		resp := s.closeSemester(req.Token, data)
		resp.ID = req.ID
		return resp
	case "archive_semester":
		var data SemesterIDData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid archive_semester payload"}
		}
		resp := s.archiveSemester(req.Token, data)
		resp.ID = req.ID
		return resp
	case "delete_semester":
		var data SemesterIDData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid delete_semester payload"}
		}
		resp := s.deleteSemester(req.Token, data)
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
	case "create_android_lesson":
		var data AndroidLessonCreateData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid create_android_lesson payload"}
		}
		resp := s.createLessonForAndroid(req.Token, data)
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
	case "mark_android_attendance":
		var data AndroidAttendanceMarkData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid mark_android_attendance payload"}
		}
		resp := s.markAttendanceForAndroid(req.Token, data)
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
	case "teacher_group_performance":
		var data GroupPerformanceData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid teacher_group_performance payload"}
		}
		resp := s.groupPerformanceForTeacher(req.Token, data)
		resp.ID = req.ID
		return resp
	case "teacher_attendance_marked_count":
		var data AttendanceSessionData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid teacher_attendance_marked_count payload"}
		}
		resp := s.attendanceMarkedCountForTeacher(req.Token, data)
		resp.ID = req.ID
		return resp
	case "teacher_attendance_session_timer":
		var data AttendanceSessionData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid teacher_attendance_session_timer payload"}
		}
		resp := s.attendanceSessionTimerForTeacher(req.Token, data)
		resp.ID = req.ID
		return resp
	case "teacher_active_attendance_session":
		resp := s.activeAttendanceSessionForTeacher(req.Token)
		resp.ID = req.ID
		return resp
	case "teacher_attendance_mark":
		var data TeacherAttendanceMarkData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid teacher_attendance_mark payload"}
		}
		resp := s.attendanceManualMarkByTeacher(req.Token, data)
		resp.ID = req.ID
		return resp
	case "student_schedule_day":
		var data StudentScheduleDayData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid student_schedule_day payload"}
		}
		resp := s.studentScheduleForDay(req.Token, data)
		resp.ID = req.ID
		return resp
	case "student_attendance_history":
		var data AttendanceHistoryData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid student_attendance_history payload"}
		}
		resp := s.attendanceHistoryForStudent(req.Token, data)
		resp.ID = req.ID
		return resp
	case "teacher_attendance_student_history":
		var data TeacherAttendanceStudentHistoryData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid teacher_attendance_student_history payload"}
		}
		resp := s.attendanceHistoryForTeacherStudent(req.Token, data)
		resp.ID = req.ID
		return resp
	case "teacher_subjects":
		resp := s.teacherSubjects(req.Token)
		resp.ID = req.ID
		return resp
	case "teacher_create_grade_item":
		var data GradeItemCreateData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid teacher_create_grade_item payload"}
		}
		resp := s.createGradeItemByTeacher(req.Token, data)
		resp.ID = req.ID
		return resp
	case "teacher_grade_items_by_subject":
		var data GradeSubjectData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid teacher_grade_items_by_subject payload"}
		}
		resp := s.gradeItemsBySubjectForTeacher(req.Token, data)
		resp.ID = req.ID
		return resp
	case "teacher_upsert_grade":
		var data GradeUpsertData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid teacher_upsert_grade payload"}
		}
		resp := s.upsertGradeByTeacher(req.Token, data)
		resp.ID = req.ID
		return resp
	case "teacher_delete_grade":
		var data GradeDeleteData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid teacher_delete_grade payload"}
		}
		resp := s.deleteGradeByTeacher(req.Token, data)
		resp.ID = req.ID
		return resp
	case "teacher_restore_grade":
		var data GradeRestoreData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid teacher_restore_grade payload"}
		}
		resp := s.restoreGradeByTeacher(req.Token, data)
		resp.ID = req.ID
		return resp
	case "teacher_delete_grade_item":
		var data GradeItemDeleteData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid teacher_delete_grade_item payload"}
		}
		resp := s.deleteGradeItemByTeacher(req.Token, data)
		resp.ID = req.ID
		return resp
	case "teacher_restore_grade_item":
		var data GradeItemRestoreData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid teacher_restore_grade_item payload"}
		}
		resp := s.restoreGradeItemByTeacher(req.Token, data)
		resp.ID = req.ID
		return resp
	case "teacher_student_grades_by_subject":
		var data TeacherStudentGradesData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid teacher_student_grades_by_subject payload"}
		}
		resp := s.gradesBySubjectForTeacher(req.Token, data)
		resp.ID = req.ID
		return resp
	case "student_grades_by_subject":
		var data GradeSubjectData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid student_grades_by_subject payload"}
		}
		resp := s.gradesBySubjectForStudent(req.Token, data)
		resp.ID = req.ID
		return resp
	case "student_performance_radar":
		var data SemesterSelectionData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid student_performance_radar payload"}
		}
		resp := s.studentPerformanceRadar(req.Token, data)
		resp.ID = req.ID
		return resp
	case "student_all_grades":
		var data SemesterSelectionData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid student_all_grades payload"}
		}
		resp := s.studentAllGrades(req.Token, data)
		resp.ID = req.ID
		return resp
	case "teacher_student_performance_radar":
		var data TeacherStudentRadarData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid teacher_student_performance_radar payload"}
		}
		resp := s.teacherStudentPerformanceRadar(req.Token, data)
		resp.ID = req.ID
		return resp
	case "staff_overview":
		resp := s.staffOverview(req.Token)
		resp.ID = req.ID
		return resp
	case "staff_students_page":
		var data StaffStudentsPageData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid staff_students_page payload"}
		}
		resp := s.staffStudentsPage(req.Token, data)
		resp.ID = req.ID
		return resp

	case "delete_email":
		resp := s.deleteEmail(req.Token)
		resp.ID = req.ID

		return resp

	default:
		return Response{ID: req.ID, OK: false, Error: "unknown_action: " + req.Action}

	}
}

func (s *Service) generateJWT(userID int32, tokenVersion int64) (string, error) {
	if userID <= 0 || tokenVersion <= 0 {
		return "", errors.New("invalid session identity")
	}
	jtiBytes := make([]byte, 16)
	if _, err := rand.Read(jtiBytes); err != nil {
		return "", err
	}
	now := time.Now().UTC()
	claims := SessionClaims{
		UserID:       userID,
		TokenVersion: tokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "ejournal",
			Subject:   strconv.FormatInt(int64(userID), 10),
			Audience:  jwt.ClaimStrings{"ejournal-web"},
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-5 * time.Second)),
			ID:        hex.EncodeToString(jtiBytes),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *Service) validateJWT(tokenString string) (int32, int64, error) {
	claims := &SessionClaims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (any, error) { return s.jwtSecret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer("ejournal"),
		jwt.WithAudience("ejournal-web"),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return 0, 0, err
	}

	if !token.Valid {
		return 0, 0, errors.New("token is not valid")
	}
	if claims.UserID <= 0 || claims.TokenVersion <= 0 {
		return 0, 0, errors.New("session claims are incomplete")
	}
	if claims.Subject != strconv.FormatInt(int64(claims.UserID), 10) {
		return 0, 0, errors.New("session subject does not match user")
	}
	return claims.UserID, claims.TokenVersion, nil
}

func (s *Service) deleteEmail(sessionToken string) Response {
	user, err := s.userBySessionToken(sessionToken)
	if err != nil {
		return Response{
			OK:    false,
			Error: err.Error(),
		}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	if err := s.store.Users.DeleteEmail(ctx, user.ID); err != nil {
		log.Printf("failed to delete email for user %d: %v", user.ID, err)
		return Response{OK: false, Error: "failed to delete email"}
	}

	return Response{
		OK:     true,
		Result: "Email has been successfully deleted",
	}
}

func (s *Service) forgotPassword(data ForgotPasswordData) Response {
	identity := strings.TrimSpace(data.Identity)
	if identity == "" {
		return Response{OK: false, Error: "Identity (login or email) is required"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	var dbUser db.User
	var ok bool
	var err error

	if strings.Contains(identity, "@") {
		dbUser, ok, err = s.store.Users.GetByEmail(ctx, identity)
	} else {
		dbUser, ok, err = s.store.Users.GetByLogin(ctx, identity)
	}

	if err != nil {
		log.Printf("password reset identity lookup failed: %v", err)
		return Response{OK: false, Error: "password reset is temporarily unavailable"}
	}
	if !ok {
		return Response{OK: true, Result: "If the account exists and has a registered email, a reset link has been sent."}
	}

	if dbUser.Email == nil || *dbUser.Email == "" {
		return Response{OK: true, Result: "If the account exists and has a registered email, a reset link has been sent."}
	}
	if s.mailer == nil || !s.mailer.Available() {
		return Response{OK: true, Result: "If the account exists and has a registered email, a reset link has been sent."}
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return Response{OK: false, Error: "password reset is temporarily unavailable"}
	}
	token := hex.EncodeToString(tokenBytes)
	expiresAt := time.Now().Add(15 * time.Minute)

	if err := s.store.Users.CreateResetToken(ctx, dbUser.ID, token, expiresAt); err != nil {
		log.Printf("failed to create password reset token: %v", err)
		return Response{OK: false, Error: "password reset is temporarily unavailable"}
	}

	email := *dbUser.Email
	go func() {
		if err := s.mailer.SendPasswordReset(email, token); err != nil {
			log.Printf("failed to send password reset email: %v", err)
		}
	}()

	return Response{OK: true, Result: "If the account exists and has a registered email, a reset link has been sent."}
}

func (s *Service) resetPassword(data ResetPasswordData) Response {
	token := strings.TrimSpace(data.Token)
	newPassword := strings.TrimSpace(data.NewPassword)
	if token == "" || newPassword == "" {
		return Response{OK: false, Error: "Token and new password are required"}
	}
	if len(newPassword) < 8 {
		return Response{OK: false, Error: "Password must be at least 8 characters long"}
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return Response{OK: false, Error: "failed to update password"}
	}

	updated, err := s.store.Users.ResetPasswordWithToken(ctx, token, string(hashedPassword))
	if err != nil {
		log.Printf("failed to consume password reset token: %v", err)
		return Response{OK: false, Error: "failed to update password"}
	}
	if !updated {
		return Response{OK: false, Error: "Invalid or expired token"}
	}

	return Response{OK: true, Result: "Password has been successfully updated"}
}

func (s *Service) updateEmail(sessionToken string, data UpdateEmailData) Response {
	return s.requestEmailBind(sessionToken, RequestEmailData{Email: data.Email})
}
