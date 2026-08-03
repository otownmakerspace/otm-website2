# =========================================================
# Backend builder
# =========================================================
FROM golang:1.25-bookworm AS backend-builder

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
# Which Hugo config-environment to use (selects config/<env>/params.toml).
# Override at build time: --build-arg HUGO_ENVIRONMENT=staging
ARG HUGO_ENVIRONMENT=production
ARG HUGO_BASE_URL=""
# SUBDOMAIN/TOPDOMAIN flow through to the Hugo subprocess so templates
# can construct https://members.${SUBDOMAIN}.${TOPDOMAIN}/login/ etc.
# via os.Getenv. The marketing site links member-related URLs directly
# at the members backend host.
ARG SUBDOMAIN=""
ARG TOPDOMAIN=""
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates curl \
 && curl -sSL "https://github.com/gohugoio/hugo/releases/download/v${HUGO_VERSION}/hugo_extended_${HUGO_VERSION}_linux-amd64.tar.gz" \
      | tar -xz -C /usr/local/bin hugo \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /app/frontend

COPY frontend/package*.json ./
RUN npm install

COPY frontend/ ./

# Expose the Go-rendered template trees to Tailwind's content scanner.
# tailwind.config.js has `'../api/internal/**/*.html'` in its content array;
# without these COPYs the path resolves to a nonexistent dir inside the
# container and modal-only classes (bg-black/50, bg-red-600, etc.) silently
# get dropped from the compiled CSS bundle. Targeted COPYs keep the layer
# cache scoped to template changes, not Go code.
COPY api/internal/stripe/templates /app/api/internal/stripe/templates
COPY api/internal/email/templates /app/api/internal/email/templates

# Tailwind must run BEFORE Hugo so that static/css/styles.css exists
# when Hugo copies the static/ tree into the publish directory.
# SUBDOMAIN/TOPDOMAIN are exposed to Hugo via the HUGO_PARAMS_* convention:
# env vars prefixed with HUGO_PARAMS_ are auto-merged into .Site.Params
# (lowercased), so templates read them as .Site.Params.subdomain /
# .Site.Params.topdomain — no security.funcs.getenv whitelist needed.
RUN npx @tailwindcss/cli -i input.css -o static/css/styles.css --minify \
 && HUGO_PARAMS_SUBDOMAIN="${SUBDOMAIN}" HUGO_PARAMS_TOPDOMAIN="${TOPDOMAIN}" \
      hugo --minify --environment "${HUGO_ENVIRONMENT}" \
      ${HUGO_BASE_URL:+--baseURL "${HUGO_BASE_URL}"}


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
