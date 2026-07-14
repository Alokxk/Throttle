package middleware

import "net/http"

// CORS allows the dashboard (a separate origin — Vite dev server or its
// built static bundle) to call this API from the browser. Every other
// client (curl, k6, another backend service) ignores CORS entirely, so this
// only matters for the one browser-based consumer.
func CORS(allowedOrigin string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	}
}
