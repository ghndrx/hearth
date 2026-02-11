# 🔥 Hearth

**A self-hosted Discord alternative. Your community, your server, your rules.**

Hearth is an open-source, self-hosted real-time communication platform that gives you complete control over your community's data and infrastructure. Built for privacy-conscious users, homelab enthusiasts, and organizations that need to own their communication stack.

---

## ✨ Features

### 🔐 Privacy First
- **DMs always encrypted** — Server never sees your private messages
- **Channels: your choice** — Enable E2EE per-channel, or keep history searchable
- **Voice/video DMs encrypted** — SRTP with E2EE key exchange
- **Zero knowledge DMs** — We can't read your private conversations. Period.

### Core Communication
- **Text Channels** — Real-time messaging with Markdown, embeds, and file sharing
- **Voice Channels** — Crystal-clear voice chat with WebRTC
- **Video Calls** — Face-to-face communication, screen sharing
- **Direct Messages** — Private 1:1 and group conversations (up to 10)
- **Threads** — Focused discussions without cluttering main channels
- **Forum Channels** — Threaded discussions with tags

### Rich Messaging
- **Markdown** — Bold, italic, code blocks, spoilers, quotes, lists
- **File Sharing** — Images, videos, documents up to 100MB
- **Link Embeds** — Auto-preview for URLs, YouTube, Twitter
- **Emoji & Reactions** — Unicode + custom server emoji
- **GIF Picker** — Tenor/Giphy integration
- **Typing Indicators** — See who's typing in real-time
- **Message Search** — Full-text with filters (from:, in:, has:, before:)

### Server Management
- **Servers** — Create isolated communities with custom branding
- **Categories** — Organize channels into logical groups
- **Roles & Permissions** — 30+ granular permissions with hierarchy
- **Channel Overrides** — Per-channel permission tweaks
- **Invites** — Time-limited, usage-limited, or permanent
- **Server Folders** — Organize your server list

### Voice & Video
- **Voice Channels** — Low-latency WebRTC audio
- **Video Chat** — Camera support with grid layout
- **Screen Sharing** — Full screen or window
- **Push-to-Talk** — Or voice activity detection
- **Noise Suppression** — AI-based background noise removal
- **Voice Moderation** — Server mute, deafen, move, disconnect

### Moderation
- **Kick/Ban/Timeout** — Full member management
- **Audit Log** — Track all moderation actions
- **Auto-Moderation** — Spam, link, and word filters
- **Verification Levels** — Email, account age, phone
- **Bulk Message Delete** — Purge up to 100 messages

### User Experience
- **User Profiles** — Avatar, banner, bio, status
- **Presence** — Online, idle, DND, invisible + custom status
- **Friends System** — Friend requests, mutual servers
- **User Notes** — Private notes on any user
- **Notifications** — Per-channel/server, desktop, mobile push
- **Dark/Light Theme** — With accessibility options

### Extensibility
- **Webhooks** — Inbound integrations for external services
- **Bot API** — Build custom bots with full API access
- **Slash Commands** — Registered bot commands with autocomplete
- **REST + WebSocket API** — Full programmatic access

### Privacy & Security
- **E2EE by Default** — All DMs, group chats, and voice encrypted end-to-end
- **Zero Knowledge** — Server never sees plaintext content
- **Data Sovereignty** — All data stays on your infrastructure
- **No Telemetry** — Zero tracking, zero analytics

### Self-Hosting
- **SQLite or Postgres** — Choose your database backend
- **S3-Compatible Storage** — AWS, MinIO, B2, R2, Wasabi
- **Easy Deployment** — Docker, Helm, systemd, or binary

---

## 🏗️ Tech Stack

| Layer | Technology |
|-------|------------|
| **Backend** | Go (Fiber/Echo) |
| **Frontend** | SvelteKit + TypeScript |
| **Database** | SQLite (dev) / PostgreSQL (prod) |
| **Real-time** | WebSocket (gorilla/websocket) |
| **Voice/Video** | WebRTC + Pion |
| **Storage** | Local FS / S3-compatible |
| **Auth** | JWT + OAuth2 (optional OIDC) |
| **Cache** | Redis (optional) |
| **Search** | Bleve (embedded) / Meilisearch |

---

## 🚀 Quick Start

### One-Line Install
```bash
curl -sSL https://get.hearth.chat | bash
```

### With Custom Domain (Caddy + Auto-SSL)
```bash
curl -sSL https://get.hearth.chat | bash -s -- --domain hearth.gregh.dev
```

### With Custom Domain (Nginx + Let's Encrypt)
```bash
curl -sSL https://get.hearth.chat/nginx | bash -s -- --domain hearth.gregh.dev --email you@example.com
```

### With Enterprise SSO (FusionAuth)
```bash
curl -sSL https://get.hearth.chat | bash -s -- --with-fusionauth
```

### Manual Docker Compose
```bash
mkdir hearth && cd hearth
curl -O https://raw.githubusercontent.com/ghndrx/hearth/main/deploy/docker-compose/docker-compose.yml
curl -O https://raw.githubusercontent.com/ghndrx/hearth/main/deploy/docker-compose/.env.example
cp .env.example .env
echo "SECRET_KEY=$(openssl rand -base64 32)" >> .env
docker-compose up -d
```

### Helm (Kubernetes)
```bash
helm repo add hearth https://ghndrx.github.io/hearth
helm install hearth hearth/hearth --set ingress.enabled=true
```

Visit `http://localhost:8080` and create your first server.

---

## 📚 Documentation

| Document | Description |
|----------|-------------|
| [Features](docs/FEATURES.md) | Complete feature specification (200+ features) |
| [PRD](docs/PRD.md) | Product requirements and user stories |
| [Architecture](docs/ARCHITECTURE.md) | System design, WebSocket, WebRTC |
| [Data Model](docs/DATA_MODEL.md) | Database schema and relationships |
| [Deployment](docs/DEPLOYMENT.md) | Docker, Helm, systemd installation |
| [Self-Hosting](docs/SELF_HOSTING.md) | Configuration and maintenance |
| [Security](docs/SECURITY.md) | Auth, encryption, attack mitigation |
| [E2EE](docs/E2EE.md) | End-to-end encryption design |
| [Roadmap](docs/ROADMAP.md) | Development phases and timeline |
| [Contributing](docs/CONTRIBUTING.md) | How to contribute |

---

## 🗺️ Roadmap

| Phase | Status | Features |
|-------|--------|----------|
| **MVP** | 🔨 In Progress | Auth, servers, text channels, basic roles |
| **v0.2** | 📋 Planned | Voice channels, DMs, file uploads |
| **v0.3** | 📋 Planned | Video, screen share, threads |
| **v1.0** | 📋 Planned | Bots, webhooks, full moderation |
| **v2.0** | 💭 Future | Federation, mobile apps, E2EE |

---

## 🤝 Contributing

Contributions welcome! See [CONTRIBUTING.md](docs/CONTRIBUTING.md) for guidelines.

---

## 📄 License

MIT License. See [LICENSE](LICENSE) for details.

---

**Built with 🔥 for the self-hosted community.**
