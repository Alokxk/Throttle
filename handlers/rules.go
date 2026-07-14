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

type CreateRuleRequest struct {
	Name      string `json:"name"`
	Algorithm string `json:"algorithm"`
	Limit     int    `json:"limit"`
	Window    int    `json:"window"`
}

func (h *Handler) CreateRule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	client := middleware.GetClientFromContext(r)

	var req CreateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "Invalid request body", "INVALID_BODY")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		httpx.WriteError(w, http.StatusBadRequest, "Rule name is required", "MISSING_NAME")
		return
	}

	if code, msg, ok := validateRateLimitParams(req.Algorithm, req.Limit, req.Window); !ok {
		httpx.WriteError(w, http.StatusBadRequest, msg, code)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), httpx.RequestTimeout)
	defer cancel()

	rule, err := models.CreateRule(ctx, h.DB, client.ID, req.Name, req.Algorithm, req.Limit, req.Window)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			httpx.WriteError(w, http.StatusConflict, "Rule name already exists", "RULE_EXISTS")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to create rule", "INTERNAL_ERROR")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rule)
}

func (h *Handler) ListRules(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	client := middleware.GetClientFromContext(r)

	ctx, cancel := context.WithTimeout(r.Context(), httpx.RequestTimeout)
	defer cancel()

	rules, err := models.ListRules(ctx, h.DB, client.ID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to fetch rules", "INTERNAL_ERROR")
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
		httpx.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	client := middleware.GetClientFromContext(r)

	name := strings.TrimPrefix(r.URL.Path, "/rules/")
	if name == "" {
		httpx.WriteError(w, http.StatusBadRequest, "Rule name is required", "MISSING_NAME")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), httpx.RequestTimeout)
	defer cancel()

	err := models.DeleteRule(ctx, h.DB, client.ID, name)
	if err != nil {
		if err == sql.ErrNoRows {
			httpx.WriteError(w, http.StatusNotFound, "Rule not found", "RULE_NOT_FOUND")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "Failed to delete rule", "INTERNAL_ERROR")
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

	httpx.WriteError(w, http.StatusNotFound, "Route not found", "NOT_FOUND")
}
