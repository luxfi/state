#!/bin/bash

echo "Stopping validators..."
for i in {1..5}; do
    if [ -f network-runner/node$i/node.pid ]; then
        PID=$(cat network-runner/node$i/node.pid)
        if kill -0 $PID 2>/dev/null; then
            echo "Stopping node$i (PID $PID)..."
            kill $PID
        fi
    fi
done

# Also kill any remaining luxd processes
pkill -f "luxd --network-id=96369" || true

echo "✅ Validators stopped"