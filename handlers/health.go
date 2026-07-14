package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	dbErr := h.DB.PingContext(ctx)
	redisErr := h.Redis.Client.Ping(ctx).Err()

	w.Header().Set("Content-Type", "application/json")

	if dbErr != nil || redisErr != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   "unhealthy",
			"postgres": pingStatus(dbErr),
			"redis":    pingStatus(redisErr),
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "ok",
		"postgres": pingStatus(nil),
		"redis":    pingStatus(nil),
	})
}

// Liveness only — must never depend on Postgres/Redis, or an outage
// would restart every pod instead of just marking them unready.
func (h *Handler) Live(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func pingStatus(err error) string {
	if err != nil {
		return "down"
	}
	return "ok"
}
