# =========================================================
# Backend builder
# =========================================================
FROM golang:1.24-bookworm AS backend-builder

WORKDIR /src

COPY api/go.mod api/go.sum ./api/
RUN cd api && go mod download

COPY api/ ./api/

RUN cd api && CGO_ENABLED=1 go build -o /out/server ./cmd/server


# =========================================================
# Frontend builder
# =========================================================
FROM node:22-bookworm AS frontend-builder

ARG HUGO_VERSION=0.151.0
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates curl \
 && curl -sSL "https://github.com/gohugoio/hugo/releases/download/v${HUGO_VERSION}/hugo_extended_${HUGO_VERSION}_linux-amd64.tar.gz" \
      | tar -xz -C /usr/local/bin hugo \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /app/frontend

COPY frontend/package*.json ./
RUN npm install

COPY frontend/ ./

# Tailwind must run BEFORE Hugo so that static/css/styles.css exists
# when Hugo copies the static/ tree into the publish directory.
RUN npx @tailwindcss/cli -i input.css -o static/css/styles.css --minify \
 && hugo --minify


# =========================================================
# Runtime
# =========================================================
FROM debian:bookworm-slim

RUN apt-get update \
 && apt-get install -y --no-install-recommends \
      gosu \
      ca-certificates \
 && rm -rf /var/lib/apt/lists/*

RUN groupadd -r appgroup && useradd -r -g appgroup appuser

WORKDIR /app

COPY --from=backend-builder /out/server ./server

# Hugo's publishDir is set to "docs" in config/_default/hugo.toml.
COPY --from=frontend-builder /app/frontend/docs ./public

COPY infra/app/entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh \
 && mkdir -p /app/db /app/config \
 && chown -R appuser:appgroup /app

EXPOSE 4242

ENV BACKEND_STATIC_DIR=/app/public \
    BACKEND_DB_PATH=/app/db/production.db \
    BACKEND_CONFIG_PATH=/app/config/config.toml

ENTRYPOINT ["/entrypoint.sh"]

CMD ["./server"]
