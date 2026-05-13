# api/ — Go backend

The makerspace.olaru.dk member-management service: Stripe checkout, webhook
fulfilment, sessions, password reset, transactional email.

## Layout

```
api/
├── cmd/server/main.go      # ~50 lines: load config, build deps, start http server
├── internal/
│   ├── config/             # TOML + env + /run/secrets resolution
│   ├── db/                 # libsql open, migrations, user repository, test seed
│   ├── session/            # cookie store, login/logout, password reset
│   ├── user/               # User type + form/session/Stripe-metadata helpers
│   ├── stripe/             # checkout, webhook fulfilment, subscriptions, cancel
│   ├── email/              # SMTP send + embedded HTML templates
│   └── http/               # ServeMux assembly
├── (templates colocated with their consuming package and embedded via
│    //go:embed: internal/email/templates/*.html, internal/stripe/templates/*.html)
├── go.mod  go.sum
└── README.md
```

`internal/` enforces import boundaries — no code outside `api/` can import these
packages, so they're free to be redesigned without breaking external users.

Packages are split by **domain** (stripe, session, user, email) rather than by
technical layer (handlers/services/repos). The dependency graph is acyclic:
`stripe`, `session`, `email` depend on `db`, `user`, `config`; nothing
imports back. `cmd/server` is the only assembly point.

## Configuration

Resolved at startup with this priority (highest wins):

1. Environment variable
2. Secret file at `/run/secrets/<name>`
3. TOML file at `BACKEND_CONFIG_PATH` (default `./config.toml`)
4. Built-in defaults

| Knob | Env var | Secret file | Notes |
|---|---|---|---|
| Bind address | `BACKEND_HOST` | — | default `localhost`. Container sets `0.0.0.0`. |
| Bind port | `BACKEND_PORT` | — | default `0`. Set explicitly in compose / dev. |
| Public URL | `BACKEND_PUBLIC_URL` | — | URL Stripe redirects to + email links use. |
| Hugo static dir | `BACKEND_STATIC_DIR` | — | default `./public`. Container `/app/public`. |
| DB file | `BACKEND_DB_PATH` | — | default `./local.db`. Container `/app/db/production.db`. |
| Cookie name | `SESSION_COOKIE_NAME` | — | default `otm-session`. Distinct per domain. |
| Cookie HMAC key | `COOKIE_STORE_KEY` | `cookie_store_key` | required (fatal if absent). 32 bytes recommended. |
| Stripe API key | `STRIPE_KEY` | `stripe_key` | `sk_test_...` or `sk_live_...` |
| Webhook secret | `STRIPE_WEBHOOK_SECRET` | `stripe_webhook_secret` | per Stripe webhook endpoint |
| Stripe price ID | `STRIPE_PRICE_ID` | — | `price_xxx`. Test/live IDs differ. |
| SMTP password | `EMAIL_PASSWORD` | `email_password` | for `User`/Host/Port from TOML |
| Captcha HMAC key | `ALTCHA_HMAC_KEY` | `altcha_hmac_key` | random ≥32 byte secret. Empty disables Altcha verification (and logs `captcha: DISABLED` at startup). Per-environment value. |
| Production flag | `SITE_RELEASE` | — | presence (any value) ⇒ release; absence ⇒ test, runs `db.SeedTest` |

## Local dev

```sh
cd api
go run ./cmd/server   # honours env vars; defaults assume cwd has config.toml + ./public
```

For the full Hugo+backend stack with Traefik + stripe-cli, use `infra/dev.sh`
from the repo root — it sets every env var consistently.

## Build

```sh
go build -o server ./cmd/server   # 32MB statically-linked binary (libsql is CGO)
```

The Dockerfile in `infra/app/` runs the same build inside a multi-stage image.
