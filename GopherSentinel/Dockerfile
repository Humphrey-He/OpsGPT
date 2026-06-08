# ============================================
# Stage 1: Builder
# ============================================
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git ca-certificates curl

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags='-w -s' -o GopherSentinel ./cmd/cli

# ============================================
# Stage 2: Runtime
# ============================================
FROM alpine:3.19

LABEL maintainer="GopherSentinel"
LABEL description="AI-powered DevOps Assistant"

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    curl \
    tzdata \
    && cp -r /usr/share/zoneinfo /etc/localtime \
    && echo $TZ > /etc/timezone

# Create non-root user for security
RUN addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -s /bin/sh -D appuser

# Copy binary from builder
COPY --from=builder /build/GopherSentinel .

# Copy config (if exists)
COPY --from=builder /build/configs ./configs 2>/dev/null || true

# Set ownership
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Environment variables
ENV GIN_MODE=release
ENV TZ=Asia/Shanghai

# Expose ports
# CLI runs on 8080 by default
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Entry point
ENTRYPOINT ["./GopherSentinel"]

# Default command (shows help)
CMD ["--help"]
