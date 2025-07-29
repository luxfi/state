#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

ACTION=${1:-start}

case $ACTION in
  start)
    echo "Starting LUX mainnet production node..."
    docker-compose -f docker-compose.prod.yml up -d
    echo "Waiting for node to be healthy..."
    sleep 10
    docker-compose -f docker-compose.prod.yml ps
    ;;
    
  stop)
    echo "Stopping LUX mainnet production node..."
    docker-compose -f docker-compose.prod.yml down
    ;;
    
  restart)
    echo "Restarting LUX mainnet production node..."
    docker-compose -f docker-compose.prod.yml restart
    ;;
    
  logs)
    docker-compose -f docker-compose.prod.yml logs -f --tail=100
    ;;
    
  status)
    docker-compose -f docker-compose.prod.yml ps
    echo ""
    echo "Testing RPC endpoint..."
    curl -s -X POST -H "Content-Type: application/json" \
      -d '{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}' \
      http://localhost:9630/ext/bc/C/rpc || echo "RPC not ready"
    ;;
    
  pull)
    echo "Pulling latest genesis image..."
    docker-compose -f docker-compose.prod.yml pull
    ;;
    
  *)
    echo "Usage: $0 {start|stop|restart|logs|status|pull}"
    exit 1
    ;;
esac