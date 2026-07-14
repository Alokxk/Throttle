package db

import (
	"database/sql"
	"embed"
	"errors"
	"log/slog"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// RunMigrations applies pending schema migrations using the same embedded
// files everywhere the app runs (local, Docker, Kubernetes) — one mechanism,
// so there's no separate init-script path that can drift out of sync.
func RunMigrations(db *sql.DB) {
	source, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		slog.Error("failed to load embedded migrations", "error", err)
		os.Exit(1)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		slog.Error("failed to init migration driver", "error", err)
		os.Exit(1)
	}

	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		slog.Error("failed to init migrator", "error", err)
		os.Exit(1)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}

	slog.Info("migrations applied")
}
