# Contributing to Hearth

Thank you for your interest in contributing to Hearth! This document provides guidelines and best practices for contributors.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Git Workflow](#git-workflow)
- [Coding Standards](#coding-standards)
- [Testing](#testing)
- [Pull Requests](#pull-requests)
- [Security](#security)

---

## Code of Conduct

Be respectful, inclusive, and constructive. We're all here to build something great.

---

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/hearth.git`
3. Add upstream: `git remote add upstream https://github.com/ghndrx/hearth.git`
4. Create a feature branch from `develop`

---

## Development Setup

### Prerequisites

- Go 1.22+
- Node.js 22+
- Docker & Docker Compose
- PostgreSQL 16+ (or use Docker)
- Redis 7+ (or use Docker)

### Quick Start

```bash
# Start dependencies
docker compose -f docker-compose.dev.yml up -d

# Backend
cd backend
go mod download
go run ./cmd/hearth

# Frontend (new terminal)
cd frontend
npm install
npm run dev
```

### Environment Variables

Copy `.env.example` to `.env` and configure:

```bash
cp .env.example .env
# Edit .env with your values
```

---

## Git Workflow

We use a modified Git Flow:

```
master (production, protected)
  ↑ PR required, 2 approvals for security changes
develop (integration)
  ↑ PR required, 1 approval
feature/* (your work)
  ↑ branch from develop
hotfix/* (urgent fixes)
  ↑ branch from master, merge to both
```

### Branch Naming

- `feature/short-description` - New features
- `fix/issue-number-description` - Bug fixes
- `hotfix/critical-issue` - Urgent production fixes
- `docs/what-changed` - Documentation only
- `refactor/what-changed` - Code refactoring

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
type(scope): description

[optional body]

[optional footer]
```

Types:
- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation
- `style` - Formatting (no code change)
- `refactor` - Code restructuring
- `test` - Adding tests
- `chore` - Maintenance

Examples:
```
feat(messages): add reaction support
fix(auth): prevent token reuse after logout
docs(api): update WebSocket documentation
```

---

## Coding Standards

### Go (Backend)

- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use `gofmt` and `golangci-lint`
- Error handling: Always check errors, wrap with context
- Naming: CamelCase, unexported = lowercase first letter
- Comments: Exported functions must have doc comments

```go
// CreateServer creates a new server with the given name.
// It returns the created server or an error if the user
// has reached their server limit.
func (s *ServerService) CreateServer(ctx context.Context, ownerID uuid.UUID, name string) (*Server, error) {
    // Implementation
}
```

### TypeScript (Frontend)

- Use TypeScript strict mode
- Follow ESLint configuration
- Use Svelte best practices
- Prefer composition over inheritance
- Keep components small and focused

```typescript
// ✅ Good
interface Props {
  message: Message;
  onReact: (emoji: string) => void;
}

// ❌ Avoid
interface Props {
  data: any;
  callback: Function;
}
```

### SQL

- Use parameterized queries (NEVER string concatenation)
- Use migrations for schema changes
- Index foreign keys and frequently queried columns
- Use transactions for multi-step operations

```sql
-- ✅ Good
SELECT * FROM users WHERE id = $1

-- ❌ Never
SELECT * FROM users WHERE id = '" + userId + "'
```

---

## Testing

### Accessibility

Before submitting UI changes, run the contrast checker:

```bash
# Check color contrast compliance (WCAG AA)
node scripts/contrast-checker.js

# CI mode (fails on violations)
node scripts/contrast-checker.js --ci
```

See [docs/accessibility/A11Y-003-contrast-audit.md](docs/accessibility/A11Y-003-contrast-audit.md) for color guidelines.

**Key requirements:**
- Text must have 4.5:1 contrast ratio on backgrounds
- UI components (icons, borders) need 3:1 minimum
- Use semantic color variables from `theme.css`

### Backend

```bash
cd backend

# Run all tests
go test ./...

# With coverage
go test -cover ./...

# Race detection
go test -race ./...

# Specific package
go test ./internal/services/...
```

### Frontend

```bash
cd frontend

# Run tests
npm run test

# Watch mode
npm run test:watch

# Coverage
npm run test:coverage
```

### Integration Tests

```bash
# Start test environment
docker compose -f docker-compose.test.yml up -d

# Run integration tests
go test -tags=integration ./...
```

---

## Pull Requests

### Before Submitting

1. ✅ Tests pass locally
2. ✅ Linter passes with no warnings
3. ✅ Documentation updated if needed
4. ✅ Commits are clean and follow conventions
5. ✅ Branch is up to date with `develop`

### PR Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation

## Testing
How did you test this?

## Screenshots (if applicable)

## Checklist
- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] No security issues introduced
```

### Review Process

1. Automated checks must pass
2. At least 1 approval required
3. Security-sensitive changes need 2 approvals
4. Maintainer merges using squash merge

---

## Security

### Reporting Vulnerabilities

**DO NOT** open public issues for security vulnerabilities.

Email: security@hearth.chat (or maintainer directly)

### Security Checklist

When contributing, ensure:

- [ ] No secrets in code or commits
- [ ] Input validation on all user data
- [ ] Parameterized database queries
- [ ] Proper authorization checks
- [ ] No sensitive data in logs
- [ ] Dependencies are up to date

### Secrets

Never commit:
- API keys
- Passwords
- Private keys
- Tokens
- Connection strings with credentials

Use environment variables or secret management.

---

## Questions?

- Open a [Discussion](https://github.com/ghndrx/hearth/discussions)
- Join our Discord (coming soon)
- Read the [Documentation](./docs/)

Thank you for contributing! 🔥
