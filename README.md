# Hearth

**Self-hosted chat. Full control.**

[![License](https://img.shields.io/github/license/ghndrx/hearth)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/ghndrx/hearth)](https://goreportcard.com/report/github.com/ghndrx/hearth)

A Discord-compatible communication platform you own. Text channels, voice, DMs, threads—all on your infrastructure.

- **Data sovereignty** — Your servers, your data
- **Single binary** — Or Docker/Kubernetes
- **E2E encryption** — Optional for DMs
- **Familiar UX** — Discord-like, zero learning curve

## Quick Start

```bash
git clone https://github.com/ghndrx/hearth.git
cd hearth && cp .env.example .env
docker compose up -d
```

Open `http://localhost:3000`

## Documentation

Full docs at [docs/](docs/):

- [Deployment Guide](docs/DEPLOYMENT.md)
- [API Reference](docs/API.md)
- [Configuration](docs/DEPLOYMENT.md#configuration)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). PRs welcome.

## License

[MIT](LICENSE)
