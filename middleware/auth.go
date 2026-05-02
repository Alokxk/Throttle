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

func Auth(db *sql.DB, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "missing API key",
			})
			return
		}

		client, err := models.GetClientByAPIKey(db, apiKey)
		if err != nil {
			if err == sql.ErrNoRows {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "invalid API key",
				})
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "internal server error",
			})
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
