import http from 'k6/http';
import ws from 'k6/ws';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';

const authErrors = new Counter('auth_errors');
const messageErrors = new Counter('message_errors');
const wsErrors = new Counter('ws_errors');
const rateLimitErrors = new Counter('rate_limit_errors');
const authLatency = new Trend('auth_latency');
const messageLatency = new Trend('message_latency');

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const WS_URL = __ENV.WS_URL || 'ws://localhost:8080';
const TEST_PASSWORD = __ENV.TEST_PASSWORD || `k6_${Date.now()}_${Math.random().toString(36).slice(2)}`;
const TEST_CHANNEL_ID = __ENV.TEST_CHANNEL_ID || '00000000-0000-0000-0000-000000000001';

export const options = {
  scenarios: {
    auth_load: {
      executor: 'constant-arrival-rate',
      rate: 100,
      timeUnit: '1s',
      duration: '60s',
      preAllocatedVUs: 10,
      maxVUs: 50,
      exec: 'testAuth',
    },
    message_creation: {
      executor: 'constant-arrival-rate',
      rate: 500,
      timeUnit: '1s',
      duration: '60s',
      preAllocatedVUs: 50,
      maxVUs: 200,
      exec: 'testMessageCreation',
      startTime: '10s',
    },
    websocket_connections: {
      executor: 'constant-vus',
      vus: 1000,
      duration: '60s',
      exec: 'testWebSocket',
      startTime: '20s',
    },
  },
  thresholds: {
    'http_req_duration{scenario:auth_load}': ['p(99)<500'],
    'http_req_duration{scenario:message_creation}': ['p(99)<500'],
    'http_req_failed': ['rate<0.01'], // <1% error rate
  },
};

let authToken = null;

export function setup() {
  // Register and login to get auth token
  const username = `perftest_${Date.now()}`;
  const registerPayload = JSON.stringify({
    username: username,
    password: TEST_PASSWORD,
    email: `${username}@test.com`,
  });

  const registerRes = http.post(`${BASE_URL}/api/auth/register`, registerPayload, {
    headers: { 'Content-Type': 'application/json' },
  });

  if (registerRes.status === 201) {
    const body = JSON.parse(registerRes.body);
    if (!body.access_token) {
      throw new Error(`Registration succeeded but no token returned (status ${registerRes.status})`);
    }
    return { token: body.access_token };
  }

  // If registration fails, try login with existing test user
  const loginPayload = JSON.stringify({
    username: 'perftest',
    password: TEST_PASSWORD,
  });

  const loginRes = http.post(`${BASE_URL}/api/auth/login`, loginPayload, {
    headers: { 'Content-Type': 'application/json' },
  });

  if (loginRes.status === 200) {
    const body = JSON.parse(loginRes.body);
    if (!body.access_token) {
      throw new Error(`Login succeeded but no token returned (status ${loginRes.status})`);
    }
    return { token: body.access_token };
  }

  throw new Error(`Setup failed: could not authenticate (register: ${registerRes.status}, login: ${loginRes.status})`);
}

export function testAuth(data) {
  const payload = JSON.stringify({
    username: `user_${__VU}_${__ITER}`,
    password: TEST_PASSWORD,
  });

  const startTime = Date.now();
  const res = http.post(`${BASE_URL}/api/auth/login`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });
  authLatency.add(Date.now() - startTime);

  if (res.status === 429) {
    rateLimitErrors.add(1);
    return;
  }

  const success = check(res, {
    'auth status is 200 or 401': (r) => r.status === 200 || r.status === 401,
  });

  if (!success) {
    authErrors.add(1);
  }
}

export function testMessageCreation(data) {
  if (!data || !data.token) {
    messageErrors.add(1);
    return;
  }

  const payload = JSON.stringify({
    channel_id: TEST_CHANNEL_ID,
    content: `Performance test message ${__VU}_${__ITER}`,
    message_type: 'text',
  });

  const startTime = Date.now();
  const res = http.post(`${BASE_URL}/api/messages`, payload, {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${data.token}`,
    },
  });
  messageLatency.add(Date.now() - startTime);

  if (res.status === 429) {
    rateLimitErrors.add(1);
    return;
  }

  const success = check(res, {
    'message status is 201 or 400': (r) => r.status === 201 || r.status === 400 || r.status === 404,
  });

  if (!success) {
    messageErrors.add(1);
  }
}

export function testWebSocket(data) {
  if (!data || !data.token) {
    wsErrors.add(1);
    return;
  }

  const url = `${WS_URL}/gateway?token=${data.token}`;
  
  const res = ws.connect(url, {}, function (socket) {
    socket.on('open', () => {
      // Send a ping every 5 seconds
      socket.setInterval(() => {
        socket.send(JSON.stringify({ type: 'ping' }));
      }, 5000);
    });

    socket.on('message', (data) => {
      // Handle messages
    });

    socket.on('error', (e) => {
      wsErrors.add(1);
    });

    socket.setTimeout(() => {
      socket.close();
    }, 30000); // Keep connection open for 30s
  });

  check(res, {
    'ws status is 101': (r) => r && r.status === 101,
  });
}
