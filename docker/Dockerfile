# Multi-stage Dockerfile for LUX Genesis deployment
# Uses official LUX images from GitHub Container Registry

# Stage 1: Genesis data preparation
FROM golang:1.24-alpine AS genesis-builder
WORKDIR /genesis

# Install dependencies
RUN apk add --no-cache git make

# Copy genesis project files
COPY . /genesis/

# Build tools and prepare data
RUN make build-tools

# Stage 2: Runtime with LUX node
FROM ghcr.io/luxfi/node:latest AS runtime

# Install additional tools
RUN apt-get update && apt-get install -y \
    curl \
    jq \
    netcat \
    && rm -rf /var/lib/apt/lists/*

# Copy genesis data and tools
COPY --from=genesis-builder /genesis/configs /genesis/configs
COPY --from=genesis-builder /genesis/chaindata /genesis/chaindata
COPY --from=genesis-builder /genesis/bin /genesis/bin

# Set working directory
WORKDIR /genesis

# Expose standard ports
EXPOSE 9630 9631

# Default to running with 96369 mainnet config
CMD ["luxd", "--dev", "--network-id=96369", "--chain-config-dir=/genesis/configs/lux-mainnet-96369", "--data-dir=/genesis/chaindata/lux-mainnet-96369/db", "--http-port=9630", "--staking-port=9631"]