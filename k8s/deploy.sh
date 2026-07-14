#!/usr/bin/env bash
# Assumes the cluster already exists — see README.md.
set -euo pipefail

cd "$(dirname "$0")/.."

echo "==> Building app image"
docker build -t throttle:local .

echo "==> Loading image into kind"
kind load docker-image throttle:local --name throttle

echo "==> Namespace"
kubectl apply -f k8s/namespace.yaml

echo "==> Secrets"
kubectl apply -f k8s/secret.yaml

echo "==> Postgres"
kubectl apply -f k8s/postgres.yaml
kubectl wait --namespace throttle --for=condition=ready pod -l app=postgres --timeout=120s

echo "==> Redis"
kubectl apply -f k8s/redis.yaml
kubectl wait --namespace throttle --for=condition=ready pod -l app=redis --timeout=60s

echo "==> Throttle app"
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
kubectl wait --namespace throttle --for=condition=ready pod -l app=throttle --timeout=120s

echo "==> Ingress controller (installs once, safe to re-run)"
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=180s

echo "==> Ingress"
kubectl apply -f k8s/ingress.yaml

echo "==> In-cluster Prometheus (scrapes the K8s app, drives autoscaling)"
kubectl apply -f k8s/prometheus.yaml
kubectl wait --namespace throttle --for=condition=ready pod -l app=prometheus --timeout=60s

echo "==> KEDA (installs once, safe to re-run)"
KEDA_VERSION=$(curl -s https://api.github.com/repos/kedacore/keda/releases/latest | grep '"tag_name"' | cut -d '"' -f 4)
kubectl apply --server-side -f "https://github.com/kedacore/keda/releases/download/${KEDA_VERSION}/keda-${KEDA_VERSION#v}.yaml"
kubectl wait --namespace keda --for=condition=available deployment --all --timeout=120s

echo "==> Autoscaling (scales on p95 /check latency, not CPU)"
kubectl apply -f k8s/scaledobject.yaml

echo "==> Loki (log storage)"
kubectl apply -f k8s/loki.yaml
kubectl wait --namespace throttle --for=condition=ready pod -l app=loki --timeout=90s

echo "==> Promtail (ships pod logs to Loki)"
kubectl apply -f k8s/promtail.yaml
kubectl wait --namespace throttle --for=condition=ready pod -l app=promtail --timeout=60s

echo "==> Done. See README.md for how to reach the app."
