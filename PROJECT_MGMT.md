# Hearth Project Management — Droid Factory Integration

This document defines how Droid Factory AI agents interact with the Hearth project for structured task management. It follows Notion database conventions for interoperability.

## Notion Kanban Sync Schema

When connecting this repo to a Notion database, use these property mappings:

| Notion Property | Type | GitHub Mapping | Droid Context |
|---|---|---|---|
| **Task Name** | Title | Issue/PR title | Feature or bug description |
| **Status** | Select | Labels + state | `Backlog`, `In Progress`, `In Review`, `Done` |
| **Priority** | Select | Priority label | `P0-Critical`, `P1-High`, `P2-Medium`, `P3-Low` |
| **Area** | Multi-select | Component label | `Federation`, `Auth`, `Messaging`, `Voice`, `UI`, `Infra` |
| **Assignee** | People | GitHub assignee | Human owner or `Droid` for AI tasks |
| **Sprint** | Relation | Milestone | Current sprint milestone |
| **Effort** | Select | Story points | `XS`, `S`, `M`, `L`, `XL` |
| **Droid Ready** | Checkbox | — | Checked when spec is approved and unblocked |
| **Droid Status** | Select | — | `Pending`, `Planning`, `Executing`, `Blocked`, `Complete` |
| **Spec URL** | URL | — | Link to Factory spec or AGENTS.md section |
| **Branch** | Rich text | — | `feat/<task-id>` per AGENTS.md convention |

## Status Workflow

```
Backlog → Ready for Droid → In Progress → In Review → Done
            (Droid Ready     (Droid Status   (PR opened    (merged/
             = true)          = Executing)     + CI green)    closed)
```

## Droid Interaction Patterns

### Starting a Droid Task

1. Create a GitHub Issue with:
   - Label: `droid-task`
   - Label: area tag (e.g., `area:federation`)
   - Priority label (e.g., `P1-High`)
   - Clear description with acceptance criteria
   - Check the **Droid Ready** checkbox when ready to hand off

2. Comment `@droid plan` on the issue to trigger Droid's planning phase
3. Droid will respond with a structured spec for approval
4. After approval, Droid updates **Droid Status** → `Executing`

### Droid Commands (use in issues/PRs)

| Command | Trigger | What Droid does |
|---|---|---|
| `@droid plan` | Issue comment | Generates implementation plan, asks clarifying questions |
| `@droid start` | Issue comment | Begins execution on an approved plan |
| `@droid review` | PR comment | Reviews the PR with AGENTS.md conventions |
| `@droid test` | PR comment | Runs tests and reports results |
| `@droid fix` | PR comment | Addresses review comments |
| `@droid status` | Any comment | Reports current task status |

### PR Lifecycle

1. Human or Droid opens PR from `feat/<task-id>` branch
2. **Droid Auto Review** workflow triggers → Droid comments review
3. Human addresses feedback or `@droid fix` for auto-fixes
4. CI must pass (`go test ./...`, `go vet`, `pnpm test`)
5. Lefthook pre-commit checks run locally
6. Squash-merge to `develop`, delete branch
7. Release-please handles CHANGELOG

## Current Sprint: Federation Phase 2

| Task | Status | Area | Effort | Assignee |
|---|---|---|---|---|
| Server signing keys | Done | Federation | M | Droid |
| .well-known discovery | Done | Federation | S | Droid |
| Room aliases / directory | Done | Federation | M | Droid |
| Outbound federation client | Done | Federation | L | Droid |
| Federation route wiring | Done | Federation | S | Droid |
| S2S event sync (Phase 3) | Backlog | Federation | XL | TBD |
| Matrix room state management | Backlog | Federation | XL | TBD |
| Megolm E2EE federation | Backlog | Federation | XL | TBD |
| Frontend Matrix client integration | Backlog | UI | L | TBD |

## Definition of Done

- [ ] Code follows AGENTS.md conventions (Go: context-first, `%w` errors; Svelte: components in `lib/components`)
- [ ] Tests colocated (`foo_test.go` for `foo.go`)
- [ ] `go vet ./...` clean
- [ ] `go test ./...` passing
- [ ] No secrets in code (use `os.Getenv`)
- [ ] Database queries via sqlc (no raw `db.Exec` with string interpolation)
- [ ] Auth middleware applies to new endpoints
- [ ] PR reviewed (human or Droid)
- [ ] CHANGELOG not manually edited (release-please manages)

## Factory Missions Integration

For multi-feature projects, use `/enter-mission` in Droid to:
1. Collaboratively plan features and milestones
2. Decompose into the sprint tasks above
3. Execute via Mission Control with validation checkpoints
4. Sync completed milestones back to Notion/GitHub

See: https://docs.factory.ai/cli/features/missions
