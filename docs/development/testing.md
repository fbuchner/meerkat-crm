---
title: Testing
parent: Development
nav_order: 4
---

# Testing

## Backend Tests

### Running Tests

```sh
cd backend
go test ./...
```

### Test Database

Tests use an in-memory SQLite database that is auto-migrated before each test. No external dependencies required.

### Writing Tests

Use the test helpers in the `_test` files alongside each package. Set up a test DB with `database.NewTestDB()` (or equivalent), create a Gin test context, and call controller functions directly. Assert on the `httptest.ResponseRecorder`.

## Frontend Tests

Frontend tests come in two kinds: Vitest unit/component tests for isolated components and hooks, and Playwright for integrated E2E tests.

### Unit Tests

```sh
cd frontend
yarn test         # run once
yarn test:watch   # watch mode
```

Test files sit next to the code they cover as `*.test.ts(x)` in `frontend/src/` (for example `EmploymentHistoryList.test.tsx`), using Testing Library with jsdom.

### E2E Tests

#### Setup

```sh
cd frontend
yarn playwright install  # install browsers (first time only)
```

The E2E tests expect the full application running on `http://localhost:7300`. `global-setup.ts` seeds a test user before the suite runs.

#### Running E2E Tests

```sh
yarn test:e2e           # headless
yarn test:e2e:headed    # with browser visible
yarn test:e2e:ui        # Playwright UI mode
yarn test:e2e:debug     # debug mode
```

#### Writing E2E Tests

Tests live in `frontend/e2e/`. Use the shared fixtures from `fixtures.ts` for login/logout helpers and the `TEST_USER` credentials from `global-setup.ts`. Group tests with `test.describe` and keep each test independent.
