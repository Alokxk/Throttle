import http from 'k6/http';
import { check } from 'k6';

export const options = {
  vus: 200,
  duration: '20s',
};

const API_KEY = __ENV.API_KEY;

export default function () {
  const identifier = `loadtest_${__VU}_${__ITER}`;
  const payload = JSON.stringify({
    identifier: identifier,
    limit: 1000000,
    refill_rate: 100000,
    algorithm: 'token_bucket',
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
