package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	pgUp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "pg_up",
		Help: "Whether the database connection is up (1) or down (0)",
	})
	pgNumBackends = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "pg_stat_database_numbackends",
		Help: "Number of active connections to the database",
	})
	pgXactCommit = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "pg_stat_database_xact_commit",
		Help: "Number of commits in the database",
	})
	pgXactRollback = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "pg_stat_database_xact_rollback",
		Help: "Number of rollbacks in the database",
	})
	pgBlksHit = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "pg_stat_database_blks_hit",
		Help: "Number of disk blocks found in buffer cache",
	})
	pgBlksRead = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "pg_stat_database_blks_read",
		Help: "Number of disk blocks read",
	})
	pgCacheHitRatio = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "pg_stat_database_cache_hit_ratio",
		Help: "Postgres buffer cache hit ratio in percent (0-100)",
	})
)

func init() {
	prometheus.MustRegister(pgUp)
	prometheus.MustRegister(pgNumBackends)
	prometheus.MustRegister(pgXactCommit)
	prometheus.MustRegister(pgXactRollback)
	prometheus.MustRegister(pgBlksHit)
	prometheus.MustRegister(pgBlksRead)
	prometheus.MustRegister(pgCacheHitRatio)
}

func main() {
	dsn := os.Getenv("DATA_SOURCE_NAME")
	if dsn == "" {
		log.Fatal("DATA_SOURCE_NAME env var is required")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Printf("Error opening DB: %v", err)
	}

	go func() {
		for {
			if db == nil {
				pgUp.Set(0)
				time.Sleep(5 * time.Second)
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			err := db.PingContext(ctx)
			cancel()
			if err != nil {
				log.Printf("DB ping failed: %v", err)
				pgUp.Set(0)
				time.Sleep(5 * time.Second)
				continue
			}
			pgUp.Set(1)

			var numBackends int
			var commits, rollbacks, blksHit, blksRead float64
			ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
			err = db.QueryRowContext(ctx, "SELECT COALESCE(sum(numbackends), 0), COALESCE(sum(xact_commit), 0), COALESCE(sum(xact_rollback), 0), COALESCE(sum(blks_hit), 0), COALESCE(sum(blks_read), 0) FROM pg_stat_database").Scan(&numBackends, &commits, &rollbacks, &blksHit, &blksRead)
			cancel()
			if err != nil {
				log.Printf("Query pg_stat_database failed: %v", err)
			} else {
				pgNumBackends.Set(float64(numBackends))
				pgXactCommit.Set(commits)
				pgXactRollback.Set(rollbacks)
				pgBlksHit.Set(blksHit)
				pgBlksRead.Set(blksRead)

				ratio := 100.0
				if blksHit+blksRead > 0 {
					ratio = (blksHit / (blksHit + blksRead)) * 100.0
				}
				pgCacheHitRatio.Set(ratio)
			}
			time.Sleep(5 * time.Second)
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	log.Println("Starting postgres-exporter on :9187")
	server := &http.Server{
		Addr:              ":9187",
		Handler:           http.DefaultServeMux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
