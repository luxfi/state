# Single Dockerfile for genesis migration and luxd launch
# Purpose: Boot working version using old subnet EVM chaindata

FROM golang:1.22.5-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev linux-headers

# Set working directory
WORKDIR /build

# Copy genesis tool source
COPY . /build/genesis

# Build genesis tool
RUN cd /build/genesis && make build

# Install luxd with canonical key patches
# Using go install to get the tagged version
RUN go install github.com/luxfi/node/main@v1.13.3-canonical-9byte || \
    go install github.com/luxfi/node/main@latest

# If go install fails, build from source with patches
RUN if [ ! -f /go/bin/main ]; then \
        git clone https://github.com/luxfi/node.git /build/node && \
        cd /build/node && \
        git checkout v1.13.3 || git checkout main && \
        # Apply canonical key patches inline \
        sed -i 's/append(append(headerPrefix, encodeBlockNumber(number)...), headerHashSuffix...)/append(headerPrefix, encodeBlockNumber(number)...)/' \
            vms/cchainvm/*.go geth/core/rawdb/*.go 2>/dev/null || true && \
        go build -o /go/bin/luxd ./main; \
    else \
        mv /go/bin/main /go/bin/luxd; \
    fi

# Runtime stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache ca-certificates bash

# Create luxd user
RUN addgroup -g 1000 luxd && \
    adduser -u 1000 -G luxd -s /bin/sh -D luxd

# Create necessary directories
RUN mkdir -p /app/data /app/chaindata /app/runtime /app/bin && \
    chown -R luxd:luxd /app

# Copy binaries
COPY --from=builder /build/genesis/bin/genesis /app/bin/genesis
COPY --from=builder /go/bin/luxd /app/bin/luxd

# Copy genesis configs
COPY --from=builder /build/genesis/configs /app/configs

# Copy entrypoint script
COPY docker/entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

# Switch to luxd user
USER luxd
WORKDIR /app

# Environment variables
ENV CHAIN_ID=X6CU5qgMJfzsTB9UWxj2ZY5hd68x41HfZ4m4hCBWbHuj1Ebc3
ENV VM_ID=mgj786NP7uDwBCcq6YwThhaN8FLyybkCa4zBWTQbNgmK6k9A6
ENV NETWORK_ID=96369

# Expose ports
EXPOSE 9630 9631

# Default command: run migration if needed, then launch luxd
CMD ["/app/entrypoint.sh"]