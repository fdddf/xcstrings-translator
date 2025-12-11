# Multi-stage build for xcstrings-translator

# UI builder stage
FROM node:20-alpine AS ui-builder

# Install git for cloning
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Clone the repository
RUN git clone https://github.com/fdddf/xcstrings-translator.git . && \
    cd web && npm install && npm run build

# Builder stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    gcc \
    musl-dev \
    nodejs \
    npm

# Set working directory
WORKDIR /app

# Clone the repository
RUN git clone https://github.com/fdddf/xcstrings-translator.git .

# Install Go dependencies
RUN go mod tidy

# Copy built UI assets from ui-builder stage
COPY --from=ui-builder /app/webui/dist ./webui/dist

# Build the application with GUI tag using Makefile
RUN make gui

# Final stage - minimal Alpine image
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    && update-ca-certificates

# Create non-root user
RUN addgroup -g 65532 --system nonroot && \
    adduser -D -u 65532 -G nonroot --system nonroot

# Copy the binary from builder stage
COPY --from=builder /app/xcstrings-translator /usr/local/bin/xcstrings-translator

# Create app directory and set permissions
RUN mkdir -p /app && \
    chown -R nonroot:nonroot /app

WORKDIR /app

# Make binary executable
RUN chmod +x /usr/local/bin/xcstrings-translator

# Switch to non-root user
USER nonroot:nonroot

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["sh", "-c", "xcstrings-translator --help"]

# Default command
ENTRYPOINT ["xcstrings-translator"]
CMD ["--help"]
