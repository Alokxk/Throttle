package algorithms_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Alokxk/Throttle/algorithms"
	"github.com/Alokxk/Throttle/config"
	"github.com/Alokxk/Throttle/db"
)

func setupRedis() *db.RedisClient {
	cfg := config.Load()
	return db.NewRedisClient(cfg.RedisURL)
}

func TestFixedWindow_ConcurrentRequests_NeverExceedsLimit(t *testing.T) {
	rc := setupRedis()
	defer rc.Client.Close()

	ctx := context.Background()
	identifier := "concurrent_fixed_test"
	apiKey := "bench_key_fixed"
	limit := 50
	concurrency := 200

	var wg sync.WaitGroup
	var allowedCount atomic.Int64

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := algorithms.FixedWindow(ctx, rc.Client, apiKey, identifier, limit, 60)
			if err != nil {
				t.Errorf("FixedWindow error: %v", err)
				return
			}
			if res.Allowed {
				allowedCount.Add(1)
			}
		}()
	}
	wg.Wait()

	if allowedCount.Load() > int64(limit) {
		t.Errorf("allowed %d requests under limit %d", allowedCount.Load(), limit)
	}
}

func TestSlidingWindow_ConcurrentRequests_NeverExceedsLimit(t *testing.T) {
	rc := setupRedis()
	defer rc.Client.Close()

	ctx := context.Background()
	identifier := "concurrent_sliding_test"
	apiKey := "bench_key_sliding"
	limit := 50
	concurrency := 200

	var wg sync.WaitGroup
	var allowedCount atomic.Int64

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := algorithms.SlidingWindow(ctx, rc.Client, apiKey, identifier, limit, 60)
			if err != nil {
				t.Errorf("SlidingWindow error: %v", err)
				return
			}
			if res.Allowed {
				allowedCount.Add(1)
			}
		}()
	}
	wg.Wait()

	// Approximation algorithm: allow 20% overshoot, still catches a broken/non-atomic implementation.
	maxAcceptable := int64(float64(limit) * 1.2)
	if allowedCount.Load() > maxAcceptable {
		t.Errorf("allowed %d requests, expected <= %d", allowedCount.Load(), maxAcceptable)
	}
}

func TestTokenBucket_ConcurrentRequests_NeverExceedsCapacity(t *testing.T) {
	rc := setupRedis()
	defer rc.Client.Close()

	ctx := context.Background()
	identifier := "concurrent_token_test"
	apiKey := "bench_key_token"
	capacity := 50
	concurrency := 200

	var wg sync.WaitGroup
	var allowedCount atomic.Int64

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Near-zero refill isolates the capacity check from refill timing.
			res, err := algorithms.TokenBucket(ctx, rc.Client, rc, rc.TokenBucketSHA, apiKey, identifier, capacity, 0.001)
			if err != nil {
				t.Errorf("TokenBucket error: %v", err)
				return
			}
			if res.Allowed {
				allowedCount.Add(1)
			}
		}()
	}
	wg.Wait()

	if allowedCount.Load() > int64(capacity) {
		t.Errorf("allowed %d requests, capacity is %d", allowedCount.Load(), capacity)
	}
}
