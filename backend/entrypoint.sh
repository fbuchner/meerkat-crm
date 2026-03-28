#!/bin/sh
set -e

PUID="${PUID:-1001}"
PGID="${PGID:-1001}"

if [ "$(id -g appuser)" != "$PGID" ]; then
    groupmod -o -g "$PGID" appgroup
fi

if [ "$(id -u appuser)" != "$PUID" ]; then
    usermod -o -u "$PUID" appuser
fi

chown -R appuser:appgroup /app/data /app/static/photos

exec su-exec appuser:appgroup "$@"
