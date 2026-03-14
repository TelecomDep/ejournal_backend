package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"os"
	"time"

	zmq "github.com/pebbe/zmq4"
	"log"
	"sync"
	"syscall"
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

var jwtSecret []byte
var sessionStore sync.Map //храним сессии юзеров
var users sync.Map        // храним юзеров (скоро будет в бд)

var userCounter int

func main() {

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET not set")
	}

	jwtSecret = []byte(secret)

	port := "20206"
	workersCount := 16
	frontAddr := "tcp://0.0.0.0:" + port
	backAddr := "inproc://backend"
	log.Printf("ZMQ start front address: %s", frontAddr)

	zctx, err := zmq.NewContext()
	if err != nil {
		log.Fatalf("error zmq.NewContext %v", err)
	}
	defer zctx.Term()

	front, err := zctx.NewSocket(zmq.ROUTER)
	if err != nil {
		log.Fatalf("error front zctx.NewSocket %v", err)
	}

	defer front.Close()
	_ = front.SetLinger(0)

	if err := front.Bind(frontAddr); err != nil {
		log.Fatalf("error binding front: %v", err)
	}

	back, err := zctx.NewSocket(zmq.DEALER)
	if err != nil {
		log.Fatalf("error back zctx.NewSocket %v", err)
	}

	defer back.Close()
	_ = back.SetLinger(0)

	if err := back.Bind(backAddr); err != nil {
		log.Fatalf("error binding back: %v", err)
	}

	//workers

	var wg sync.WaitGroup
	wg.Add(workersCount)

	for i := 0; i < workersCount; i++ {
		go func(workerId int) {
			defer wg.Done()
			runWorker(zctx, backAddr, workerId)
		}(i)
	}

	log.Printf("Proxy start")

	if err := zmq.Proxy(front, back, nil); err != nil {
		log.Fatalf("zmq.Proxy error: %v", err)
	}

	wg.Wait()
}

func runWorker(zctx *zmq.Context, backend string, workerId int) {
	s, err := zctx.NewSocket(zmq.REP)
	if err != nil {
		log.Fatalf("error zctx.NewSocket %v", err)
		return
	}

	defer s.Close()
	_ = s.SetLinger(0)

	if err := s.Connect(backend); err != nil {
		log.Fatalf("Connect error: %v", err)
	}

	for {
		frames, err := s.RecvMessage(0)
		if err != nil {
			log.Printf("RecvMessage error: %v", err)
			return
		}

		rawJson := frames[len(frames)-1]
		resp := handleRequest(rawJson)

		out, err := json.Marshal(resp)
		if err != nil {
			out = []byte(`{"ok": false, "error": "server json marshal error"}`)
		}

		_, err = s.SendMessage(string(out))
		if err != nil {
			if zmq.AsErrno(err) == zmq.Errno(syscall.EAGAIN) {
				log.Printf("Send timeout: %v", err)
				continue
			}
			log.Printf("Send error: %v", err)
			return
		}
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
			Result: map[string]any{"pong": true}}
	case "register":
		var data LoginData
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return Response{OK: false, Error: "EROR reg: " + err.Error()}
		}

		_, exist := users.Load(data.Login)
		if exist {
			return Response{
				ID:    req.ID,
				OK:    false,
				Error: "user exist",
			}
		}

		userCounter++
		userID := fmt.Sprintf("user-%d", userCounter)

		user := User{
			UserID: userID,
			Login:  data.Login,
			Pass:   data.Password,
		}

		users.Store(data.Login, user)

		return Response{
			ID: req.ID,
			OK: true,
			Result: map[string]any{"user_id": userID,
				"login": data.Login}}

	case "login":
		var data LoginData
		err := json.Unmarshal(req.Data, &data)
		if err != nil {
			return Response{OK: false, Error: "EROR_login: " + err.Error()}
		}
		//if data.Login != "admin" || data.Password != "admin" {
		//	return Response{OK: false, Error: "EROR_login: wrong password or login"}
		//}

		val, ok := users.Load(data.Login)
		if !ok {
			return Response{ID: req.ID, OK: false, Error: "user does not exist"}
		}

		user := val.(User)

		if user.Pass != data.Password {
			return Response{ID: req.ID, OK: false, Error: "wrong password"}
		}

		token, err := generateJWT(user.UserID) //на всяк случай по юзайди
		if err != nil {
			return Response{OK: false, Error: "EROR_generateJWT: " + err.Error()}
		}

		sessionStore.Store(token, user)

		return Response{
			ID: req.ID,
			OK: true,
			Result: map[string]any{"token": token,
				"user_ID": user.UserID,
				"login":   user.Login,
			}}
	default:
		return Response{
			ID:    req.ID,
			OK:    false,
			Error: "unknown_action: " + req.Action}

	}

}

func generateJWT(userID string) (string, error) {
	cl := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 12).Unix(), //12 часлсв до истечения токена
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, cl)

	return token.SignedString(jwtSecret)
}

func isok_JWT(tokenString string) (string, error) { //на будущее в новые кейсы
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
		return "", err
	}

	userID, ok := cl["user_id"].(string)

	if !ok {
		return "", fmt.Errorf("no user id found in claims")
	}
	return userID, nil

}
