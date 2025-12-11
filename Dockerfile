# Multi-stage build for xcstrings-translator
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    gcc \
    musl-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o bin/xcstrings-translator .

# Final stage - minimal Alpine image
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add \
    ca-certificates \
    tzdata

# Create non-root user
RUN addgroup -g 65532 nonroot && \
    adduser -D -u 65532 -G nonroot nonroot

# Copy the binary from builder stage
COPY --from=builder /app/bin/xcstrings-translator /usr/local/bin/xcstrings-translator

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
    CMD xcstrings-translator --help || exit 1

# Default command
ENTRYPOINT ["xcstrings-translator"]
CMD ["--help"]
