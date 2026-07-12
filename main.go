package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/Alokxk/Throttle/config"
	"github.com/Alokxk/Throttle/db"
	"github.com/Alokxk/Throttle/handlers"
	"github.com/Alokxk/Throttle/middleware"
)

func main() {
	cfg := config.Load()

	pgDB := db.NewPostgresDB(cfg.DatabaseURL)
	defer pgDB.Close()

	redisClient := db.NewRedisClient(cfg.RedisURL)
	defer redisClient.Client.Close()

	h := handlers.NewHandler(pgDB, redisClient)

	http.HandleFunc("/health", h.Health)
	http.HandleFunc("/register", h.Register)
	http.HandleFunc("/check", middleware.Auth(pgDB, h.Check))
	http.HandleFunc("/stats/", middleware.Auth(pgDB, h.Stats))
	http.HandleFunc("/rules", middleware.Auth(pgDB, h.CreateRule))
	http.HandleFunc("/rules/", middleware.Auth(pgDB, h.RulesRouter))
	http.HandleFunc("/check/ip", middleware.Auth(pgDB, h.CheckIP))
	http.HandleFunc("/reset", middleware.Auth(pgDB, h.Reset))
	http.HandleFunc("/exemptions", middleware.Auth(pgDB, h.CreateExemption))
	http.HandleFunc("/exemptions/", middleware.Auth(pgDB, h.ExemptionsRouter))

	server := &http.Server{Addr: ":" + cfg.Port}

	go func() {
		log.Printf("Server starting on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	log.Println("Shutdown signal received, draining in-flight requests...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	} else {
		log.Println("Server shut down cleanly")
	}
}
