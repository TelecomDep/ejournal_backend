package config

import (
	"log"
	"os"
	"strings"
)

type AppConfig struct {
	JWTSecret        string
	SiteBaseURL      string
	AppPort          string
	CORSAllowOrigins string
	DBDSN            string
}

func Load() AppConfig {
	cfg := AppConfig{
		JWTSecret:        strings.TrimSpace(os.Getenv("JWT_SECRET")),
		SiteBaseURL:      getEnv("SITE_BASE_URL", "http://localhost:3000"),
		AppPort:          getEnv("APP_PORT", "8888"),
		CORSAllowOrigins: getEnv("CORS_ALLOW_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000"),
		DBDSN:            strings.TrimSpace(os.Getenv("DB_DSN")),
	}

	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET not set")
	}

	return cfg
}

func getEnv(key, fallback string) string {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return fallback
	}
	return val
}
