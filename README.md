# otm-website

The website for **O'Town Makerspace** at [makerspace.olaru.dk](https://makerspace.olaru.dk):
public marketing/info site, member sign-up flow, payment, and login — all in
one repository, three peer concerns:

```
otm-website/
├── frontend/    Hugo (content + layouts + theme; produces frontend/docs/)
├── api/         Go backend (Stripe checkout, webhook fulfilment, sessions, email)
└── infra/       Docker/Traefik runtime + Terraform/Ansible provisioning
```

Each peer has its own README:

- [frontend/](frontend/) — Hugo content site (uses the `hugo-up-business-main`
  Tailwind theme; bilingual EN+DA)
- [api/README.md](api/README.md) — Go service layout (`cmd/server/` +
  layered `internal/` packages: config, db, session, user, stripe, email, http)
- [infra/provisioning/README.md](infra/provisioning/README.md) — one-time
  server setup (Hetzner Terraform + Ansible roles for Docker / Tailscale /
  UFW / fail2ban / SSH hardening)

For local development jump straight to [Local dev](#local-dev) below.

## CI/CD

Branch-per-environment:

| Push to       | Workflow                                              | Effect |
|---------------|-------------------------------------------------------|--------|
| `master`      | [`main.yml`](.github/workflows/main.yml)              | Run tests (Go + Hugo). No deploy. |
| `staging`     | [`deploy-staging-app.yml`](.github/workflows/deploy-staging-app.yml) | test → build-hugo → publish image → deploy to **Staging** environment. |
| `production`  | [`deploy-production-app.yml`](.github/workflows/deploy-production-app.yml) | test → build-hugo → publish image → deploy to **Production** environment. |
| pull request  | [`backend_test.yml`](.github/workflows/backend_test.yml), [`hugo_test.yml`](.github/workflows/hugo_test.yml) | PR validation. |

Reverse-proxy deploys are manual (`workflow_dispatch`):

- [`deploy-staging-reverseproxy.yml`](.github/workflows/deploy-staging-reverseproxy.yml)
- [`deploy-production-reverseproxy.yml`](.github/workflows/deploy-production-reverseproxy.yml)

The legacy [`hugo.yml`](.github/workflows/hugo.yml) keeps publishing
`makerspace.olaru.dk` to GitHub Pages from `frontend/docs/` for now.
It gets retired during the production cutover (see
[Production cutover](#production-cutover) below).

### Promotion flow

```
master  → tests only
   ↓ (merge / fast-forward)
staging → builds Hugo with --environment staging, publishes image,
          deploys to Hetzner staging box at staging.makerspace.olaru.dk
   ↓ (merge / fast-forward, after staging is verified)
production → builds Hugo with --environment production, deploys to
             Hetzner production box (post-cutover)
```

Each deploy job declares `environment: <Env>` so secrets and variables
resolve from the corresponding GitHub Environment.

## Required GitHub configuration

### Repository variables (`Settings → Variables → Actions`)

These are non-secret strings. They can be set per environment to differ
between staging and production, or at the repo level to share.

| Variable           | Used by | Example | Notes |
|--------------------|---------|---------|-------|
| `IMAGE`            | publish + deploy | `otm-website:latest` | Local Docker tag on the VPS; the GHCR image is retagged to this before `docker compose up`. |
| `SUBDOMAIN`        | Hugo build + deploy | `staging` or `www` | Combined with TOPDOMAIN for full host. |
| `TOPDOMAIN`        | Hugo build + deploy | `makerspace.olaru.dk` | Apex domain (no leading dot). |
| `STRIPE_PRICE_ID`  | deploy | `price_1S…` | Test-mode in Staging, live-mode in Production. |
| `SESSION_COOKIE_NAME` | deploy | `otm-session` | Distinct per deployment domain so sessions don't bleed. |

### Production environment secrets

Both deploy workflows declare `environment: <Staging|Production>`, so the
secrets below must be set at **Settings → Environments → \<env\>**, not at
the repo level.

| Secret                                | Used by | What it is |
|---------------------------------------|---------|------------|
| `GHCR_TOKEN`                          | publish + deploy | GitHub PAT (classic) with `write:packages` (push) and `read:packages` (pull). |
| `DEPLOYMENT_REMOTE_HOST`              | deploys | Hetzner VPS Tailscale MagicDNS name or `100.x.y.z` IPv4. |
| `DEPLOYMENT_REMOTE_USERNAME`          | deploys | SSH login user on the VPS. Must have **passwordless `sudo`**. |
| `DEPLOYMENT_REMOTE_SSH_PRIVATE_KEY`   | deploys | OpenSSH private key (full PEM body). The public half is in the deploy user's `authorized_keys`. |
| `TAILSCALE_OAUTH_CLIENT_ID`           | deploys | Tailscale OAuth client ID (with `auth_keys` write scope, `tag:ci` permitted). |
| `TAILSCALE_OAUTH_CLIENT_SECRET`       | deploys | Companion secret for the OAuth client. |
| `SSL_CERTIFICATE_EMAIL`               | reverse-proxy deploys | Email Let's Encrypt registers ACME notifications under. |
| `STRIPE_KEY`                          | app deploys | Stripe secret API key (`sk_test_…` for staging, `sk_live_…` for prod). |
| `STRIPE_WEBHOOK_SECRET`               | app deploys | Stripe webhook signing secret (`whsec_…`). **Per-endpoint:** staging and production each need their own webhook registered in Stripe. |
| `COOKIE_STORE_KEY`                    | app deploys | 32 random bytes (e.g. `openssl rand -hex 32`) for HMAC signing of session cookies. |
| `EMAIL_PASSWORD`                      | app deploys | SMTP password for the Zoho account that sends transactional mail. |

These secrets are never written to the repo — the deploy step reads them from
the runner's env and writes each into `/opt/secrets/app/<file>` on the VPS via
SSH heredoc, then `chmod 600` + `chown root:root`. Inside the container they
appear at `/run/secrets/<file>` via bind mounts.

## Local dev

```sh
git clone …otm-website && cd otm-website
cd frontend && hugo --minify       # builds frontend/docs/
cd ..
./infra/dev.sh build               # Hugo + Docker image
./infra/dev.sh app up --dev        # app + reverse-proxy + stripe-cli
```

Visit `http://localhost/` for the Hugo site, `http://localhost/login/` for the
backend-served login page. The `--dev` flag also starts a Stripe CLI container
that forwards webhook events to the local backend; copy the `whsec_…` it
prints into `secrets/app/stripe_webhook_secret` and restart the app.

Local dev secrets live in `secrets/app/` (gitignored). On first `app up`,
`infra/dev.sh` creates the directory with `REPLACE_ME` placeholders; fill in
test-mode values for whatever services you want to exercise.

See [api/README.md](api/README.md) for backend-only dev (`go run ./cmd/server`
in the `api/` directory).

## Production cutover (deferred)

Currently `makerspace.olaru.dk` is served by the legacy
[`hugo.yml`](.github/workflows/hugo.yml) GitHub Pages workflow, and the
member-portal "Sign In" button on production points to the existing external
[`members.theotowngarage.com`](https://members.theotowngarage.com).

The new full-stack deployment runs on Hetzner via this repo's
branch-per-environment pipeline. The cutover is a deliberate, ordered series
of steps — not pushed yet:

1. **Provision the Hetzner staging box** via `infra/provisioning/`
   (see its [README](infra/provisioning/README.md)). Get `terraform output
   server_ip`.
2. **Configure GitHub Environments**: create `Staging` and `Production`
   under Settings → Environments. Populate the secrets and variables listed
   above. For **Staging**, register a Stripe test-mode webhook endpoint at
   `https://staging.makerspace.olaru.dk/webhook` and copy its new `whsec_…`
   into `STRIPE_WEBHOOK_SECRET`.
3. **DNS**: add an A/AAAA record for `staging.makerspace.olaru.dk` pointing
   at the Hetzner staging IP. Production keeps its existing record (GitHub
   Pages) for now.
4. **First reverse-proxy deploy**: manually run the
   `deploy-staging-reverseproxy.yml` workflow (it brings Traefik up and
   creates the `app_network` external Docker network).
5. **First app deploy**: push `master` → `staging` (`git push origin master:staging`).
   This fires `deploy-staging-app.yml`. Verify
   `https://staging.makerspace.olaru.dk/`, `/login/`, signup with Stripe
   test card `4242 4242 4242 4242`, webhook hits `/webhook`, DB row created.
6. **Test for a few days.** Hammer the staging environment: signups,
   logins, password resets, subscription cancel.
7. **Provision the production Hetzner box.** Same playbook, larger
   server type, different Tailscale tag (`tag:otm-production`).
8. **Production cutover** (the irreversible step):
   - Register the production Stripe webhook at
     `https://makerspace.olaru.dk/webhook` and store its `whsec_` in the
     Production environment's `STRIPE_WEBHOOK_SECRET`.
   - **Delete** `frontend/static/CNAME` and merge — wait for GitHub Pages
     to release the custom domain claim (usually a few minutes).
   - Update DNS: A/AAAA for `makerspace.olaru.dk` → Hetzner production IP.
   - Push `staging` → `production` to fire the production deploy.
   - Wait for Let's Encrypt to issue a fresh cert via Traefik.
   - Verify production end-to-end.
   - Delete [`hugo.yml`](.github/workflows/hugo.yml) and the legacy GH
     Pages site config.
   - Switch `frontend/config/production/params.toml` `memberPortalUrl`
     from `https://members.theotowngarage.com` to `/login/` and redeploy.
9. **Retire `members.theotowngarage.com`** once production has handled
   real traffic for a period the team is comfortable with.

## Rollback

| Scenario | Recovery |
|----------|---------|
| Staging app crashes | `ssh staging 'docker compose -f infra/app/docker-compose.yml down'` (503 is fine for staging). Revert offending commit on `staging` and push; workflow redeploys prior image. |
| Bad CI config | Revert the offending commit. Production GH Pages keeps working independently. |
| Production deploy regresses | `ssh prod 'docker tag ghcr.io/…@<previous-sha> otm-website:latest && docker compose -f infra/app/docker-compose.yml up -d'`. Then revert the breaking commit on `production`. |
| GH Pages broken after the Hugo move | Revert the relevant commit; `git mv` is reversible. |
| Total abandonment | Stop merging migration branch, `terraform destroy` the staging Hetzner box, remove the staging DNS record. otm's `master` branch and the existing `makerspace.olaru.dk` GH Pages site stay live and untouched. |

## Migration history

This repo absorbed the backend + infrastructure from
`theotowngarage-com/otg-website` in a single feature branch. Backend
history is preserved via `git subtree` under `api/`; deeper context for
each phase lives in the individual commit messages.

The migration plan that produced this layout lives at
`/home/iustinian/.claude/plans/fluttering-leaping-storm.md` (local-only).
