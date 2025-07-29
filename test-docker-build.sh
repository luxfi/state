#!/bin/bash
set -euo pipefail

echo "Testing Docker build from genesis directory..."
cd /home/z/work/lux/genesis

echo "Building Docker image..."
make docker-build

echo "Listing Docker images..."
docker images | grep lux-genesis

echo "Done!"