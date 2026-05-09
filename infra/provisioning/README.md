# infra/provisioning/

One-time server provisioning for the otm-website Hetzner VPSes
(staging + production). Two tools, one pipeline:

- **Terraform** creates the Hetzner Cloud server and uploads the SSH key. A
  minimal cloud-init creates the deploy user and installs `python3` so
  Ansible can talk to the box.
- **Ansible** connects over SSH and configures the server: SSH hardening,
  unattended-upgrades, Docker, Tailscale, UFW, fail2ban, and a GitHub-runner
  deploy keypair.

Once provisioning completes, ongoing app deploys come from the GitHub
Actions workflows in `.github/workflows/deploy-{staging,production}-app.yml`,
not from this directory.

## Layout

```
infra/provisioning/
├── terraform/
│   ├── main.tf, variables.tf, outputs.tf, versions.tf
│   ├── .terraform.lock.hcl                # provider version pin (tracked)
│   ├── terraform.tfvars.example           # template — copy to terraform.tfvars
│   └── (terraform.tfvars, terraform.tfstate*, .terraform/ — gitignored)
├── ansible/
│   ├── ansible.cfg, playbook.yml
│   ├── inventory.ini.example              # template — copy to inventory.ini
│   ├── group_vars/all.yml.example         # template — copy to all.yml
│   └── roles/{base,docker,tailscale,firewall,fail2ban,github_runner_key}/
└── scripts/
    └── setup.sh                           # orchestration: terraform apply + cloud-init wait + ansible-playbook
```

`scripts/docker-build.sh` from the upstream terraform-provisioning project was
intentionally not copied — `infra/dev.sh build` (parent directory) is the
canonical app-stack build script for otm-website and is already path-aware
for this repo's three-peer layout.

## Prerequisites

### Tooling

```bash
# Terraform (HashiCorp apt repo)
wget -O- https://apt.releases.hashicorp.com/gpg | sudo gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" \
  | sudo tee /etc/apt/sources.list.d/hashicorp.list
sudo apt update && sudo apt install terraform

# Ansible
sudo apt install ansible
ansible-galaxy collection install community.general community.crypto ansible.posix
```

### Credentials

Gather these once before provisioning:

| Credential | Where |
|---|---|
| Hetzner API token (read+write) | Hetzner Cloud Console → Security → API Tokens → Generate API Token |
| Tailscale auth key (reusable, with `tag:otm-staging` or `tag:otm-production`) | Tailscale Admin Console → Settings → Keys → Generate auth key |
| SSH keypair for the deploy user | `ssh-keygen -t ed25519 -f ~/.ssh/id_to_otm` |
| Linux user password hash | `openssl passwd -6` |

## Quick start (staging)

```bash
cd infra/provisioning

# 1. Fill in templates with your real values (these are gitignored — never commit them)
cp terraform/terraform.tfvars.example terraform/terraform.tfvars
cp ansible/group_vars/all.yml.example ansible/group_vars/all.yml

# 2. Edit terraform/terraform.tfvars: paste the Hetzner token, point at the SSH key
#    you just generated, set user_name + user_password_hash.

# 3. Edit ansible/group_vars/all.yml: ansible_user/user_name (must match Terraform's
#    user_name), user_email, server_hostname (e.g. otm-staging), UFW rules.

# 4. Encrypt the Tailscale auth key with Ansible Vault and paste into all.yml:
ansible-vault encrypt_string 'tskey-auth-YOUR-KEY-HERE' --name tailscale_auth_key

# 5. Run end-to-end provisioning (creates the server + waits for cloud-init + runs all roles)
chmod +x scripts/setup.sh
bash scripts/setup.sh
```

After completion: the staging Hetzner box is up, accessible only via
Tailscale (UFW allows SSH on the tailscale0 interface only). Note the
`server_ip` from `terraform output` and add an A-record for
`staging.makerspace.olaru.dk` pointing at it.

Next step is the GitHub Actions deploy:

```bash
# Run the reverse-proxy deploy once manually so Traefik is up:
gh workflow run deploy-staging-reverseproxy.yml

# Then push to the staging branch — deploy-staging-app.yml fires automatically:
git push origin main:staging
```

## Production

Same playbook with different inputs:
- New `terraform.tfvars` values (different `server_name`, possibly larger
  `server_type`).
- New `inventory.ini` group `[makerspace_production]`.
- `ansible-playbook -i inventory.ini playbook.yml --limit makerspace_production --ask-vault-pass`.
- Different Tailscale tag (`tag:otm-production`) so ACLs can scope CI access
  per environment.

## Re-running individual roles

After the initial setup, you can re-run any subset of roles:

```bash
cd infra/provisioning/ansible
# Available tags: base, docker, tailscale, firewall, fail2ban, github_runner_key
ansible-playbook -i inventory.ini playbook.yml --ask-vault-pass --tags docker
```

## State backend

By default Terraform keeps state in `terraform.tfstate` next to the
configuration — fine for a single operator. For team use or CI-driven applies,
switch to a remote backend (S3+DynamoDB or Terraform Cloud) by uncommenting
the `backend "s3"` block in `terraform/versions.tf` and running
`terraform init -migrate-state`. The example key is `otm-website/terraform.tfstate`.

## Destroying infrastructure

```bash
cd infra/provisioning/terraform
terraform destroy   # tears down the Hetzner server + SSH key
```

This does **not** remove DNS records or release the Tailscale machine — do
those manually via the Hetzner DNS console and Tailscale Admin Console.

## What the playbook installs

| Role | Installs / configures |
|---|---|
| `base` | SSH hardening (`PasswordAuthentication no`, key-only); unattended-upgrades; system packages |
| `docker` | Docker Engine + Compose v2 (apt install from Docker's official repo) |
| `tailscale` | Tailscale daemon, joins tailnet via the vault-encrypted auth key |
| `firewall` | UFW: defaults deny-incoming/allow-outgoing; only the ports in `ufw_allowed_ports` (public) and `ufw_tailscale_ports` (tailscale0-only) |
| `fail2ban` | sshd jail, traefik-404 jail, ufw-blocked jail, recidive |
| `github_runner_key` | Generates an SSH keypair for CI/CD; the public key goes into the deploy user's `authorized_keys`, the private key is read into the GitHub `DEPLOYMENT_REMOTE_SSH_PRIVATE_KEY` secret |

## Footguns

- **`terraform.tfvars` is plaintext.** The Hetzner token is in there. Make sure
  it's gitignored (it is, by `.gitignore` rules at the project root) and that
  your home directory has FDE.
- **`terraform.tfstate` contains everything Terraform created** (server IPs,
  resource IDs, sometimes secrets). Use a remote backend for team use.
- **`ansible/group_vars/all.yml`** holds the Ansible Vault-encrypted Tailscale
  key. Vault encryption is solid, but the file is still gitignored
  defensively — committing it is harmless if vaulted, harmful if you
  accidentally save plaintext.
- **DNS comes last.** Don't point the staging A-record at the new server until
  Traefik is running, otherwise Let's Encrypt's HTTP-01/TLS challenge fails
  and you wait an hour for ratelimits to reset.
