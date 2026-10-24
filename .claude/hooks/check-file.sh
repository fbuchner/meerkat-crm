#!/usr/bin/env bash
# PostToolUse hook: cheap per-file checks right after a file is written.
# Formats Go in place; reports i18n drift back to Claude as context.
set -uo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

ROOT=$(project_root)
input=$(cat)
file=$(printf '%s' "$input" | jq -r '.tool_response.filePath // .tool_input.file_path // empty' 2>/dev/null)
[ -n "$file" ] && [ -f "$file" ] || exit 0

notes=()

case "$file" in
  *.go)
    if command -v gofmt >/dev/null 2>&1; then
      before=$(shasum "$file" 2>/dev/null | cut -d' ' -f1)
      gofmt -w "$file" 2>/dev/null
      after=$(shasum "$file" 2>/dev/null | cut -d' ' -f1)
      [ "$before" != "$after" ] && notes+=("gofmt reformatted $(basename "$file") - re-read it before further edits")
    fi
    ;;
  */i18n/locales/*.json)
    if ! jq empty "$file" >/dev/null 2>&1; then
      printf 'Invalid JSON written to %s. Fix it before continuing.\n' "$file" >&2
      exit 2
    fi
    while IFS= read -r line; do
      [ -n "$line" ] && notes+=("$line")
    done < <(locale_parity_report "$ROOT/frontend/src/i18n/locales")
    ;;
esac

if [ ${#notes[@]} -gt 0 ]; then
  msg="i18n/format check: $(printf '%s | ' "${notes[@]}")"
  jq -n --arg m "$msg" \
    '{hookSpecificOutput:{hookEventName:"PostToolUse",additionalContext:$m}}'
fi
exit 0
