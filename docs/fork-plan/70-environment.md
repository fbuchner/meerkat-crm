# 70 — Environment, branch target, and CI (operational facts, not architecture)

Everything in `00`–`60` is architecture and content. This doc captures the operational facts an
implementing agent needs and would otherwise have to rediscover: how to actually build/test Go code in
this environment, where the work lands, and what (if anything) CI needs.

## 70.1 No `go` toolchain on PATH — build/test via Docker

This sandbox has no `go` binary. All `go build`/`go vet`/`go test` verification must run inside a
`golang` container, using the **already-created, already-populated** Docker volumes from prior work on
this repo so dependencies aren't re-downloaded every run:

```bash
docker volume create meerkat-go-mod-cache   # idempotent if it already exists
docker volume create meerkat-go-build-cache

docker run --rm \
  -v "/home/drew/github/meerkat-crm:/app" \
  -v meerkat-go-mod-cache:/go/pkg/mod \
  -v meerkat-go-build-cache:/root/.cache/go-build \
  -w /app/backend \
  -e GOFLAGS="-mod=mod -buildvcs=false" \
  golang:1.25 sh -c "go build ./... && go vet ./... && go test ./..."
```

Notes:
- `-buildvcs=false` avoids a spurious VCS-stamping error when the container's git identity differs from
  the mounted repo's.
- Scope `go test`/`go vet` to specific new packages during iterative work, e.g.
  `go test ./contactmodel/... ./correspondence/...`; run the full `./...` before declaring a WP done.
- This is the **only** way to verify P0 work in this environment. A "the code looks right" report
  without an actual green run through this container is not acceptance — see `60-review-gates.md`.
- If frontend verification is needed later (P3 onward), `frontend/node_modules` must be installed with
  `npm install --legacy-peer-deps` (the repo's `react-scripts`/TypeScript version pins conflict under
  plain `npm install`) — `go` isn't the only toolchain gap; note it here so it isn't rediscovered twice.

## 70.2 Where this work lands

- Repo root: `/home/drew/github/meerkat-crm`, currently on branch `main`, clean.
- Two git remotes already exist from prior session work: `origin` → `fbuchner/meerkat-crm` (upstream,
  no push access) and `fork` → `DrewBrunning/meerkat-crm` (the user's own fork, push access confirmed).
- **P0 (WP-10 through WP-60) should be developed on a new branch off `main`**, e.g.
  `feature/rfc9553-contacts`, pushed to `fork` (not `origin`). It touches only new files under
  `backend/{contactmodel,correspondence,jscontact,vcard4,vcard3,internal/rfctest}` and
  `docs/fork-plan`/`docs/specs` — per `00-overview.md` §0.9's isolation check, `git status` should show
  no modifications to any existing file for the entirety of P0.
- The rebrand (`WP-74`, `50-integration-and-rebrand.md`) — including any repo rename — happens **last**,
  after P0–P4 land, specifically so the bulk of the implementation history isn't attributed to a
  transitional name.

## 70.3 CI

**No CI changes are needed for P0.** Confirmed by reading `.github/workflows/unit-tests.yml`: its
backend job runs `go test ./...` from the `backend/` working directory, which already recurses into any
new package directory automatically. Same for `go vet`/`go build` if later added there. Revisit CI only
at P1+ if new top-level commands (e.g. a one-shot `cmd/migrate-contacts` per WP-70) need their own job
step, or if `e2e-tests.yml`'s Playwright run needs new scenarios wired in (P3).

## 70.4 Quick-start checklist for an implementing agent picking up WP-10

1. `cd /home/drew/github/meerkat-crm && git status` — confirm clean, on `main`.
2. `git checkout -b feature/rfc9553-contacts` (only if this branch doesn't already exist from a prior
   WP in the same effort — check `git branch` first).
3. Read `00-overview.md` in full, then only your WP's row in §0.7 and the doc section(s) it names.
4. Create only the files your WP lists.
5. Verify via §70.1's Docker command, scoped to your package.
6. Report the diff + test output. Do not push/commit unless asked — see the standing git safety rules.
