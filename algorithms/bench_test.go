package algorithms_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/Alokxk/Throttle/algorithms"
	"github.com/Alokxk/Throttle/config"
	"github.com/Alokxk/Throttle/db"
)

func BenchmarkFixedWindow(b *testing.B) {
	cfg := config.Load()
	rc := db.NewRedisClient(cfg.RedisURL)
	defer rc.Client.Close()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		identifier := fmt.Sprintf("bench_fixed_%d", i%1000)
		algorithms.FixedWindow(ctx, rc.Client, "bench_api_key", identifier, 1000000, 60)
	}
}

func BenchmarkSlidingWindow(b *testing.B) {
	cfg := config.Load()
	rc := db.NewRedisClient(cfg.RedisURL)
	defer rc.Client.Close()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		identifier := fmt.Sprintf("bench_sliding_%d", i%1000)
		algorithms.SlidingWindow(ctx, rc.Client, "bench_api_key", identifier, 1000000, 60)
	}
}

func BenchmarkTokenBucket(b *testing.B) {
	cfg := config.Load()
	rc := db.NewRedisClient(cfg.RedisURL)
	defer rc.Client.Close()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		identifier := fmt.Sprintf("bench_token_%d", i%1000)
		algorithms.TokenBucket(ctx, rc.Client, rc, rc.TokenBucketSHA, "bench_api_key", identifier, 1000000, 1000)
	}
}
