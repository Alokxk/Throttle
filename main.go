package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/Alokxk/Throttle/config"
	"github.com/Alokxk/Throttle/db"
	"github.com/Alokxk/Throttle/handlers"
	"github.com/Alokxk/Throttle/middleware"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg := config.Load()

	pgDB := db.NewPostgresDB(cfg.DatabaseURL)
	defer pgDB.Close()
	db.RunMigrations(pgDB)

	redisClient := db.NewRedisClient(cfg.RedisURL)
	defer redisClient.Client.Close()

	h := handlers.NewHandler(pgDB, redisClient)

	prometheus.MustRegister(collectors.NewDBStatsCollector(pgDB, "throttle"))
	http.Handle("/metrics", promhttp.Handler())

	cors := func(path string, next http.HandlerFunc) http.HandlerFunc {
		return middleware.CORS(cfg.CORSAllowedOrigin, middleware.RequestID(middleware.Metrics(path, next)))
	}

	http.HandleFunc("/health", cors("/health", h.Health))
	http.HandleFunc("/live", cors("/live", h.Live))
	http.HandleFunc("/register", cors("/register", h.Register))
	http.HandleFunc("/me", cors("/me", middleware.Auth(pgDB, h.Me)))
	http.HandleFunc("/check", cors("/check", middleware.Auth(pgDB, h.Check)))
	http.HandleFunc("/stats/", cors("/stats/", middleware.Auth(pgDB, h.Stats)))
	http.HandleFunc("/rules", cors("/rules", middleware.Auth(pgDB, h.CreateRule)))
	http.HandleFunc("/rules/", cors("/rules/", middleware.Auth(pgDB, h.RulesRouter)))
	http.HandleFunc("/check/ip", cors("/check/ip", middleware.Auth(pgDB, h.CheckIP)))
	http.HandleFunc("/reset", cors("/reset", middleware.Auth(pgDB, h.Reset)))
	http.HandleFunc("/exemptions", cors("/exemptions", middleware.Auth(pgDB, h.CreateExemption)))
	http.HandleFunc("/exemptions/", cors("/exemptions/", middleware.Auth(pgDB, h.ExemptionsRouter)))

	server := &http.Server{Addr: ":" + cfg.Port}

	go func() {
		slog.Info("server starting", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	slog.Info("shutdown signal received, draining in-flight requests")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	} else {
		slog.Info("server shut down cleanly")
	}
}
