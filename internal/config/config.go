package config

import (
	"log"
	"os"
	"strconv"
	"strings"
)

type AppConfig struct {
	JWTSecret            string
	SiteBaseURL          string
	AppPort              string
	CORSAllowOrigins     string
	DBDSN                string
	RoleHashTeacher      string
	RoleHashStudent      string
	DefaultGroupID       int32
	AllowEarlyAttendance bool
}

func Load() AppConfig {
	cfg := AppConfig{
		JWTSecret:            strings.TrimSpace(os.Getenv("JWT_SECRET")),
		SiteBaseURL:          getEnv("SITE_BASE_URL", "http://localhost:3000"),
		AppPort:              getEnv("APP_PORT", "8888"),
		CORSAllowOrigins:     getEnv("CORS_ALLOW_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000"),
		DBDSN:                strings.TrimSpace(os.Getenv("DB_DSN")),
		RoleHashTeacher:      getEnv("ROLE_HASH_TEACHER", "TEACHER-HASH-2026"),
		RoleHashStudent:      getEnv("ROLE_HASH_STUDENT", "STUDENT-HASH-2026"),
		DefaultGroupID:       getEnvInt32("DEFAULT_STUDENT_GROUP_ID", 1),
		AllowEarlyAttendance: getEnvBool("ALLOW_EARLY_ATTENDANCE", false),
	}

	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET not set")
	}
	if strings.EqualFold(strings.TrimSpace(cfg.RoleHashTeacher), strings.TrimSpace(cfg.RoleHashStudent)) {
		log.Fatal("ROLE_HASH_TEACHER and ROLE_HASH_STUDENT must be different")
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

func getEnvInt32(key string, fallback int32) int32 {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return fallback
	}

	parsed, err := strconv.ParseInt(val, 10, 32)
	if err != nil {
		return fallback
	}
	return int32(parsed)
}

func getEnvBool(key string, fallback bool) bool {
	val := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if val == "" {
		return fallback
	}
	switch val {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}
