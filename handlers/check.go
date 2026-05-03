package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/Alokxk/Throttle/algorithms"
	"github.com/Alokxk/Throttle/middleware"
)

type CheckRequest struct {
	Identifier string  `json:"identifier"`
	Limit      int     `json:"limit"`
	Window     int     `json:"window"`
	Algorithm  string  `json:"algorithm"`
	RefillRate float64 `json:"refill_rate"`
}

func (h *Handler) Check(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	client := middleware.GetClientFromContext(r)

	var req CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Identifier == "" || req.Limit <= 0 || req.Algorithm == "" {
		http.Error(w, "identifier, limit, and algorithm are required", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	var result *algorithms.Result
	var err error

	switch req.Algorithm {
	case "fixed_window":
		if req.Window <= 0 {
			http.Error(w, "window is required for fixed_window", http.StatusBadRequest)
			return
		}
		result, err = algorithms.FixedWindow(ctx, h.Redis.Client, client.APIKey, req.Identifier, req.Limit, req.Window)

	case "sliding_window":
		if req.Window <= 0 {
			http.Error(w, "window is required for sliding_window", http.StatusBadRequest)
			return
		}
		result, err = algorithms.SlidingWindow(ctx, h.Redis.Client, client.APIKey, req.Identifier, req.Limit, req.Window)

	case "token_bucket":
		refillRate := req.RefillRate
		if refillRate <= 0 {
			refillRate = float64(req.Limit) / 60.0
		}
		result, err = algorithms.TokenBucket(ctx, h.Redis.Client, h.Redis.TokenBucketSHA, client.APIKey, req.Identifier, req.Limit, refillRate)

	default:
		http.Error(w, "algorithm must be fixed_window, sliding_window, or token_bucket", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Printf("algorithm error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	go h.logUsage(client.ID, req.Identifier, req.Algorithm, result.Allowed)
	go h.incrementStats(client.ID, req.Algorithm, result.Allowed)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"allowed":     result.Allowed,
		"remaining":   result.Remaining,
		"reset_at":    result.ResetAt,
		"algorithm":   req.Algorithm,
		"retry_after": result.RetryAfter,
	})
}

func (h *Handler) logUsage(clientID, identifier, algorithm string, allowed bool) {
	_, err := h.DB.Exec(`
		INSERT INTO usage_logs (client_id, identifier, algorithm, allowed)
		VALUES ($1, $2, $3, $4)
	`, clientID, identifier, algorithm, allowed)
	if err != nil {
		log.Printf("failed to log usage: %v", err)
	}
}

func (h *Handler) incrementStats(clientID, algorithm string, allowed bool) {
	ctx := context.Background()
	pipe := h.Redis.Client.Pipeline()

	pipe.Incr(ctx, "stats:"+clientID+":total")
	pipe.Incr(ctx, "stats:"+clientID+":algo:"+algorithm)

	if allowed {
		pipe.Incr(ctx, "stats:"+clientID+":allowed")
	} else {
		pipe.Incr(ctx, "stats:"+clientID+":rejected")
	}

	if _, err := pipe.Exec(ctx); err != nil {
		log.Printf("failed to increment stats: %v", err)
	}
}
