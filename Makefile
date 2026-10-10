.PHONY: help install format format-check lint typecheck test test-e2e precommit build

YARN ?= corepack yarn
# Base for format-check: only files changed since it are checked, so legacy
# unformatted files do not block unrelated work.
BASE ?= $(shell git merge-base HEAD origin/main 2>/dev/null || git merge-base HEAD main 2>/dev/null)

help: ## Show this help message
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-14s %s\n", $$1, $$2}'

install: ## Install backend and frontend dependencies
	cd backend && go mod download
	cd frontend && $(YARN) install --frozen-lockfile

format: ## gofmt Go files changed since BASE
	@files=$$(git diff --name-only --diff-filter=d $(BASE) -- 'backend/*.go'; git ls-files --others --exclude-standard -- 'backend/*.go'); \
	[ -z "$$files" ] || gofmt -w $$files

format-check: ## Fail if Go files changed since BASE are not gofmt-clean
	@files=$$(git diff --name-only --diff-filter=d $(BASE) -- 'backend/*.go'; git ls-files --others --exclude-standard -- 'backend/*.go'); \
	bad=$$([ -z "$$files" ] || gofmt -l $$files); \
	if [ -n "$$bad" ]; then echo "Not gofmt-clean:"; echo "$$bad"; exit 1; fi

lint: ## go vet + eslint
	cd backend && go vet ./...
	cd frontend && npx eslint --max-warnings=0 src

typecheck: ## TypeScript type check
	cd frontend && npx tsc --noEmit

test: ## Backend and frontend unit tests
	cd backend && go test ./...
	cd frontend && $(YARN) test

test-e2e: ## Playwright E2E against a local stack (no Docker; see scripts/e2e-stack.mjs)
	node scripts/e2e-stack.mjs -- npx playwright test $(ARGS)

precommit: format-check lint typecheck test ## All checks except E2E

build: ## Build backend binary and frontend bundle
	cd backend && go build -o meerkat .
	cd frontend && REACT_APP_API_URL= $(YARN) build
