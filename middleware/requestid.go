package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

type requestIDKey struct{}

func RequestID(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := generateRequestID()
		w.Header().Set("X-Request-ID", id)

		ctx := context.WithValue(r.Context(), requestIDKey{}, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

func generateRequestID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
