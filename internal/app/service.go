package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/TelecomDep/ejournal_backend/internal/db"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	RoleHash string `json:"role_hash,omitempty"`
}

type RegisterByInviteData struct {
	InviteCode string `json:"invite_code"`
	Login      string `json:"login"`
	Password   string `json:"password"`
}

type User struct {
	ID     int32
	UserID string
	Login  string
	Pass   string
	Role   string
}

type AttendanceCreateData struct {
	SubjectID      int32   `json:"subject_id,omitempty"`
	GroupIDs       []int32 `json:"group_ids,omitempty"`
	LessonName     string  `json:"lesson_name,omitempty"`
	ExpiresMinutes int     `json:"expires_minutes,omitempty"`
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
	requestQueue         chan requestJob
}

var appTimeLocation = loadAppTimeLocation()

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

func NewService(jwtSecret, siteBaseURL, roleHashTeacher, roleHashStudent string, defaultGroupID int32, allowEarlyAttendance bool, store *db.Store) *Service {
	return &Service{
		jwtSecret:            []byte(strings.TrimSpace(jwtSecret)),
		siteBaseURL:          strings.TrimSpace(siteBaseURL),
		roleHashTeacher:      normalizeRoleHash(roleHashTeacher),
		roleHashStudent:      normalizeRoleHash(roleHashStudent),
		defaultGroupID:       defaultGroupID,
		allowEarlyAttendance: allowEarlyAttendance,
		store:                store,
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
		"login":   user.Login,
		"role":    user.Role,
	}

	ctx, cancel := s.dbContext()
	defer cancel()

	switch user.Role {
	case "student":
		var studentID int32
		var studentName sql.NullString
		var groupID sql.NullInt32
		var groupName sql.NullString
		err = s.store.Pool().QueryRow(
			ctx,
			`SELECT s.student_id, s.student_name, s.group_id, g.group_name
			 FROM students s
			 LEFT JOIN groups g ON g.group_id = s.group_id
			 WHERE s.user_id = $1 OR s.student_id = $1
			 ORDER BY CASE WHEN s.user_id = $1 THEN 0 ELSE 1 END
			 LIMIT 1`,
			user.ID,
		).Scan(&studentID, &studentName, &groupID, &groupName)
		if err == nil {
			result["student_id"] = studentID
			if studentName.Valid {
				result["name"] = studentName.String
				result["student_name"] = studentName.String
			}
			if groupID.Valid {
				result["group_id"] = groupID.Int32
			}
			if groupName.Valid {
				result["group_name"] = groupName.String
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

	var studentID int32
	var studentName string
	var groupID *int32
	var groupName sql.NullString
	err = tx.QueryRow(
		ctx,
		`SELECT s.student_id, s.student_name, s.group_id, g.group_name
		 FROM students s
		 LEFT JOIN groups g ON g.group_id = s.group_id
		 WHERE s.user_id IS NULL
		   AND s.invite_code_used_at IS NULL
		   AND s.invite_code_hash IS NOT NULL
		   AND s.invite_code_hash = crypt($1, s.invite_code_hash)
		 LIMIT 1
		 FOR UPDATE`,
		inviteCode,
	).Scan(&studentID, &studentName, &groupID, &groupName)
	if errors.Is(err, pgx.ErrNoRows) {
		return Response{OK: false, Error: "invalid or used invite_code"}
	}
	if err != nil {
		return Response{OK: false, Error: "failed to validate invite_code"}
	}

	var created db.User
	err = tx.QueryRow(
		ctx,
		`INSERT INTO users (login, password_hash, role)
		 VALUES ($1, $2, 'student')
		 RETURNING id, login, password_hash, role, created_at`,
		login,
		string(hashedPassword),
	).Scan(&created.ID, &created.Login, &created.PasswordHash, &created.Role, &created.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Response{OK: false, Error: "user exist"}
		}
		return Response{OK: false, Error: "failed to create user"}
	}

	cmd, err := tx.Exec(
		ctx,
		`UPDATE students
		 SET user_id = $2,
		     invite_code_used_at = NOW(),
		     invite_code = NULL,
		     invite_code_hash = NULL
		 WHERE student_id = $1
		   AND user_id IS NULL
		   AND invite_code_used_at IS NULL`,
		studentID,
		created.ID,
	)
	if err != nil {
		return Response{OK: false, Error: "failed to bind student profile"}
	}
	if cmd.RowsAffected() == 0 {
		return Response{OK: false, Error: "failed to bind student profile"}
	}

	if err = tx.Commit(ctx); err != nil {
		return Response{OK: false, Error: "failed to commit registration"}
	}

	result := map[string]any{
		"user_id":      strconv.FormatInt(int64(created.ID), 10),
		"login":        created.Login,
		"role":         created.Role,
		"student_id":   studentID,
		"student_name": studentName,
	}
	if groupID != nil {
		result["group_id"] = *groupID
	}
	if groupName.Valid {
		result["group_name"] = groupName.String
	}

	return Response{
		OK:     true,
		Result: result,
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
	if requestedSubjectID <= 0 {
		requestedSubjectID = nearestLesson.SubjectID
	}
	if requestedSubjectID != nearestLesson.SubjectID {
		return Response{OK: false, Error: "subject_id does not match nearest scheduled lesson"}
	}

	groupIDs := normalizeGroupIDs(data.GroupIDs)
	if len(groupIDs) == 0 {
		groupIDs = normalizeGroupIDs(nearestLesson.GroupIDs)
	}
	if len(groupIDs) == 0 {
		return Response{OK: false, Error: "nearest scheduled lesson has no groups"}
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

	effectiveTTL := normalizeInviteTTL(data.ExpiresMinutes)
	expiresAt := time.Now().Add(time.Duration(effectiveTTL) * time.Minute)
	session, rosterSize, err := s.store.Attendance.CreateSessionWithGroups(ctx, teacherProfile.ID, subject.ID, groupIDs, expiresAt)
	if err != nil {
		return Response{OK: false, Error: "failed to create attendance session"}
	}

	lessonID := strconv.FormatInt(int64(session.ID), 10)
	teacherID := strconv.FormatInt(int64(teacherProfile.ID), 10)
	inviteToken, signedExpiresAt, err := s.generateAttendanceInviteToken(lessonID, teacherID, effectiveTTL)
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
			"teacher_id":      teacherID,
			"schedule_start":  formatAPITime(nearestLesson.StartAt),
			"schedule_end":    formatAPITime(nearestLesson.EndAt),
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

	stats, err := s.store.Attendance.GetTeacherGroupAttendanceStats(ctx, teacherProfile.ID, data.GroupID, data.SubjectID)
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
		"timezone": "Asia/Novosibirsk",
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
