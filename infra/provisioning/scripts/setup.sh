#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="${SCRIPT_DIR}/.."

echo "==> Initializing Terraform..."
cd "${REPO_ROOT}/terraform"
terraform init

echo "==> Applying Terraform (creates server + generates inventory)..."
terraform apply

echo "==> Waiting for cloud-init to finish (this can take several minutes)..."
PRIVATE_KEY=$(grep ansible_ssh_private_key_file "${REPO_ROOT}/ansible/inventory.ini" | head -1 | sed 's/.*ansible_ssh_private_key_file=//')
SERVER_IP=$(grep -oP '^\d+\.\d+\.\d+\.\d+' "${REPO_ROOT}/ansible/inventory.ini" | head -1)
REMOTE_USER=$(grep -oP 'ansible_user=\K\S+' "${REPO_ROOT}/ansible/inventory.ini" | head -1)

# Poll until cloud-init reports done (timeout after 10 minutes)
TRIES=0
MAX_TRIES=60
until ssh -o StrictHostKeyChecking=accept-new -o ConnectTimeout=5 \
    -i "${PRIVATE_KEY}" "${REMOTE_USER}@${SERVER_IP}" \
    "cloud-init status --wait" 2>/dev/null; do
  TRIES=$((TRIES + 1))
  if [ "$TRIES" -ge "$MAX_TRIES" ]; then
    echo "==> WARNING: Timed out waiting for cloud-init. Proceeding anyway..."
    break
  fi
  echo "==> Cloud-init not ready yet, retrying in 10s... ($TRIES/$MAX_TRIES)"
  sleep 10
done

echo "==> Running Ansible playbook..."
cd "${REPO_ROOT}/ansible"
ansible-playbook -i inventory.ini playbook.yml --ask-vault-pass

echo "==> Done! Server is provisioned."
