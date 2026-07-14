import http from 'k6/http';
import { check } from 'k6';
import type { Options } from 'k6/options';

// Targets the in-cluster app through the Ingress (not localhost:8080 like the
// other scripts) — the point of this test is to observe KEDA's Prometheus
// scaler reacting to real p95 latency, which only the in-cluster Prometheus
// (k8s/prometheus.yaml) sees.
export const options: Options = {
  stages: [
    { duration: '20s', target: 150 },
    { duration: '20s', target: 400 },
    { duration: '4m', target: 400 },
    { duration: '2m', target: 400 },
    { duration: '20s', target: 0 },
  ],
};

const API_KEY: string = __ENV.API_KEY;

export default function () {
  const identifier = `k8s_autoscale_${__VU}_${__ITER}`;
  const payload = JSON.stringify({
    identifier: identifier,
    limit: 1000000,
    window: 60,
    algorithm: 'fixed_window',
  });

  const res = http.post('http://localhost/check', payload, {
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': API_KEY,
      Host: 'throttle.local',
    },
    timeout: '10s',
  });

  check(res, {
    'status is 200': (r) => r.status === 200,
  });
}
