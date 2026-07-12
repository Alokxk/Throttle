package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Alokxk/Throttle/httpx"
	"github.com/Alokxk/Throttle/middleware"
	"github.com/Alokxk/Throttle/models"
)

type CreateExemptionRequest struct {
	Identifier string `json:"identifier"`
	Reason     string `json:"reason"`
}

func (h *Handler) ExemptionsRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/exemptions/")

	if path == "list" && r.Method == http.MethodGet {
		h.ListExemptions(w, r)
		return
	}

	if path != "" && r.Method == http.MethodDelete {
		h.DeleteExemption(w, r)
		return
	}

	httpx.WriteError(w, http.StatusNotFound, "Route not found", "NOT_FOUND")
}

func (h *Handler) CreateExemption(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	client := middleware.GetClientFromContext(r)

	var req CreateExemptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid request body", "INVALID_BODY")
		return
	}

	req.Identifier = strings.TrimSpace(req.Identifier)
	if req.Identifier == "" {
		httpx.WriteError(w, http.StatusBadRequest, "Identifier is required", "MISSING_IDENTIFIER")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), httpx.RequestTimeout)
	defer cancel()

	exemption, err := models.CreateExemption(ctx, h.DB, client.ID, req.Identifier, req.Reason)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			httpx.WriteError(w, http.StatusConflict, "Identifier already exempted", "ALREADY_EXEMPTED")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to create exemption", "INTERNAL_ERROR")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(exemption)
}

func (h *Handler) ListExemptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	client := middleware.GetClientFromContext(r)

	ctx, cancel := context.WithTimeout(r.Context(), httpx.RequestTimeout)
	defer cancel()

	exemptions, err := models.ListExemptions(ctx, h.DB, client.ID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to fetch exemptions", "INTERNAL_ERROR")
		return
	}

	if exemptions == nil {
		exemptions = []*models.Exemption{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"exemptions": exemptions,
	})
}

func (h *Handler) DeleteExemption(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	client := middleware.GetClientFromContext(r)

	identifier := strings.TrimPrefix(r.URL.Path, "/exemptions/")
	if identifier == "" {
		httpx.WriteError(w, http.StatusBadRequest, "Identifier is required", "MISSING_IDENTIFIER")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), httpx.RequestTimeout)
	defer cancel()

	err := models.DeleteExemption(ctx, h.DB, client.ID, identifier)
	if err != nil {
		if err == sql.ErrNoRows {
			httpx.WriteError(w, http.StatusNotFound, "Exemption not found", "EXEMPTION_NOT_FOUND")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to delete exemption", "INTERNAL_ERROR")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Exemption deleted successfully",
	})
}
