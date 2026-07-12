# Load Test Findings

We wanted to find out what actually breaks first when Throttle gets a lot of traffic, instead of guessing. Turns out it's not what we expected.

## What we tested

Hit `/check` with k6, ramping from 50 up to 500 concurrent users over a minute, and also steady 200-user runs for 20 seconds each. Every request used a unique identifier so we weren't just hammering one Redis key.

## What we found

- No failed requests at any point — nothing was actually crashing or erroring out.
- But latency got rough under load: average response time around 70-100ms, with the slowest 5% of requests taking 150-290ms. Compare that to the `/health` endpoint, which stays under 10ms even under load.
- **First guess: the Postgres connection pool running out (only 25 connections allowed).** Checked with `pg_stat_activity` — nope, active connections never went past 7. Ruled out.
- **Second guess: our own laptop being slow because the load generator (k6) and the server were fighting over the same CPU cores.** Tried pinning them to separate cores with `taskset` so they couldn't compete. Latency barely changed. Ruled out (mostly).
- **Then we checked each piece's CPU usage separately** during the load: the Go app itself was only using about 25% of a core, Redis about 30% — but **Postgres was using 260-280% CPU**, more than two and a half cores' worth.
- Digging into why with `EXPLAIN ANALYZE`: every `/check` request makes 2 Postgres queries — one to check the API key, one to check if the identifier is exempt. Each query only takes a fraction of a millisecond to actually *run*, but planning the query (figuring out how to run it) took noticeably longer than running it.
- **We turned on `pg_stat_statements`** (Postgres's built-in query stats tracker) to get real numbers instead of a one-off manual check, and re-ran the test. Across 57,144 real requests:

  | Query | Calls | Mean plan time | Mean exec time |
  |---|---|---|---|
  | Auth lookup (checks the API key) | 57,144 | 0.098ms | 0.024ms (~4x planning vs running) |
  | Exemption check | 57,144 | 0.082ms | 0.031ms (~3x planning vs running) |

  We're not using prepared statements anywhere, so Postgres re-plans these same two queries from scratch on every single request instead of reusing a saved plan. At ~2,000-2,900 requests/sec x 2 queries each, that repeated planning work is what's actually eating the CPU.

- **Bonus finding we weren't looking for:** `pg_stat_statements` also showed a third, hidden query — a Postgres-internal check that runs whenever `usage_logs` inserts a row (since it has a foreign key pointing at `clients`). It only ran 6,328 times out of 57,144 requests, about 11%. That's hard proof that our bounded worker pool (10 workers, queue of 1000, added back in Phase 0) is dropping roughly 89% of usage/stats records at this load level — exactly the "usage job queue full, dropping" behavior we saw in the logs earlier, now confirmed with real numbers instead of just log lines.

- **Compared fixed_window against token_bucket** under the same fixed 200-user load: nearly identical latency (fixed_window avg 70ms / p95 168ms vs token_bucket avg 67ms / p95 153ms — token_bucket was actually a touch faster, well within normal noise). This tells us the algorithm itself (simple `INCR` vs a Lua script round trip) barely matters here — Postgres dominates the cost regardless of which rate-limiting algorithm runs. Didn't bother testing sliding_window separately since it uses the same kind of simple Redis calls as fixed_window and would very likely show the same thing.

## Bottom line

The bottleneck isn't too much data, running out of connections, missing indexes, or the rate-limiting algorithm itself. It's that every request pays for two Postgres queries to be *re-planned from scratch* instead of reusing a saved plan, and at real request volume that adds up to being the single biggest cost in the whole request.

## What we did to check this properly (not just guess)

- Ruled out connection pool exhaustion with `pg_stat_activity` before assuming it.
- Ruled out "it's just our laptop / k6 stealing CPU" by actually isolating processes onto separate CPU cores with `taskset` and re-measuring, not just assuming.
- Confirmed which process was actually burning CPU (Postgres, not the Go app or Redis) with `docker stats` and `ps`, instead of guessing from vague symptoms.
- Turned on `pg_stat_statements` for real, live numbers instead of trusting a single manual `EXPLAIN ANALYZE` sample — and that correction mattered, since the live numbers (3-4x) were less dramatic than the one-off sample suggested (15x).

## Known gaps (being upfront about them)

- The `clients` table only has a handful of rows right now from all our testing today. Right now the auth lookup query does a sequential scan instead of using its index — which is actually the *correct* choice by Postgres for a tiny table, but at real scale (thousands of clients) this could behave differently and is worth re-checking then.
- We only load-tested from one laptop, so the absolute numbers (2,000-2,900 req/s) are specific to this hardware, not a general "this is Throttle's max throughput" claim.

## The fix, and proof it worked

Switched the Postgres driver from `lib/pq` to `pgx` (`github.com/jackc/pgx/v5/stdlib`), which caches query plans per connection automatically — no manual prepared-statement code needed, just a driver swap in `db/postgres.go`.

Reset `pg_stat_statements` and ran the exact same fixed-load test again to prove it actually worked, not just assume it did:

| | lib/pq (before) | pgx (after) |
|---|---|---|
| Requests handled | 57,144 | 90,209 |
| Times queries were re-planned | 57,144 (every call) | **150 total** |
| Total planning CPU-time (both queries) | ~10,258ms | **~112ms** |
| Throughput | ~2,834 req/s | ~4,500 req/s (+59%) |
| p95 latency | 168ms | 98ms (-42%) |

The `calls` counter (90,209) and `plans` counter (150) are separate in `pg_stat_statements` — most calls now reuse a cached plan and skip the planner entirely, instead of re-planning on every single request. That's about a 99% cut in planning-related CPU work, while handling more requests than the previous run, not fewer.

Went with `pgx` over manually managing prepared statements ourselves because it fixes this for every current and future query with one driver-level change, and it's the more actively maintained, widely-used Postgres driver in the Go ecosystem today (`lib/pq` is in maintenance-only mode).

## Finding the actual ceiling

Everything above found *a* bottleneck and fixed it, but never answered the original question: where does Throttle actually break? All our tests up to this point topped out at 500 virtual users with 0% failures — we hadn't pushed hard enough to find out.

Ran a much bigger ramp after the pgx fix: 200 → 500 → 1,000 → 2,000 → 4,000 concurrent users over ~110 seconds, isolating the app+databases and the load generator onto separate CPU cores again (same `taskset` technique as before) for a cleaner signal.

**Result:**

| | |
|---|---|
| Total requests | 468,808 |
| Peak throughput | ~4,260 req/s |
| Failures | 23 (0.005%) |
| Failure type | Client-side timeouts (10s) — no server errors, no crashes, nothing in the app log |
| p95 latency at peak | 1.01s (worst case 4.54s) |

**Checked the Postgres-pool-exhaustion hypothesis again** (now that queries are fast, maybe *more* requests could be in flight at once, queuing for one of the 25 pool connections) — watched `pg_stat_activity` for the entire test. Active connections never exceeded **8**, even at the 4,000-VU peak. Ruled out, cleanly, with data, same as the first time. Since each query now takes well under a millisecond, a single connection can service thousands of sequential requests per second, so the pool never comes close to saturating.

We didn't chase the exact next bottleneck (CPU on the app itself vs. Redis) beyond this — diminishing returns given how much ground this load-testing phase already covered, and the actual result is already a strong, complete story: **at roughly 8x more concurrent load than anything tested before, 99.995% of requests still succeeded, and the failures that did happen were timeouts, not crashes.** The system degrades gracefully rather than falling over.

Numbers are specific to this laptop, same caveat as before — not a general "Throttle's max throughput" claim, just what this hardware could sustain.
