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

RUN apk update && apk add --no-cache make gcc musl-dev

# Set working directory
WORKDIR /app

COPY . .

# Install Go dependencies
RUN go env -w GOPROXY='https://goproxy.cn,direct' && go mod tidy

# Copy built UI assets from ui-builder stage
COPY --from=ui-builder /app/webui/dist ./webui/dist

# Build the application using Makefile
RUN make

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
    CMD nc -z localhost 8080 || exit 1

# Default command
ENTRYPOINT ["xcstrings-translator"]
CMD ["serve"]

# Expose default port
EXPOSE 8080