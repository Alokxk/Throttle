package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/Alokxk/Throttle/config"
	"github.com/Alokxk/Throttle/db"
	"github.com/Alokxk/Throttle/handlers"
)

func main() {
	cfg := config.Load()

	pgDB := db.NewPostgresDB(cfg.DatabaseURL)
	defer pgDB.Close()

	redisClient := db.NewRedisClient(cfg.RedisURL)
	defer redisClient.Client.Close()

	h := &handlers.Handler{DB: pgDB}

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/register", h.Register)

	log.Printf("Server starting on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, nil); err != nil {
		log.Fatal(err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
