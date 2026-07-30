---
title: Deployment
nav_order: 6
has_children: false
---

# Deployment

For initial setup see [Getting Started](getting-started.html). This page covers production-specific concerns.

## How the Docker Setup Works

Mycorrhizal CRM runs as a single all-in-one container. Inside it, nginx serves the React SPA and proxies all `/api/`, `/carddav/`, and `/.well-known/carddav` requests to the Go backend on `127.0.0.1:8080`; the backend is never exposed to the host directly. Only the nginx port is published (default `7300`). This same-origin proxy is built in, so nothing extra is required.

Rate limiters (auth, API, CardDAV, account lockout) are in-memory and per-process; they reset on restart and are not shared across replicas if you run more than one backend instance.

You only need an external reverse proxy for TLS termination. Point it at the published port (default `7300`):

```nginx
server {
    listen 443 ssl;
    server_name mycorrhizal.example.com;

    location / {
        proxy_pass http://localhost:7300;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto https;
    }
}
```

## Production Environment

Set these variables in `.env` when running over HTTPS:

| Variable | Value |
|---|---|
| `FRONTEND_URL` | Exact origin, e.g. `https://mycorrhizal.example.com` (never `*`) |
| `COOKIE_SECURE` | `true` |
| `COOKIE_DOMAIN` | Your domain |
| `JWT_SECRET_KEY` | Generate with `openssl rand -base64 32` |


## Single Sign-On (OIDC)

Mycorrhizal CRM supports SSO via any OpenID Connect provider (Keycloak, Google, Authentik, Authelia, etc.). When enabled, a **Sign in with provider** button appears on the login page.

### Setup

1. Register a new OAuth2 client with your provider. Set the redirect URI to:
   ```
   https://mycorrhizal.example.com/api/v1/auth/oidc/callback
   ```
   This is derived automatically from `FRONTEND_URL`, no separate variable needed. If your provider
   supports RP-Initiated Logout and requires a `post_logout_redirect_uri` to be pre-registered too, it's
   likewise derived from `FRONTEND_URL`:
   ```
   https://mycorrhizal.example.com/login
   ```

2.  Set the OIDC environment variables in the docker compose. See [Getting-Started → Environment variables](getting-started.html#environment-variables) for details. SSO is disabled if any of the first three variables are missing.

### Account linking

On first SSO login, the backend attempts to match the OIDC identity to an existing account in this order:

1. **Subject match** — if the user has logged in via this provider before, their account is found directly.
2. **Email match** — if the provider returns a *verified* email that matches an existing account, the OIDC identity is linked to that account automatically. Unverified emails are ignored to prevent account takeover (except if `OIDC_TRUST_EMAIL=true` is set).
3. **Auto-provision** — if `OIDC_AUTO_PROVISION=true` and no account matched, a new account is created using the email/name from the provider.

If auto-provisioning is disabled and no match is found, the user sees an error and must be registered manually first.

### Passwords

Accounts created through SSO have no password and can only log in via SSO. Existing password-based accounts that get linked retain their password.

## Upgrades

```sh
docker compose pull
docker compose up -d
```

Database migrations run automatically on startup.

## Backups

Back up the SQLite database file and photo directory:

```sh
cp /path/to/data/mycorrhizal.db /backups/mycorrhizal-$(date +%F).db
rsync -a /path/to/photos/ /backups/photos/
```

The database can be copied while the app is running (SQLite WAL mode).
