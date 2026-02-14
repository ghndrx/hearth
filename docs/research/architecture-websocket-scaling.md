# WebSocket Scaling Architecture Research

**Research Date:** February 11, 2026  
**Focus Area:** WebSocket Scaling (Redis pub/sub, horizontal scaling)  
**Sources:** Slack Engineering, Dyte, WebSocket.org, ByteByteGo

---

## Executive Summary

Scaling WebSockets for a chat platform like Hearth requires fundamentally different patterns than HTTP-based services. WebSocket connections are **stateful**, **long-lived**, and require **bidirectional message delivery in real-time**. This research documents production patterns from Slack (5M+ concurrent connections), Dyte, and other large-scale systems.

---

## Core Challenges

### Why WebSockets Are Hard to Scale

1. **Per-connection memory overhead** - Each connection requires buffers for send/receive
2. **File descriptors** - OS limits (default 256-1024) per process
3. **Stateful nature** - Connections maintain session state, complicating failover
4. **Fan-out complexity** - One message may need delivery to thousands of clients
5. **Geographic distribution** - Latency requires multi-region deployment

### The 10K Problem

A single server can theoretically handle millions of TCP connections, but practical limits kick in:
- Default kernel limits
- Memory per connection (~10KB-50KB with buffers)
- CPU for keepalive processing
- Application-level state

---

## Architecture Patterns

### Pattern 1: Slack's Channel Server Architecture

Slack's production architecture separates concerns into specialized server types:

```
┌─────────────────────────────────────────────────────────────────┐
│                        Slack Architecture                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐       │
│  │ Gateway      │    │ Gateway      │    │ Gateway      │       │
│  │ Server (GS)  │    │ Server (GS)  │    │ Server (GS)  │       │
│  │ [Region A]   │    │ [Region B]   │    │ [Region C]   │       │
│  │              │    │              │    │              │       │
│  │ WebSocket    │    │ WebSocket    │    │ WebSocket    │       │
│  │ Connections  │    │ Connections  │    │ Connections  │       │
│  └──────┬───────┘    └──────┬───────┘    └──────┬───────┘       │
│         │                   │                   │                │
│         └───────────────────┼───────────────────┘                │
│                             │                                    │
│                    ┌────────▼────────┐                           │
│                    │ Consistent Hash │                           │
│                    │ Ring (CHARM)    │                           │
│                    └────────┬────────┘                           │
│                             │                                    │
│    ┌────────────────────────┼────────────────────────┐           │
│    │                        │                        │           │
│  ┌─▼──────────┐    ┌────────▼───────┐    ┌──────────▼─┐         │
│  │ Channel    │    │ Channel        │    │ Channel    │         │
│  │ Server     │    │ Server         │    │ Server     │         │
│  │ (16M chan) │    │ (16M chan)     │    │ (16M chan) │         │
│  └────────────┘    └────────────────┘    └────────────┘         │
│                                                                  │
│  Channel Servers: Stateful, in-memory, hold channel history      │
│  Gateway Servers: Hold WebSocket connections, user subscriptions │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Key Components:**

| Server Type | State | Purpose |
|-------------|-------|---------|
| **Gateway Server (GS)** | Stateful | WebSocket connections, user subscriptions |
| **Channel Server (CS)** | Stateful | Channel message history, consistent-hash mapped |
| **Admin Server (AS)** | Stateless | Interface between webapp and channel servers |
| **Presence Server (PS)** | Stateful | Online/offline status tracking |
| **CHARM** | Coordinator | Manages consistent hash ring for channel servers |

**Critical Insight:** Slack serves **16 million channels per host** at peak. Channel-to-server mapping uses consistent hashing - each channel ID hashes to a specific server.

### Pattern 2: Dyte's Three-Layer Architecture

```
┌─────────────────────────────────────────────────────────┐
│                 Dyte WebSocket Architecture              │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  Layer 1: EDGE                                           │
│  ┌─────────────────────────────────────────────────┐    │
│  │  Load Balancer                                   │    │
│  │     │                                            │    │
│  │  ┌──▼───┐  ┌───────┐  ┌───────┐  ┌───────┐     │    │
│  │  │Edge  │  │ Edge  │  │ Edge  │  │ Edge  │     │    │
│  │  │Server│  │Server │  │Server │  │Server │     │    │
│  │  └──────┘  └───────┘  └───────┘  └───────┘     │    │
│  │  • WebSocket termination only                   │    │
│  │  • Lightweight, scales rapidly                  │    │
│  │  • TLS termination handled separately           │    │
│  └─────────────────────────────────────────────────┘    │
│                          │                               │
│  Layer 2: MESSAGE BROKER │                               │
│  ┌───────────────────────▼─────────────────────────┐    │
│  │              RabbitMQ Cluster                    │    │
│  │  • Routes messages between edge and hub         │    │
│  │  • Topic-based routing per channel/room         │    │
│  │  • Maintains message order                      │    │
│  └─────────────────────────────────────────────────┘    │
│                          │                               │
│  Layer 3: HUB            │                               │
│  ┌───────────────────────▼─────────────────────────┐    │
│  │           Hub Servers                            │    │
│  │  • Process messages                              │    │
│  │  • Broadcast to required connections             │    │
│  │  • Business logic                                │    │
│  └─────────────────────────────────────────────────┘    │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

