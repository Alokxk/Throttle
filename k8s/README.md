# Throttle on Kubernetes

Deploys the full Throttle stack to a local [kind](https://kind.sigs.k8s.io/)
cluster: the app, Postgres, Redis, an in-cluster Prometheus + KEDA for
autoscaling, and Loki + Promtail for centralized logs — with proper
readiness/liveness probes, persistent storage, secrets, and external access
via Ingress.

## Architecture

Everything runs inside the cluster, in its own `throttle` namespace:

| Resource | What | Why |
|---|---|---|
| `throttle` Deployment | 2-6 replicas of the app | Readiness probe hits `/health` (checks Postgres+Redis); liveness probe hits `/live` (checks only that the process itself is responsive) — these are deliberately different endpoints, see [handlers/health.go](../handlers/health.go) |
| `postgres` Deployment + PVC | Single Postgres instance, persistent storage | Client records, rules, and exemptions need to survive a pod restart |
| `redis` Deployment | Single Redis instance, **no** persistent storage | Rate-limit counters are TTL'd and short-lived by design — losing them on a restart just resets limits once, not a correctness issue |
| `throttle-secrets` Secret | DB connection strings | Keeps credentials out of the Deployment specs. Note: these are local dev placeholder credentials (same values already public in `docker-compose.yml`) — a real production setup would inject secrets via a cloud KMS or Sealed Secrets, never commit them as plain `stringData` like this file does |
| `Ingress` (nginx) | Routes `throttle.local` → the app | Only way to reach the app from outside the cluster; without it you're limited to `kubectl exec`/debug pods |
| `prometheus` (in-cluster) | Scrapes the K8s app's `/metrics` | Separate from the `docker-compose` Prometheus, which watches the locally-run app instead — two workflows, each with its own observability, same reasoning as Postgres/Redis above |
| KEDA + `ScaledObject` | Autoscales `throttle` on p95 `/check` latency | Deliberately **not** CPU-based — load testing showed the app barely uses CPU even under heavy load, so a default CPU trigger would almost never fire. Proven with a controlled before/after load test, which also caught a real connection-pool bug — see [`loadtest/FINDINGS.md`](../loadtest/FINDINGS.md#does-the-kubernetes-autoscaler-actually-help) |
| Loki + Promtail | Centralized logs across all replicas | Searching each of up to 6 pods individually via `kubectl logs` doesn't scale; Promtail ships every pod's logs to Loki with labels, one place to search |

Migrations aren't a ConfigMap or an init step at all — they're embedded into
the app binary (`db/migrate.go`, via `go:embed`) and applied automatically
with [`golang-migrate`](https://github.com/golang-migrate/migrate) before the
server starts accepting traffic. Same mechanism locally, in `docker-compose`,
and here — no separate file that can drift out of sync.

## Prerequisites

- Docker (with Docker Desktop's `kind`-compatible engine)
- [kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation)
- [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl)

## One-time setup: create the cluster

`kind`'s port mappings (needed for Ingress to reach port 80 on your machine)
can only be set when the cluster is created, so this is a separate, one-time
step before deploying:

```bash
kind create cluster --name throttle --config k8s/kind-cluster-config.yaml
```

## Deploy

```bash
./k8s/deploy.sh
```

This builds the app image, loads it into the cluster, and applies everything
in the right order (namespace → secrets → migrations → Postgres → Redis →
app → ingress controller → Ingress → in-cluster Prometheus → KEDA →
autoscaling → Loki → Promtail), waiting for each piece to actually be ready
before moving to the next. Safe to re-run — every step is idempotent.

## Using the app

The Ingress routes requests for the host `throttle.local`. Since that's not
a real DNS name, pass it as a header instead of adding it to `/etc/hosts`:

```bash
curl -H "Host: throttle.local" http://localhost/health

curl -H "Host: throttle.local" -X POST http://localhost/register \
  -H "Content-Type: application/json" \
  -d '{"name":"my-app","email":"dev@example.com"}'

curl -H "Host: throttle.local" -X POST http://localhost/check \
  -H "X-API-Key: <api_key from register>" \
  -H "Content-Type: application/json" \
  -d '{"identifier":"user_123","limit":100,"window":60,"algorithm":"fixed_window"}'
```

(If you'd rather use a real hostname, add `127.0.0.1 throttle.local` to
`/etc/hosts` and drop the `-H "Host: ..."` flag.)

## Useful commands

```bash
# Overall status
kubectl get all -n throttle

# Watch pods come up
kubectl get pods -n throttle -w

# Logs for one pod (structured JSON, one line per log entry)
kubectl logs -n throttle -l app=throttle -f

# Current autoscaler status — replica count, current vs. target latency
kubectl get hpa -n throttle
kubectl get scaledobject -n throttle

# Search logs across every replica at once, via Loki (needs a running pod
# to curl from, e.g. any debug pod inside the cluster):
#   curl -G http://loki.throttle.svc.cluster.local:3100/loki/api/v1/query_range \
#     --data-urlencode 'query={app="throttle"} |= "error"'

# Tear everything down (keeps the cluster itself)
kubectl delete namespace throttle

# Delete the cluster entirely
kind delete cluster --name throttle
```

Note: don't `kubectl scale deployment throttle` manually — KEDA owns replica
count via the `ScaledObject`, and will override a manual scale on its next
reconcile. To change the scaling range, edit `minReplicaCount`/
`maxReplicaCount` in `scaledobject.yaml` instead.