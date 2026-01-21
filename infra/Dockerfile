# Stage 1 - Build Go binary
FROM golang:1.24 AS builder
WORKDIR /app

# Copy module files and download deps
COPY ../../go.mod ../../go.sum ./
RUN go mod download

# Copy source
COPY ../../ .

# Note: Swagger docs are managed manually - do not regenerate
# To regenerate: swag init -g cmd/api/main.go --output docs

# Build both binaries
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/api ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -o /bin/worker ./cmd/worker


# Stage 2 - Runtime
FROM debian:bookworm-slim
WORKDIR /app

# ✅ Install PostgreSQL client and CA certificates
RUN apt-get update && apt-get install -y --no-install-recommends \
    postgresql-client \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Copy binaries and entrypoint
COPY --from=builder /bin/api /app/api
COPY --from=builder /bin/worker /app/worker
COPY infra/entrypoint.sh /app/entrypoint.sh

RUN chmod +x /app/entrypoint.sh

EXPOSE 8080
ENTRYPOINT ["/app/entrypoint.sh"]
