package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DatabaseURL string
	RedisURL    string
}

func Load() *Config {
	_, filename, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(filename), "..")
	envPath := filepath.Join(root, ".env")

	godotenv.Load(envPath)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		slog.Error("DATABASE_URL environment variable is required")
		os.Exit(1)
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		slog.Error("REDIS_URL environment variable is required")
		os.Exit(1)
	}

	return &Config{
		Port:        port,
		DatabaseURL: databaseURL,
		RedisURL:    redisURL,
	}
}
