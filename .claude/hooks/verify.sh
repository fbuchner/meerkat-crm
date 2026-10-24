#!/usr/bin/env bash
# Stop hook: before Claude can finish, run the checks for whatever this session
# actually changed. Blocks (exit 2) with the failure output so Claude must fix it.
# Gives up after MAX_ATTEMPTS blocks so a stubborn failure cannot trap the session.
set -uo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

MAX_ATTEMPTS=3
ROOT=$(project_root)
STATE=$(state_dir)

input=$(cat 2>/dev/null)
session=$(printf '%s' "$input" | jq -r '.session_id // "unknown"' 2>/dev/null || echo unknown)
fingerprints="$STATE/fingerprint-$session"
attempts_file="$STATE/attempts-$session"

cd "$ROOT" 2>/dev/null || exit 0
git rev-parse --git-dir >/dev/null 2>&1 || exit 0

attempts=$(cat "$attempts_file" 2>/dev/null || echo 0)
if [ "$attempts" -ge "$MAX_ATTEMPTS" ]; then
  echo '{"systemMessage":"Verification hook gave up after 3 attempts - run the checks by hand."}'
  exit 0
fi

changed() {
  local area="$1" old new
  new=$(area_fingerprint "$area")
  if [ -f "$fingerprints" ]; then
    old=$(grep "^$area " "$fingerprints" 2>/dev/null | cut -d' ' -f2)
  else
    old=""
  fi
  # No snapshot (hook added mid-session): fall back to "is the area dirty at all".
  if [ -z "$old" ]; then
    [ -n "$(git status --porcelain -- "$area" 2>/dev/null)" ]
  else
    [ "$old" != "$new" ]
  fi
}

failures=()

run_check() {
  local label="$1"; shift
  local out
  if ! out=$("$@" 2>&1); then
    failures+=("--- $label ---"$'\n'"$(printf '%s\n' "$out" | tail -40)")
  fi
}

if changed backend; then
  run_check "go vet ./... (backend)" env -C "$ROOT/backend" go vet ./...
  run_check "go test ./... (backend)" env -C "$ROOT/backend" go test ./...
fi

if changed frontend; then
  if [ -d "$ROOT/frontend/node_modules" ]; then
    run_check "yarn tsc --noEmit (frontend)" env -C "$ROOT/frontend" yarn --silent tsc --noEmit
    run_check "yarn test (frontend)" env -C "$ROOT/frontend" env CI=true yarn --silent test --run
  else
    failures+=("frontend/node_modules is missing - run 'yarn install' in frontend/ before relying on these checks")
  fi
  parity=$(locale_parity_report "$ROOT/frontend/src/i18n/locales")
  [ -n "$parity" ] && failures+=("--- i18n locale parity ---"$'\n'"$parity")
fi

if [ ${#failures[@]} -gt 0 ]; then
  echo $((attempts + 1)) > "$attempts_file"
  {
    echo "Verification failed for the changes in this session. Fix these, then finish:"
    echo
    printf '%s\n\n' "${failures[@]}"
    echo "(Definition of done is in CLAUDE.md. This check ran because backend/ or frontend/ changed.)"
  } >&2
  exit 2
fi

rm -f "$attempts_file" 2>/dev/null
exit 0
