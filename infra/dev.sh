#!/usr/bin/env bash
# Helper script to manage local stacks for the otm-website (otownmakerspace.dk).
# Location: infra/dev.sh
# Usage examples:
#   ./infra/dev.sh app up             # start app + reverse-proxy
#   ./infra/dev.sh app up --dev       # start with stripe-cli for webhook forwarding
#   ./infra/dev.sh app down           # stop app stack + reverse-proxy
#   ./infra/dev.sh app logs           # tail app logs
#   ./infra/dev.sh proxy up|down      # start/stop only the reverse-proxy
#   ./infra/dev.sh build              # build Hugo site (frontend/) + Docker image
#   ./infra/dev.sh rebuild            # down, build, up
#   ./infra/dev.sh export-env         # print command to export app env vars
#   ./infra/dev.sh status             # show running containers
#
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"
APP_COMPOSE="$ROOT_DIR/infra/app/docker-compose.yml"
APP_ENV="$ROOT_DIR/infra/app/.env"
PROXY_COMPOSE="$ROOT_DIR/infra/reverse-proxy/reverse-proxy.yml"
PROXY_ENV="$ROOT_DIR/infra/reverse-proxy/.env"

# Colors
c_green="\033[0;32m"; c_yellow="\033[0;33m"; c_red="\033[0;31m"; c_nc="\033[0m"

msg()  { echo -e "${c_green}==>${c_nc} $*"; }
warn() { echo -e "${c_yellow}==> WARN:${c_nc} $*"; }
err()  { echo -e "${c_red}==> ERROR:${c_nc} $*"; }

usage() {
  cat <<EOF
otm-website dev helper

Stacks and files:
  App stack:        infra/app/docker-compose.yml  (env: infra/app/.env)
  Reverse proxy:    infra/reverse-proxy/reverse-proxy.yml (env: infra/reverse-proxy/.env)
  Secrets:          secrets/app/ (gitignored — placeholder files auto-created on first up)

Commands:
  app up [--dev]    Start app stack (starts reverse-proxy first)
                    --dev: also starts stripe-cli for webhook forwarding
  app down          Stop app stack, then stop reverse-proxy
  app logs [svc]    Tail logs (default: otm-app)
  app ps            Show app stack containers

  proxy up|down     Start/stop only the reverse-proxy stack

  build             Build Hugo site (from frontend/) + Docker image
  rebuild           app down; build; app up

  export-env        Print command to export app envs into your shell
  status            Shortcut for 'app ps'
  help              Show this help
EOF
}

need_compose() {
  if ! command -v docker >/dev/null 2>&1; then err "docker is required"; exit 1; fi
  if ! docker compose version >/dev/null 2>&1; then err "docker compose v2 is required"; exit 1; fi
}

ensure_network() {
  if ! docker network inspect app_network >/dev/null 2>&1; then
    msg "Creating external network 'app_network'"
    docker network create app_network >/dev/null
  fi
}

ensure_secrets() {
  local secrets_dir="$ROOT_DIR/secrets/app"
  # Per-file (not just first-run) so a checkout that predates a newly added
  # secret still gets the file — compose bind mounts fail on missing sources.
  mkdir -p "$secrets_dir"
  for f in cookie_store_key stripe_key stripe_publishable_key stripe_webhook_secret smtp_username smtp_password; do
    if [[ ! -f "$secrets_dir/$f" ]]; then
      echo "REPLACE_ME" > "$secrets_dir/$f"
      warn "  Created $secrets_dir/$f — fill in the real value"
    fi
  done
  # Google Contacts sync creds are optional: empty file = feature disabled.
  # Created empty (not REPLACE_ME) so the backend cleanly disables the sync
  # instead of hammering Google with a garbage token.
  for f in google_client_id google_client_secret google_refresh_token; do
    if [[ ! -f "$secrets_dir/$f" ]]; then
      : > "$secrets_dir/$f"
      msg "  Created empty $secrets_dir/$f (Google Contacts sync disabled until filled)"
    fi
  done
}

compose_app() {
  need_compose
  ensure_network
  ensure_secrets

  local abs_secrets_dir="$ROOT_DIR/secrets/app"

  local env_args=()
  if [[ -f "$APP_ENV" ]]; then
    env_args+=("--env-file" "$APP_ENV")
    set -a; . "$APP_ENV"; set +a
  else
    warn "Env file not found at $APP_ENV (copy infra/app/.env.example to infra/app/.env)"
  fi

  : "${TOPDOMAIN:?TOPDOMAIN must be set in infra/app/.env}"
  : "${SUBDOMAIN?SUBDOMAIN must be set in infra/app/.env (leave empty for an apex deploy)}"

  SECRETS_FOLDER="$abs_secrets_dir" \
  NETWORK_NAME="${NETWORK_NAME:-app_network}" \
  docker compose -f "$APP_COMPOSE" "${env_args[@]}" "$@"
}

