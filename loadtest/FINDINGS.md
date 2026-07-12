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

## What's next

The likely fix is prepared statements (or a Postgres driver like `pgx` that caches query plans automatically), so Postgres stops re-planning the same two queries every time. Not doing that yet — this file is just the finding, the fix is a separate decision to make.
