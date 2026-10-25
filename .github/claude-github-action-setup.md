# Claude Code GitHub Action — setup and triggers

Two workflows, both using [`anthropics/claude-code-action`](https://github.com/anthropics/claude-code-action)
pinned to the `v1` major tag (currently resolving to `v1.0.252`).

## What triggers what

| Workflow | Fires on | Does |
| --- | --- | --- |
| [`claude.yml`](workflows/claude.yml) | an issue opened or assigned whose title or body contains `@claude`; an issue comment, PR review comment, or submitted PR review containing `@claude` | Claude reads the thread, implements the change, pushes a `claude/**` branch and opens a PR against `main` |
| [`claude-review.yml`](workflows/claude-review.yml) | `pull_request_target` — `opened`, `synchronize`, `ready_for_review` | Claude reviews the diff and posts a top-level comment plus inline comments |

Neither workflow merges anything, and auto-merge is not enabled anywhere.

### Guards worth knowing

- **Only users with write access can trigger `claude.yml`.** The action enforces
  this itself for issue, comment and review events, so a drive-by `@claude` from a
  stranger on this public repo does nothing.
- **`claude-review.yml` reviews fork PRs too.** It used to skip them: the
  `pull_request` event gives a fork no access to repository secrets, so the run
  could only fail, which left exactly the contributions that most need review
  unreviewed. It now runs on `pull_request_target`, which executes in this
  repository's context where the secret is available.
- **`pull_request_target` is only safe while the PR's code never reaches that
  job**, so three properties in `claude-review.yml` are load-bearing and the
  comment at the top of the file says so. The checkout pins the **base** commit,
  never the PR head; there is no build, install, test or lint step; and
  `--allowedTools` carries no code-executing tool, so Claude reads the base tree
  and talks to the GitHub API and nothing else. Claude sees the change through
  `gh pr diff`, as data. CodeQL enforces the first of those — it flags a PR-head
  checkout here as `actions/untrusted-checkout/high`, and it did exactly that on
  the first draft of the change.
- **`claude-review.yml` skips drafts, and skips PRs opened by bots other than
  `claude[bot]`.** The action refuses a bot actor by exiting non-zero, which
  showed every dependency bump a red `Review pull request` check for what is
  intended behaviour — and made a broken token indistinguishable from normal
  operation. The job condition now decides it instead, so bot PRs resolve as a
  clean skip. `allowed_bots: 'claude[bot]'` stays set and `claude[bot]` is let
  through the condition, so PRs Claude opened from `claude.yml` are still
  reviewed. To review dependency bumps as well, drop the `user.type != 'Bot'`
  clause and add `dependabot[bot]` to `allowed_bots`, at the cost of a Claude run
  per bump.

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

One secret, and nothing else:

| Secret | Value |
| --- | --- |
| `CLAUDE_CODE_OAUTH_TOKEN` | Output of `claude setup-token`, run locally. Works with a Claude Pro or Max subscription. |

If you would rather bill via the API than your subscription, add `ANTHROPIC_API_KEY`
instead and swap the `claude_code_oauth_token:` line in both workflows for
`anthropic_api_key: ${{ secrets.ANTHROPIC_API_KEY }}`. Do not set both.

### Steps

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

Until the secret exists, both workflows will run and fail at the Claude step. That
is the only thing missing — the YAML itself needs no further edits.

## Repository rules in CI

`CLAUDE.md` at the repo root is gitignored (`.gitignore` line 86), so a CI checkout
never sees it and neither does Claude running in a workflow. Everything that has to
reach CI therefore lives in the committed
[`.github/CLAUDE_INSTRUCTIONS.md`](CLAUDE_INSTRUCTIONS.md), which both workflow
prompts tell Claude to read first: the definition of done, the two-allowlist
contact-field trap, locale parity, `DateFormatProvider`, the backend and frontend
conventions, and the git boundaries these workflows assume (push a `claude/**` branch
and open a PR; never `main`, never force-push, never merge, no auto-merge).

That file is the authority for CI and for contributors, and it is self-contained —
it is not a mirror of a personal `CLAUDE.md`, so nothing breaks if yours says
something different or says nothing on a subject at all.

The `.claude/hooks/` verification hooks are local-only — they are a `Stop` hook in
your own Claude Code session and do not run in Actions. That is why both prompts
spell out the test commands and why the workflows set up Go and Node and run
`yarn install` before the action, so Claude can actually run the suites.

## Cost knobs

- `--max-turns` is set to `40` for `claude.yml` and `20` for `claude-review.yml`.
  Lower them if runs cost more than they are worth; Claude stops gracefully at the cap.
- No `--model` is pinned, so the action's own default applies. Add
  `--model <id>` to `claude_args` to force a cheaper model for reviews.
- `claude-review.yml` runs on every push to a PR. If that is too chatty, drop
  `synchronize` from its `types:` so it only reviews on open and on ready-for-review.
