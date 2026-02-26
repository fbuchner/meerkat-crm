---
title: Frontend Development
parent: Development
nav_order: 3
---

# Frontend Development

## Setup

Requires Node.js and Yarn.

```sh
cd frontend
cp .env.example .env  # set REACT_APP_API_URL if backend isn't on localhost:8080
yarn install
```

## Running Locally

```sh
yarn start  # dev server on port 3000
```

Hot reload is enabled. The dev server proxies nothing so requests go directly to `REACT_APP_API_URL`.

## Adding a New Page

1. Create `src/pages/FooPage.tsx`.
2. Add a route in the router (typically in `App.tsx`).
3. Add a custom hook in `src/hooks/useFoo.ts` for data fetching (see pattern below).
4. Add an API module in `src/api/foo.ts` if needed.

## API Client

All requests go through `src/api/client.ts`. It handles:
- httpOnly cookie auth (`credentials: 'include'`, no Authorization header)
- Configurable timeout via `REACT_APP_REQUEST_TIMEOUT` (default 30s)
- Automatic redirect to `/login` on 401
- Structured `ApiError` with `code` and `details` fields

Call the client from API modules, not directly from components.

## Custom Hooks

Hooks in `src/hooks/` encapsulate data fetching.
Hooks own their loading/error state. Components call `refetch()` after mutations.

## Components

Reusable components live in `src/components/`. They are MUI-based and receive data via props and do not fetch data themselves. Dialog components manage their own open/close state when passed an `open` prop and `onClose` callback.

## Translations

All user-facing strings must be translated. Add keys to both `src/i18n/locales/en.json` and `de.json`. Do not hardcode English strings in components.

## Testing

The frontend is tested via Playwright E2E tests against a running application. See [Testing](testing.md).
