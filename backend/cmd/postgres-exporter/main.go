package main

import (
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
)

func init() {
	prometheus.MustRegister(pgUp)
	prometheus.MustRegister(pgNumBackends)
	prometheus.MustRegister(pgXactCommit)
	prometheus.MustRegister(pgXactRollback)
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
			err := db.Ping()
			if err != nil {
				log.Printf("DB ping failed: %v", err)
				pgUp.Set(0)
				time.Sleep(5 * time.Second)
				continue
			}
			pgUp.Set(1)

			var numBackends int
			var commits, rollbacks float64
			err = db.QueryRow("SELECT COALESCE(sum(numbackends), 0), COALESCE(sum(xact_commit), 0), COALESCE(sum(xact_rollback), 0) FROM pg_stat_database").Scan(&numBackends, &commits, &rollbacks)
			if err != nil {
				log.Printf("Query pg_stat_database failed: %v", err)
			} else {
				pgNumBackends.Set(float64(numBackends))
				pgXactCommit.Set(commits)
				pgXactRollback.Set(rollbacks)
			}
			time.Sleep(5 * time.Second)
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	log.Println("Starting postgres-exporter on :9187")
	if err := http.ListenAndServe(":9187", nil); err != nil {
		log.Fatal(err)
	}
}
