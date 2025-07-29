#!/bin/bash
# Check the status of the ongoing migration

echo "🔍 Migration Status Check"
echo "========================"
echo ""

# Check if migration is running
if pgrep -f "genesis.*migrate.*add-evm-prefix" > /dev/null; then
    echo "✅ Migration is currently RUNNING"
    echo ""
    
    # Get process info
    PID=$(pgrep -f "genesis.*migrate.*add-evm-prefix")
    echo "Process ID: $PID"
    echo "Running for: $(ps -o etime= -p $PID | xargs)"
    echo ""
else
    echo "❌ Migration is NOT running"
    echo ""
fi

# Check database sizes
echo "📊 Database Sizes:"
if [ -d "chaindata/lux-mainnet-96369/db/pebbledb" ]; then
    SRC_SIZE=$(du -sh chaindata/lux-mainnet-96369/db/pebbledb 2>/dev/null | cut -f1)
    echo "  Source DB: $SRC_SIZE"
fi

if [ -d "runtime/evm/pebbledb" ]; then
    DST_SIZE=$(du -sh runtime/evm/pebbledb 2>/dev/null | cut -f1)
    echo "  Migrated DB: $DST_SIZE"
    
    # Estimate progress
    if [ -n "$SRC_SIZE" ] && [ -n "$DST_SIZE" ]; then
        SRC_BYTES=$(du -sb chaindata/lux-mainnet-96369/db/pebbledb 2>/dev/null | cut -f1)
        DST_BYTES=$(du -sb runtime/evm/pebbledb 2>/dev/null | cut -f1)
        if [ "$SRC_BYTES" -gt 0 ]; then
            PROGRESS=$((DST_BYTES * 100 / SRC_BYTES))
            echo "  Progress: ~$PROGRESS%"
        fi
    fi
fi

echo ""

# Check last migration log
if [ -f "runtime/tip.txt" ]; then
    echo "📈 Last known tip: $(cat runtime/tip.txt)"
else
    echo "📈 Tip not yet determined"
fi

echo ""

# Estimate time remaining
if pgrep -f "genesis.*migrate" > /dev/null; then
    echo "⏱️  Estimation:"
    echo "  Based on 31M+ keys, this may take 15-30 minutes"
    echo "  The process is I/O bound, so SSD speed matters"
fi

echo ""
echo "💡 Tips:"
echo "  - Migration must complete before launching luxd"
echo "  - Watch progress with: tail -f nohup.out (if using nohup)"
echo "  - Check specific progress: ps aux | grep genesis"
echo ""
echo "When migration completes, run:"
echo "  make mainnet  (to continue with launch)"
echo "  OR"
echo "  ./scripts/validate-rpc.sh  (to verify after launch)"