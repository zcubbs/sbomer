# ────────────────────────────────────────────────────────────────────────
# 🚀 Stage: Runtime
# ────────────────────────────────────────────────────────────────────────
FROM golang:1.24-alpine

WORKDIR /app

# Copy pre-built binaries and necessary resources
COPY sbomer /app/bin/sbomer
COPY fetcher /app/bin/fetcher
COPY go.mod go.sum ./
COPY scripts/migrate.go ./scripts/migrate.go
COPY migrations ./migrations
COPY docker/entrypoint.sh /entrypoint.sh

# Start the application using the entrypoint script
ENTRYPOINT ["/entrypoint.sh"]
