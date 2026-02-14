# Contributing to Hearth

Thank you for your interest in contributing to Hearth! This document provides guidelines for contributing.

---

## Code of Conduct

Be respectful. Be constructive. We're all here to build something great.

---

## How to Contribute

### Reporting Bugs
1. Check existing issues first
2. Use the bug report template
3. Include reproduction steps
4. Attach logs if relevant

### Suggesting Features
1. Check existing issues/discussions
2. Use the feature request template
3. Explain the use case, not just the solution

### Code Contributions

#### Setup
```bash
# Clone the repo
git clone https://github.com/ghndrx/hearth.git
cd hearth

# Install dependencies
make setup

# Start development environment
make dev
```

#### Development Workflow
1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-thing`)
3. Make your changes
4. Write/update tests
5. Run linters (`make lint`)
6. Commit with conventional commits
7. Push and open a PR

#### Commit Messages
Use [Conventional Commits](https://conventionalcommits.org/):
```
feat: add voice channel support
fix: resolve WebSocket reconnection issue
docs: update self-hosting guide
chore: update dependencies
```

#### Code Style
- **Go:** Follow `gofmt` and `golangci-lint`
- **TypeScript:** ESLint + Prettier
- **SQL:** Lowercase keywords, snake_case names

---

## Architecture Guidelines

### Backend (Go)
- Use dependency injection
- Interfaces for testability
- Errors should be wrapped with context
- Use structured logging

### Frontend (SvelteKit)
- Component-based architecture
- TypeScript for all new code
- CSS modules or Tailwind
- Accessible by default

### Database
- Migrations for all schema changes
- No raw SQL in service layer (use repository)
- Index frequently queried columns

---

## Testing

### Running Tests
```bash
# All tests
make test

# Backend only
make test-backend

# Frontend only
make test-frontend

# With coverage
make test-coverage
```

### Test Requirements
- New features need tests
- Bug fixes should include regression test
- Aim for 80%+ coverage on new code

---

## Documentation

- Update relevant docs with code changes
- API changes need OpenAPI spec updates
- User-facing features need user guide updates

---

## Review Process

1. All PRs require at least one approval
2. CI must pass (tests, lint, build)
3. Breaking changes need discussion first
4. Security-sensitive changes need security review

---

## Release Process

Maintainers handle releases. If you want something released:
1. Ensure it's merged to main
2. Open an issue requesting release
3. Include what features/fixes to highlight

---

## Getting Help

- **Discord:** (link when available)
- **Discussions:** GitHub Discussions
- **Issues:** For bugs and features

---

Thank you for contributing! 🔥
