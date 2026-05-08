package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Alokxk/Throttle/middleware"
	"github.com/Alokxk/Throttle/models"
)

var validAlgorithms = map[string]bool{
	"fixed_window":   true,
	"sliding_window": true,
	"token_bucket":   true,
}

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

func (h *Handler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	client := middleware.GetClientFromContext(r)

	name := strings.TrimPrefix(r.URL.Path, "/rules/")
	if name == "" {
		writeError(w, http.StatusBadRequest, "Rule name is required", "MISSING_NAME")
		return
	}

	err := models.DeleteRule(h.DB, client.ID, name)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "Rule not found", "RULE_NOT_FOUND")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to delete rule", "INTERNAL_ERROR")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Rule deleted successfully",
	})
}

func (h *Handler) RulesRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/rules/")

	if path == "list" && r.Method == http.MethodGet {
		h.ListRules(w, r)
		return
	}

	if path != "" && r.Method == http.MethodDelete {
		h.DeleteRule(w, r)
		return
	}

	writeError(w, http.StatusNotFound, "Route not found", "NOT_FOUND")
}
