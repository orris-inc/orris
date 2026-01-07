# Runtime stage only - binary is pre-built
FROM alpine:3.21

ARG TARGETARCH=amd64

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy pre-built binary
# Multi-arch: build/linux/{amd64,arm64}/orris
# Single-arch fallback: orris (when TARGETARCH dir doesn't exist)
COPY build/linux/${TARGETARCH}/orris /app/orris

# Copy migration scripts
COPY internal/infrastructure/migration/scripts /app/internal/infrastructure/migration/scripts

# Expose port
EXPOSE 8080

# Default command
ENTRYPOINT ["/app/orris"]
CMD ["server"]
