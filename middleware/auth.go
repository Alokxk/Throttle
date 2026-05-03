package middleware

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/Alokxk/Throttle/models"
)

type contextKey string

const clientContextKey contextKey = "client"

type errorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func writeError(w http.ResponseWriter, status int, message, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorResponse{
		Error: message,
		Code:  code,
	})
}

func Auth(db *sql.DB, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			writeError(w, http.StatusUnauthorized, "Missing API key", "MISSING_API_KEY")
			return
		}

		client, err := models.GetClientByAPIKey(db, apiKey)
		if err != nil {
			if err == sql.ErrNoRows {
				writeError(w, http.StatusUnauthorized, "Invalid API key", "INVALID_API_KEY")
				return
			}
			writeError(w, http.StatusInternalServerError, "Internal server error", "INTERNAL_ERROR")
			return
		}

		ctx := context.WithValue(r.Context(), clientContextKey, client)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func GetClientFromContext(r *http.Request) *models.Client {
	client, _ := r.Context().Value(clientContextKey).(*models.Client)
	return client
}
