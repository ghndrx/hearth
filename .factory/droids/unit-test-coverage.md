# Unit Test Coverage Droid

Analyzes code changes and generates/updates unit tests to maintain coverage for both Go backend and SvelteKit frontend.

## Trigger
Run on pull requests when source files change (excluding tests, configs, docs).

## Capabilities
- Analyze changed files to identify functions/components needing tests
- Generate idiomatic unit tests (Go table-driven tests, Vitest for TypeScript)
- Check coverage thresholds and report gaps
- Update existing tests when signatures change

## Instructions

### Context Gathering
1. Get the list of changed files from the PR
2. Filter to source files: `backend/**/*.go` (exclude `*_test.go`), `frontend/src/**/*.{ts,svelte}` (exclude `*.test.ts`)
3. Read each changed file to understand the code

### Go Backend Testing
- Location: `backend/` directory
- Test files: colocated as `foo_test.go` next to `foo.go`
- Style: table-driven tests with `t.Run()` subtests
- Use `context.Context` as first arg where applicable
- Mock external dependencies (Redis, DB) using interfaces
- Run: `cd backend && go test ./... -cover`

### SvelteKit Frontend Testing
- Location: `frontend/src/`
- Test files: colocated as `foo.test.ts` next to `foo.ts`
- Framework: Vitest
- Use `@testing-library/svelte` for component tests
- Mock stores and API calls
- Run: `cd frontend && bun run test`

### Coverage Requirements
- Minimum 70% line coverage for new code
- Report uncovered lines in PR comment

### Output
1. Create/update test files for changed source files
2. Run tests and report results
3. Comment on PR with coverage summary

## Model
inherit

## Tools
- Read, Edit, Create, Grep, Glob, Execute
