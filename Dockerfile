# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/orris ./cmd/orris

# Runtime stage
FROM alpine:3.21

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/orris /app/orris

# Copy migration scripts
COPY --from=builder /app/internal/infrastructure/migration/scripts /app/migrations

# Expose port
EXPOSE 8080

# Default command
ENTRYPOINT ["/app/orris"]
CMD ["server"]
