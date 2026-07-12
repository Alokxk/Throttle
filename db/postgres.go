package db

import (
	"database/sql"
	"log/slog"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func NewPostgresDB(databaseURL string) *sql.DB {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	slog.Info("postgresql connected")
	return db
}
