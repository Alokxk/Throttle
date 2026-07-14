import http from 'k6/http';
import { check } from 'k6';
import type { Options } from 'k6/options';

export const options: Options = {
  stages: [
    { duration: '15s', target: 200 },
    { duration: '20s', target: 500 },
    { duration: '20s', target: 1000 },
    { duration: '20s', target: 2000 },
    { duration: '20s', target: 4000 },
    { duration: '15s', target: 0 },
  ],
};

const API_KEY: string = __ENV.API_KEY;

export default function () {
  const identifier = `loadtest_${__VU}_${__ITER}`;
  const payload = JSON.stringify({
    identifier: identifier,
    limit: 1000000,
    window: 60,
    algorithm: 'fixed_window',
  });

  const res = http.post('http://localhost:8080/check', payload, {
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': API_KEY,
    },
    timeout: '10s',
  });

  check(res, {
    'status is 200': (r) => r.status === 200,
  });
}
