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
}

type User struct {
	UserID string
	Login  string
	Pass   string
}

type requestJob struct {
	rawRequest string
	resultCh   chan Response
}

var jwtSecret []byte
var sessionStore sync.Map
var users sync.Map
var userCounter atomic.Int64
var requestQueue chan requestJob

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

func normalizeAuthHeader(token string) string {
	token = strings.TrimSpace(token)
	token = strings.TrimPrefix(token, "Bearer ")
	return strings.TrimSpace(token)
}

func profileByToken(token string) Response {
	token = normalizeAuthHeader(token)
	if token == "" {
		return Response{OK: false, Error: "missing token"}
	}

	userID, err := isok_JWT(token)
	if err != nil {
		return Response{OK: false, Error: "invalid token"}
	}

	if val, ok := sessionStore.Load(token); ok {
		user := val.(User)
		return Response{
			OK: true,
			Result: map[string]any{
				"user_id": user.UserID,
				"login":   user.Login,
			},
		}
	}

	return Response{
		OK: true,
		Result: map[string]any{
			"user_id": userID,
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
		user := User{UserID: userID, Login: data.Login, Pass: data.Password}
		users.Store(data.Login, user)

		return Response{
			ID: req.ID,
			OK: true,
			Result: map[string]any{
				"user_id": userID,
				"login":   data.Login,
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
			},
		}
	case "profile":
		resp := profileByToken(req.Token)
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
