#!/usr/bin/env bash
# SessionStart hook: fingerprint each area of the working tree, so the Stop hook
# can tell what THIS session changed and stay silent on read-only sessions.
set -uo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

ROOT=$(project_root)
STATE=$(state_dir)
session=$(cat 2>/dev/null | jq -r '.session_id // "unknown"' 2>/dev/null || echo unknown)

cd "$ROOT" 2>/dev/null || exit 0
git rev-parse --git-dir >/dev/null 2>&1 || exit 0

{
  for area in backend frontend; do
    printf '%s %s\n' "$area" "$(area_fingerprint "$area")"
  done
} > "$STATE/fingerprint-$session" 2>/dev/null
rm -f "$STATE/attempts-$session" 2>/dev/null
exit 0
