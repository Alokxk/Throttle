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

	// Sized against Postgres's default max_connections (100) divided by
	// KEDA's maxReplicaCount (6, see k8s/scaledobject.yaml): 15 * 6 = 90,
	// leaving headroom for Postgres's own reserved connections and manual
	// psql access. Found this the hard way — at 25/replica, scaling to 6
	// replicas under load hit "sorry, too many clients already" and new
	// pods crash-looped instead of serving traffic.
	db.SetMaxOpenConns(15)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	slog.Info("postgresql connected")
	return db
}
