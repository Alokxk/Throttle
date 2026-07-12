package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Alokxk/Throttle/db"
	"github.com/Alokxk/Throttle/httpx"
	"github.com/Alokxk/Throttle/models"
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

	ctx, cancel := context.WithTimeout(r.Context(), httpx.RequestTimeout)
	defer cancel()

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
