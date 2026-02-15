package config

import (
	"fmt"
	"os"
)

type Config struct {
	AppEnv    string
	Port      string
	MongoURI  string
	RedisAddr string
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:    getenv("APP_ENV", "dev"),
		Port:      getenv("PORT", "8080"),
		MongoURI:  getenv("MONGO_URI", ""),
		RedisAddr: getenv("REDIS_ADDR", ""),
	}

	if cfg.MongoURI == "" {
		return Config{}, fmt.Errorf("missing env MONGO_URI")
	}

	if cfg.RedisAddr == "" {
		return Config{}, fmt.Errorf("missing env REDIS_ADDR")
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
