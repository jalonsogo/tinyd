#!/usr/bin/env bash
#
# Deploy the landing page.
#
#   1. Push source to origin/main (jalonsogo/tinyd-landing).
#   2. Mirror to jalonsogo/tinyd@gh-pages so jalonsogo.github.io/tinyd updates.
#
# Requires the local repo to have both remotes configured:
#   origin -> https://github.com/jalonsogo/tinyd-landing.git
#   tinyd  -> https://github.com/jalonsogo/tinyd.git

set -euo pipefail

cd "$(dirname "$0")"

if [[ -n "$(git status --porcelain)" ]]; then
  echo "✗ Working tree is dirty. Commit or stash before deploying." >&2
  git status --short >&2
  exit 1
fi

branch="$(git rev-parse --abbrev-ref HEAD)"
if [[ "$branch" != "main" ]]; then
  echo "✗ Refusing to deploy from branch '$branch' (expected 'main')." >&2
  exit 1
fi

echo "→ Pushing source to origin/main"
git push origin main

echo "→ Mirroring to jalonsogo/tinyd@gh-pages"
git push tinyd main:gh-pages

echo "✓ Done. Live at https://jalonsogo.github.io/tinyd  (and source at https://github.com/jalonsogo/tinyd-landing)"
