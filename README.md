# 🔥 Hearth

**A self-hosted Discord alternative. Your community, your server, your rules.**

Hearth is an open-source, self-hosted real-time communication platform that gives you complete control over your community's data and infrastructure. Built for privacy-conscious users, homelab enthusiasts, and organizations that need to own their communication stack.

---

## ✨ Features

### Core Communication
- **Text Channels** — Real-time messaging with rich text, embeds, and file sharing
- **Voice Channels** — Crystal-clear voice chat with WebRTC
- **Video Calls** — Face-to-face communication, screen sharing
- **Direct Messages** — Private 1:1 and group conversations
- **Threads** — Focused discussions without cluttering main channels

### Server Management
- **Servers (Instances)** — Create isolated communities with custom branding
- **Categories** — Organize channels into logical groups
- **Roles & Permissions** — Granular access control with role hierarchy
- **Invites** — Time-limited or permanent invite links
- **Moderation Tools** — Ban, kick, mute, timeout, audit logs

### User Experience
- **User Profiles** — Customizable profiles with status and bio
- **Reactions** — Emoji reactions on messages
- **Mentions** — @user, @role, @everyone, @here
- **Search** — Full-text search across messages
- **Notifications** — Configurable push, email, and in-app alerts
- **Presence** — Online, idle, DND, invisible status

### Extensibility
- **Webhooks** — Inbound integrations for external services
- **Bot API** — Build custom bots with full API access
- **Plugins** — Extend functionality with community plugins
- **Themes** — Customizable UI themes

### Self-Hosting Benefits
- **Data Sovereignty** — All data stays on your infrastructure
- **No Telemetry** — Zero tracking, zero analytics sent externally
- **Federation Ready** — Future support for server-to-server communication
- **Single Binary** — Easy deployment with Docker or standalone binary
- **SQLite or Postgres** — Choose your database backend

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

### Docker Compose (Recommended)
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

### Systemd (Bare Metal)
```bash
curl -sSL https://raw.githubusercontent.com/ghndrx/hearth/main/deploy/systemd/install.sh | sudo bash
sudo systemctl start hearth
```

Visit `http://localhost:8080` and create your first server.

---

## 📚 Documentation

- [Product Requirements (PRD)](docs/PRD.md)
- [Architecture](docs/ARCHITECTURE.md)
- [Data Model](docs/DATA_MODEL.md)
- [API Reference](docs/API.md)
- [Features Deep Dive](docs/FEATURES.md)
- [Security Model](docs/SECURITY.md)
- [Roadmap](docs/ROADMAP.md)
- [Self-Hosting Guide](docs/SELF_HOSTING.md)
- [Contributing](docs/CONTRIBUTING.md)

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
