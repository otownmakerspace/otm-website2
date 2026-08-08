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

Both deploy workflows declare `environment: <Staging|Production>`. Variables
and secrets must therefore be set under **Settings → Environments → \<env\>**
(*not* repo-wide), so Staging and Production keep distinct credentials, DNS,
and Stripe modes despite running identical workflow code.

The two environments need the same *keys*; the *values* differ. Create both
environments now even if you're only deploying Staging today — it stops
later production pushes from failing on missing secrets.

### Setup order

Don't fill the environments first. The values depend on infrastructure that
doesn't exist yet:

1. **Provision the server** — `./infra/provisioning/scripts/setup.sh` (see
   [infra/provisioning/README.md](infra/provisioning/README.md)). Output
   gives you the IP, and the `otmadmin` SSH key pair you'll feed back in.
2. **Generate a Tailscale OAuth client** at Tailscale Admin → Settings →
   OAuth clients (scope `auth_keys` write, tag `tag:ci`). This is *separate*
   from the auth key the playbook used to register the server; the OAuth
   client lets each CI run mint a one-shot ephemeral key for the runner.
3. **DNS**: point `<SUBDOMAIN>.<TOPDOMAIN>` at the server's public IPv4
   (and IPv6 if you have one). Traefik handles cert provisioning on the
   first request.
4. **Register Stripe webhooks** at `https://<host>/webhook` for *each*
   environment. Each endpoint emits its own `whsec_…` — staging and
   production cannot share one.
5. **Populate the GitHub Environment** with the variables and secrets below.
6. **Push to `staging`** to trigger the first deploy. Workflow logs name
   any missing secret/var on the spot.

### Variables — 11 per environment

`Settings → Environments → <env> → Add variable`. Plain strings, visible
in the GitHub UI.

| Variable                  | Staging example          | Production example     | Notes |
|---------------------------|--------------------------|------------------------|-------|
| `IMAGE`                   | `otm-website:latest`     | `otm-website:latest`   | Local Docker tag on the VPS; the GHCR image is retagged to this before `docker compose up`. |
| `SUBDOMAIN`               | `staging`                | `www`                  | Combined with `TOPDOMAIN` to form the full host. Used by Hugo `--baseURL` and Traefik routing. |
| `TOPDOMAIN`               | `makerspace.olaru.dk`    | `makerspace.olaru.dk`  | Apex domain (no leading dot). |
| `STRIPE_PRICE_ID`         | `price_…` (test mode)    | `price_…` (live mode)  | Stripe Dashboard → Products → choose product → copy price ID. |
| `SESSION_COOKIE_NAME`     | `otm-session-staging`    | `otm-session`          | Distinct names stop staging/production cookies colliding if you ever browse both. |
| `EMAIL_HOST`              | `smtppro.zoho.com`       | `smtppro.zoho.com`     | Provider's SMTP host. `smtppro.zoho.com` for Zoho Mail Business, `smtp.zoho.com` for free/personal, `smtp.zoho.eu` for EU-datacenter accounts. |
| `EMAIL_PORT`              | `465`                    | `465`                  | `465` for implicit-SSL, `587` for STARTTLS. Match what your provider documents. |
| `BRAND_NAME`              | `O'Town Makerspace`      | `O'Town Makerspace`    | Public-facing brand displayed in transactional emails (footer, body copy, subject lines like *"Welcome to {brand}!"*). Optional — backend defaults to `O'Town Makerspace`. Set to override at rebrand. |
| `BRAND_WORDMARK_LEADING`  | `O'TOWN`                 | `O'TOWN`               | Dark/primary part of the two-tone email wordmark. Optional — defaults to `O'TOWN`. |
| `BRAND_WORDMARK_ACCENT`   | `MAKERSPACE`             | `MAKERSPACE`           | Orange-coloured second part of the wordmark. Optional — defaults to `MAKERSPACE`. |
| `BRAND_LOGO_URL`          | `https://makerspace.olaru.dk/images/branding/logos/wordmark-consolidated.svg` | `https://makerspace.olaru.dk/images/branding/logos/wordmark-consolidated.svg` | Absolute URL to a horizontal wordmark image rendered at the top of every email (PNG or SVG). The site's marketing host serves this from `frontend/branding/logos/`. Empty disables the image — clients fall back to the text wordmark above. Optional — defaults to the production URL. |

