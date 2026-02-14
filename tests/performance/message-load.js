import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const messageLatency = new Trend('message_latency', true);
const messagesCreated = new Counter('messages_created');

export const options = {
  scenarios: {
    message_load: {
      executor: 'constant-arrival-rate',
      rate: 100,           // 100 requests per second (reduced due to rate limiting)
      timeUnit: '1s',
      duration: '60s',
      preAllocatedVUs: 50,
      maxVUs: 200,
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<200', 'p(99)<500'],
    errors: ['rate<0.01'],  // <1% error rate
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Store authenticated users with servers/channels
const userSessions = [];

export function setup() {
  // Create test users with servers and channels
  const sessions = [];
  
  for (let i = 0; i < 10; i++) {
    const session = createTestSession(i);
    if (session) {
      sessions.push(session);
    }
  }
  
  if (sessions.length === 0) {
    console.error('Failed to create any test sessions');
  }
  
  return { sessions, startTime: Date.now() };
}

function createTestSession(index) {
  const uniqueId = `msgtest_${index}_${Date.now()}`;
  
  // Register user
  const regPayload = JSON.stringify({
    email: `${uniqueId}@test.local`,
    username: uniqueId,
    password: 'TestPassword123!',
  });
  
  const regRes = http.post(`${BASE_URL}/api/v1/auth/register`, regPayload, {
    headers: { 'Content-Type': 'application/json' },
  });
  
  if (regRes.status !== 201) {
    console.log(`Registration failed for ${uniqueId}: ${regRes.status}`);
    return null;
  }
  
  let token;
  try {
    token = JSON.parse(regRes.body).access_token;
  } catch {
    return null;
  }
  
  // Create a server
  const serverPayload = JSON.stringify({
    name: `Test Server ${uniqueId}`,
  });
  
  const serverRes = http.post(`${BASE_URL}/api/v1/servers`, serverPayload, {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
    },
  });
  
  if (serverRes.status !== 201) {
    console.log(`Server creation failed: ${serverRes.status} - ${serverRes.body}`);
    return null;
  }
  
  let serverId;
  try {
    serverId = JSON.parse(serverRes.body).id;
  } catch {
    return null;
  }
  
  // Create a text channel
  const channelPayload = JSON.stringify({
    name: 'general',
    type: 0, // TEXT channel
  });
  
  const channelRes = http.post(`${BASE_URL}/api/v1/servers/${serverId}/channels`, channelPayload, {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
    },
  });
  
  if (channelRes.status !== 201) {
    console.log(`Channel creation failed: ${channelRes.status} - ${channelRes.body}`);
    return null;
  }
  
  let channelId;
  try {
    channelId = JSON.parse(channelRes.body).id;
  } catch {
    return null;
  }
  
  return { token, serverId, channelId };
}

export default function (data) {
  if (!data.sessions || data.sessions.length === 0) {
    errorRate.add(true);
    return;
  }
  
  // Pick a random session
  const session = data.sessions[Math.floor(Math.random() * data.sessions.length)];
  
  const testType = Math.random();
  
  if (testType < 0.6) {
    // 60% send messages
    sendMessage(session);
  } else if (testType < 0.9) {
    // 30% get messages
    getMessages(session);
  } else {
    // 10% typing indicator
    sendTyping(session);
  }
}

function sendMessage(session) {
  const payload = JSON.stringify({
    content: `Load test message ${Date.now()} from VU ${__VU}`,
  });
  
  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${session.token}`,
    },
    tags: { endpoint: 'send_message' },
  };
  
  const start = Date.now();
  const res = http.post(`${BASE_URL}/api/v1/channels/${session.channelId}/messages`, payload, params);
  messageLatency.add(Date.now() - start);
  
  const success = check(res, {
    'message sent (201)': (r) => r.status === 201,
    'message has id': (r) => {
      try {
        return JSON.parse(r.body).id !== undefined;
      } catch {
        return false;
      }
    },
  });
  
  if (success) {
    messagesCreated.add(1);
  }
  errorRate.add(!success);
}

function getMessages(session) {
  const params = {
    headers: {
      'Authorization': `Bearer ${session.token}`,
    },
    tags: { endpoint: 'get_messages' },
  };
  
  const start = Date.now();
  const res = http.get(`${BASE_URL}/api/v1/channels/${session.channelId}/messages?limit=50`, params);
  messageLatency.add(Date.now() - start);
  
  const success = check(res, {
    'messages retrieved (200)': (r) => r.status === 200,
    'messages is array': (r) => {
      try {
        return Array.isArray(JSON.parse(r.body));
      } catch {
        return false;
      }
    },
  });
  
  errorRate.add(!success);
}

function sendTyping(session) {
  const params = {
    headers: {
      'Authorization': `Bearer ${session.token}`,
    },
    tags: { endpoint: 'typing' },
  };
  
  const start = Date.now();
  const res = http.post(`${BASE_URL}/api/v1/channels/${session.channelId}/typing`, null, params);
  messageLatency.add(Date.now() - start);
  
  const success = check(res, {
    'typing indicator sent (204 or 200)': (r) => r.status === 204 || r.status === 200,
  });
  
  errorRate.add(!success);
}

export function teardown(data) {
  console.log(`Test duration: ${(Date.now() - data.startTime) / 1000}s`);
  console.log(`Sessions used: ${data.sessions ? data.sessions.length : 0}`);
}
