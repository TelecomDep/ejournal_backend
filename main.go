package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang-jwt/jwt/v5"
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
	UserID string
	Login  string
	Pass   string
	Role   string
}

type AttendanceCreateData struct {
	LessonName     string `json:"lesson_name"`
	ExpiresMinutes int    `json:"expires_minutes"`
}

type AttendanceConfirmData struct {
	InviteToken string `json:"invite_token"`
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

var jwtSecret []byte
var sessionStore sync.Map
var users sync.Map
var userCounter atomic.Int64
var lessonCounter atomic.Int64
var requestQueue chan requestJob
var attendanceMarks sync.Map // key: lesson_id:student_id, value: time.Time

func main() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET not set")
	}
	jwtSecret = []byte(secret)

	workersCount := runtime.NumCPU() * 2
	if workersCount < 1 {
		workersCount = 1
	}

	startWorkerPool(workersCount)
	log.Printf("Internal worker pool started with %d workers", workersCount)
	startHTTPServer()
}

func startWorkerPool(workersCount int) {
	requestQueue = make(chan requestJob, 1024)
	for i := 0; i < workersCount; i++ {
		go func() {
			for job := range requestQueue {
				job.resultCh <- handleRequest(job.rawRequest)
			}
		}()
	}
}

func dispatchRequest(raw string, timeout time.Duration) (Response, error) {
	job := requestJob{
		rawRequest: raw,
		resultCh:   make(chan Response, 1),
	}

	select {
	case requestQueue <- job:
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

func nextUserID() string {
	id := userCounter.Add(1)
	return fmt.Sprintf("user-%d", id)
}

func nextLessonID() string {
	id := lessonCounter.Add(1)
	return fmt.Sprintf("lesson-%d", id)
}

func normalizeRole(role string) string {
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "teacher" {
		return "teacher"
	}
	return "student"
}

func normalizeAuthHeader(token string) string {
	token = strings.TrimSpace(token)
	token = strings.TrimPrefix(token, "Bearer ")
	return strings.TrimSpace(token)
}

func userBySessionToken(token string) (User, error) {
	token = normalizeAuthHeader(token)
	if token == "" {
		return User{}, errors.New("missing token")
	}

	_, err := isok_JWT(token)
	if err != nil {
		return User{}, errors.New("invalid token")
	}

	val, ok := sessionStore.Load(token)
	if !ok {
		return User{}, errors.New("session not found")
	}

	return val.(User), nil
}

func profileByToken(token string) Response {
	user, err := userBySessionToken(token)
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

func generateAttendanceInviteToken(lessonID, teacherID string, expiresMinutes int) (string, time.Time, error) {
	if expiresMinutes <= 0 {
		expiresMinutes = 15
	}
	if expiresMinutes > 180 {
		expiresMinutes = 180
	}

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
	signed, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", time.Time{}, err
	}

	return signed, exp, nil
}

func parseAttendanceInviteToken(inviteToken string) (*AttendanceInviteClaims, error) {
	inviteToken = strings.TrimSpace(inviteToken)
	if inviteToken == "" {
		return nil, errors.New("missing invite token")
	}

	parsed, err := jwt.ParseWithClaims(inviteToken, &AttendanceInviteClaims{}, func(token *jwt.Token) (any, error) {
		return jwtSecret, nil
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

func createAttendanceLinkByTeacher(sessionToken string, data AttendanceCreateData) Response {
	teacher, err := userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if teacher.Role != "teacher" {
		return Response{OK: false, Error: "forbidden: teacher role required"}
	}

	lessonID := nextLessonID()
	inviteToken, expiresAt, err := generateAttendanceInviteToken(lessonID, teacher.UserID, data.ExpiresMinutes)
	if err != nil {
		return Response{OK: false, Error: "failed to generate invite token"}
	}

	siteBaseURL := strings.TrimSpace(os.Getenv("SITE_BASE_URL"))
	if siteBaseURL == "" {
		siteBaseURL = "http://localhost:3000"
	}
	url := fmt.Sprintf("%s/attendance/join?token=%s", strings.TrimRight(siteBaseURL, "/"), inviteToken)

	return Response{
		OK: true,
		Result: map[string]any{
			"lesson_id":       lessonID,
			"lesson_name":     data.LessonName,
			"invite_token":    inviteToken,
			"url":             url,
			"teacher_id":      teacher.UserID,
			"expires_at":      expiresAt.UTC().Format(time.RFC3339),
			"expires_minutes": data.ExpiresMinutes,
		},
	}
}

func confirmAttendanceByStudent(sessionToken string, data AttendanceConfirmData) Response {
	student, err := userBySessionToken(sessionToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if student.Role != "student" {
		return Response{OK: false, Error: "forbidden: student role required"}
	}

	claims, err := parseAttendanceInviteToken(data.InviteToken)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	markKey := claims.LessonID + ":" + student.UserID
	markedAt := time.Now().UTC()
	_, loaded := attendanceMarks.LoadOrStore(markKey, markedAt)
	if loaded {
		return Response{OK: false, Error: "attendance already confirmed"}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"lesson_id":  claims.LessonID,
			"student_id": student.UserID,
			"teacher_id": claims.TeacherID,
			"marked_at":  markedAt.Format(time.RFC3339),
			"attendance": "confirmed",
		},
	}
}

func handleRequest(raw string) Response {
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

		_, exist := users.Load(data.Login)
		if exist {
			return Response{ID: req.ID, OK: false, Error: "user exist"}
		}

		userID := nextUserID()
		user := User{UserID: userID, Login: data.Login, Pass: data.Password, Role: normalizeRole(data.Role)}
		users.Store(data.Login, user)

		return Response{
			ID: req.ID,
			OK: true,
			Result: map[string]any{
				"user_id": userID,
				"login":   data.Login,
				"role":    user.Role,
			},
		}
	case "login":
		var data LoginData
		err := json.Unmarshal(req.Data, &data)
		if err != nil {
			return Response{ID: req.ID, OK: false, Error: "EROR_login: " + err.Error()}
		}

		val, ok := users.Load(data.Login)
		if !ok {
			return Response{ID: req.ID, OK: false, Error: "user does not exist"}
		}

		user := val.(User)
		if user.Pass != data.Password {
			return Response{ID: req.ID, OK: false, Error: "wrong password"}
		}

		token, err := generateJWT(user.UserID)
		if err != nil {
			return Response{ID: req.ID, OK: false, Error: "EROR_generateJWT: " + err.Error()}
		}

		sessionStore.Store(token, user)
		return Response{
			ID: req.ID,
			OK: true,
			Result: map[string]any{
				"token":   token,
				"user_ID": user.UserID,
				"login":   user.Login,
				"role":    user.Role,
			},
		}
	case "profile":
		resp := profileByToken(req.Token)
		resp.ID = req.ID
		return resp
	case "create_attendance_link":
		var data AttendanceCreateData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid create_attendance_link payload"}
		}
		resp := createAttendanceLinkByTeacher(req.Token, data)
		resp.ID = req.ID
		return resp
	case "confirm_attendance":
		var data AttendanceConfirmData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{ID: req.ID, OK: false, Error: "invalid confirm_attendance payload"}
		}
		resp := confirmAttendanceByStudent(req.Token, data)
		resp.ID = req.ID
		return resp
	default:
		return Response{ID: req.ID, OK: false, Error: "unknown_action: " + req.Action}
	}
}

func generateJWT(userID string) (string, error) {
	cl := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 12).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, cl)
	return token.SignedString(jwtSecret)
}

func isok_JWT(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		return jwtSecret, nil
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

	userID, ok := cl["user_id"].(string)
	if !ok {
		return "", fmt.Errorf("no user id found in claims")
	}

	return userID, nil
}
