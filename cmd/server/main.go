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

	var dbStore *db.Store
	if cfg.DBDSN != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		store, err := db.NewStore(ctx, cfg.DBDSN)
		if err != nil {
			log.Fatalf("database init failed: %v", err)
		}
		if err := store.Ping(ctx); err != nil {
			log.Fatalf("database ping failed: %v", err)
		}

		dbStore = store
		defer dbStore.Close()
		log.Printf("PostgreSQL connection initialized")
	}

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
