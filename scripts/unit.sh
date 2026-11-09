#!/usr/bin/env bash
# Usage: scripts/unit.sh U2
# Creates or reopens the unit's worktree from origin/qa and starts Claude Code on it.
set -euo pipefail
unit="${1:?Usage: scripts/unit.sh <U1-U7|GATE|INT|QA|RELEASE>}"
root="$(git rev-parse --show-toplevel)"
branch="$(python3 -c "import json,sys;print(json.load(open(sys.argv[1]))[sys.argv[2]]['branch'])" "$root/.claude/boundaries.json" "$unit")"
dir="$(dirname "$root")/sr-${branch#*/}"
if [ ! -d "$dir" ]; then
  git -C "$root" fetch origin
  if git -C "$root" show-ref --verify --quiet "refs/heads/$branch"; then
    git -C "$root" worktree add "$dir" "$branch"
  else
    git -C "$root" worktree add "$dir" -b "$branch" origin/qa
  fi
  if [ -f "$root/.env" ]; then cp "$root/.env" "$dir/.env"; fi
fi
cd "$dir"
exec claude "/unit $unit"
