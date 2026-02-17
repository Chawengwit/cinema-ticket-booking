package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppEnv             string
	Port               string
	MongoURI           string
	RedisAddr          string
	JWTSecret          string
	FrontendURL        string
	CORSOrigins        []string
	SeatLockTTLSeconds int
	AdminEmails        []string
}

func Load() (Config, error) {
	frontendURL := getenv("FRONTEND_URL", "http://localhost:5173")
	corsOrigins := getenv("CORS_ORIGINS", frontendURL)

	ttlStr := getenv("SEAT_LOCK_TTL_SECONDS", "300")
	ttlSec, err := strconv.Atoi(ttlStr)
	if err != nil || ttlSec <= 0 {
		return Config{}, fmt.Errorf("invalid SEAT_LOCK_TTL_SECONDS: %s", ttlStr)
	}

	adminEmailsRaw := getenv("ADMIN_EMAILS", "")
	adminEmails := normalizeEmails(splitCSV(adminEmailsRaw))

	cfg := Config{
		AppEnv:             getenv("APP_ENV", "dev"),
		Port:               getenv("PORT", "8080"),
		MongoURI:           getenv("MONGO_URI", ""),
		RedisAddr:          getenv("REDIS_ADDR", ""),
		JWTSecret:          getenv("JWT_SECRET", ""),
		FrontendURL:        frontendURL,
		CORSOrigins:        splitCSV(corsOrigins),
		SeatLockTTLSeconds: ttlSec,
		AdminEmails:        adminEmails,
	}

	if cfg.MongoURI == "" {
		return Config{}, fmt.Errorf("missing env MONGO_URI")
	}
	if cfg.RedisAddr == "" {
		return Config{}, fmt.Errorf("missing env REDIS_ADDR")
	}
	if cfg.JWTSecret == "" {
		return Config{}, fmt.Errorf("missing env JWT_SECRET")
	}
	if len(cfg.JWTSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}

	return cfg, nil
}

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{}
	}
	return out
}

func normalizeEmails(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))

	for _, e := range in {
		e = strings.ToLower(strings.TrimSpace(e))
		if e == "" {
			continue
		}
		if _, ok := seen[e]; ok {
			continue
		}
		seen[e] = struct{}{}
		out = append(out, e)
	}
	return out
}
