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
	Client         *redis.Client
	TokenBucketSHA string
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

	sha := loadTokenBucketScript(client, ctx)

	return &RedisClient{
		Client:         client,
		TokenBucketSHA: sha,
	}
}

func loadTokenBucketScript(client *redis.Client, ctx context.Context) string {
	_, filename, _, _ := runtime.Caller(0)
	scriptPath := filepath.Join(filepath.Dir(filename), "..", "algorithms", "scripts", "token_bucket.lua")

	script, err := os.ReadFile(scriptPath)
	if err != nil {
		log.Fatalf("Failed to read token bucket Lua script: %v", err)
	}

	sha, err := client.ScriptLoad(ctx, string(script)).Result()
	if err != nil {
		log.Fatalf("Failed to load token bucket script into Redis: %v", err)
	}

	log.Printf("Token bucket Lua script loaded, SHA: %s", sha)
	return sha
}
