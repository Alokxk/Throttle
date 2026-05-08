package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

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

	writeError(w, http.StatusNotFound, "Route not found", "NOT_FOUND")
}

func (h *Handler) CreateExemption(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	client := middleware.GetClientFromContext(r)

	var req CreateExemptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", "INVALID_BODY")
		return
	}

	req.Identifier = strings.TrimSpace(req.Identifier)
	if req.Identifier == "" {
		writeError(w, http.StatusBadRequest, "Identifier is required", "MISSING_IDENTIFIER")
		return
	}

	exemption, err := models.CreateExemption(h.DB, client.ID, req.Identifier, req.Reason)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			writeError(w, http.StatusConflict, "Identifier already exempted", "ALREADY_EXEMPTED")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to create exemption", "INTERNAL_ERROR")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(exemption)
}

func (h *Handler) ListExemptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	client := middleware.GetClientFromContext(r)

	exemptions, err := models.ListExemptions(h.DB, client.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch exemptions", "INTERNAL_ERROR")
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
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	client := middleware.GetClientFromContext(r)

	identifier := strings.TrimPrefix(r.URL.Path, "/exemptions/")
	if identifier == "" {
		writeError(w, http.StatusBadRequest, "Identifier is required", "MISSING_IDENTIFIER")
		return
	}

	err := models.DeleteExemption(h.DB, client.ID, identifier)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "Exemption not found", "EXEMPTION_NOT_FOUND")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to delete exemption", "INTERNAL_ERROR")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Exemption deleted successfully",
	})
}
