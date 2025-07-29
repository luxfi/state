# Development Dockerfile for Lux Network with genesis
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev linux-headers

# Set working directory
WORKDIR /build

# Copy genesis tool source
COPY . .

# Build genesis tool
RUN make build

# Runtime stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache ca-certificates curl bash

# Create luxd user
RUN addgroup -g 1000 luxd && \
    adduser -u 1000 -G luxd -s /bin/sh -D luxd

# Create directories
RUN mkdir -p /app/data /app/configs /app/bin && \
    chown -R luxd:luxd /app

# Copy genesis tool
COPY --from=builder /build/bin/genesis /app/bin/genesis

# Copy genesis configs
COPY configs /app/configs

# Switch to luxd user
USER luxd

# Set working directory
WORKDIR /app

# Expose ports
EXPOSE 9630 9631

# Default command (can be overridden)
CMD ["/app/bin/genesis", "launch", "L1"]