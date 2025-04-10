# ────────────────────────────────────────────────────────────────────────
# 🛠️ Stage 1: Build
# ────────────────────────────────────────────────────────────────────────
FROM harbor.forge.berger-levrault.com/diac-labs/base-images/alpine:latest AS build

WORKDIR /app

# Install Go and Git for building
RUN apk add --no-cache go git

# Install Task runner using Go
RUN go install github.com/go-task/task/v3/cmd/task@latest && \
  mv /root/go/bin/task /usr/local/bin/task

# Copy Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod tidy

# Copy the full source code
COPY . .

# Build binaries using Task commands
RUN task build:sbomer && task build:fetcher

# ────────────────────────────────────────────────────────────────────────
# 🚀 Stage 2: Runtime
# ────────────────────────────────────────────────────────────────────────
FROM harbor.forge.berger-levrault.com/diac-labs/base-images/alpine:latest

WORKDIR /app

# Install minimal runtime dependencies
RUN apk add --no-cache go git syft

# Copy built binaries and necessary resources from build stage
COPY --from=build /app/go.mod ./
COPY --from=build /app/go.sum ./
COPY --from=build /app/bin/sbomer ./bin/sbomer
COPY --from=build /app/bin/fetcher ./bin/fetcher
COPY --from=build /app/scripts/migrate.go ./scripts/migrate.go
COPY --from=build /app/migrations ./migrations

# Pre-fetch required Go dependency if used at runtime
RUN go get github.com/jackc/pgx/v5/pgxpool

# Create required runtime directory
RUN mkdir -p /app/tmp/sbomer

# Copy the entrypoint script and make it executable
COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Start the application using the entrypoint script
ENTRYPOINT ["/entrypoint.sh"]
