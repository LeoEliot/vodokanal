package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port                 int
	Env                  string
	DatabaseURL          string
	RedisURL             string
	JWTSecret            string
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration
}

func Load() (*Config, error) {
	port, err := strconv.Atoi(getEnv("PORT", "8081"))
	if err != nil {
		return nil, fmt.Errorf("invalid PORT: %w", err)
	}

	accessDuration, err := time.ParseDuration(getEnv("ACCESS_TOKEN_DURATION", "15m"))
	if err != nil {
		accessDuration = 15 * time.Minute
	}

	refreshDuration, err := time.ParseDuration(getEnv("REFRESH_TOKEN_DURATION", "168h")) // 7 days
	if err != nil {
		refreshDuration = 168 * time.Hour
	}

	dbURL := getEnv("DATABASE_URL", "postgres://vodokanal:vodokanal_password@localhost:5432/vodokanal?sslmode=disable")
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379")
	jwtSecret := getEnv("JWT_SECRET", "change-me-in-production")

	if jwtSecret == "change-me-in-production" {
		// Warn but allow for development
		if os.Getenv("ENV") != "production" {
			fmt.Println("WARNING: Using default JWT secret. Set JWT_SECRET in production!")
		}
	}

	return &Config{
		Port:                 port,
		Env:                  getEnv("ENV", "development"),
		DatabaseURL:          dbURL,
		RedisURL:             redisURL,
		JWTSecret:            jwtSecret,
		AccessTokenDuration:  accessDuration,
		RefreshTokenDuration: refreshDuration,
	}, nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
