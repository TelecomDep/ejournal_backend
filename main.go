package main

import (
	"encoding/json"
	"log"
	"sync"
	"syscall"

	zmq "github.com/pebbe/zmq4"
)

type Request struct {
	ID     string          `json:"id"`
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
}

type Response struct {
	ID     string `json:"id"`
	OK     bool   `json:"ok"`
	Result any    `json:"result,omitempty"`
	Error  string `json:"error"`
}

func main() {
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
		return Response{ID: req.ID, OK: true, Result: map[string]any{"pong": true}}
	case "register":
		return Response{ID: req.ID, OK: true, Result: map[string]any{"status": "registered"}}
	case "login":
		return Response{ID: req.ID, OK: true, Result: map[string]any{"token": "JWT"}}
	default:
		return Response{ID: req.ID, OK: false, Error: "unknown_action: " + req.Action}

	}

}
