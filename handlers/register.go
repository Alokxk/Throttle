package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/Alokxk/Throttle/algorithms"
	"github.com/Alokxk/Throttle/db"
	"github.com/Alokxk/Throttle/httpx"
	"github.com/Alokxk/Throttle/models"
)

// Registration abuse limit: 5 signups per hour per IP. Deliberately low
// since legitimate registration is a rare, one-time action per client.
const (
	registerLimit         = 5
	registerWindowSeconds = 3600
)

type Handler struct {
	DB        *sql.DB
	Redis     *db.RedisClient
	usageJobs chan usageJob
}

func NewHandler(database *sql.DB, redis *db.RedisClient) *Handler {
	h := &Handler{
		DB:        database,
		Redis:     redis,
		usageJobs: make(chan usageJob, usageQueueSize),
	}
	h.startUsageWorkers()
	return h
}

type RegisterRequest struct {
	Name             string `json:"name"`
	Email            string `json:"email"`
	DefaultAlgorithm string `json:"default_algorithm"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), httpx.RequestTimeout)
	defer cancel()

	ip := extractIP(r)
	limitResult, err := algorithms.FixedWindow(ctx, h.Redis.Client, "register", ip, registerLimit, registerWindowSeconds)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Internal server error", "INTERNAL_ERROR")
		return
	}
	if !limitResult.Allowed {
		w.Header().Set("Retry-After", strconv.Itoa(limitResult.RetryAfter))
		httpx.WriteError(w, http.StatusTooManyRequests, "Too many registration attempts, please try again later", "RATE_LIMITED")
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid request body", "INVALID_BODY")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)

	if req.Name == "" || req.Email == "" {
		httpx.WriteError(w, http.StatusBadRequest, "Name and email are required", "MISSING_FIELDS")
		return
	}

	if !strings.Contains(req.Email, "@") {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid email address", "INVALID_EMAIL")
		return
	}

	if req.DefaultAlgorithm == "" {
		req.DefaultAlgorithm = "fixed_window"
	}

	if !validAlgorithms[req.DefaultAlgorithm] {
		httpx.WriteError(w, http.StatusBadRequest, "Algorithm must be fixed_window, sliding_window, or token_bucket", "INVALID_ALGORITHM")
		return
	}

	apiKey, keyPrefix, keyHash, err := generateAPIKey()
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to generate API key", "INTERNAL_ERROR")
		return
	}

	client, err := models.CreateClient(ctx, h.DB, req.Name, req.Email, apiKey, keyPrefix, keyHash, req.DefaultAlgorithm)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			httpx.WriteError(w, http.StatusConflict, "Email already registered", "EMAIL_EXISTS")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to create client", "INTERNAL_ERROR")
		return
	}

	client.APIKey = apiKey

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(client)
}

func generateAPIKey() (apiKey, keyPrefix, keyHash string, err error) {
	bytes := make([]byte, 12)
	if _, err = rand.Read(bytes); err != nil {
		return
	}

	apiKey = "thr_" + hex.EncodeToString(bytes)
	keyPrefix = apiKey[:8]

	hash := sha256.Sum256([]byte(apiKey))
	keyHash = hex.EncodeToString(hash[:])
	return
}