compose_proxy() {
  need_compose
  ensure_network

  local env_args=()
  if [[ -f "$PROXY_ENV" ]]; then
    env_args+=("--env-file" "$PROXY_ENV")
  else
    warn "Env file not found at $PROXY_ENV."
  fi

  NETWORK_NAME="${NETWORK_NAME:-app_network}" \
  docker compose -f "$PROXY_COMPOSE" "${env_args[@]}" "$@"
}

cmd="${1:-help}"
shift || true

case "$cmd" in
  help|-h|--help)
    usage ;;

  status)
    "$0" app ps ;;

  export-env)
    if [[ -f "$APP_ENV" ]]; then
      echo "set -a; . '$APP_ENV'; set +a"
    else
      err "Env file not found at $APP_ENV"; exit 1
    fi ;;

  build)
    # Determine image tag from env file
    local_tag="otm-website:latest"
    if [[ -f "$APP_ENV" ]]; then
      set -a; . "$APP_ENV"; set +a
      if [[ -n "${IMAGE:-}" ]]; then local_tag="$IMAGE"; fi
    fi

    : "${TOPDOMAIN:?TOPDOMAIN must be set in infra/app/.env}"
    : "${SUBDOMAIN?SUBDOMAIN must be set in infra/app/.env (leave empty for an apex deploy)}"

    base_url="https://${SUBDOMAIN:+${SUBDOMAIN}.}${TOPDOMAIN}"
    hugo_env="${HUGO_ENV:-development}"
    msg "Building Hugo static site (env: $hugo_env, baseURL: $base_url)..."
    if command -v hugo >/dev/null 2>&1; then
      # HUGO_PARAMS_SUBDOMAIN / HUGO_PARAMS_TOPDOMAIN feed
      # .Site.Params.subdomain and .topdomain at build time so the
      # marketing site can compose absolute members.<sub>.<top> URLs.
      (cd "$ROOT_DIR/frontend" && \
        HUGO_PARAMS_SUBDOMAIN="${SUBDOMAIN}" \
        HUGO_PARAMS_TOPDOMAIN="${TOPDOMAIN}" \
        hugo build --minify --cleanDestinationDir --environment "$hugo_env" -b "$base_url")
    else
      err "Hugo not found. Install Hugo first: https://gohugo.io/installation/"
      exit 1
    fi

    msg "Building Docker image $local_tag"
    # Forward the Hugo env/baseURL into the Docker build so the
    # frontend-builder stage's in-container Hugo run matches what we
    # built on the host above. Without these flags Hugo inside Docker
    # defaults to production and overwrites the dev-built docs/.
    # SUBDOMAIN/TOPDOMAIN go through too so the Hugo build can compose
    # https://members.<SUBDOMAIN>.<TOPDOMAIN>/... links via os.Getenv.
    docker build \
      --build-arg HUGO_ENVIRONMENT="$hugo_env" \
      --build-arg HUGO_BASE_URL="$base_url" \
      --build-arg SUBDOMAIN="${SUBDOMAIN}" \
      --build-arg TOPDOMAIN="${TOPDOMAIN}" \
      -t "$local_tag" -f "$ROOT_DIR/infra/app/Dockerfile" "$ROOT_DIR" ;;

  rebuild)
    "$0" app down || true
    "$0" build
    "$0" app up ;;

  proxy)
    sub="${1:-}" ; shift || true
    case "$sub" in
      up)   compose_proxy up -d ;;
      down) compose_proxy down || true ;;
      *)    err "Usage: proxy up|down"; exit 1 ;;
    esac ;;

  app)
    sub="${1:-up}"; shift || true
    # Parse flags
    USE_DEV_PROFILE=""
    new_args=()
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --dev) USE_DEV_PROFILE="true"; shift ;;
        *) new_args+=("$1"); shift ;;
      esac
    done
    set -- "${new_args[@]+"${new_args[@]}"}"

    case "$sub" in
      up)
        compose_proxy up -d
        if [[ -n "$USE_DEV_PROFILE" ]]; then
          compose_app --profile dev up -d "$@"
        else
          compose_app up -d "$@"
        fi ;;
      down)
        compose_app down || true
        compose_proxy down || true ;;
      ps)
        compose_app ps ;;
      logs)
        svc="${1:-otm-app}"; shift || true
        compose_app logs -f "$svc" ;;
      *)
        err "Unknown: app $sub"; exit 1 ;;
    esac ;;

  *)
    err "Unknown command: $cmd"; usage; exit 1 ;;
esac
