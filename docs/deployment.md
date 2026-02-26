---
title: Deployment
nav_order: 6
has_children: false
---

# Deployment

For initial setup see [Getting Started](getting-started.md). This page covers production-specific concerns.

## How the Docker Setup Works

The frontend container runs nginx, which both serves the React SPA and proxies all `/api/`, `/carddav/`, and `/.well-known/carddav` requests to the backend container. The backend is not port-mapped to the host — it is only reachable within the Docker network. Leave `REACT_APP_API_URL` empty in `.env.docker` to use this built-in proxy.

You only need an external reverse proxy for **TLS termination**. Point it at the frontend container port (default `3000`):

```nginx
server {
    listen 443 ssl;
    server_name meerkat.example.com;

    location / {
        proxy_pass http://localhost:3000;
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

## File Permissions

The backend container runs as group `1001`. Host directories for data and photos must be writable:

```sh
chown -R :1001 /path/to/data /path/to/photos
chmod -R 775 /path/to/data /path/to/photos
```

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
