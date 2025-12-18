import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const TOKEN = __ENV.API_TOKEN || ''; // API_KEY token para cenário com token
const DURATION = __ENV.DURATION || '30s';
const RPS = Number(__ENV.RPS || 10);

export const options = {
  scenarios: {
    ip_limit: {
      executor: 'constant-arrival-rate',
      rate: RPS, // requisições por segundo
      timeUnit: '1s',
      duration: DURATION,
      preAllocatedVUs: Math.max(1, RPS * 2),
      maxVUs: Math.max(4, RPS * 4),
      exec: 'ipScenario',
    },
    token_limit: {
      executor: 'constant-arrival-rate',
      rate: RPS,
      timeUnit: '1s',
      duration: DURATION,
      preAllocatedVUs: Math.max(1, RPS * 2),
      maxVUs: Math.max(4, RPS * 4),
      exec: 'tokenScenario',
      // passa TOKEN ao cenário via env
      env: { TOKEN },
      startTime: '0s',
    },
  },
  thresholds: {
    // poucos erros inesperados (status != 200 e != 429)
    http_req_failed: ['rate<0.05'],
  },
};

const allowed = new Counter('allowed_reqs');
const blocked = new Counter('blocked_reqs');
const unexpected = new Counter('unexpected_reqs');
const rt = new Trend('rt_ms');

function record(res, label) {
  rt.add(res.timings.duration, { label });
  if (res.status === 200) {
    allowed.add(1);
  } else if (res.status === 429) {
    // valida mensagem exata
    check(res, {
      '429 mensagem correta': (r) => r && r.body === '{"message": "you have reached the maximum number of requests or actions allowed within a certain time frame"}',
    });
    blocked.add(1);
  } else {
    unexpected.add(1);
  }
}

export function ipScenario() {
  const res = http.get(`${BASE_URL}/`, { tags: { scenario: 'ip' } });
  check(res, { 'status é 200 ou 429 (IP)': (r) => r.status === 200 || r.status === 429 });
  record(res, 'ip');
  sleep(0.05);
}

export function tokenScenario() {
  const headers = TOKEN ? { API_KEY: TOKEN } : {};
  const res = http.get(`${BASE_URL}/`, { headers, tags: { scenario: 'token' } });
  check(res, { 'status é 200 ou 429 (TOKEN)': (r) => r.status === 200 || r.status === 429 });
  record(res, 'token');
  sleep(0.05);
}

export function handleSummary(data) {
  const lines = [];
  lines.push('--- k6 Rate Limiter Summary ---');
  lines.push(`Allowed: ${data.metrics.allowed_reqs?.count || 0}`);
  lines.push(`Blocked (429): ${data.metrics.blocked_reqs?.count || 0}`);
  lines.push(`Unexpected: ${data.metrics.unexpected_reqs?.count || 0}`);
  lines.push(`p95 RT (ms): ${Math.round(data.metrics.rt_ms?.p(95) || 0)}`);
  return { stdout: lines.join('\n') + '\n' };
}
