# Throttle dashboard

A small operator UI over the Throttle API: live usage stats, and CRUD for
rules and exemptions. React + TypeScript + Tailwind, no backend of its own —
it's a thin client over the existing `/me`, `/stats`, `/rules`, and
`/exemptions` endpoints, authenticated the same way any other API caller is
(an API key, pasted in on the login screen and kept in `sessionStorage`).

## Running it

Via `docker-compose up` from the repo root — starts alongside the API,
Postgres, Redis, Prometheus, and Grafana. Reachable at
`http://localhost:5173`.

For local frontend development with hot reload:

```bash
cp .env.example .env   # only needed if the API isn't on localhost:8080
npm install
npm run dev
```

The Go API needs `CORS_ALLOWED_ORIGIN=http://localhost:5173` set (this is
already the default — see `config/config.go`) so the browser is allowed to
call it from a different origin.

## Why polling, not a WebSocket/SSE feed

The "live" stats on the Overview tab poll `GET /stats/:client_id` every 3
seconds rather than pushing over a socket. The API only tracks aggregate
Redis counters, not a per-request event stream, so there's no live feed to
subscribe to in the first place — and for a dashboard refreshing a few
numbers, polling is simpler to reason about and debug than a persistent
connection, at a granularity a human can't tell apart from "live" anyway.

## Structure

```
src/
  lib/api.ts        API client (fetch wrapper, typed responses)
  components/        Layout, StatTile, Logo, icons
  pages/              Login, Overview, Rules, Exemptions
```

Tab navigation is plain React state, not a router — three tabs behind one
API key doesn't need URL-addressable routes.
