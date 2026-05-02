package algorithms

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func TokenBucket(ctx context.Context, client *redis.Client, sha, apiKey, identifier string, capacity int, refillRate float64) (*Result, error) {
	now := float64(time.Now().UnixNano()) / 1e9

	key := fmt.Sprintf("token:%s:%s", apiKey, identifier)

	result, err := client.EvalSha(ctx, sha, []string{key},
		capacity,
		refillRate,
		now,
	).Result()
	if err != nil {
		return nil, fmt.Errorf("EVALSHA failed: %w", err)
	}

	values, ok := result.([]interface{})
	if !ok || len(values) != 2 {
		return nil, fmt.Errorf("unexpected Lua script response format")
	}

	allowed := values[0].(int64) == 1
	remaining := int(values[1].(int64))

	refillSeconds := float64(capacity-remaining) / refillRate
	retryAfter := 0
	if !allowed {
		retryAfter = int(1/refillRate) + 1
	}

	resetAt := int64(now + refillSeconds)

	return &Result{
		Allowed:    allowed,
		Remaining:  remaining,
		ResetAt:    resetAt,
		RetryAfter: retryAfter,
	}, nil
}
