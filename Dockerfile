# ==========================================
# Build Stage
# ==========================================
FROM golang:1.24-alpine AS builder

# Install build dependencies
# gcc and musl-dev are needed for CGO (if any dependencies require it)
RUN apk add --no-cache \
    git \
    ca-certificates \
    gcc \
    musl-dev

# Set working directory
WORKDIR /build

# Copy dependency files first for better layer caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the binary
# - CGO_ENABLED=0 for static binary (no C dependencies)
# - -ldflags="-w -s" strips debug info and symbol table (smaller binary)
# - -trimpath removes file system paths from binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -trimpath \
    -o /build/api \
    ./cmd/api/main.go

# ==========================================
# Runtime Stage
# ==========================================
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    && addgroup -g 1000 appuser \
    && adduser -D -u 1000 -G appuser appuser

# Set working directory
WORKDIR /app

# Copy binary from build stage
COPY --from=builder /build/api /app/api

# Change ownership to non-root user
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose application port
EXPOSE 8080

# Add healthcheck
# - Checks /health endpoint every 30s
# - 3s timeout for the check
# - 3 retries before marking unhealthy
# - 30s startup grace period
HEALTHCHECK --interval=30s --timeout=3s --start-period=30s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["/app/api"]
