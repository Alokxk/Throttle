package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Alokxk/Throttle/db"
	"github.com/Alokxk/Throttle/models"
)

type Handler struct {
	DB    *sql.DB
	Redis *db.RedisClient
}

type RegisterRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", "INVALID_BODY")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)

	if req.Name == "" || req.Email == "" {
		writeError(w, http.StatusBadRequest, "Name and email are required", "MISSING_FIELDS")
		return
	}

	if !strings.Contains(req.Email, "@") {
		writeError(w, http.StatusBadRequest, "Invalid email address", "INVALID_EMAIL")
		return
	}

	apiKey, err := generateAPIKey()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate API key", "INTERNAL_ERROR")
		return
	}

	client, err := models.CreateClient(h.DB, req.Name, req.Email, apiKey)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			writeError(w, http.StatusConflict, "Email already registered", "EMAIL_EXISTS")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to create client", "INTERNAL_ERROR")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(client)
}

func generateAPIKey() (string, error) {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "thr_" + hex.EncodeToString(bytes), nil
}
