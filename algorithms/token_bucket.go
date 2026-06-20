package algorithms

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type ScriptReloader interface {
	ReloadTokenBucketScript(ctx context.Context) (string, error)
}

func TokenBucket(ctx context.Context, client *redis.Client, reloader ScriptReloader, sha, apiKey, identifier string, capacity int, refillRate float64) (*Result, error) {
	now := float64(time.Now().UnixNano()) / 1e9
	key := fmt.Sprintf("token:%s:%s", apiKey, identifier)

	result, err := client.EvalSha(ctx, sha, []string{key}, capacity, refillRate, now).Result()
	if err != nil && strings.Contains(err.Error(), "NOSCRIPT") {
		newSHA, reloadErr := reloader.ReloadTokenBucketScript(ctx)
		if reloadErr != nil {
			return nil, fmt.Errorf("script reload failed after NOSCRIPT: %w", reloadErr)
		}
		result, err = client.EvalSha(ctx, newSHA, []string{key}, capacity, refillRate, now).Result()
	}
	if err != nil {
		return nil, fmt.Errorf("EVALSHA failed: %w", err)
	}

	values, ok := result.([]interface{})
	if !ok || len(values) != 2 {
		return nil, fmt.Errorf("unexpected Lua script response shape: %T", result)
	}

	allowedRaw, ok1 := values[0].(int64)
	remainingRaw, ok2 := values[1].(int64)
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("unexpected Lua script response types: %T, %T", values[0], values[1])
	}

	allowed := allowedRaw == 1
	remaining := int(remainingRaw)

	refillSeconds := float64(capacity-remaining) / refillRate
	retryAfter := 0
	if !allowed {
		retryAfter = int(1/refillRate) + 1
	}

	return &Result{
		Allowed:    allowed,
		Remaining:  remaining,
		ResetAt:    int64(now + refillSeconds),
		RetryAfter: retryAfter,
	}, nil
}
