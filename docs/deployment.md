---
title: Deployment
nav_order: 6
has_children: false
---

# Deployment

For initial setup see [Getting Started](getting-started.md). This page covers production-specific concerns.

## How the Docker Setup Works

The frontend container runs nginx, which both serves the React SPA and proxies all `/api/`, `/carddav/`, and `/.well-known/carddav` requests to the backend container. The backend is not port-mapped to the host — it is only reachable within the Docker network. Leave `REACT_APP_API_URL` empty in `.env.docker` to use this built-in proxy.

You only need an external reverse proxy for TLS termination. Point it at the frontend container port (default `7300`):

```nginx
server {
    listen 443 ssl;
    server_name meerkat.example.com;

    location / {
        proxy_pass http://localhost:7300;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto https;
    }
}
```

## Production Environment

Set these variables in `.env.docker` when running over HTTPS:

| Variable | Value |
|---|---|
| `FRONTEND_URL` | Exact origin, e.g. `https://meerkat.example.com` (never `*`) |
| `COOKIE_SECURE` | `true` |
| `COOKIE_DOMAIN` | Your domain |
| `JWT_SECRET_KEY` | Generate with `openssl rand -base64 32` |


## Single Sign-On (OIDC)

Meerkat CRM supports SSO via any OpenID Connect provider (Keycloak, Google, Authentik, Authelia, etc.). When enabled, a **Sign in with provider** button appears on the login page.

### Setup

1. Register a new OAuth2 client with your provider. Set the redirect URI to:
   ```
   https://meerkat.example.com/api/v1/auth/oidc/callback
   ```
   This is derived automatically from `FRONTEND_URL`, no separate variable needed.

2.  Set the OIDC environment variables in the docker compose. See [Getting-Started → Environment variables](getting-started.md#environment-variables) for details. SSO is disabled if any of the first three variables are missing.

### Account linking

On first SSO login, the backend attempts to match the OIDC identity to an existing account in this order:

1. **Subject match** — if the user has logged in via this provider before, their account is found directly.
2. **Email match** — if the provider returns a *verified* email that matches an existing account, the OIDC identity is linked to that account automatically. Unverified emails are ignored to prevent account takeover.
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
cp /path/to/data/meerkat.db /backups/meerkat-$(date +%F).db
rsync -a /path/to/photos/ /backups/photos/
```

The database can be copied while the app is running (SQLite WAL mode).
