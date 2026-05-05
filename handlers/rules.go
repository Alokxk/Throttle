package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Alokxk/Throttle/middleware"
	"github.com/Alokxk/Throttle/models"
)

type CreateRuleRequest struct {
	Name      string `json:"name"`
	Algorithm string `json:"algorithm"`
	Limit     int    `json:"limit"`
	Window    int    `json:"window"`
}

func (h *Handler) CreateRule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	client := middleware.GetClientFromContext(r)

	var req CreateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", "INVALID_BODY")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "Rule name is required", "MISSING_NAME")
		return
	}

	if !validAlgorithms[req.Algorithm] {
		writeError(w, http.StatusBadRequest, "Algorithm must be fixed_window, sliding_window, or token_bucket", "INVALID_ALGORITHM")
		return
	}

	if req.Limit <= 0 {
		writeError(w, http.StatusBadRequest, "Limit must be greater than 0", "INVALID_LIMIT")
		return
	}

	if req.Algorithm != "token_bucket" && req.Window <= 0 {
		writeError(w, http.StatusBadRequest, "Window must be greater than 0 for fixed_window and sliding_window", "INVALID_WINDOW")
		return
	}

	rule, err := models.CreateRule(h.DB, client.ID, req.Name, req.Algorithm, req.Limit, req.Window)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			writeError(w, http.StatusConflict, "Rule name already exists", "RULE_EXISTS")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to create rule", "INTERNAL_ERROR")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rule)
}

func (h *Handler) ListRules(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	client := middleware.GetClientFromContext(r)

	rules, err := models.ListRules(h.DB, client.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch rules", "INTERNAL_ERROR")
		return
	}

	if rules == nil {
		rules = []*models.Rule{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"rules": rules,
	})
}
