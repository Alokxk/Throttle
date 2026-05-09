# Throttle

Rate limiting as a service — Token Bucket, Sliding Window, and Fixed Window algorithms backed by Redis atomic operations.

![Throttle](assets/throttle-dark.png)

![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go&logoColor=white)
![Redis](https://img.shields.io/badge/Redis-7+-DC382D?style=flat&logo=redis&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-14-336791?style=flat&logo=postgresql&logoColor=white)

## What is Throttle

Throttle is a standalone HTTP service that handles rate limiting for any application. Instead of building rate limiting logic into each of your services, you register with Throttle once and call a single endpoint before processing each request.

Any application in any language can use it — Python, Ruby, Node, Java — by making one HTTP call. Throttle handles all the Redis counters, algorithm logic, and distributed state.

## How it works

### The Algorithms

**1. Fixed Window**

Divides time into fixed chunks — windows. Each window gets a fresh counter. Simple and fast, but has an edge case: a user can send double the limit in a short burst at the window boundary (100 requests at 00:59, 100 more at 01:01 — 200 requests in 2 seconds against a 100/min limit).

```
Window 1          Window 2
00:00 ──── 01:00  01:00 ──── 02:00
[████████░░]      [░░░░░░░░░░]
70/100 used       0/100 used
```

**2. Sliding Window**

Fixes the boundary spike using a weighted average across the current and previous window. More accurate, same Redis memory footprint as fixed window.

```
weighted = previous_count × overlap + current_count
```

If you're 15 seconds into a 60-second window, the previous window contributes 75% of its count to the total. Smooth, accurate, no boundary spikes.

**3. Token Bucket**

A bucket fills with tokens at a constant rate. Each request consumes one token. If the bucket is empty, the request is rejected. Allows short bursts while enforcing a long-term average rate — this is what AWS, Stripe, and Twilio use.

```
Capacity: 100 tokens
Refill:   10 tokens/second
Request --> consume 1 token --> allowed
No tokens --> rejected --> retry after N seconds
```

### Why Redis Lua for token bucket

The token bucket needs three operations: read token count → calculate refill → write new count. If two servers run these simultaneously without atomicity, both read the same count and both allow — one too many requests goes through.

We use a Redis Lua script via `EVALSHA`. Lua scripts execute atomically on the Redis server side — no other command runs between the read and the write, regardless of how many servers are calling it simultaneously.

### Why not WATCH/MULTI/EXEC

`WATCH/MULTI/EXEC` uses optimistic locking — if another client modifies the key between `WATCH` and `EXEC`, the transaction aborts and retries. Under high load this creates a retry storm. Lua scripts never abort — they just run.

## Running locally

### Option 1 — Docker Compose (recommended)

One command, no setup:

```bash
git clone https://github.com/Alokxk/Throttle.git
cd Throttle
docker-compose up --build
```

PostgreSQL, Redis, and the server start together. Migrations run automatically.

### Option 2 — Manual setup

**Prerequisites:** Go 1.24+, PostgreSQL, Redis

```bash
git clone https://github.com/Alokxk/Throttle.git
cd Throttle

cp .env.example .env
# Edit .env with your database credentials
```

`.env` variables:

```
PORT=8080
DATABASE_URL=postgresql://postgres:postgres@localhost/throttle?sslmode=disable
REDIS_URL=redis://localhost:6379/0
```

Create the database and run migrations:

```bash
make createdb
make migrate
```

Start the server:

```bash
make run
```

### Makefile commands

- `make run` — Start the server
- `make build` — Compile the binary
- `make test` — Run all tests
- `make migrate` — Run database migrations
- `make docker-up` — Start with Docker Compose
- `make docker-down` — Stop Docker Compose

## Tech stack

- Language: Go 1.24 — standard library HTTP server, no framework
- Rate limit state: Redis — atomic `INCR`, sorted sets, Lua scripts
- Client storage: PostgreSQL — registration, rules, exemptions
- Algorithms: Fixed window, sliding window, token bucket

**Why Go's standard library over Gin/Echo:** Forces explicit understanding of how HTTP works. No magic — every line is intentional.

**Why Redis for counters:** `INCR` is atomic. Two servers checking the same counter simultaneously will always get consistent results. Sub-millisecond latency keeps rate checks off the critical path.

**Why PostgreSQL for clients only:** Rules and client data are relational, written once, read often. Redis handles the hot path; PostgreSQL handles the cold path.

## API reference

### `GET /health`

Health check endpoint. No authentication required. Returns `{"status": "ok"}`.

```bash
curl http://localhost:8080/health
```

### `POST /register`

Register your application and get an API key.

```bash
curl -X POST http://localhost:8080/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-app",
    "email": "dev@example.com",
    "default_algorithm": "sliding_window"
  }'
```

```json
{
  "client_id": "uuid",
  "api_key": "thr_xxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "default_algorithm": "sliding_window",
  "created_at": "2024-01-01T00:00:00Z"
}
```

### `POST /check`

Check if a request should be allowed or rejected.

```bash
curl -X POST http://localhost:8080/check \
  -H "X-API-Key: thr_xxxxxxxxxxxxxxxxxxxxxxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "identifier": "user_123",
    "limit": 100,
    "window": 60,
    "algorithm": "sliding_window"
  }'
```

```json
{
  "allowed": true,
  "remaining": 47,
  "reset_at": 1234567890,
  "algorithm": "sliding_window",
  "warning": false,
  "retry_after": 0
}
```

**Fields:**

- `identifier` — The end-user or entity being rate limited. Required for user-based checks; omit for IP-based checks.
- `limit` — Maximum number of requests allowed in the window.
- `window` — Window size in seconds. Not applicable when using the `token_bucket` algorithm.
- `algorithm` — Rate limiting algorithm to use: `fixed_window`, `sliding_window`, or `token_bucket`.
- `rule` — Name of a pre-configured rule. If provided, it overrides `limit`, `window`, and `algorithm`.
- `warn_threshold` — Decimal between 0 and 1 (default: 0.2). When the fraction of remaining allowance falls below this value, the response includes a warning.
- `refill_rate` — Tokens-per-second refill rate for the `token_bucket` algorithm. Defaults to `limit/60` if not specified.

**Response headers**

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 47
X-RateLimit-Reset: 1234567890
X-RateLimit-Algorithm: sliding_window
X-RateLimit-Warning: true        (only when warning threshold crossed)
Retry-After: 13                  (only when rejected)
```

### `POST /check/ip`

Rate limit by IP address — no identifier needed. Throttle extracts the real IP automatically, handling `X-Forwarded-For` and `X-Real-IP` headers from proxies.

```bash
curl -X POST http://localhost:8080/check/ip \
  -H "X-API-Key: thr_xxxxxxxxxxxxxxxxxxxxxxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "limit": 10,
    "window": 60,
    "algorithm": "fixed_window"
  }'
