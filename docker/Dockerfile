# ────────────────────────────────────────────────────────────────────────
# 🛠️ Stage 1: Build
# ────────────────────────────────────────────────────────────────────────
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS build

# Add build arguments for targeting specific platform
ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

# Install Task runner using Go
RUN go install github.com/go-task/task/v3/cmd/task@latest

# Copy Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod tidy

# Copy the full source code
COPY . .

# Set environment variables for cross-compilation
ENV CGO_ENABLED=0
ENV GOOS=${TARGETOS}
ENV GOARCH=${TARGETARCH}

# Build binaries
RUN go build -o bin/sbomer cmd/sbomer/main.go && \
    go build -o bin/fetcher cmd/fetcher/main.go

# ────────────────────────────────────────────────────────────────────────
# 🚀 Stage 2: Runtime
# ────────────────────────────────────────────────────────────────────────
FROM --platform=$TARGETPLATFORM golang:1.24-alpine

WORKDIR /app

# Install minimal runtime dependencies
RUN apk add --no-cache git

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
