// @title EJournal Backend API
// @version 1.0
// @description Backend API for e-journal, attendance links and invite registration.
// @BasePath /
// @schemes http https
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
package main

import (
	"context"
	"log"
	"runtime"
	"time"

	"github.com/TelecomDep/ejournal_backend/internal/app"
	"github.com/TelecomDep/ejournal_backend/internal/config"
	"github.com/TelecomDep/ejournal_backend/internal/db"
	"github.com/TelecomDep/ejournal_backend/internal/httpserver"
)

func main() {
	cfg := config.Load()

	if cfg.DBDSN == "" {
		log.Fatalf("DB_DSN not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbStore, err := db.NewStore(ctx, cfg.DBDSN)
	if err != nil {
		log.Fatalf("database init failed: %v", err)
	}
	if err := dbStore.Ping(ctx); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}
	defer dbStore.Close()
	log.Printf("PostgreSQL connection initialized")

	workersCount := runtime.NumCPU() * 2
	if workersCount < 1 {
		workersCount = 1
	}

	svc := app.NewService(cfg.JWTSecret, cfg.SiteBaseURL, dbStore)
	svc.StartWorkerPool(workersCount)
	log.Printf("Internal worker pool started with %d workers", workersCount)

	server := httpserver.New(cfg, svc)
	server.Start()
}
