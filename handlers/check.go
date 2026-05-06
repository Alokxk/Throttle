package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/Alokxk/Throttle/algorithms"
	"github.com/Alokxk/Throttle/middleware"
	"github.com/Alokxk/Throttle/models"
)

type CheckRequest struct {
	Identifier    string  `json:"identifier"`
	Limit         int     `json:"limit"`
	Window        int     `json:"window"`
	Algorithm     string  `json:"algorithm"`
	RefillRate    float64 `json:"refill_rate"`
	Rule          string  `json:"rule"`
	WarnThreshold float64 `json:"warn_threshold"`
}

func (h *Handler) Check(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	client := middleware.GetClientFromContext(r)

	var req CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", "INVALID_BODY")
		return
	}

	if req.Identifier == "" {
		writeError(w, http.StatusBadRequest, "Identifier is required", "MISSING_IDENTIFIER")
		return
	}

	if req.Rule != "" {
		rule, err := models.GetRuleByName(h.DB, client.ID, req.Rule)
		if err != nil {
			if err == sql.ErrNoRows {
				writeError(w, http.StatusBadRequest, "Rule not found", "RULE_NOT_FOUND")
				return
			}
			writeError(w, http.StatusInternalServerError, "Failed to fetch rule", "INTERNAL_ERROR")
			return
		}
		req.Algorithm = rule.Algorithm
		req.Limit = rule.Limit
		req.Window = rule.Window
	} else if req.Algorithm == "" {
		req.Algorithm = client.DefaultAlgorithm
	}

	if req.Limit <= 0 {
		writeError(w, http.StatusBadRequest, "Limit must be greater than 0", "INVALID_LIMIT")
		return
	}

	ctx := context.Background()
	var result *algorithms.Result
	var err error

	switch req.Algorithm {
	case "fixed_window":
		if req.Window <= 0 {
			writeError(w, http.StatusBadRequest, "Window must be greater than 0 for fixed_window", "INVALID_WINDOW")
			return
		}
		result, err = algorithms.FixedWindow(ctx, h.Redis.Client, client.APIKey, req.Identifier, req.Limit, req.Window)

	case "sliding_window":
		if req.Window <= 0 {
			writeError(w, http.StatusBadRequest, "Window must be greater than 0 for sliding_window", "INVALID_WINDOW")
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
		writeError(w, http.StatusBadRequest, "Algorithm must be fixed_window, sliding_window, or token_bucket", "INVALID_ALGORITHM")
		return
	}

	if err != nil {
		log.Printf("algorithm error: %v", err)
		writeError(w, http.StatusInternalServerError, "Internal server error", "INTERNAL_ERROR")
		return
	}

	warnThreshold := req.WarnThreshold
	if warnThreshold <= 0 || warnThreshold >= 1 {
		warnThreshold = 0.2
	}
	warnAt := int(float64(req.Limit) * warnThreshold)
	result.Warning = result.Allowed && result.Remaining <= warnAt
	result.WarnAt = warnAt

	go h.logUsage(client.ID, req.Identifier, req.Algorithm, result.Allowed)
	go h.incrementStats(client.ID, req.Algorithm, result.Allowed)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(req.Limit))
	w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt, 10))
	w.Header().Set("X-RateLimit-Algorithm", req.Algorithm)

	if result.Warning {
		w.Header().Set("X-RateLimit-Warning", "true")
	}

	if !result.Allowed {
		w.Header().Set("Retry-After", strconv.Itoa(result.RetryAfter))
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"allowed":     result.Allowed,
		"remaining":   result.Remaining,
		"reset_at":    result.ResetAt,
		"algorithm":   req.Algorithm,
		"retry_after": result.RetryAfter,
		"warning":     result.Warning,
	})
}

func (h *Handler) CheckIP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	client := middleware.GetClientFromContext(r)
	ip := extractIP(r)

	var req CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", "INVALID_BODY")
		return
	}

	req.Identifier = ip

	if req.Rule != "" {
		rule, err := models.GetRuleByName(h.DB, client.ID, req.Rule)
		if err != nil {
			if err == sql.ErrNoRows {
				writeError(w, http.StatusBadRequest, "Rule not found", "RULE_NOT_FOUND")
				return
			}
			writeError(w, http.StatusInternalServerError, "Failed to fetch rule", "INTERNAL_ERROR")
			return
		}
		req.Algorithm = rule.Algorithm
		req.Limit = rule.Limit
		req.Window = rule.Window
	} else if req.Algorithm == "" {
		req.Algorithm = client.DefaultAlgorithm
	}

	if req.Limit <= 0 {
		writeError(w, http.StatusBadRequest, "Limit must be greater than 0", "INVALID_LIMIT")
		return
	}

	ctx := context.Background()
	var result *algorithms.Result
	var err error

	switch req.Algorithm {
	case "fixed_window":
		if req.Window <= 0 {
			writeError(w, http.StatusBadRequest, "Window must be greater than 0 for fixed_window", "INVALID_WINDOW")
			return
		}
		result, err = algorithms.FixedWindow(ctx, h.Redis.Client, client.APIKey, req.Identifier, req.Limit, req.Window)

	case "sliding_window":
		if req.Window <= 0 {
			writeError(w, http.StatusBadRequest, "Window must be greater than 0 for sliding_window", "INVALID_WINDOW")
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
		writeError(w, http.StatusBadRequest, "Algorithm must be fixed_window, sliding_window, or token_bucket", "INVALID_ALGORITHM")
		return
	}

	if err != nil {
		log.Printf("algorithm error: %v", err)
		writeError(w, http.StatusInternalServerError, "Internal server error", "INTERNAL_ERROR")
		return
	}

	warnThreshold := req.WarnThreshold
	if warnThreshold <= 0 || warnThreshold >= 1 {
		warnThreshold = 0.2
	}
	warnAt := int(float64(req.Limit) * warnThreshold)
	result.Warning = result.Allowed && result.Remaining <= warnAt
	result.WarnAt = warnAt

	go h.logUsage(client.ID, req.Identifier, req.Algorithm, result.Allowed)
	go h.incrementStats(client.ID, req.Algorithm, result.Allowed)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(req.Limit))
	w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt, 10))
	w.Header().Set("X-RateLimit-Algorithm", req.Algorithm)

	if result.Warning {
		w.Header().Set("X-RateLimit-Warning", "true")
	}

	if !result.Allowed {
		w.Header().Set("Retry-After", strconv.Itoa(result.RetryAfter))
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"allowed":     result.Allowed,
		"remaining":   result.Remaining,
		"reset_at":    result.ResetAt,
		"algorithm":   req.Algorithm,
		"identifier":  ip,
		"retry_after": result.RetryAfter,
		"warning":     result.Warning,
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
