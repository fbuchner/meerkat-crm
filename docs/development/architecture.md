---
title: Architecture
parent: Development
nav_order: 1
---

# Architecture

## Overview

Two services: a Go API (`backend/`) and a React SPA (`frontend/`). They communicate over HTTP, the frontend calls `/api/v1/*` endpoints. In Docker, only the frontend port is published while the backend is internal.

## Backend

### Project Structure

```
backend/
  main.go              # Init: logger, config, DB, scheduler, router, graceful shutdown
  config/              # Environment variable loading and validation
  routes/routes.go     # All route registrations
  middleware/          # Auth, rate limiting, validation, logging, request ID
  controllers/         # HTTP handlers — thin, delegate to services or query DB directly
  models/              # GORM models and input DTOs
  services/            # Business logic (reminders, import, birthdays, password reset)
  errors/              # AppError type and error handler middleware
  database/migrations/ # Embedded SQL migrations, auto-applied on startup
  carddav/             # CardDAV protocol implementation
  i18n/                # Backend translations (email notifications)
```

### Error Handling

Controllers return `*apperrors.AppError`. The error handler middleware at the top of the stack catches these and writes a structured JSON response. Unhandled panics are also caught.

```go
return nil, apperrors.New(http.StatusNotFound, "CONTACT_NOT_FOUND", "contact not found")
```

### Database Layer

SQLite via GORM. All SQL migrations live in `database/migrations/` as embedded files and run automatically on startup in version order. Use `make migrate-create NAME=xxx` to add a new migration.

Every table includes `user_id` for multi-tenant isolation and all queries must filter by it.

## Frontend

### Project Structure

```
frontend/src/
  api/          # One module per resource (contacts, activities, …) + client.ts
  hooks/        # Data-fetching hooks (useContacts, useActivities, …)
  components/   # Reusable dialog and display components
  pages/        # Top-level route components (one per page)
  types/        # Shared TypeScript types
  i18n/locales/ # en.json, de.json
```

### State Management

No global store. Each page owns its state via custom hooks. Hooks encapsulate API calls and expose `{ data, loading, error, refetch }`. Dialogs manage local open/close state in `useContactDialogs` and similar hooks.

### Routing

React Router. Each page component maps to a route. Protected routes redirect to `/login` on 401 (handled automatically in `api/client.ts`).

### Internationalization

`i18next` with `react-i18next`. All UI strings live in `i18n/locales/en.json` and `de.json`. Use the `useTranslation()` hook. The user's language preference is stored server-side and applied to backend email notifications too.

## Security

**Authentication** — JWT (HMAC-signed). Tokens are stored in an `httpOnly` cookie for the browser; API clients (e.g. CardDAV) use a `Bearer` header fallback. Tokens carry `user_id` and `username` claims and expire automatically (expiration set by env variable).

**Multi-tenant isolation** — every table has a `user_id` column; all queries filter by the ID extracted from the validated JWT, so one user cannot access another's data.

**Password policy** — entropy-based validation (minimum 50 bits). Passwords are rejected at registration/change time before they are hashed.

**Account lockout** — after 5 failed login attempts the account is locked with exponential back-off (1 min base, doubling up to 30 min). The lockout state is reset after 1 hour of inactivity.

**Rate limiting** — per-IP token-bucket limiter applied globally; tighter limits on auth endpoints.

**Request size limits** — JSON bodies capped at 1 MB, other bodies at 10 MB to prevent DoS via large payloads.
