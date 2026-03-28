import { check, sleep } from 'k6';
import http from 'k6/http';
import ws from 'k6/ws';
import { Counter, Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('ws_errors');
const connectionLatency = new Trend('ws_connect_latency', true);
const messageLatency = new Trend('ws_message_latency', true);
const connectionsActive = new Counter('ws_connections_active');
const messagesReceived = new Counter('ws_messages_received');

export const options = {
  scenarios: {
    websocket_load: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 500 },   // Ramp up to 500
        { duration: '30s', target: 1000 },  // Ramp to 1000 concurrent
        { duration: '60s', target: 1000 },  // Hold at 1000
        { duration: '30s', target: 0 },     // Ramp down
      ],
    },
  },
  thresholds: {
    ws_connect_latency: ['p(95)<1000', 'p(99)<2000'],
    ws_errors: ['rate<0.01'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const WS_URL = __ENV.WS_URL || 'ws://localhost:8080';

export function setup() {
  // Test HTTP connectivity first
  const res = http.get(`${BASE_URL}/health`);
  check(res, {
    'health check ok': (r) => r.status === 200,
  });
  return { startTime: Date.now() };
}

export default function () {
  // Create a unique user for this VU
  const uniqueId = `wstest_${__VU}_${__ITER}_${Date.now()}`;
  
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
    errorRate.add(true);
    sleep(1);
    return;
  }
  
  let token;
  try {
    token = JSON.parse(regRes.body).access_token;
  } catch {
    errorRate.add(true);
    sleep(1);
    return;
  }
  
  // Connect to WebSocket gateway
  const connectStart = Date.now();
  
  const res = ws.connect(`${WS_URL}/gateway`, {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
  }, function (socket) {
    connectionLatency.add(Date.now() - connectStart);
    connectionsActive.add(1);
    
    let identified = false;
    let heartbeatInterval = null;
    
    socket.on('open', function () {
      // Send identify payload (Discord-style gateway)
      socket.send(JSON.stringify({
        op: 2, // IDENTIFY
        d: {
          token: token,
          properties: {
            os: 'k6',
            browser: 'k6',
            device: 'k6',
          },
        },
      }));
    });
    
    socket.on('message', function (msg) {
      messagesReceived.add(1);
      const msgStart = Date.now();
      
      try {
        const data = JSON.parse(msg);
        
        // Handle different opcodes
        switch (data.op) {
          case 10: // HELLO
            // Start heartbeating
            const interval = data.d?.heartbeat_interval || 45000;
            heartbeatInterval = socket.setInterval(function () {
              socket.send(JSON.stringify({ op: 1, d: null })); // HEARTBEAT
            }, interval);
            break;
            
          case 0: // DISPATCH
            if (data.t === 'READY') {
              identified = true;
            }
            break;
            
          case 11: // HEARTBEAT_ACK
            // Connection is alive
            break;
            
          case 9: // INVALID_SESSION
            errorRate.add(true);
            socket.close();
            break;
        }
        
        messageLatency.add(Date.now() - msgStart);
      } catch (e) {
        // Non-JSON message or parse error
      }
    });
    
    socket.on('error', function (e) {
      errorRate.add(true);
    });
    
    socket.on('close', function () {
      if (heartbeatInterval) {
        clearInterval(heartbeatInterval);
      }
    });
    
    // Keep connection open for 30-60 seconds
    const connectionDuration = 30 + Math.random() * 30;
    socket.setTimeout(function () {
      socket.close();
    }, connectionDuration * 1000);
  });
  
  const success = check(res, {
    'WebSocket connected': (r) => r && r.status === 101,
  });
  
  if (!success) {
    errorRate.add(true);
  }
  
  sleep(1);
}

export function teardown(data) {
  console.log(`Test duration: ${(Date.now() - data.startTime) / 1000}s`);
}
