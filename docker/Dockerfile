# ────────────────────────────────────────────────────────────────────────
# 🚀 Stage: Runtime
# ────────────────────────────────────────────────────────────────────────
FROM alpine:3.19

WORKDIR /app

# Install minimal runtime dependencies
RUN apk add --no-cache git

# Create required runtime directory
RUN mkdir -p /app/tmp/sbomer

# Copy pre-built binaries and necessary resources
COPY sbomer /app/bin/sbomer
COPY fetcher /app/bin/fetcher
COPY go.mod go.sum ./
COPY scripts/migrate.go ./scripts/migrate.go
COPY migrations ./migrations

# Copy the entrypoint script and make it executable
COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Start the application using the entrypoint script
ENTRYPOINT ["/entrypoint.sh"]
