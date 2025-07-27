#!/bin/bash
# Lux CLI wrapper for import operations

set -e

# Configuration
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
IMPORT_SCRIPT="$SCRIPT_DIR/import-chain-data.sh"
MONITOR_SCRIPT="$SCRIPT_DIR/monitor-node.sh"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Functions
usage() {
    echo "Lux Import CLI"
    echo ""
    echo "Usage: $0 <command> [options]"
    echo ""
    echo "Commands:"
    echo "  import <path>      Import chain data from specified path"
    echo "  monitor            Start monitoring the node"
    echo "  status             Check current node status"
    echo "  backup             Backup the current database"
    echo "  help               Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 import /path/to/chaindata"
    echo "  $0 monitor"
    echo "  $0 status"
}

log() {
    echo -e "${GREEN}[Lux Import]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
    exit 1
}

# Command handlers
cmd_import() {
    if [ -z "$1" ]; then
        error "Import path required"
    fi
    
    log "Starting chain data import from: $1"
    exec "$IMPORT_SCRIPT" "$1"
}

cmd_monitor() {
    log "Starting node monitoring..."
    exec "$MONITOR_SCRIPT"
}

cmd_status() {
    log "Checking node status..."
    
    # Check if node is running
    if pgrep -f "luxd.*data-dir" > /dev/null; then
        echo "✅ Node is running"
        
        # Get PID
        PID=$(pgrep -f "luxd.*data-dir" | head -1)
        echo "   PID: $PID"
        
        # Check RPC
        if curl -s http://localhost:9650/ext/health > /dev/null 2>&1; then
            echo "✅ RPC is accessible"
            
            # Get block height
            RESPONSE=$(curl -s -X POST -H "Content-Type: application/json" \
                -d '{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}' \
                http://localhost:9650/ext/bc/C/rpc 2>/dev/null || echo "{}")
            
            HEX_HEIGHT=$(echo "$RESPONSE" | grep -o '"result":"[^"]*"' | cut -d'"' -f4)
            if [ -n "$HEX_HEIGHT" ]; then
                HEIGHT=$(printf "%d\n" "$HEX_HEIGHT" 2>/dev/null || echo "unknown")
                echo "   Block height: $HEIGHT"
            fi
            
            # Get bootstrap status
            BOOTSTRAP=$(curl -s -X POST -H "Content-Type: application/json" \
                -d '{"jsonrpc":"2.0","id":1,"method":"info.isBootstrapped","params":{"chain":"C"}}' \
                http://localhost:9650/ext/info 2>/dev/null || echo "{}")
            
            if echo "$BOOTSTRAP" | grep -q '"isBootstrapped":true'; then
                echo "✅ Node is bootstrapped"
            else
                echo "⏳ Node is bootstrapping..."
            fi
        else
            echo "❌ RPC is not accessible"
        fi
    else
        echo "❌ Node is not running"
    fi
}

cmd_backup() {
    log "Creating database backup..."
    
    BACKUP_DIR="${BACKUP_DIR:-./backups}"
    DATA_DIR="${DATA_DIR:-$HOME/.luxd-import}"
    TIMESTAMP=$(date +%Y%m%d-%H%M%S)
    
    mkdir -p "$BACKUP_DIR"
    
    if [ ! -d "$DATA_DIR" ]; then
        error "Data directory not found: $DATA_DIR"
    fi
    
    BACKUP_FILE="$BACKUP_DIR/luxd-import-$TIMESTAMP.tar.gz"
    
    log "Creating backup: $BACKUP_FILE"
    tar -czf "$BACKUP_FILE" -C "$DATA_DIR" . || error "Backup failed"
    
    SIZE=$(du -h "$BACKUP_FILE" | cut -f1)
    log "✅ Backup created successfully (Size: $SIZE)"
}

# Main command dispatcher
case "${1:-help}" in
    import)
        shift
        cmd_import "$@"
        ;;
    monitor)
        cmd_monitor
        ;;
    status)
        cmd_status
        ;;
    backup)
        cmd_backup
        ;;
    help|--help|-h)
        usage
        ;;
    *)
        error "Unknown command: $1"
        ;;
esac