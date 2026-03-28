# Hearth

**Self-hosted chat. Full control.**

[![License](https://img.shields.io/github/license/ghndrx/hearth)](LICENSE)
[![CI](https://github.com/ghndrx/hearth/actions/workflows/ci.yml/badge.svg)](https://github.com/ghndrx/hearth/actions/workflows/ci.yml)

A Discord-compatible communication platform you own. Text channels, voice, DMs, threads—all on your infrastructure.

## Features

- **Data sovereignty** — Your servers, your data
- **Single binary** — Or Docker/Kubernetes
- **E2E encryption** — Optional for DMs
- **Familiar UX** — Discord-like interface
- **Accessible** — WCAG 2.1 compliant
- **Scalable** — Redis Pub/Sub, horizontal scaling

## Quick Start

```bash
git clone https://github.com/ghndrx/hearth.git
cd hearth
cp .env.example .env
docker compose up -d
```

Open `http://localhost:3000`

## Documentation

- [Deployment Guide](docs/DEPLOYMENT.md) — Docker, Kubernetes, bare metal
- [Self-Hosting Guide](docs/SELF_HOSTING.md) — Full setup walkthrough
- [API Reference](docs/api/README.md) — REST & WebSocket API

## Tech Stack

| Layer | Tech |
|-------|------|
| Frontend | SvelteKit, Tailwind, TypeScript |
| Backend | Go, Chi, WebSocket |
| Database | PostgreSQL |
| Cache | Redis |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE)
