---
title: Getting Started
nav_order: 2
---

# Getting Started

## Installation

### Docker compose

The official container image is published in the github registry: `ghcr.io/fbuchner/meerkat-crm-backend` and `ghcr.io/fbuchner/meerkat-crm-frontend`
Images are available for linux/amd64, other architectures can be added on demand.

You can use either the `:latest` tag or a specific version (e.g. `:0.9` or `:0.9.1`).

Copy the [sample docker compose file](https://github.com/fbuchner/meerkat-crm/blob/main/docker-compose.yml) as well as [sample env file](https://github.com/fbuchner/meerkat-crm/blob/main/.env.docker.example) and rename the env file to `.env.docker`.

After adjusting the environment variables as needed you can run:
```docker compose --env-file .env.docker up -d```

### Environment variables

| Variable | Description |
|---|---|
| `JWT_SECRET_KEY` | Random string used for JWT signing (minimum 32 characters) |
| `FRONTEND_URL` | Used for CORS headers. Wildcard (`*`) is allowed but not recommended for production use |
| `RESEND_API_KEY` | API key for [Resend](https://resend.com), used to send e-mail notifications. The generous free tier is more than enough for any personal setup |
| `RESEND_FROM_EMAIL` | Sender e-mail address for Resend, needs to be configured in Resend |
| `CARDDAV_ENABLED` | When set to `true` the application acts as a CardDAV server which allows contacts to be synced with your phone |
| `DATA_PATH` | Host directory where the database file should be stored |
| `PHOTOS_PATH` | Host directory where the contact photos should be stored |
| `JWT_EXPIRY_HOURS` | Token expiry, i.e. after how many hours you will need to sign into the application again. Default is 96 hours (4 days) |

Other variables are found in the [sample env file](https://github.com/fbuchner/meerkat-crm/blob/main/.env.docker.example).

The containers run as non-root containers. Make sure the user group 1001 has write access to the data and photo directories.
```
chown -R :1001 /path/to/data  /path/to/photos
chmod -R 775 /path/to/data  /path/to/photos
```

## Post-Installation Setup

When running Meerkat-CRM you can access the application under the specified port (default is `3000`). 
To get started you need to register a user. The first user will automatically receive administrator rights and therefore be able to access the admin panel in the settings menu.

## Backup

Make regular backups of your data by copying the database file in your data directory as well as the contents of the photo directory to a separate device.
