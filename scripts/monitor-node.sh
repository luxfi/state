#!/bin/bash
# Script to monitor luxd node after import

set -e

# Configuration
RPC_URL="${RPC_URL:-http://localhost:9630}"
CHECK_INTERVAL="${CHECK_INTERVAL:-60}"  # Check every 60 seconds
ALERT_THRESHOLD="${ALERT_THRESHOLD:-5}" # Alert after 5 consecutive failures
LOG_FILE="${LOG_FILE:-./monitoring.log}"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Counters
FAIL_COUNT=0
TOTAL_CHECKS=0
START_TIME=$(date +%s)

# Functions
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1" | tee -a "$LOG_FILE"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1" | tee -a "$LOG_FILE"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOG_FILE"
}

check_node_status() {
    # Check if node is bootstrapped
    local response=$(curl -s -X POST -H "Content-Type: application/json" \
        -d '{"jsonrpc":"2.0","id":1,"method":"info.isBootstrapped","params":{"chain":"C"}}' \
        "$RPC_URL/ext/info" 2>/dev/null)
    
    if [ $? -ne 0 ]; then
        return 1
    fi
    
    # Parse response
    local is_bootstrapped=$(echo "$response" | grep -o '"isBootstrapped":[^,}]*' | cut -d: -f2)
    
    if [ "$is_bootstrapped" = "true" ]; then
        return 0
    else
        return 1
    fi
}

get_block_height() {
    local response=$(curl -s -X POST -H "Content-Type: application/json" \
        -d '{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}' \
        "$RPC_URL/ext/bc/C/rpc" 2>/dev/null)
    
    if [ $? -eq 0 ]; then
        local hex_height=$(echo "$response" | grep -o '"result":"[^"]*"' | cut -d'"' -f4)
        if [ -n "$hex_height" ]; then
            printf "%d\n" "$hex_height" 2>/dev/null || echo "0"
        else
            echo "0"
        fi
    else
        echo "0"
    fi
}

get_peers_count() {
    local response=$(curl -s -X POST -H "Content-Type: application/json" \
        -d '{"jsonrpc":"2.0","id":1,"method":"info.peers","params":[]}' \
        "$RPC_URL/ext/info" 2>/dev/null)
    
    if [ $? -eq 0 ]; then
        echo "$response" | grep -o '"peers":\[[^]]*\]' | grep -o '{' | wc -l
    else
        echo "0"
    fi
}

get_node_version() {
    local response=$(curl -s -X POST -H "Content-Type: application/json" \
        -d '{"jsonrpc":"2.0","id":1,"method":"info.getNodeVersion","params":[]}' \
        "$RPC_URL/ext/info" 2>/dev/null)
    
    if [ $? -eq 0 ]; then
        echo "$response" | grep -o '"nodeVersion":"[^"]*"' | cut -d'"' -f4
    else
        echo "unknown"
    fi
}

# Main monitoring loop
log "Starting node monitoring..."
log "RPC URL: $RPC_URL"
log "Check interval: $CHECK_INTERVAL seconds"
log "Alert threshold: $ALERT_THRESHOLD consecutive failures"

# Get initial node info
NODE_VERSION=$(get_node_version)
log "Node version: $NODE_VERSION"

while true; do
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    
    # Check node status
    if check_node_status; then
        # Node is healthy
        BLOCK_HEIGHT=$(get_block_height)
        PEERS_COUNT=$(get_peers_count)
        UPTIME=$(($(date +%s) - START_TIME))
        UPTIME_HOURS=$((UPTIME / 3600))
        UPTIME_MINS=$(((UPTIME % 3600) / 60))
        
        log "Node healthy - Block: $BLOCK_HEIGHT, Peers: $PEERS_COUNT, Uptime: ${UPTIME_HOURS}h ${UPTIME_MINS}m"
        
        # Reset fail count
        if [ $FAIL_COUNT -gt 0 ]; then
            log "Node recovered after $FAIL_COUNT failures"
        fi
        FAIL_COUNT=0
        
        # Check for specific milestones
        if [ $BLOCK_HEIGHT -gt 0 ] && [ $((BLOCK_HEIGHT % 1000)) -eq 0 ]; then
            log "Milestone reached: Block $BLOCK_HEIGHT"
        fi
        
        # After 48 hours, suggest enabling indexing
        if [ $UPTIME_HOURS -ge 48 ] && [ -f ".monitoring_48h_flag" ]; then
            rm -f ".monitoring_48h_flag"
            log "✅ 48 hours milestone reached! Node is stable."
            log "You can now enable indexing and spin up additional validators."
        fi
        
    else
        # Node is not healthy
        FAIL_COUNT=$((FAIL_COUNT + 1))
        error "Node check failed ($FAIL_COUNT/$ALERT_THRESHOLD)"
        
        if [ $FAIL_COUNT -ge $ALERT_THRESHOLD ]; then
            error "CRITICAL: Node has failed $FAIL_COUNT consecutive checks!"
            # You could add alerting here (email, webhook, etc.)
        fi
    fi
    
    # Show summary every 10 checks
    if [ $((TOTAL_CHECKS % 10)) -eq 0 ]; then
        log "Summary: Total checks: $TOTAL_CHECKS, Current fail count: $FAIL_COUNT"
    fi
    
    sleep $CHECK_INTERVAL
done