**Key Insight:** Dyte separates TLS termination from WebSocket handling. TLS is CPU-intensive; keeping it separate allows independent scaling.

---

## Redis Pub/Sub for Cross-Server Broadcasting

When scaling horizontally, messages received by one server must reach clients on other servers.

### Basic Redis Pub/Sub Pattern

```javascript
// Server-side pub/sub implementation
const redis = require('redis');
const WebSocket = require('ws');

class PubSubWebSocketServer {
  constructor(port) {
    this.wss = new WebSocket.Server({ port });
    this.publisher = redis.createClient();
    this.subscriber = redis.createClient();
    this.channels = new Map(); // channel -> Set of clients
    
    this.setupSubscriber();
  }

  setupSubscriber() {
    this.subscriber.on('message', (channel, message) => {
      const clients = this.channels.get(channel) || new Set();
      clients.forEach(ws => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(message);
        }
      });
    });
  }

  subscribe(ws, channelId) {
    if (!this.channels.has(channelId)) {
      this.channels.set(channelId, new Set());
      this.subscriber.subscribe(channelId);
    }
    this.channels.get(channelId).add(ws);
  }

  publish(channelId, message) {
    this.publisher.publish(channelId, JSON.stringify(message));
  }
}
```

### Redis Pub/Sub Limitations

| Limitation | Impact | Mitigation |
|------------|--------|------------|
| No persistence | Messages lost if no subscribers | Use Redis Streams for durability |
| No acknowledgment | Fire-and-forget | Implement app-level ACKs |
| Single point of failure | Downtime = no messages | Redis Cluster or Sentinel |
| Memory bound | Large channels consume RAM | Sharding by channel prefix |

### Redis Streams Alternative

For durability and replay capability:

```javascript
// Using Redis Streams for durable messaging
async function publishMessage(channelId, message) {
  await redis.xadd(
    `stream:${channelId}`,
    'MAXLEN', '~', 1000, // Keep ~1000 messages
    '*', // Auto-generate ID
    'data', JSON.stringify(message)
  );
}

async function consumeMessages(channelId, lastId = '$') {
  const results = await redis.xread(
    'BLOCK', 0,
    'STREAMS', `stream:${channelId}`, lastId
  );
  return results;
}
```

---

## Load Balancing Strategies

### Algorithm Comparison

| Algorithm | Use Case | Pros | Cons |
|-----------|----------|------|------|
| **Round Robin** | Uniform load | Simple | Ignores connection weight |
| **Least Connections** | Variable message rates | Better distribution | More state tracking |
| **Consistent Hashing** | Channel affinity | Predictable routing | Rebalancing on node changes |
| **Least Bandwidth** | Heavy media streams | Accurate load measure | Complex monitoring |

