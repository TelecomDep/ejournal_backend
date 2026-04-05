package main

import (
	"log"
	"runtime"

	"github.com/TelecomDep/ejournal_backend/internal/app"
	"github.com/TelecomDep/ejournal_backend/internal/config"
	"github.com/TelecomDep/ejournal_backend/internal/httpserver"
)

func main() {
	cfg := config.Load()

	workersCount := runtime.NumCPU() * 2
	if workersCount < 1 {
		workersCount = 1
	}

	svc := app.NewService(cfg.JWTSecret, cfg.SiteBaseURL)
	svc.StartWorkerPool(workersCount)
	log.Printf("Internal worker pool started with %d workers", workersCount)

	server := httpserver.New(cfg, svc)
	server.Start()
}
