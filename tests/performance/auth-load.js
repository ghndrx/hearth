import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const authLatency = new Trend('auth_latency', true);

export const options = {
  scenarios: {
    auth_load: {
      executor: 'constant-arrival-rate',
      rate: 100,           // 100 requests per second
      timeUnit: '1s',
      duration: '60s',
      preAllocatedVUs: 50,
      maxVUs: 200,
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<300', 'p(99)<500'],
    errors: ['rate<0.01'],  // <1% error rate
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const users = new Map();
let userCounter = 0;

export function setup() {
  // Test that the service is up
  const res = http.get(`${BASE_URL}/health`);
  check(res, {
    'health check ok': (r) => r.status === 200,
  });
  return { startTime: Date.now() };
}

export default function () {
  const testType = Math.random();
  
  if (testType < 0.3) {
    // 30% registration attempts
    testRegistration();
  } else if (testType < 0.8) {
    // 50% login attempts
    testLogin();
  } else {
    // 20% refresh token
    testRefresh();
  }
  
  sleep(0.01); // Small delay between requests
}

function testRegistration() {
  const uniqueId = `${__VU}_${userCounter++}_${Date.now()}`;
  const payload = JSON.stringify({
    email: `user_${uniqueId}@test.local`,
    username: `user_${uniqueId}`,
    password: 'TestPassword123!',
  });
  
  const params = {
    headers: { 'Content-Type': 'application/json' },
    tags: { endpoint: 'register' },
  };
  
  const start = Date.now();
  const res = http.post(`${BASE_URL}/api/v1/auth/register`, payload, params);
  authLatency.add(Date.now() - start);
  
  const success = check(res, {
    'register status is 201': (r) => r.status === 201,
    'register returns token': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.access_token !== undefined;
      } catch {
        return false;
      }
    },
  });
  
  errorRate.add(!success);
  
  if (success && res.status === 201) {
    try {
      const body = JSON.parse(res.body);
      users.set(uniqueId, {
        email: `user_${uniqueId}@test.local`,
        password: 'TestPassword123!',
        token: body.access_token,
        refresh: body.refresh_token,
      });
    } catch {}
  }
}

function testLogin() {
  // Try to login with a fresh user or create one
  const uniqueId = `login_${__VU}_${Date.now()}`;
  const email = `user_${uniqueId}@test.local`;
  
  // First register
  const regPayload = JSON.stringify({
    email: email,
    username: `user_${uniqueId}`,
    password: 'TestPassword123!',
  });
  
  http.post(`${BASE_URL}/api/v1/auth/register`, regPayload, {
    headers: { 'Content-Type': 'application/json' },
  });
  
  // Then login
  const loginPayload = JSON.stringify({
    email: email,
    password: 'TestPassword123!',
  });
  
  const params = {
    headers: { 'Content-Type': 'application/json' },
    tags: { endpoint: 'login' },
  };
  
  const start = Date.now();
  const res = http.post(`${BASE_URL}/api/v1/auth/login`, loginPayload, params);
  authLatency.add(Date.now() - start);
  
  const success = check(res, {
    'login status is 200': (r) => r.status === 200,
    'login returns token': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.access_token !== undefined;
      } catch {
        return false;
      }
    },
  });
  
  errorRate.add(!success);
}

function testRefresh() {
  // Create user and get refresh token
  const uniqueId = `refresh_${__VU}_${Date.now()}`;
  const email = `user_${uniqueId}@test.local`;
  
  const regPayload = JSON.stringify({
    email: email,
    username: `user_${uniqueId}`,
    password: 'TestPassword123!',
  });
  
  const regRes = http.post(`${BASE_URL}/api/v1/auth/register`, regPayload, {
    headers: { 'Content-Type': 'application/json' },
  });
  
  if (regRes.status !== 201) {
    errorRate.add(true);
    return;
  }
  
  let refreshToken;
  try {
    const body = JSON.parse(regRes.body);
    refreshToken = body.refresh_token;
  } catch {
    errorRate.add(true);
    return;
  }
  
  const refreshPayload = JSON.stringify({
    refresh_token: refreshToken,
  });
  
  const params = {
    headers: { 'Content-Type': 'application/json' },
    tags: { endpoint: 'refresh' },
  };
  
  const start = Date.now();
  const res = http.post(`${BASE_URL}/api/v1/auth/refresh`, refreshPayload, params);
  authLatency.add(Date.now() - start);
  
  const success = check(res, {
    'refresh status is 200': (r) => r.status === 200,
    'refresh returns new token': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.access_token !== undefined;
      } catch {
        return false;
      }
    },
  });
  
  errorRate.add(!success);
}

export function teardown(data) {
  console.log(`Test duration: ${(Date.now() - data.startTime) / 1000}s`);
}
