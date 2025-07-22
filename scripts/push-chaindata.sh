#!/usr/bin/env bash
##
# Incrementally commit and push each chaindata subdirectory to avoid GitHub's per-push size limit (~2GB).
# Run this from the repo root on the 'main' branch with a clean working tree.
set -euo pipefail

# Ensure we are at the repository root
cd "$(git rev-parse --show-toplevel)"

# Abort if there are uncommitted changes
if ! git diff-index --quiet HEAD --; then
  echo "Error: working tree is not clean. Commit or stash changes first." >&2
  exit 1
fi

# Verify current branch
MAIN_BRANCH=main
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [[ "$CURRENT_BRANCH" != "$MAIN_BRANCH" ]]; then
  echo "Error: not on '$MAIN_BRANCH' branch (current: $CURRENT_BRANCH)." >&2
  exit 1
fi

echo "Starting incremental chaindata push on branch '$MAIN_BRANCH'..."

# Iterate chaindata subdirectories (excluding non-dir entries)
for dir in chaindata/*/; do
  name=$(basename "$dir")
  # Skip hidden or dot dirs if any
  [[ "$name" = .* ]] && continue

  echo "--> Committing and pushing chaindata/$name"
  git add "chaindata/$name"
  git commit -m "chore(genesis): add chaindata for $name"
  git push origin "$MAIN_BRANCH"
done

echo "All chaindata subdirectories have been pushed."
