# Claude Code GitHub Action and AI review — setup and triggers

Two workflows. Development runs on Claude through
[`anthropics/claude-code-action`](https://github.com/anthropics/claude-code-action),
pinned to the `v1` major tag (currently resolving to `v1.0.252`). Review deliberately
runs on a different vendor's model through OpenRouter (GLM-5.3, with DeepSeek V4 Pro
as fallback), because a reviewer from the
same vendor as the author tends to share its blind spots.

## What triggers what

| Workflow | Fires on | Does |
| --- | --- | --- |
| [`claude.yml`](workflows/claude.yml) | an issue opened or assigned whose title or body contains `@claude`; an issue comment, PR review comment, or submitted PR review containing `@claude` | Claude reads the thread, implements the change, pushes a `claude/**` branch and opens a PR against `main` |
| [`ai-review.yml`](workflows/ai-review.yml) | `pull_request_target` — `opened`, `synchronize`, `ready_for_review`, and `labeled` with the `ai-review` label | A non-Anthropic model reviews the diff, posts one summary comment plus inline comments, and sets the `ai-review` commit status. A daily `schedule` run also resumes reviews postponed by the budget |

Neither workflow merges anything, and auto-merge is not enabled anywhere.

### Guards worth knowing

- **Only users with write access can trigger `claude.yml`.** The action enforces
  this itself for issue, comment and review events, so a drive-by `@claude` from a
  stranger on this public repo does nothing.
- **`ai-review.yml` reviews fork PRs too.** It runs on `pull_request_target`,
  because `pull_request` gives a fork no access to repository secrets.
- **`pull_request_target` is only safe while the PR's code never reaches that
  job**, so three properties in `ai-review.yml` are load-bearing and the comment at
  the top of the file says so. The checkout pins the **base** commit, never the PR
  head; there is no build, install, test or lint step; and no PR-controlled value
  is interpolated into a `run:` line. The script reads the PR through the GitHub API
  as text, and the model gets no tools. It also means a PR cannot edit its own
  reviewer: changes to the script, prompt or config only apply once merged, so the
  PR that introduces or changes them is reviewed by the previous version.
- **Who spends OpenRouter credit.** Reviews run automatically for the owner,
  members, collaborators and `trusted_authors` in
  [`ai-review/config.json`](ai-review/config.json). Anyone else's PR sits on a
  pending status until a maintainer adds the `ai-review` label. The label is
  removed after the run, so a later push needs a fresh decision. Add contributors to
  `trusted_authors` as they appear.
- **Bots.** `dependabot[bot]` gets a passing status with no review, through
  `skip_authors`. Other bot accounts are treated as strangers, because anyone can
  register a GitHub App and open a PR with it. `claude[bot]` is in `trusted_authors`,
  so PRs from `claude.yml` are reviewed.
- **What fails the `ai-review` status:** a finding at or above `fail_on` in the config
  (`blocker` by default), a suspected prompt injection, or a PR too large to send
  completely. What fits is still reviewed. The script works the verdict out from the
  findings, so the model cannot just declare a pass. If the reviewer itself breaks,
  the status is `error` and the job goes red, so re-run it.
- **When the budget is used up.** The key has a daily limit, and OpenRouter rejects
  requests once it is spent. That is not a failure: the job stays green, the summary
  comment says the review is postponed, and the status stays `pending`, so a required
  check keeps blocking. The scheduled run at 00:17 UTC reviews postponed PRs, oldest
  first, until the budget runs out again; whatever is left waits for the next day. To
  resume sooner, run the `AI Review` workflow by hand. A new push also retries. GitHub
  disables scheduled workflows after 60 days without repository activity.
- **`needs-human-review` label.** Added when a PR touches `human_review_paths` in the
  config (migrations, dependencies, `e2e/`, auth, deployment files, `.github/`), when
  the model raises a flag listed in `human_review_risk_flags`, or when the review was
  incomplete.
- **Not a required check yet.** To make it blocking, add `ai-review` to the
  `Mandatory merge requests` ruleset, and keep a bypass actor for oversized or emergency
  PRs, which otherwise cannot pass. The check is posted with `GITHUB_TOKEN`, so a
  workflow on a branch of this repository with `statuses: write` could fake it. A
  dedicated GitHub App as the required source would close that gap.

## CodeQL

[`workflows/codeql.yml`](workflows/codeql.yml) is an **advanced setup**, committed
here rather than configured in the UI, because CodeQL's default setup does not
analyze pull requests from forks — so fork PRs produced no CodeQL check at all.
With `CodeQL` listed as a required check in the `Mandatory merge requests`
ruleset, those PRs could not satisfy it and were only mergeable by admin
override.

Two things follow from that and are easy to undo by accident:

- **Default setup must stay disabled.** GitHub rejects advanced-setup uploads
  while it is on, with `CodeQL analyses from advanced configurations cannot be
  processed when the default setup is enabled`. Re-enabling it in the UI breaks
  every CodeQL run.
- **The trigger is `pull_request`, never `pull_request_target`.** Fork code is
  analyzed, but it must never run with a write token or with access to secrets.
  `permissions` is `contents: read` at the top level; only the analyze job asks
  for `security-events: write`.

## Required repository secrets