### Sticky Sessions vs. Shared State

**Sticky Sessions:**
- Connection always goes to same server
- Simple but fragile (server failure = reconnect)
- Hard to rebalance

**Shared State (Recommended):**
- Any server can handle any connection
- Redis/message broker maintains state
- Stream continuity on failover

---

## OS-Level Tuning

### File Descriptor Limits

```bash
# Check current limits
ulimit -n

# Temporary increase
ulimit -n 1000000

# Permanent (add to /etc/security/limits.conf)
* soft nofile 1000000
* hard nofile 1000000
```

### Programmatic Adjustment (Go)

```go
if runtime.GOOS == "linux" {
    var rLimit syscall.Rlimit
    syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
    rLimit.Cur = rLimit.Max
    syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
}
```

### Kernel Parameters

```bash
# /etc/sysctl.conf
net.core.somaxconn = 65535
net.ipv4.tcp_max_syn_backlog = 65535
net.core.netdev_max_backlog = 65535
net.ipv4.ip_local_port_range = 1024 65535
```

---

## Reconnection & Resilience

### Exponential Backoff

Critical for mass reconnection scenarios (deployment, outage):

```javascript
class ReconnectingWebSocket {
  constructor(url) {
    this.url = url;
    this.baseDelay = 1000;
    this.maxDelay = 30000;
    this.attempts = 0;
    this.messageQueue = [];
  }

  connect() {
    this.ws = new WebSocket(this.url);
    
    this.ws.onopen = () => {
      this.attempts = 0;
      this.flushQueue();
    };
    
    this.ws.onclose = () => {
      this.scheduleReconnect();
    };
  }

  scheduleReconnect() {
    const delay = Math.min(
      this.baseDelay * Math.pow(2, this.attempts) + Math.random() * 1000,
      this.maxDelay
    );
    this.attempts++;
    setTimeout(() => this.connect(), delay);
  }

  send(message) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(message);
    } else {
      this.messageQueue.push(message);
    }
  }

  flushQueue() {
    while (this.messageQueue.length > 0) {
      this.ws.send(this.messageQueue.shift());
    }
  }
}
```

### Ghost Connection Cleanup

Detect and clean up connections that appear open but are dead:

```javascript
const HEARTBEAT_INTERVAL = 30000;
const HEARTBEAT_TIMEOUT = 10000;

wss.on('connection', (ws) => {
  ws.isAlive = true;
  
  ws.on('pong', () => {
    ws.isAlive = true;
  });
});

setInterval(() => {
  wss.clients.forEach((ws) => {
    if (!ws.isAlive) {
      return ws.terminate();
    }
    ws.isAlive = false;
    ws.ping();
  });
}, HEARTBEAT_INTERVAL);
```

---

## Hearth Implementation Recommendations

### Phase 1: Small Self-Hosted (< 1000 connections)

```
┌─────────────────────────────────────────┐
│           Single Server Setup            │
├─────────────────────────────────────────┤
│                                          │
│  ┌────────────────────────────────┐     │
│  │         Hearth Server           │     │
│  │                                 │     │
│  │  ┌───────────┐  ┌───────────┐  │     │
│  │  │ WebSocket │  │   HTTP    │  │     │
│  │  │  Handler  │  │   API     │  │     │
│  │  └─────┬─────┘  └─────┬─────┘  │     │
│  │        │              │        │     │
│  │  ┌─────▼──────────────▼─────┐  │     │
│  │  │    In-Memory Pub/Sub     │  │     │
│  │  │    (Event Emitter)       │  │     │
│  │  └──────────────────────────┘  │     │
│  │                                 │     │
│  └────────────────────────────────┘     │
│                                          │
│  • No external dependencies              │
│  • Simple deployment                     │
│  • Good for home/small team use          │
│                                          │
└─────────────────────────────────────────┘
```

### Phase 2: Medium Scale (1K-100K connections)

