package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/Alokxk/Throttle/config"
)

func main() {
	cfg := config.Load()

	http.HandleFunc("/health", healthHandler)

	log.Printf("Server starting on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, nil); err != nil {
		log.Fatal(err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