| Secret | Used by | Value |
| --- | --- | --- |
| `CLAUDE_CODE_OAUTH_TOKEN` | `claude.yml` | Output of `claude setup-token`, run locally. Works with a Claude Pro or Max subscription. |
| `OPENROUTER_API_KEY` | `ai-review.yml` | A key from <https://openrouter.ai/settings/keys>. Give it its own credit limit. |

If you would rather bill via the API than your subscription, add `ANTHROPIC_API_KEY`
instead and swap the `claude_code_oauth_token:` line in `claude.yml` for
`anthropic_api_key: ${{ secrets.ANTHROPIC_API_KEY }}`. Do not set both.

### AI review steps

1. **Create a key** at <https://openrouter.ai/settings/keys> with a credit limit that
   resets daily, then add it:
   ```bash
   gh secret set OPENROUTER_API_KEY --repo fbuchner/meerkat-crm
   ```
2. **Create the labels:**
   ```bash
   gh label create ai-review --repo fbuchner/meerkat-crm --color 1D76DB --description "Run AI review on this PR"
   ```
   ```bash
   gh label create needs-human-review --repo fbuchner/meerkat-crm --color D93F0B --description "Sensitive paths, or flagged by AI review"
   ```
3. **Try it on an existing PR.** This prints the review and posts nothing:
   ```bash
   GITHUB_TOKEN=$(gh auth token) OPENROUTER_API_KEY=... python3 .github/ai-review/review.py --dry-run --repo fbuchner/meerkat-crm --pr 123
   ```
4. **Merge the workflow.** Review starts on the next PR opened or pushed after that.

The script's unit tests need no keys or network:
```bash
python3 -m unittest discover -s .github/ai-review
```

### Claude Code steps

1. **Install the Claude GitHub App** on `fbuchner/meerkat-crm`:
   <https://github.com/apps/claude>. Repository-admin rights are required. The app
   supplies the short-lived, repo-scoped token the action uses for git and the API;
   the workflows do not need a personal access token.
2. **Mint the token locally:**
   ```bash
   claude setup-token
   ```
3. **Add it as a repository secret** — GitHub → Settings → Secrets and variables →
   Actions → *New repository secret* → name `CLAUDE_CODE_OAUTH_TOKEN`, paste the
   token as the value. Or from the CLI:
   ```bash
   gh secret set CLAUDE_CODE_OAUTH_TOKEN --repo fbuchner/meerkat-crm
   ```
4. **Allow Actions to open pull requests** if it is not already allowed: Settings →
   Actions → General → Workflow permissions → tick *Allow GitHub Actions to create
   and approve pull requests*. Without this, `claude.yml` pushes a branch but cannot
   open the PR.
5. **Smoke-test it** by commenting `@claude` on a small, low-risk issue.

Until the secret exists, `claude.yml` will run and fail at the Claude step. That is
the only thing missing — the YAML itself needs no further edits.

## Repository rules in CI

`CLAUDE.md` at the repo root is gitignored (`.gitignore` line 86), so a CI checkout
never sees it and neither does Claude running in a workflow. Everything that has to
reach CI therefore lives in the committed
[`.github/CLAUDE_INSTRUCTIONS.md`](CLAUDE_INSTRUCTIONS.md), which the `claude.yml`
prompt tells Claude to read first and `review.py` sends to the reviewer as project rules: the definition of done, the two-allowlist
contact-field trap, locale parity, `DateFormatProvider`, the backend and frontend
conventions, and the git boundaries these workflows assume (push a `claude/**` branch
and open a PR; never `main`, never force-push, never merge, no auto-merge).

That file is the authority for CI and for contributors, and it is self-contained —
it is not a mirror of a personal `CLAUDE.md`, so nothing breaks if yours says
something different or says nothing on a subject at all.

To keep paid reviews from catching what a local one would, `review-reminder.sh` in
`.claude/hooks/` stops the first `git push` or `gh pr create` per commit and asks Claude
to run `/code-review`, then lets the retry through. It is a nudge, not a safeguard: it
only reaches people using Claude Code, and a retry gets past it. The limits that hold
are in `ai-review/config.json` and the OpenRouter key.

The `.claude/hooks/` verification hooks are local-only — they are a `Stop` hook in
your own Claude Code session and do not run in Actions. That is why that file
spells out the test commands and why `claude.yml` sets up Go and Node and runs
`yarn install` before the action, so Claude can actually run the suites.

## Cost knobs

- `--max-turns` is set to `40` for `claude.yml`. Lower it if runs cost more than they
  are worth; Claude stops gracefully at the cap.
- No `--model` is pinned for `claude.yml`, so the action's own default applies.
- [`ai-review/config.json`](ai-review/config.json) picks the review models, first
  choice then fallback, the `reasoning_effort`, and the limits on how much of a PR is
  sent. Check IDs against <https://openrouter.ai/models>; changing model is a one-line
  edit. `provider_quantizations` keeps reviews off heavily quantized hosts; adding `fp4`
  makes them cheaper, but quality may vary between runs. A model that keeps returning unusable output costs at most one call per model
  and mode, so four with the current config. The tokens and
  cost of each review are at the bottom of its summary comment. Near the end of the
  daily budget, OpenRouter may refuse a request whose `max_output_tokens` it could not
  cover. That counts as used up, and the review is postponed.
- `ai-review.yml` runs on every push to a PR. If that is too chatty, drop
  `synchronize` from its `types:` so it only reviews on open and on ready-for-review.
