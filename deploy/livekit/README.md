# LiveKit Voice Server Deployment

This directory contains configuration for Hearth's voice channel infrastructure using LiveKit.

## Quick Start

1. **Copy config to server:**
   ```bash
   scp -r deploy/livekit root@your-server:/root/hearth/
   ```

2. **Create the network (if not exists):**
   ```bash
   docker network create hearth-network
   ```

3. **Start LiveKit:**
   ```bash
   cd /root/hearth/livekit
   docker compose -f docker-compose.livekit.yml up -d
   ```

4. **Configure Caddy** (add to /etc/caddy/Caddyfile):
   ```
   hearth.gregh.dev {
       handle /livekit* {
           reverse_proxy localhost:7880
       }
       # ... existing routes
   }
   ```

5. **Set backend environment variables:**
   ```bash
   LIVEKIT_API_KEY=APIcKPP67nS9WWXH9mWx3XGc
   LIVEKIT_API_SECRET=8fDdP39mJYSBcW4w2hDFP8pUB0WYR0tvbVZ0tFSdHXTzDPap
   LIVEKIT_URL=wss://hearth.gregh.dev/livekit
   ```

## Architecture

```
                    ┌─────────────────┐
                    │     Caddy       │
                    │  (SSL + Proxy)  │
                    └────────┬────────┘
                             │
      ┌──────────────────────┼──────────────────────┐
      │                      │                      │
      ▼                      ▼                      ▼
┌───────────┐         ┌───────────┐          ┌───────────┐
│  Frontend │         │  Backend  │          │  LiveKit  │
│   :3000   │         │   :8080   │          │   :7880   │
└───────────┘         └─────┬─────┘          └─────┬─────┘
                            │                      │
                            └──────────────────────┘
                                    Token Gen
```

## Ports

| Port | Protocol | Purpose |
|------|----------|---------|
| 7880 | TCP | HTTP API / WebSocket signaling |
| 7881 | TCP | WebRTC over TCP |
| 7882 | UDP | WebRTC media |
| 50000-50100 | UDP | RTP media ports |

## Credentials

**API Key:** `APIcKPP67nS9WWXH9mWx3XGc`
**API Secret:** `8fDdP39mJYSBcW4w2hDFP8pUB0WYR0tvbVZ0tFSdHXTzDPap`

⚠️ **Important:** Rotate these credentials in production!

## Monitoring

Check LiveKit status:
```bash
docker logs hearth-livekit
curl http://localhost:7880/
```

List active rooms:
```bash
docker exec hearth-livekit livekit-cli list-rooms --url http://localhost:7880 --api-key APIcKPP67nS9WWXH9mWx3XGc --api-secret 8fDdP39mJYSBcW4w2hDFP8pUB0WYR0tvbVZ0tFSdHXTzDPap
```
