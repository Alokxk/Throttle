import http from 'k6/http';
import { check } from 'k6';
import type { Options } from 'k6/options';

// Steady fixed load (no ramp) against the Ingress, used to compare
// pinned-replica-count latency against KEDA-autoscaled latency under the
// exact same load — see loadtest/FINDINGS.md.
export const options: Options = {
  stages: [
    { duration: '15s', target: 400 },
    { duration: '3m', target: 400 },
    { duration: '10s', target: 0 },
  ],
};

const API_KEY: string = __ENV.API_KEY;

export default function () {
  const identifier = `k8s_fixed_${__VU}_${__ITER}`;
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
