#!/bin/bash
# Check CI status for the genesis migration workflow

REPO="luxfi/genesis"
WORKFLOW_NAME="Build Genesis Migration"
BRANCH="main"

echo "🔍 Checking CI status for $REPO..."
echo

# Check if gh CLI is installed
if ! command -v gh &> /dev/null; then
    echo "❌ GitHub CLI (gh) is not installed"
    echo "Install with: sudo apt install gh"
    echo ""
    echo "Alternative: Check manually at:"
    echo "https://github.com/$REPO/actions/workflows/build-genesis-migration.yml"
    exit 1
fi

# Get the latest workflow run
echo "Fetching latest workflow runs..."
RUNS=$(gh run list --repo "$REPO" --workflow "$WORKFLOW_NAME" --branch "$BRANCH" --limit 5)

if [ -z "$RUNS" ]; then
    echo "No workflow runs found."
    echo "Check: https://github.com/$REPO/actions"
    exit 1
fi

echo "$RUNS"
echo

# Get the latest run ID
RUN_ID=$(echo "$RUNS" | head -n 1 | awk '{print $NF}')

if [ -n "$RUN_ID" ]; then
    echo "📊 Latest run details (ID: $RUN_ID):"
    gh run view "$RUN_ID" --repo "$REPO"
    
    echo ""
    echo "🔗 View in browser:"
    echo "https://github.com/$REPO/actions/runs/$RUN_ID"
fi

echo ""
echo "✅ To view all workflows:"
echo "https://github.com/$REPO/actions"