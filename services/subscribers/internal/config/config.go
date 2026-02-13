package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port        int
	Env         string
	DatabaseURL string
	RedisURL    string
}

func Load() (*Config, error) {
	port, err := strconv.Atoi(getEnv("PORT", "8082"))
	if err != nil {
		return nil, fmt.Errorf("invalid PORT: %w", err)
	}

	dbURL := getEnv("DATABASE_URL", "postgres://vodokanal:vodokanal_password@localhost:5432/vodokanal?sslmode=disable")
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379")

	return &Config{
		Port:        port,
		Env:         getEnv("ENV", "development"),
		DatabaseURL: dbURL,
		RedisURL:    redisURL,
	}, nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
