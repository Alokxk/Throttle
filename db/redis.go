package db

import (
	"context"
	"log/slog"
	"os"

	"github.com/Alokxk/Throttle/algorithms"
	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	Client            *redis.Client
	TokenBucketSHA    string
	tokenBucketScript string
}

func NewRedisClient(redisURL string) *RedisClient {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		slog.Error("failed to parse Redis URL", "error", err)
		os.Exit(1)
	}

	client := redis.NewClient(opts)

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		slog.Error("failed to connect to Redis", "error", err)
		os.Exit(1)
	}

	slog.Info("redis connected")

	sha, err := client.ScriptLoad(ctx, algorithms.TokenBucketScript).Result()
	if err != nil {
		slog.Error("failed to load token bucket script into Redis", "error", err)
		os.Exit(1)
	}
	slog.Info("token bucket Lua script loaded", "sha", sha)

	return &RedisClient{
		Client:            client,
		TokenBucketSHA:    sha,
		tokenBucketScript: algorithms.TokenBucketScript,
	}
}

// Call on EVALSHA NOSCRIPT errors (e.g. after a Redis restart clears the script cache).
func (r *RedisClient) ReloadTokenBucketScript(ctx context.Context) (string, error) {
	sha, err := r.Client.ScriptLoad(ctx, r.tokenBucketScript).Result()
	if err != nil {
		return "", err
	}
	r.TokenBucketSHA = sha
	return sha, nil
}
