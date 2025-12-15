package config

import (
	"os"
)

type Config struct {
	DatabaseURL string
	RedisURL    string
	ServerAddr  string
	JWTSecret   string
}

func Load() *Config {
	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://root:password@localhost:5432/rickshaw?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "localhost:6379"),
		ServerAddr:  getEnv("SERVER_ADDR", "0.0.0.0:8080"),
		JWTSecret:   getEnv("JWT_SECRET", "your-super-secret-key-change-in-production"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
