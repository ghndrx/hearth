#!/bin/bash
# Hearth Worktree Pool Manager
# Manages a pool of git worktrees for parallel agent development

set -e

POOL_DIR="${HEARTH_POOL_DIR:-$HOME/hearth-pool}"
REPO_DIR="$HOME/clawd/hearth"
NUM_WORKTREES="${NUM_WORKTREES:-5}"

usage() {
    echo "Usage: $0 <command>"
    echo "Commands: create, list, sync, cleanup, destroy"
}

create_pool() {
    echo "Creating $NUM_WORKTREES worktrees..."
    mkdir -p "$POOL_DIR"
    for i in $(seq 1 $NUM_WORKTREES); do
        git worktree add "$POOL_DIR/worker-$i" -b "agent/worker-$i"
        cp CLAUDE.md "$POOL_DIR/worker-$i/"
    done
    echo "Pool ready!"
}

case "${1:-}" in
    create) create_pool ;;
    list) git worktree list ;;
    sync) 
        for wt in $(git worktree list --porcelain | awk '{print $1}' | grep -v "$POOL_DIR"); do
            cd "$wt" && git fetch origin develop && git rebase origin/develop
        done
        ;;
    *) usage ;;
esac
