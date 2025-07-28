#!/bin/bash
# Script to import chain data into luxd with proper steps

set -e

# Configuration
LUXD_PATH="${LUXD_PATH:-/home/z/work/lux/node/build/luxd}"
DATA_DIR="${DATA_DIR:-$HOME/.luxd-import}"
IMPORT_SOURCE="${1:-}"
NETWORK_ID="${NETWORK_ID:-96369}"
LOG_DIR="${LOG_DIR:-./logs}"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Functions
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
    exit 1
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

usage() {
    echo "Usage: $0 <import-data-path>"
    echo ""
    echo "Import chain data into luxd node"
    echo ""
    echo "Arguments:"
    echo "  import-data-path    Path to the chain data to import"
    echo ""
    echo "Environment variables:"
    echo "  LUXD_PATH          Path to luxd binary (default: /home/z/work/lux/node/build/luxd)"
    echo "  DATA_DIR           Data directory for node (default: ~/.luxd-import)"
    echo "  NETWORK_ID         Network ID (default: 96369)"
    echo "  LOG_DIR            Log directory (default: ./logs)"
    exit 1
}

# Check arguments
if [ -z "$IMPORT_SOURCE" ]; then
    usage
fi

if [ ! -d "$IMPORT_SOURCE" ]; then
    error "Import source directory does not exist: $IMPORT_SOURCE"
fi

# Create directories
mkdir -p "$LOG_DIR"
mkdir -p "$DATA_DIR"

# Kill any existing luxd process
log "Stopping any existing luxd processes..."
pkill -f "luxd.*data-dir=$DATA_DIR" || true
sleep 2

# Start node in import mode
log "Starting node in import mode..."
log "Import source: $IMPORT_SOURCE"
log "Data directory: $DATA_DIR"
log "Network ID: $NETWORK_ID"

# Create log file with timestamp
IMPORT_LOG="$LOG_DIR/import-$(date +%Y%m%d-%H%M%S).log"

# Start luxd with import flag
nohup "$LUXD_PATH" \
    --network-id="$NETWORK_ID" \
    --data-dir="$DATA_DIR" \
    --import-chain-data="$IMPORT_SOURCE" \
    --http-host=0.0.0.0 \
    --http-port=9630 \
    --staking-ephemeral-cert-enabled \
    --public-ip=127.0.0.1 \
    --log-level=info \
    > "$IMPORT_LOG" 2>&1 &

LUXD_PID=$!
log "Started luxd with PID: $LUXD_PID"
log "Logs: $IMPORT_LOG"

# Monitor import progress
log "Monitoring import progress..."
log "Waiting for trie rebuild/snapshot to complete..."

# Function to check if import is complete
check_import_status() {
    # Check logs for completion markers
    if grep -q "Rebuilding state snapshot" "$IMPORT_LOG"; then
        echo "rebuilding"
    elif grep -q "Generated state snapshot" "$IMPORT_LOG"; then
        echo "complete"
    elif grep -q "Failed to import" "$IMPORT_LOG"; then
        echo "failed"
    elif grep -q "shutting down" "$IMPORT_LOG"; then
        echo "shutdown"
    else
        echo "running"
    fi
}

# Monitor loop
WAIT_TIME=0
MAX_WAIT=3600  # 1 hour max
while [ $WAIT_TIME -lt $MAX_WAIT ]; do
    STATUS=$(check_import_status)
    
    case $STATUS in
        "rebuilding")
            log "State snapshot rebuild in progress..."
            ;;
        "complete")
            log "✅ Import completed successfully!"
            break
            ;;
        "failed")
            error "Import failed! Check logs: $IMPORT_LOG"
            ;;
        "shutdown")
            error "Node shut down unexpectedly! Check logs: $IMPORT_LOG"
            ;;
        "running")
            # Show progress
            if [ $((WAIT_TIME % 30)) -eq 0 ]; then
                log "Import still running... ($WAIT_TIME seconds elapsed)"
                # Show last few log lines
                tail -5 "$IMPORT_LOG" | sed 's/^/  > /'
            fi
            ;;
    esac
    
    sleep 5
    WAIT_TIME=$((WAIT_TIME + 5))
done

if [ $WAIT_TIME -ge $MAX_WAIT ]; then
    error "Import timeout after $MAX_WAIT seconds"
fi

# Stop the node
log "Stopping node after import..."
kill $LUXD_PID 2>/dev/null || true

# Wait for clean shutdown
sleep 10

# Start node normally without import mode
log "Starting node in normal mode..."
NORMAL_LOG="$LOG_DIR/normal-$(date +%Y%m%d-%H%M%S).log"

nohup "$LUXD_PATH" \
    --network-id="$NETWORK_ID" \
    --data-dir="$DATA_DIR" \
    --http-host=0.0.0.0 \
    --http-port=9630 \
    --staking-ephemeral-cert-enabled \
    --public-ip=127.0.0.1 \
    --pruning-enabled \
    --state-sync-enabled=false \
    --log-level=info \
    > "$NORMAL_LOG" 2>&1 &

NORMAL_PID=$!
log "Started node normally with PID: $NORMAL_PID"
log "Logs: $NORMAL_LOG"

# Wait for node to be ready
log "Waiting for node to be ready..."
sleep 20

# Check if node is running
if curl -s -X POST -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","id":1,"method":"info.isBootstrapped","params":{"chain":"C"}}' \
    http://localhost:9630/ext/info > /dev/null 2>&1; then
    log "✅ Node is running successfully!"
else
    error "Node failed to start properly"
fi

# Create backup script
cat > "$LOG_DIR/backup-database.sh" << 'EOF'
#!/bin/bash
# Backup the imported database

BACKUP_DIR="${BACKUP_DIR:-./backups}"
DATA_DIR="${DATA_DIR:-$HOME/.luxd-import}"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)

mkdir -p "$BACKUP_DIR"

echo "Creating backup of imported database..."
tar -czf "$BACKUP_DIR/luxd-import-$TIMESTAMP.tar.gz" -C "$DATA_DIR" .
echo "Backup created: $BACKUP_DIR/luxd-import-$TIMESTAMP.tar.gz"
echo "Size: $(du -h "$BACKUP_DIR/luxd-import-$TIMESTAMP.tar.gz" | cut -f1)"
EOF

chmod +x "$LOG_DIR/backup-database.sh"

log "✅ Import process completed!"
log ""
log "Next steps:"
log "1. Monitor the node: tail -f $NORMAL_LOG"
log "2. Check node status: curl -X POST -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"info.isBootstrapped\",\"params\":{\"chain\":\"C\"}}' http://localhost:9630/ext/info"
log "3. Backup database: $LOG_DIR/backup-database.sh"
log "4. Monitor for 48h before enabling indexing"
log ""
log "Node PID: $NORMAL_PID"
log "Data directory: $DATA_DIR"