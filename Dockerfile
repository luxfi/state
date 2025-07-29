# Production Dockerfile for Lux Network with migrated blockchain data
FROM ghcr.io/luxfi/node:latest AS base

# Switch to root to set up data
USER root

# Create data directories
RUN mkdir -p /data/db/chains/X6CU5qgMJfzsTB9UWxj2ZY5hd68x41HfZ4m4hCBWbHuj1Ebc3/db \
    /data/db/chains/X6CU5qgMJfzsTB9UWxj2ZY5hd68x41HfZ4m4hCBWbHuj1Ebc3/vm/mgj786NP7uDwBCcq6YwThhaN8FLyybkCa4zBWTQbNgmK6k9A6/evm \
    /data/configs/C \
    /app/plugins

# Copy consensus database
COPY runtime/luxd-final/db/chains/X6CU5qgMJfzsTB9UWxj2ZY5hd68x41HfZ4m4hCBWbHuj1Ebc3/db /data/db/chains/X6CU5qgMJfzsTB9UWxj2ZY5hd68x41HfZ4m4hCBWbHuj1Ebc3/db

# Copy EVM database (this is the large one - we'll need to be selective)
# We'll copy only essential SST files and the MANIFEST
COPY runtime/luxd-final/db/chains/X6CU5qgMJfzsTB9UWxj2ZY5hd68x41HfZ4m4hCBWbHuj1Ebc3/vm/mgj786NP7uDwBCcq6YwThhaN8FLyybkCa4zBWTQbNgmK6k9A6/evm/MANIFEST* \
     runtime/luxd-final/db/chains/X6CU5qgMJfzsTB9UWxj2ZY5hd68x41HfZ4m4hCBWbHuj1Ebc3/vm/mgj786NP7uDwBCcq6YwThhaN8FLyybkCa4zBWTQbNgmK6k9A6/evm/CURRENT \
     runtime/luxd-final/db/chains/X6CU5qgMJfzsTB9UWxj2ZY5hd68x41HfZ4m4hCBWbHuj1Ebc3/vm/mgj786NP7uDwBCcq6YwThhaN8FLyybkCa4zBWTQbNgmK6k9A6/evm/OPTIONS* \
     /data/db/chains/X6CU5qgMJfzsTB9UWxj2ZY5hd68x41HfZ4m4hCBWbHuj1Ebc3/vm/mgj786NP7uDwBCcq6YwThhaN8FLyybkCa4zBWTQbNgmK6k9A6/evm/

# Copy config
COPY runtime/luxd-final/configs/C/config.json /data/configs/C/

# Copy geth plugin
COPY runtime/plugins/mgj786NP7uDwBCcq6YwThhaN8FLyybkCa4zBWTQbNgmK6k9A6 /app/plugins/

# Set ownership
RUN chown -R luxd:luxd /data /app

# Switch back to luxd user
USER luxd

# Set environment variables for imported blockchain
ENV LUX_IMPORTED_BLOCK_ID="646572b42a6210ac8efea0ab0df2a028acde2297c3ae07bc8dd1fc3e120b802a"
ENV LUX_IMPORTED_HEIGHT="1082780"
ENV LUX_IMPORTED_TIMESTAMP="1717148410"

# Expose ports
EXPOSE 9630 9631

# Set working directory
WORKDIR /app

# Launch luxd with production settings
CMD ["/app/luxd", \
    "--network-id=96369", \
    "--db-dir=/data/db", \
    "--chain-config-dir=/data/configs", \
    "--plugin-dir=/app/plugins", \
    "--http-host=0.0.0.0", \
    "--http-port=9630", \
    "--staking-port=9631", \
    "--log-level=info", \
    "--api-admin-enabled=false", \
    "--api-auth-required=false", \
    "--api-metrics-enabled=true", \
    "--health-check-frequency=30s", \
    "--network-allow-private-ips=false", \
    "--dev"]