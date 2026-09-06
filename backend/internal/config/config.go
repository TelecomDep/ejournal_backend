package config

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

type AppConfig struct {
	JWTSecret                   string
	SiteBaseURL                 string
	AppPort                     string
	CORSAllowOrigins            string
	DBDSN                       string
	RoleHashTeacher             string
	RoleHashStudent             string
	DefaultGroupID              int32
	AllowEarlyAttendance        bool
	UploadDir                   string
	SMTPHost                    string
	SMTPPort                    string
	SMTPUser                    string
	SMTPPassword                string
	SMTPFrom                    string
	SMTPTLSServerName           string
	SMTPCAFile                  string
	MetricsToken                string
	TrustedProxies              string
	AllowDemoAccounts           bool
	AllowLegacyRoleRegistration bool
}

func Load() AppConfig {
	loadDotEnv(".env")

	cfg := AppConfig{
		JWTSecret:                   strings.TrimSpace(os.Getenv("JWT_SECRET")),
		SiteBaseURL:                 getEnv("SITE_BASE_URL", "http://localhost:3000"),
		AppPort:                     getEnv("APP_PORT", "8888"),
		CORSAllowOrigins:            getEnv("CORS_ALLOW_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000"),
		DBDSN:                       strings.TrimSpace(os.Getenv("DB_DSN")),
		RoleHashTeacher:             strings.TrimSpace(os.Getenv("ROLE_HASH_TEACHER")),
		RoleHashStudent:             strings.TrimSpace(os.Getenv("ROLE_HASH_STUDENT")),
		DefaultGroupID:              getEnvInt32("DEFAULT_STUDENT_GROUP_ID", 1),
		AllowEarlyAttendance:        getEnvBool("ALLOW_EARLY_ATTENDANCE", false),
		UploadDir:                   getEnv("UPLOAD_DIR", "uploads"),
		SMTPHost:                    getEnv("SMTP_HOST", ""),
		SMTPPort:                    getEnv("SMTP_PORT", "587"),
		SMTPUser:                    getEnv("SMTP_USER", ""),
		SMTPPassword:                getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:                    getEnv("SMTP_FROM", ""),
		SMTPTLSServerName:           getEnv("SMTP_TLS_SERVER_NAME", ""),
		SMTPCAFile:                  getEnv("SMTP_CA_FILE", ""),
		MetricsToken:                strings.TrimSpace(os.Getenv("METRICS_TOKEN")),
		TrustedProxies:              getEnv("TRUSTED_PROXIES", "127.0.0.1,::1"),
		AllowDemoAccounts:           getEnvBool("ALLOW_DEMO_ACCOUNTS", false),
		AllowLegacyRoleRegistration: getEnvBool("ALLOW_LEGACY_ROLE_REGISTRATION", false),
	}

	if err := cfg.ValidateSecurity(); err != nil {
		log.Fatal(err)
	}
	if !cfg.AllowLegacyRoleRegistration {
		cfg.RoleHashTeacher = ""
		cfg.RoleHashStudent = ""
	}

	return cfg
}

// ValidateSecurity rejects deployable configurations with documented placeholder
// secrets. Production disables seeded demo accounts and legacy role codes by default.
func (cfg AppConfig) ValidateSecurity() error {
	secret := strings.TrimSpace(cfg.JWTSecret)
	lower := strings.ToLower(secret)
	if len(secret) < 32 || strings.Contains(lower, "change-me") ||
		strings.Contains(lower, "generate-a-") || strings.Contains(lower, "replace-with-") ||
		strings.Count(secret, secret[:1]) == len(secret) {
		return fmt.Errorf("JWT_SECRET must be a newly generated random secret of at least 32 characters (use openssl rand -hex 32)")
	}
	if cfg.AllowLegacyRoleRegistration && (cfg.RoleHashTeacher == "" || cfg.RoleHashStudent == "" ||
		strings.EqualFold(cfg.RoleHashTeacher, cfg.RoleHashStudent)) {
		return fmt.Errorf("legacy registration requires two distinct non-empty role codes")
	}
	return nil
}

func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}

		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		_ = os.Setenv(key, value)
	}
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
