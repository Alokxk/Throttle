package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/Alokxk/Throttle/middleware"
	"github.com/redis/go-redis/v9"
)

func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	client := middleware.GetClientFromContext(r)

	clientID := strings.TrimPrefix(r.URL.Path, "/stats/")
	if clientID == "" {
		http.Error(w, "client_id is required", http.StatusBadRequest)
		return
	}

	if clientID != client.ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	ctx := context.Background()

	total := redisGetInt(h.Redis.Client, ctx, "stats:"+clientID+":total")
	allowed := redisGetInt(h.Redis.Client, ctx, "stats:"+clientID+":allowed")
	rejected := redisGetInt(h.Redis.Client, ctx, "stats:"+clientID+":rejected")
	fixedWindow := redisGetInt(h.Redis.Client, ctx, "stats:"+clientID+":algo:fixed_window")
	slidingWindow := redisGetInt(h.Redis.Client, ctx, "stats:"+clientID+":algo:sliding_window")
	tokenBucket := redisGetInt(h.Redis.Client, ctx, "stats:"+clientID+":algo:token_bucket")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"client_id":    clientID,
		"total_checks": total,
		"allowed":      allowed,
		"rejected":     rejected,
		"by_algorithm": map[string]int{
			"fixed_window":   fixedWindow,
			"sliding_window": slidingWindow,
			"token_bucket":   tokenBucket,
		},
	})
}

func redisGetInt(client *redis.Client, ctx context.Context, key string) int {
	val, err := client.Get(ctx, key).Result()
	if err != nil {
		return 0
	}
	n, _ := strconv.Atoi(val)
	return n
}
