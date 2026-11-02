#!/usr/bin/env bash
# PreToolUse hook: before commits leave the machine, ask Claude to review them
# locally. Every review on GitHub is paid for by the maintainer, so problems are
# cheapest to catch here. Refuses the first attempt per commit and lets the retry
# through, so it can never trap a session. additionalContext would not do: it only
# reaches Claude after the push has already run.
set -uo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

# CI has its own review, and no /code-review to run.
[ -z "${GITHUB_ACTIONS:-}" ] || exit 0

input=$(cat)
command=$(printf '%s' "$input" | jq -r '.tool_input.command // empty' 2>/dev/null)
[ -n "$command" ] || exit 0

# Only at the start of a command, so a commit message mentioning these does not count.
start='(^|[;&|(])[[:space:]]*([A-Za-z_]+=[^[:space:]]*[[:space:]]+)*'
printf '%s\n' "$command" | grep -Eq "${start}(git([[:space:]]+-[cC][[:space:]]+[^[:space:]]+)*[[:space:]]+push([[:space:]]|$)|gh[[:space:]]+pr[[:space:]]+create([[:space:]]|$))" || exit 0

cwd=$(printf '%s' "$input" | jq -r '.cwd // empty' 2>/dev/null)
cd "${cwd:-$(project_root)}" 2>/dev/null || exit 0
head=$(git rev-parse HEAD 2>/dev/null) || exit 0

# Nothing to review, such as a push from main or of a tag.
ahead=$(git rev-list --count origin/main..HEAD 2>/dev/null || echo 1)
[ "$ahead" != "0" ] || exit 0

marker="$(state_dir)/review-reminded-$head"
[ ! -e "$marker" ] || exit 0
touch "$marker" 2>/dev/null || exit 0

reason="Paused once so these commits get a local review first: reviews on GitHub cost the maintainer money per run. Run /code-review on this branch against origin/main (if that skill is unavailable, review the diff yourself), fix what it finds, then retry this command. If you already reviewed these changes, or the user asked to skip this, retry now. This reminder appears once per commit, so the retry will go through."
jq -n --arg r "$reason" \
  '{hookSpecificOutput:{hookEventName:"PreToolUse",permissionDecision:"deny",permissionDecisionReason:$r}}'
exit 0
