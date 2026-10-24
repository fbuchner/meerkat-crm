#!/usr/bin/env bash
# Shared helpers for the Meerkat CRM verification hooks.

project_root() {
  if [ -n "${CLAUDE_PROJECT_DIR:-}" ] && [ -d "$CLAUDE_PROJECT_DIR" ]; then
    printf '%s' "$CLAUDE_PROJECT_DIR"
  else
    (cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
  fi
}

state_dir() {
  local d="${TMPDIR:-/tmp}/meerkat-crm-hooks"
  mkdir -p "$d" 2>/dev/null
  printf '%s' "$d"
}

# Leaf key paths of a JSON file, sorted. Used for i18n parity.
json_leaf_keys() {
  jq -r '[paths(scalars) | join(".")] | sort | .[]' "$1" 2>/dev/null
}

# Compares every locale against en.json. Prints one line per problem.
locale_parity_report() {
  local loc="$1" en="$1/en.json" f base missing extra
  [ -f "$en" ] || return 0
  local ref
  ref=$(json_leaf_keys "$en")
  [ -n "$ref" ] || return 0
  for f in "$loc"/*.json; do
    [ "$f" = "$en" ] && continue
    base=$(basename "$f")
    if ! jq empty "$f" >/dev/null 2>&1; then
      printf '%s is not valid JSON\n' "$base"
      continue
    fi
    local cur
    cur=$(json_leaf_keys "$f")
    missing=$(comm -23 <(printf '%s\n' "$ref") <(printf '%s\n' "$cur") | head -15 | tr '\n' ' ')
    extra=$(comm -13 <(printf '%s\n' "$ref") <(printf '%s\n' "$cur") | head -15 | tr '\n' ' ')
    [ -n "${missing// /}" ] && printf '%s is missing keys present in en.json: %s\n' "$base" "$missing"
    [ -n "${extra// /}" ] && printf '%s has keys absent from en.json: %s\n' "$base" "$extra"
  done
}

# Content fingerprint of one area of the working tree: tracked modifications
# against HEAD plus the contents of untracked files. Tool-agnostic, so it also
# catches edits made through shell commands rather than the Edit tool.
area_fingerprint() {
  local area="$1"
  {
    git diff HEAD -- "$area" 2>/dev/null
    git ls-files --others --exclude-standard -- "$area" 2>/dev/null | sort | while IFS= read -r f; do
      shasum "$f" 2>/dev/null
    done
  } | shasum | cut -d' ' -f1
}
