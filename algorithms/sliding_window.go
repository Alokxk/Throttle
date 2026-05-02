package algorithms

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func SlidingWindow(ctx context.Context, client *redis.Client, apiKey, identifier string, limit int, windowSeconds int) (*Result, error) {
	now := time.Now().Unix()
	windowSize := int64(windowSeconds)

	currentWindowStart := (now / windowSize) * windowSize
	previousWindowStart := currentWindowStart - windowSize

	currentKey := fmt.Sprintf("sliding:%s:%s:%d", apiKey, identifier, currentWindowStart)
	previousKey := fmt.Sprintf("sliding:%s:%s:%d", apiKey, identifier, previousWindowStart)

	pipe := client.Pipeline()
	currentIncrCmd := pipe.Incr(ctx, currentKey)
	pipe.Expire(ctx, currentKey, time.Duration(windowSeconds*2)*time.Second)
	previousGetCmd := pipe.Get(ctx, previousKey)
	_, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("redis pipeline failed: %w", err)
	}

	currentCount, err := currentIncrCmd.Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get current count: %w", err)
	}

	var previousCount int64
	if val, err := previousGetCmd.Result(); err == nil {
		fmt.Sscanf(val, "%d", &previousCount)
	}

	elapsed := now - currentWindowStart
	overlap := float64(windowSize-elapsed) / float64(windowSize)
	weightedCount := float64(previousCount)*overlap + float64(currentCount)

	windowEnd := currentWindowStart + windowSize
	remaining := int(float64(limit) - weightedCount)
	if remaining < 0 {
		remaining = 0
	}

	return &Result{
		Allowed:    weightedCount <= float64(limit),
		Remaining:  remaining,
		ResetAt:    windowEnd,
		RetryAfter: int(windowEnd - now),
	}, nil
}
