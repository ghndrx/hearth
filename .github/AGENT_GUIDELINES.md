# Agent Workflow Guidelines

## Core Principle
All commits, branches, and pull requests are made under **Greg Hendrickson** identity.

## Branch Naming
- Feature branches: `feat/<short-description>` (e.g., `feat/user-status`, `feat/websocket-auth`)
- Fix branches: `fix/<short-description>` (e.g., `fix/gofmt-errors`)
- Test branches: `test/<package>` (e.g., `test/handlers-coverage`)
- Chore branches: `chore/<description>` (e.g., `chore/ci-fix`)
- **Never** commit directly to `master` or `develop`

## Commit Messages
- Subject line: imperative mood, lowercase, no period (e.g., `feat: add user status with emoji`)
- Body: explain WHY, not WHAT
- Reference issues: `Fixes #123` or `Closes #123`
- **No AI attribution** in commit messages (no "AI-generated", "Claude-assisted", etc.)

## Author Identity (REQUIRED)
Every agent session MUST configure before any git operations:
```bash
git config user.name "Greg Hendrickson"
git config user.email "greg@gregh.dev"
```

For repos using SOPS (infrastructure):
```bash
git config user.name "Greg Hendrickson"
git config user.email "greg@hndrx.co"
```

## Workflow

### Feature Development
1. Create feature branch from `develop`:
   ```bash
   git checkout develop && git pull origin develop
   git checkout -b feat/my-feature
   ```
2. Make commits (always with Greg identity)
3. Push branch:
   ```bash
   git push -u origin feat/my-feature
   ```
4. Open PR targeting `develop` via GitHub UI or `gh pr create`

### PR Requirements
- Title: clear, concise description
- Description: summarize changes and motivation
- Link any related issues
- Request review from `@ghndrx`
- Do NOT use draft PRs for final submission

### Keeping PRs Updated
- If `develop` moves ahead, rebase your feature branch:
  ```bash
  git checkout feat/my-feature
  git fetch origin
  git rebase origin/develop
  # Resolve conflicts if any
  git push --force-with-lease
  ```
- **Never** merge `develop` into your feature branch (use rebase)

### After Merge
- GitHub Actions `branch-autocleanup` workflow auto-deletes merged branches
- If manual cleanup needed:
  ```bash
  git checkout develop && git pull
  git branch -d feat/my-feature
  ```

## Preventing Conflicts

### Before Starting Work
```bash
# Always start fresh from latest develop
git checkout develop && git pull origin develop
```

### During Development
- Commit frequently with meaningful messages
- Keep commits atomic (one logical change per commit)
- Rebase on `develop` if working across multiple days

### Before Opening PR
```bash
# Ensure clean rebase
git fetch origin
git rebase origin/develop
# Fix any conflicts
git push --force-with-lease
```

## Repository Sync Checklist
When working in a local clone:
- [ ] `git remote -v` shows correct origin
- [ ] On a feature branch (not master/develop)
- [ ] `git log --format='%an <%ae>'` shows only Greg Hendrickson
- [ ] `git branch` shows only expected branches
- [ ] `git status` is clean before major operations

## Workflows That Create Commits
Any GitHub Action that configures git user identity MUST use Greg's identity:
```yaml
- name: Set up git identity
  run: |
    git config --global user.email "greg@gregh.dev"
    git config --global user.name "Greg Hendrickson"
```

## Anti-Patterns to Avoid
- ❌ `git commit -m "fix: fix stuff"` — vague messages
- ❌ Committing directly to `develop` or `master`
- ❌ Using bot/agent identities in commit author
- ❌ "AI-generated" or "Claude Code" in commit messages
- ❌ Merge commits in feature branches (use rebase)
- ❌ Force pushing to `master` or `develop`
- ❌ Working across multiple feature branches simultaneously
