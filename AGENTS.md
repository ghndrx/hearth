# AGENTS.md — Hearth backend + web frontend

## Stack
- Backend: Go 1.25 in `backend/` — entry `backend/cmd/...`, module name `hearth`
- Frontend: SvelteKit in `frontend/`
- Default branch: `develop` (not `main`)
- Pre-commit: lefthook (`lefthook.yml` at repo root)

## Commands
- Test backend: `cd backend && go test ./...`
- Test frontend: `cd frontend && pnpm test`
- Lint backend: `cd backend && go vet ./...`
- Format: `gofmt -w backend/` and `cd frontend && pnpm format`
- Build: `make build`

## Conventions
- Commit format: Conventional Commits (`feat:`, `fix:`, `chore:`). Scope optional.
- Branch from `develop`, name `feat/<feature-id>`.
- Go: use `context.Context` as first arg, wrap errors with `fmt.Errorf("... %w", err)`.
- Svelte: components in `frontend/src/lib/components/`, routes in `frontend/src/routes/`.
- Tests colocated: `foo.go` ↔ `foo_test.go`.

## Do not touch without explicit task
- `k8s/` — production manifests, deploy via separate pipeline
- `deploy/` — infrastructure bootstrap scripts
- `CHANGELOG.md` — release-please manages this

## Security
- No secrets in code. Use `os.Getenv` + `.env` (gitignored).
- All inbound HTTP goes through the auth middleware in `backend/internal/http/middleware/`.
- Database queries via `sqlc`-generated code; no raw `db.Exec` with string interpolation.

## Hearth-specific
- This is a self-hosted, federated Discord/Slack alternative. Federation via Matrix.
- Voice via LiveKit; don't reinvent WebRTC.
- E2EE via Megolm/Vodozemac for federated channels.
