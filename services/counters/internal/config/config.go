package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port             int
	Env              string
	DatabaseURL       string
	RedisURL         string
	VerificationDays int // Days before verification is required
}

func Load() (*Config, error) {
	port, err := strconv.Atoi(getEnv("PORT", "8087"))
	if err != nil {
		return nil, fmt.Errorf("invalid PORT: %w", err)
	}

	verificationDays, err := strconv.Atoi(getEnv("VERIFICATION_DAYS", "1460")) // 4 years
	if err != nil {
		verificationDays = 1460
	}

	dbURL := getEnv("DATABASE_URL", "postgres://vodokanal:vodokanal_password@localhost:5432/vodokanal?sslmode=disable")
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379")

	return &Config{
		Port:             port,
		Env:              getEnv("ENV", "development"),
		DatabaseURL:       dbURL,
		RedisURL:         redisURL,
		VerificationDays: verificationDays,
	}, nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
