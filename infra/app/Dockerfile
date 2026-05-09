# Multi-stage build for the otm-website backend.
#
# Build context: repo root (so we can COPY both api/ for the build and
# frontend/docs/ for the static site).

FROM golang:1.24-bookworm AS builder

WORKDIR /src
COPY api/go.mod api/go.sum ./api/
RUN cd api && go mod download

COPY api/ ./api/
RUN cd api && CGO_ENABLED=1 go build -o /out/server ./cmd/server

# --- Runtime ---
FROM debian:bookworm-slim

RUN apt-get update \
 && apt-get install -y --no-install-recommends gosu ca-certificates \
 && rm -rf /var/lib/apt/lists/*
RUN groupadd -r appgroup && useradd -r -g appgroup appuser

WORKDIR /app

COPY --from=builder /out/server ./server
COPY frontend/docs/ ./public/
COPY infra/app/entrypoint.sh /entrypoint.sh

RUN mkdir -p /app/db /app/config && chown -R appuser:appgroup /app

EXPOSE 4242

# Path defaults for the new config-driven backend. The compose env block can
# override any of these.
ENV BACKEND_STATIC_DIR=/app/public \
    BACKEND_DB_PATH=/app/db/production.db \
    BACKEND_CONFIG_PATH=/app/config/config.toml

ENTRYPOINT ["/entrypoint.sh"]
CMD ["./server"]
