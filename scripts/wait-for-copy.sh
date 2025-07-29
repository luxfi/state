#!/bin/bash
# Wait for copy to complete

TARGET_DIR="runtime/db/pebble/v1.0.0/chains/X6CU5qgMJfzsTB9UWxj2ZY5hd68x41HfZ4m4hCBWbHuj1Ebc3/vm/mgj786NP7uDwBCcq6YwThhaN8FLyybkCa4zBWTQbNgmK6k9A6/evm"
SOURCE_SIZE=$(du -sb runtime/evm/pebbledb/ | cut -f1)

echo "⏳ Waiting for copy to complete..."
echo "   Source size: $(du -sh runtime/evm/pebbledb/ | cut -f1)"

while true; do
    if [ -f "$TARGET_DIR/CURRENT" ]; then
        echo "✅ Copy complete!"
        echo "   Target size: $(du -sh $TARGET_DIR | cut -f1)"
        break
    fi
    
    CURRENT_SIZE=$(du -sb "$TARGET_DIR" 2>/dev/null | cut -f1 || echo 0)
    PROGRESS=$((CURRENT_SIZE * 100 / SOURCE_SIZE))
    echo -ne "\r   Progress: $PROGRESS% ($(du -sh $TARGET_DIR 2>/dev/null | cut -f1 || echo 0))"
    
    sleep 2
done

echo ""
echo "Ready to launch!"