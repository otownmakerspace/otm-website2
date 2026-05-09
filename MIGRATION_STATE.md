# Migration state — paused for review

Snapshot taken right after the otg-website backend + infrastructure
migration into otm-website finished its 9 planned phases. Everything is
committed locally; **nothing has been pushed yet**. The legacy
GitHub Pages deployer (`hugo.yml`) remains live and untouched, so
production at `makerspace.olaru.dk` is unaffected.

Pick this file up when you come back to decide whether to push, open a
PR, run the staging cutover, or change direction.

---

## Where we are right now

- **Branch:** `feat/backend-infra-migration` (local-only)
- **HEAD:** `2711c91 docs: top-level README documenting three-peer architecture and cutover plan`
- **Working tree:** clean (no uncommitted changes)
- **Remote:** `origin → git@github.com:iustin94/makerspace.git`
- **Safety tag:** `pre-migration-master` points at the otm `master` HEAD
  before any migration work. `git reset --hard pre-migration-master` on
  master would undo *everything* if you abandon.

## Commits introduced by this migration (newest first)

The 8 logical commits + 1 subtree-merge commit + 35 pre-existing backend
commits surfaced through the subtree merge:

```
2711c91 docs: top-level README documenting three-peer architecture and cutover plan
cfa4560 ci: import branch-per-environment workflows from otg, adapted for new layout
2df5b7d frontend: add per-env config overlays + import backend-served Hugo pages
73a1592 infra(provisioning): import terraform/ansible/scripts from terraform-provisioning
6d88db4 infra: import otg docker stack adapted for three-peer layout
89641bc refactor(api): split monolithic backend into cmd/server + internal layered packages
33d822e Add 'api/' from commit 'fd407d74…'        ← subtree merge from otg/backend
954400a chore: relocate Hugo site under frontend/
+ 35 prior backend commits visible through 33d822e's parent
```

`git log master..HEAD` reproduces this list at any time.

## What's verified

- `cd api && go vet ./... && go build ./cmd/server` → clean (32 MB binary).
- `cd frontend && hugo --minify --environment staging` → emits
  `memberPortalUrl=/login/`. `--environment production` → emits
  `https://members.theotowngarage.com`.
- All seven member pages generated: `/login/`, `/checkout/`, `/dashboard`,
  `/request-reset/`, `/reset-password/`, `/reset-sent`, `/reset-success`.
- `infra/app/Dockerfile` passes `docker build --check`.
- `cd infra/provisioning/terraform && terraform init -backend=false && terraform validate`
  → "Success! The configuration is valid".
- `ansible-playbook --syntax-check` against `inventory.ini.example` → ok.
- Every workflow + action YAML file: `python3 -c 'import yaml; yaml.safe_load(open(f))'` → ok.

## Layout

```
otm-website/
├── frontend/                       Hugo (theme: hugo-up-business-main, EN+DA)
│   ├── config/_default/hugo.toml
│   ├── config/staging/params.toml      memberPortalUrl=/login/
│   └── config/production/params.toml   memberPortalUrl=https://members.theotowngarage.com
├── api/                            Go module: github.com/iustin94/makerspace/api
│   ├── cmd/server/main.go              ~50 lines: bootstrap only
│   └── internal/{config,db,session,user,stripe,email,http}/
├── infra/
│   ├── app/                            Dockerfile + docker-compose.yml + dev .env.example
│   ├── reverse-proxy/                  Traefik
│   ├── dev.sh                          Local stack manager (cd frontend, ${SUBDOMAIN}.${TOPDOMAIN})
│   └── provisioning/                   Terraform + Ansible for one-time Hetzner setup
├── .github/
│   ├── workflows/  main.yml, deploy-{staging,production}-{app,reverseproxy}.yml,
│   │              backend_test.yml, hugo_test.yml,
│   │              hugo.yml  ← legacy GH Pages deployer; stays live until cutover
│   └── actions/   test, deploy-app, deploy-reverse-proxy, publish, tailscale-prime
└── README.md                       Full architecture + cutover walkthrough
```

## What's left to do (Phase 8 of the plan — operator steps)

Everything below was deliberately not auto-run because each step touches
a real account, real DNS, or real money. The README's "Production
cutover" section has the same list with more detail.

