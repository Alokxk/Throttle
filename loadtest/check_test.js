import http from 'k6/http';
import { check } from 'k6';

export const options = {
  stages: [
    { duration: '10s', target: 50 },
    { duration: '20s', target: 200 },
    { duration: '20s', target: 500 },
    { duration: '10s', target: 0 },
  ],
};

const API_KEY = __ENV.API_KEY;

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
  });

  check(res, {
    'status is 200': (r) => r.status === 200,
  });
}