```
┌─────────────────────────────────────────────────────────┐
│              Multi-Server with Redis                     │
├─────────────────────────────────────────────────────────┤
│                                                          │
│              ┌──────────────────┐                        │
│              │   Load Balancer  │                        │
│              │   (HAProxy/Nginx)│                        │
│              └────────┬─────────┘                        │
│                       │                                  │
│        ┌──────────────┼──────────────┐                  │
│        │              │              │                  │
│   ┌────▼────┐   ┌────▼────┐   ┌────▼────┐             │
│   │ Hearth  │   │ Hearth  │   │ Hearth  │             │
│   │ Node 1  │   │ Node 2  │   │ Node 3  │             │
│   └────┬────┘   └────┬────┘   └────┬────┘             │
│        │              │              │                  │
│        └──────────────┼──────────────┘                  │
│                       │                                  │
│              ┌────────▼────────┐                        │
│              │  Redis Cluster  │                        │
│              │  (Pub/Sub +     │                        │
│              │   Streams)      │                        │
│              └─────────────────┘                        │
│                                                          │
│  • Horizontal scaling                                    │
│  • Redis handles cross-server messaging                  │
│  • Optional Redis Sentinel for HA                        │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### Phase 3: Large Scale (100K+ connections)

```
┌───────────────────────────────────────────────────────────────┐
│           Multi-Region Slack-Style Architecture                │
├───────────────────────────────────────────────────────────────┤
│                                                                │
│  Region A                          Region B                    │
│  ┌─────────────────────┐          ┌─────────────────────┐     │
│  │  Gateway Servers    │          │  Gateway Servers    │     │
│  │  ┌─────┐ ┌─────┐   │          │  ┌─────┐ ┌─────┐   │     │
│  │  │ GS  │ │ GS  │   │          │  │ GS  │ │ GS  │   │     │
│  │  └──┬──┘ └──┬──┘   │          │  └──┬──┘ └──┬──┘   │     │
│  └─────┼───────┼──────┘          └─────┼───────┼──────┘     │
│        │       │                       │       │             │
│        └───────┼───────────────────────┼───────┘             │
│                │                       │                      │
│        ┌───────▼───────────────────────▼───────┐             │
│        │         Channel Servers                │             │
│        │  (Consistent Hash Ring)                │             │
│        │  ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐     │             │
│        │  │ CS  │ │ CS  │ │ CS  │ │ CS  │     │             │
│        │  └─────┘ └─────┘ └─────┘ └─────┘     │             │
│        └────────────────────────────────────────┘             │
│                                                                │
│  • Gateway servers per region (low latency)                    │
│  • Channel servers centralized (consistent hashing)            │
│  • CHARM manages hash ring                                     │
│  • 500ms global message delivery                               │
│                                                                │
└───────────────────────────────────────────────────────────────┘
```

---

## Key Metrics to Monitor

| Metric | Target | Alert Threshold |
|--------|--------|-----------------|
| Active connections | Varies | >80% of max |
| Message latency (p99) | <100ms | >500ms |
| Messages/second | Varies | Sustained >90% capacity |
| Connection churn rate | Low | >10% of connections/min |
| Memory per connection | <50KB | >100KB |
| File descriptor usage | <80% | >90% |

---

## References

1. [Slack Engineering: Real-time Messaging](https://slack.engineering/real-time-messaging/)
2. [ByteByteGo: How Slack Supports Billions of Messages](https://blog.bytebytego.com/p/how-slack-supports-billions-of-daily)
3. [Dyte: Scaling WebSockets to Millions](https://dyte.io/blog/scaling-websockets-to-millions/)
4. [WebSocket.org: WebSockets at Scale](https://websocket.org/guides/websockets-at-scale/)
5. [Ably: Challenges of Scaling WebSockets](https://ably.com/topic/the-challenge-of-scaling-websockets)

---

## Next Research Topics

- [ ] Database sharding for message storage
- [ ] Caching strategies for user/channel data
- [ ] Event sourcing for chat history
