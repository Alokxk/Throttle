package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Alokxk/Throttle/algorithms"
	"github.com/Alokxk/Throttle/httpx"
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
		httpx.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	client := middleware.GetClientFromContext(r)

	var req CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid request body", "INVALID_BODY")
		return
	}

	if req.Identifier == "" {
		httpx.WriteError(w, http.StatusBadRequest, "Identifier is required", "MISSING_IDENTIFIER")
		return
	}

	h.runCheck(w, r, client, req, req.Identifier)
}

func (h *Handler) CheckIP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	client := middleware.GetClientFromContext(r)
	ip := extractIP(r)

	var req CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid request body", "INVALID_BODY")
		return
	}

	h.runCheck(w, r, client, req, ip)
}

func (h *Handler) runCheck(w http.ResponseWriter, r *http.Request, client *models.Client, req CheckRequest, identifier string) {
	ctx, cancel := context.WithTimeout(r.Context(), httpx.RequestTimeout)
	defer cancel()

	exempted, exemptErr := models.IsExempted(ctx, h.DB, client.ID, identifier)
	if exemptErr != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Internal server error", "INTERNAL_ERROR")
		return
	}
	if exempted {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"allowed":    true,
			"exempted":   true,
			"remaining":  -1,
			"reset_at":   0,
			"algorithm":  "none",
			"identifier": identifier,
		})
		return
	}

	if req.Rule != "" {
		rule, err := models.GetRuleByName(ctx, h.DB, client.ID, req.Rule)
		if err != nil {
			if err == sql.ErrNoRows {
				httpx.WriteError(w, http.StatusBadRequest, "Rule not found", "RULE_NOT_FOUND")
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, "Failed to fetch rule", "INTERNAL_ERROR")
			return
		}
		req.Algorithm = rule.Algorithm
		req.Limit = rule.Limit
		req.Window = rule.Window
	} else if req.Algorithm == "" {
		req.Algorithm = client.DefaultAlgorithm
	}

	if code, msg, ok := validateRateLimitParams(req.Algorithm, req.Limit, req.Window); !ok {
		httpx.WriteError(w, http.StatusBadRequest, msg, code)
		return
	}

	var result *algorithms.Result
	var err error

	switch req.Algorithm {
	case "fixed_window":
		result, err = algorithms.FixedWindow(ctx, h.Redis.Client, client.APIKey, identifier, req.Limit, req.Window)

	case "sliding_window":
		result, err = algorithms.SlidingWindow(ctx, h.Redis.Client, client.APIKey, identifier, req.Limit, req.Window)

	case "token_bucket":
		refillRate := req.RefillRate
		if refillRate <= 0 {
			refillRate = float64(req.Limit) / 60.0
		}
		result, err = algorithms.TokenBucket(ctx, h.Redis.Client, h.Redis, h.Redis.TokenBucketSHA, client.APIKey, identifier, req.Limit, refillRate)
	}

	if err != nil {
		slog.Error("algorithm error", "error", err, "algorithm", req.Algorithm, "request_id", middleware.RequestIDFromContext(ctx))
		httpx.WriteError(w, http.StatusInternalServerError, "Internal server error", "INTERNAL_ERROR")
		return
	}

	warnThreshold := req.WarnThreshold
	if warnThreshold <= 0 || warnThreshold >= 1 {
		warnThreshold = 0.2
	}
	warnAt := int(float64(req.Limit) * warnThreshold)
	result.Warning = result.Allowed && result.Remaining <= warnAt
	result.WarnAt = warnAt

	h.enqueueUsage(usageJob{
		clientID:   client.ID,
		identifier: identifier,
		algorithm:  req.Algorithm,
		allowed:    result.Allowed,
		requestID:  middleware.RequestIDFromContext(ctx),
	})

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
		"identifier":  identifier,
		"retry_after": result.RetryAfter,
		"warning":     result.Warning,
	})
}

func (h *Handler) logUsage(clientID, identifier, algorithm string, allowed bool, requestID string) {
	ctx, cancel := context.WithTimeout(context.Background(), httpx.RequestTimeout)
	defer cancel()

	_, err := h.DB.ExecContext(ctx, `
		INSERT INTO usage_logs (client_id, identifier, algorithm, allowed)
		VALUES ($1, $2, $3, $4)
	`, clientID, identifier, algorithm, allowed)
	if err != nil {
		slog.Error("failed to log usage", "error", err, "client_id", clientID, "request_id", requestID)
	}
}

func (h *Handler) incrementStats(clientID, algorithm string, allowed bool, requestID string) {
	ctx, cancel := context.WithTimeout(context.Background(), httpx.RequestTimeout)
	defer cancel()

	pipe := h.Redis.Client.Pipeline()

	pipe.Incr(ctx, "stats:"+clientID+":total")
	pipe.Incr(ctx, "stats:"+clientID+":algo:"+algorithm)

	if allowed {
		pipe.Incr(ctx, "stats:"+clientID+":allowed")
	} else {
		pipe.Incr(ctx, "stats:"+clientID+":rejected")
	}

	if _, err := pipe.Exec(ctx); err != nil {
		slog.Error("failed to increment stats", "error", err, "client_id", clientID, "request_id", requestID)
	}
}
