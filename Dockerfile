# Runtime stage only - binary is pre-built
FROM alpine:3.21

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy pre-built binary
COPY orris /app/orris

# Copy migration scripts
COPY internal/infrastructure/migration/scripts /app/internal/infrastructure/migration/scripts

# Expose port
EXPOSE 8080

# Default command
ENTRYPOINT ["/app/orris"]
CMD ["server"]