```

```json
{
  "allowed": true,
  "identifier": "203.0.113.1",
  "remaining": 9,
  "reset_at": 1234567890
}
```

### `POST /rules`

Create a reusable rate limit policy. Reference it by name on `/check` instead of passing limit/window/algorithm every time.

```bash
curl -X POST http://localhost:8080/rules \
  -H "X-API-Key: thr_xxxxxxxxxxxxxxxxxxxxxxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "api_default",
    "algorithm": "sliding_window",
    "limit": 100,
    "window": 60
  }'
```

Use the rule:

```bash
curl -X POST http://localhost:8080/check \
  -H "X-API-Key: thr_xxxxxxxxxxxxxxxxxxxxxxxxxxxx" \
  -d '{"identifier": "user_123", "rule": "api_default"}'
```

### `GET /rules/list`

List all rules for your account.

```bash
curl http://localhost:8080/rules/list \
  -H "X-API-Key: thr_xxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```

```json
[
  {
    "name": "api_default",
    "algorithm": "sliding_window",
    "limit": 100,
    "window": 60
  }
]
```

### `DELETE /rules/:name`

Delete a rule by name.

```bash
curl -X DELETE http://localhost:8080/rules/api_default \
  -H "X-API-Key: thr_xxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```

### `POST /reset`

Clear a specific identifier's rate limit counter. Useful for testing or giving a user a clean slate after a support interaction.

```bash
curl -X POST http://localhost:8080/reset \
  -H "X-API-Key: thr_xxxxxxxxxxxxxxxxxxxxxxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "identifier": "user_123",
    "algorithm": "fixed_window"
  }'
```

```json
{
  "message": "Identifier reset successfully",
  "identifier": "user_123",
  "keys_deleted": 1
}
```

### `POST /exemptions`

Whitelist an identifier from rate limiting entirely. Useful for internal services, admin users, or health check bots.

```bash
curl -X POST http://localhost:8080/exemptions \
  -H "X-API-Key: thr_xxxxxxxxxxxxxxxxxxxxxxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "identifier": "internal-service",
    "reason": "Internal microservice, no rate limiting needed"
  }'
```

### `GET /exemptions/list`

List all exemptions for your account.

```bash
curl http://localhost:8080/exemptions/list \
  -H "X-API-Key: thr_xxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```

```json
[
  {
    "identifier": "internal-service",
    "reason": "Internal microservice, no rate limiting needed"
  }
]
```

### `DELETE /exemptions/:identifier`

Remove an exemption.

```bash
curl -X DELETE http://localhost:8080/exemptions/internal-service \
  -H "X-API-Key: thr_xxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```

### `GET /stats/:client_id`

Usage statistics for your account. Reads from Redis counters — instant regardless of request volume. The `client_id` is returned when you call `/register`.

```bash
curl http://localhost:8080/stats/550e8400-e29b-41d4-a716-446655440000 \
  -H "X-API-Key: thr_xxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```

```json
{
  "client_id": "uuid",
  "total_checks": 15234,
  "allowed": 14891,
  "rejected": 343,
  "by_algorithm": {
    "fixed_window": 8000,
    "sliding_window": 6000,
    "token_bucket": 1234
  }
}
```

## Known limitations

**Single Redis instance** — A single point of failure. Production deployments should use Redis Sentinel for automatic failover or Redis Cluster for horizontal scaling.

**Functional tests, not integration tests** — Tests run against real local PostgreSQL and Redis. True integration tests would spin up isolated containers per test run using `testcontainers-go`.

**Sliding window is an approximation** — Uses a weighted average across two windows rather than tracking every request timestamp. Accurate enough for production use but not mathematically exact.

**No request timeouts** — Database calls don't have context deadlines. Under extreme load, slow database responses could cause requests to hang.
