package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Alokxk/Throttle/middleware"
)

type ResetRequest struct {
	Identifier string `json:"identifier"`
	Algorithm  string `json:"algorithm"`
}

func (h *Handler) Reset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	client := middleware.GetClientFromContext(r)

	var req ResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", "INVALID_BODY")
		return
	}

	if req.Identifier == "" {
		writeError(w, http.StatusBadRequest, "Identifier is required", "MISSING_IDENTIFIER")
		return
	}

	ctx := context.Background()
	deleted, err := resetIdentifier(ctx, h, client.APIKey, req.Identifier, req.Algorithm)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to reset identifier", "INTERNAL_ERROR")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":      "Identifier reset successfully",
		"identifier":   req.Identifier,
		"keys_deleted": deleted,
	})
}

func resetIdentifier(ctx context.Context, h *Handler, apiKey, identifier, algorithm string) (int, error) {
	deleted := 0

	patterns := []string{}

	switch algorithm {
	case "fixed_window":
		patterns = append(patterns, "fixed:"+apiKey+":"+identifier+":*")
	case "sliding_window":
		patterns = append(patterns, "sliding:"+apiKey+":"+identifier+":*")
	case "token_bucket":
		patterns = append(patterns, "token:"+apiKey+":"+identifier)
	default:
		patterns = append(patterns,
			"fixed:"+apiKey+":"+identifier+":*",
			"sliding:"+apiKey+":"+identifier+":*",
			"token:"+apiKey+":"+identifier,
		)
	}

	for _, pattern := range patterns {
		var cursor uint64
		for {
			keys, nextCursor, err := h.Redis.Client.Scan(ctx, cursor, pattern, 100).Result()
			if err != nil {
				return deleted, err
			}

			if len(keys) > 0 {
				if err := h.Redis.Client.Del(ctx, keys...).Err(); err != nil {
					return deleted, err
				}
				deleted += len(keys)
			}

			cursor = nextCursor
			if cursor == 0 {
				break
			}
		}
	}

	return deleted, nil
}
