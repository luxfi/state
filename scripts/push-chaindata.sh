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
  # Skip hidden/dot dirs and the configs folder
  if [[ "$name" = .* ]] || [[ "$name" = "configs" ]]; then
    continue
  fi

  echo "Processing chaindata/$name..."
  pdir="$dir/db/pebbledb"
  # If this chain uses PebbleDB, commit metadata and split SSTs into batches
  if [[ -d "$pdir" ]]; then
    # Commit top-level metadata.json if present
    if [[ -f "$dir/metadata.json" ]]; then
      echo "--> Adding metadata.json for $name"
      git add "$dir/metadata.json"
      git commit -m "chore(genesis): add metadata.json for $name"
      git push origin "$MAIN_BRANCH"
    fi

    # Commit PebbleDB logs, manifests, options
    extras=("$pdir"/*.log "$pdir"/MANIFEST-* "$pdir"/OPTIONS-*)
    if compgen -G "$pdir"/*.log > /dev/null || compgen -G "$pdir"/MANIFEST-* > /dev/null || compgen -G "$pdir"/OPTIONS-* > /dev/null; then
      echo "--> Adding PebbleDB manifests/logs for $name"
      git add "${extras[@]}"
      git commit -m "chore(genesis): add $name pebbledb manifests and logs"
      git push origin "$MAIN_BRANCH"
    fi

    # Split SST files into batches to keep each push <2GB
    mapfile -t sst_files < <(ls "$pdir"/*.sst | sort)
    total=${#sst_files[@]}
    if (( total > 0 )); then
      batch=100
      for ((i=0; i<total; i+=batch)); do
        j=$((i+batch))
        (( j>total )) && j=$total
        echo "--> Adding SST files $((i+1))-$j of $total for $name"
        git add "${sst_files[@]:i:batch}"
        git commit -m "chore(genesis): add $name SST files $((i+1))-$j of $total"
        git push origin "$MAIN_BRANCH"
      done
    fi

    continue
  fi

  # Fallback: commit entire directory for small chains
  echo "--> Adding & pushing entire chaindata/$name"
  git add "$dir"
  git commit -m "chore(genesis): add chaindata for $name"
  git push origin "$MAIN_BRANCH"
done

echo "All chaindata subdirectories have been pushed."
