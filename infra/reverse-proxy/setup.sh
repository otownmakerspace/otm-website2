#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   setup.sh --network-name <name> --acme-dir <path> --username <user> [--base-dir <path>]

NETWORK_NAME=""
ACME_DIR=""
USERNAME=""
BASE_DIR=""

print_help() {
  cat <<EOF
Usage: $0 --network-name <name> --acme-dir <path> --username <user> [--base-dir <path>]

Required arguments:
  --network-name    Docker network name to create/use for Traefik
  --acme-dir        Directory for ACME storage (will create and chmod 600 acme.json)
  --username        Linux user on remote host; used for ownership and default base dir

Optional arguments:
  --base-dir        Base directory for reverse-proxy files (default: /home/<username>/infra/reverse-proxy)
  -h, --help        Show this help and exit
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --network-name)   NETWORK_NAME="$2"; shift 2;;
    --network-name=*) NETWORK_NAME="${1#*=}"; shift;;
    --acme-dir)       ACME_DIR="$2"; shift 2;;
    --acme-dir=*)     ACME_DIR="${1#*=}"; shift;;
    --username)       USERNAME="$2"; shift 2;;
    --username=*)     USERNAME="${1#*=}"; shift;;
    --base-dir)       BASE_DIR="$2"; shift 2;;
    --base-dir=*)     BASE_DIR="${1#*=}"; shift;;
    -h|--help)        print_help; exit 0;;
    --)               shift; break;;
    *)                echo "Unknown option: $1" >&2; print_help; exit 2;;
  esac
done

if [[ -z "$NETWORK_NAME" || -z "$ACME_DIR" || -z "$USERNAME" ]]; then
  echo "Missing required arguments." >&2
  print_help
  exit 2
fi

if [[ -z "$BASE_DIR" ]]; then
  BASE_DIR="/home/${USERNAME}/infra/reverse-proxy"
fi

if ! command -v docker >/dev/null 2>&1; then
  echo "Docker CLI not found in PATH" >&2
  exit 1
fi

# Create network if it does not exist (idempotent)
if ! docker network inspect "$NETWORK_NAME" >/dev/null 2>&1; then
  docker network create "$NETWORK_NAME"
fi

# Prepare ACME storage (idempotent)
sudo mkdir -p "$ACME_DIR"
sudo touch "$ACME_DIR/acme.json"
sudo chmod 600 "$ACME_DIR/acme.json"

# Ensure base directory exists and is writable by the target user
sudo mkdir -p "$BASE_DIR"
sudo chown -R "${USERNAME}":"${USERNAME}" "$BASE_DIR" || true
chmod -R u+rwX "$BASE_DIR"

echo "Setup complete:
  NETWORK_NAME = $NETWORK_NAME
  ACME_DIR     = $ACME_DIR
  USERNAME     = $USERNAME
  BASE_DIR     = $BASE_DIR" >&2
