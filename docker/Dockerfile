# Multi-stage build for optimized production image
FROM golang:1.21-alpine AS builder

# Install build dependencies and CA certificates
RUN apk add --no-cache git ca-certificates tzdata && update-ca-certificates

# Create non-root user for security
RUN adduser -D -u 10001 appuser

WORKDIR /app

# Download dependencies first (better layer caching)
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application with optimizations
# Remove the git version for now to simplify
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=1.0.0" \
    -o /app/bin/server ./cmd/server

# Create minimal runtime image
FROM scratch

# Import from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /app/bin/server /app/server

# Use non-root user
USER appuser

# Expose port (should match your config)
EXPOSE 8081

# Health check (simpler approach)
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/server", "healthcheck"]

# Command to run the application
CMD ["/app/server"]