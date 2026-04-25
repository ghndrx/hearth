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

## Try voice locally (Phase 0 smoke test)

```bash
docker compose -f docker-compose.dev.yml up -d   # postgres + redis + minio + livekit
cd backend && go run ./cmd/hearth &
cd frontend && pnpm install && pnpm dev
```

Open two browser tabs on `http://localhost:3000`, log in as two different
users, join the same voice channel. If you can hear yourself round-trip,
the voice stack is live end-to-end. If the `/voice/token` endpoint 500s
or the LiveKitManager WebSocket connect hangs, check that
`LIVEKIT_API_SECRET` matches between `.env` and `docker-compose.dev.yml`
(32-char minimum).

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