### Secrets — 13 per environment

`Settings → Environments → <env> → Add secret`. Write-only, masked in logs.

| Secret                              | Where to get the value                                                                                              | Notes |
|-------------------------------------|---------------------------------------------------------------------------------------------------------------------|-------|
| `GHCR_TOKEN`                        | GitHub Settings → Developer settings → PATs (classic), scopes `write:packages` + `read:packages`.                   | One token can serve both environments. |
| `DEPLOYMENT_REMOTE_HOST`            | Tailscale MagicDNS name (`otm-staging.tailXXXX.ts.net`) or `100.x.y.z` IPv4 from `tailscale ip -4` on the server.   | Tailnet-internal; not the public IP. |
| `DEPLOYMENT_REMOTE_USERNAME`        | The Linux user Terraform created — `otmadmin` (must have passwordless `sudo`, set up by the Ansible `base` role).   | Same in both envs unless you customised `user_name`. |
| `DEPLOYMENT_REMOTE_SSH_PRIVATE_KEY` | Contents of the private key referenced by `ssh_private_key_path` in `terraform.tfvars` (full PEM, header→footer).   | The public half is already in `~otmadmin/.ssh/authorized_keys` on the server. |
| `TAILSCALE_OAUTH_CLIENT_ID`         | Tailscale Admin → Settings → OAuth clients → Generate.                                                              | *Not* the same as the auth key in `group_vars/all.yml`. |
| `TAILSCALE_OAUTH_CLIENT_SECRET`     | Same screen — shown once at creation; copy immediately.                                                             | Lose it = revoke and regenerate. |
| `SSL_CERTIFICATE_EMAIL`             | Any address you receive Let's Encrypt expiry notices on.                                                            | Only used by the reverse-proxy workflow. |
| `STRIPE_KEY`                        | Stripe Dashboard → Developers → API keys → Secret key. **Test mode** for Staging (`sk_test_…`), live for Production (`sk_live_…`). | Toggle the test/live switch in Stripe before copying. |
| `STRIPE_WEBHOOK_SECRET`             | Stripe Dashboard → Developers → Webhooks → your endpoint → Signing secret (`whsec_…`).                              | **Per-endpoint** — each environment needs its own registered webhook. |
| `COOKIE_STORE_KEY`                  | Generate fresh: `openssl rand -hex 32`.                                                                              | Use a *different* value in each environment. |
| `EMAIL_ADDRESS`                     | The mailbox you authenticate to SMTP with — full email address (e.g. `info@theotowngarage.com`).                    | SMTP username *and* the default `From:` header on outbound mail. Deploy writes it to `/opt/secrets/app/email_address` (bind-mounted into the container at `/run/secrets/email_address`). Pairs with `EMAIL_PASSWORD`. |
| `EMAIL_PASSWORD`                    | SMTP password from your transactional-mail provider (Zoho).                                                         | One mailbox can serve both envs; consider separate accounts to scope blast radius. |
| `ALTCHA_HMAC_KEY`                   | Generate fresh: `openssl rand -hex 32`.                                                                              | HMAC key for the Altcha proof-of-work captcha on the signup + reset forms. **Different value per environment**; rotate to invalidate outstanding challenges. Leaving it empty disables captcha verification — the backend logs `captcha: DISABLED` at startup and accepts any submission. |

### Reverse-proxy workflow (manual)

`deploy-<env>-reverseproxy.yml` runs only on `workflow_dispatch`. It uses a
subset of the secrets above:
`DEPLOYMENT_REMOTE_*`, `TAILSCALE_OAUTH_*`, `SSL_CERTIFICATE_EMAIL`. Run it
once after server provisioning to bring up Traefik + the `app_network`
Docker network, then leave it alone — the app workflow doesn't touch the
proxy.

### How secrets reach the container

Secrets never live in the repo or the image. The deploy step reads them from
the runner's env, SSHes to the VPS, and writes each to `/opt/secrets/app/<file>`
via heredoc with `chmod 600` + `chown root:root`. The container bind-mounts
that directory read-only at `/run/secrets/`, where the Go server reads them
at startup.

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
