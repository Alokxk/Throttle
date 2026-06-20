package db

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"runtime"

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
		log.Fatalf("Failed to parse Redis URL: %v", err)
	}

	client := redis.NewClient(opts)

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	log.Println("Redis connected")

	script := readTokenBucketScript()
	sha, err := client.ScriptLoad(ctx, script).Result()
	if err != nil {
		log.Fatalf("Failed to load token bucket script into Redis: %v", err)
	}
	log.Printf("Token bucket Lua script loaded, SHA: %s", sha)

	return &RedisClient{
		Client:            client,
		TokenBucketSHA:    sha,
		tokenBucketScript: script,
	}
}

func readTokenBucketScript() string {
	_, filename, _, _ := runtime.Caller(0)
	scriptPath := filepath.Join(filepath.Dir(filename), "..", "algorithms", "scripts", "token_bucket.lua")

	script, err := os.ReadFile(scriptPath)
	if err != nil {
		log.Fatalf("Failed to read token bucket Lua script: %v", err)
	}
	return string(script)
}

// Reload re-registers the script with Redis and updates the cached SHA.
// Call this when EVALSHA fails with NOSCRIPT (e.g. after a Redis restart
// flushed the script cache).
func (r *RedisClient) ReloadTokenBucketScript(ctx context.Context) (string, error) {
	sha, err := r.Client.ScriptLoad(ctx, r.tokenBucketScript).Result()
	if err != nil {
		return "", err
	}
	r.TokenBucketSHA = sha
	return sha, nil
}
