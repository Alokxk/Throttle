package algorithms

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Result struct {
	Allowed    bool
	Remaining  int
	ResetAt    int64
	RetryAfter int
}

func FixedWindow(ctx context.Context, client *redis.Client, apiKey, identifier string, limit int, windowSeconds int) (*Result, error) {
	now := time.Now().Unix()
	windowStart := (now / int64(windowSeconds)) * int64(windowSeconds)
	windowEnd := windowStart + int64(windowSeconds)

	key := fmt.Sprintf("fixed:%s:%s:%d", apiKey, identifier, windowStart)

	count, err := client.Incr(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("redis INCR failed: %w", err)
	}

	if count == 1 {
		client.Expire(ctx, key, time.Duration(windowSeconds)*time.Second)
	}

	remaining := limit - int(count)
	if remaining < 0 {
		remaining = 0
	}

	return &Result{
		Allowed:    count <= int64(limit),
		Remaining:  remaining,
		ResetAt:    windowEnd,
		RetryAfter: int(windowEnd - now),
	}, nil
}