1. **Decide whether to push the branch.**
   - `git push origin feat/backend-infra-migration` (open as a PR for review).
   - Or merge directly into `master` (otm's default branch).
2. **Provision the Hetzner staging box.** See
   [`infra/provisioning/README.md`](infra/provisioning/README.md):
   - `cd infra/provisioning && cp terraform/terraform.tfvars.example terraform/terraform.tfvars`
   - Fill in: Hetzner API token, SSH key paths, Linux user name +
     `openssl passwd -6` hash.
   - `cp ansible/group_vars/all.yml.example ansible/group_vars/all.yml`
   - Fill in: ansible_user, user_email, server_hostname, UFW ports, and
     vault-encrypt the Tailscale auth key
     (`ansible-vault encrypt_string 'tskey-…' --name tailscale_auth_key`).
   - `bash scripts/setup.sh` (terraform apply → wait for cloud-init →
     ansible-playbook).
3. **Configure the GitHub `Staging` environment.**
   `Settings → Environments → New environment "Staging"`. Add the
   secrets and variables listed in `README.md` → Required GitHub
   configuration. For Stripe specifically:
   - Register a **new test-mode webhook endpoint** in the Stripe
     Dashboard pointing at `https://staging.makerspace.olaru.dk/webhook`
   - Copy the `whsec_…` it generates into the Staging env's
     `STRIPE_WEBHOOK_SECRET`.
4. **DNS.** Add an A/AAAA record:
   `staging.makerspace.olaru.dk → <hetzner staging IP from `terraform output server_ip`>`.
5. **First reverse-proxy deploy** (manual, one-time):
   `gh workflow run deploy-staging-reverseproxy.yml`. Wait for green.
6. **First app deploy.** Push `master → staging`:
   `git push origin master:staging`. Watch
   `deploy-staging-app.yml` run test → build-hugo → publish → deploy.
7. **Verify staging end-to-end.**
   - `curl -I https://staging.makerspace.olaru.dk/` → 200, valid Let's
     Encrypt cert.
   - `curl -I https://staging.makerspace.olaru.dk/login/` → 200, login
     form.
   - Sign up via the form using Stripe test card
     `4242 4242 4242 4242`. Verify webhook fires (Stripe Dashboard →
     Events → most recent → 200) and DB row is created
     (`ssh staging 'docker exec otm-app sqlite3 /app/db/production.db
      "SELECT email, customer_id FROM user;"'`).
   - Test login → dashboard → cancel-subscription flows.
8. **Sit on staging for several days.** Watch logs (`ssh staging
   'docker logs otm-app --tail 100 -f'`) for any unexpected errors;
   exercise password reset; verify the welcome and goodbye emails
   actually arrive.
9. **Production cutover** (irreversible-ish — read README's "Production
   cutover" section before starting):
   - Provision the production Hetzner box with the same Terraform
     module but a different `server_name`/Tailscale tag.
   - Configure GitHub `Production` environment with prod values
     (`sk_live_…` Stripe key, prod webhook secret, prod cookie key).
   - Delete `frontend/static/CNAME` and merge to master so GH Pages
     releases the custom-domain claim.
   - Wait, then DNS-flip `makerspace.olaru.dk` → Hetzner production IP.
   - Push `staging → production` to fire `deploy-production-app.yml`.
   - Wait for Traefik to issue a fresh Let's Encrypt cert.
   - Verify production end-to-end.
   - Delete `.github/workflows/hugo.yml`.
   - Switch `frontend/config/production/params.toml`'s
     `memberPortalUrl` from `https://members.theotowngarage.com` to
     `/login/` and redeploy.

## How to resume — common entry points

```sh
# See where you are:
cd /home/iustinian/Projects/otm-website
git log --oneline master..HEAD       # the migration commits
git status                           # working tree status
cat MIGRATION_STATE.md               # this file

# Re-verify everything still builds (run from repo root):
(cd api && go build ./cmd/server)               # → 32 MB binary
(cd frontend && hugo --minify --environment staging)
(cd infra/provisioning/terraform && terraform validate)

# Push for code review / GH Actions visibility:
git push origin feat/backend-infra-migration

# Or, decide to abandon entirely:
git checkout master
git branch -D feat/backend-infra-migration
git tag -d pre-migration-master                  # optional cleanup
```

## Where the planning artifact lives

The full migration plan that produced this state file:
`/home/iustinian/.claude/plans/fluttering-leaping-storm.md`
(local-only, not in this repo). That file has the original phase
breakdown, the rejected alternatives, the footgun list, and the
rollback story for each phase.

## Open questions worth thinking about before pushing

1. **Default-branch rename.** otm uses `master`; otg uses `main`. The
   workflows accept either, but standardising on one before pushing
   would be cleaner.
2. **Branch protection rules** for `staging` and `production` (require
   PR review, prevent force-push) — recommend enabling before any
   non-trivial team forms.
3. **Stripe webhook idempotency.** The current `FulfillCheckout`
   handler doesn't dedupe events. Stripe occasionally re-delivers
   webhooks; a retry would call `db.AddUser` twice and fail the second
   time on the email-unique constraint. Acceptable for MVP, worth
   tightening before live.
4. **Hugo theme styling for member pages.** The login/checkout/dashboard
   pages use Tailwind utility classes that work but aren't visually
   polished to the same level as the rest of the site. A design pass
   before production cutover would smooth this.
