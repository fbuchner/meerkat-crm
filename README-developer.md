**Project Overview**
- Meerkat CRM is a split Go backend and React frontend; the API sits under /api/v1 and serves a single-page app that manages contacts, activities, notes, reminders, and photos.
- The backend boots in [backend/main.go](backend/main.go) where config, database migrations, cron-style reminders, and the Gin router are wired together.
- The frontend lives in [frontend/src](frontend/src) with React 19, TypeScript, and MUI components backed by a typed API layer and custom hooks.

**Backend**
- Route definitions in [backend/routes/routes.go](backend/routes/routes.go) apply layered middleware: request IDs, structured logging, rate limiting, JWT auth, and JSON validation.
- Controllers (see [backend/controllers/contact_controller.go](backend/controllers/contact_controller.go)) expect `validated` payloads injected by [backend/middleware/validation.go](backend/middleware/validation.go); pull inputs from the context instead of decoding again.
- Custom errors from [backend/errors/errors.go](backend/errors/errors.go) plus [backend/middleware/middleware.go](backend/middleware/middleware.go) map failures to consistent JSON envelopes; prefer returning `*apperrors.AppError`.
- Database access is via GORM models in [backend/models](backend/models) with JSON arrays (contacts.circles) and manual cascade cleanup in delete flows; wrap multi-entity writes in transactions.
- Scheduled reminders run from [backend/services/reminder_service.go](backend/services/reminder_service.go) using gocron; honor `REMINDER_TIME` and Resend email toggles from [backend/config/config.go](backend/config/config.go).

**Frontend**
- All network calls go through [frontend/src/api/client.ts](frontend/src/api/client.ts) which enforces auth headers, request timeouts, and auto-logout on 401; reuse it for new endpoints.
- Resource-specific modules in [frontend/src/api](frontend/src/api) pair with hooks in [frontend/src/hooks](frontend/src/hooks); pages like [frontend/src/ContactsPage.tsx](frontend/src/ContactsPage.tsx) consume `{ data, loading, error, refetch }` contracts.
- Auth helpers in [frontend/src/auth.ts](frontend/src/auth.ts) persist JWTs in localStorage; frontend assumes `REACT_APP_API_URL` when constructing base URLs.
- Styling blends global CSS (App.css/index.css) with MUI theming; photo uploads land in backend static storage under `static/photos`.

**Workflows**
- Source backend/my_environment.env to `.env` before running the server
- Start the backend with `go run main.go` (or `make dev`) from backend/ after `go mod tidy`; migrations are embedded in the binary and auto-run on boot. Use `make migrate-up` or cmd/migrate for manual control during development.
- Frontend uses Yarn: `yarn install` then `yarn start` from frontend/; CRA proxies should point at the backend URL defined in `.env`.
- Logs use zerolog via [backend/logger/logger.go](backend/logger/logger.go); set LOG_LEVEL and LOG_PRETTY for debugging, and rely on request IDs threaded through middleware.
- Rate limiting is IP-based via [backend/middleware/rate_limiter.go](backend/middleware/rate_limiter.go); respect separate auth/general buckets when adding endpoints.

**Docker Build (Local)**
- Copy `.env.docker.example` to `.env.docker` and configure `JWT_SECRET_KEY`, `FRONTEND_URL`, and optionally `DATA_PATH`/`PHOTOS_PATH` for volume locations.
- Build and run locally: `docker compose -f docker-compose.build.yml up -d --build`
- Push images to GHCR: `docker compose -f docker-compose.build.yml push`
- Container defaults (`PORT`, `SQLITE_DB_PATH`, `PROFILE_PHOTO_DIR`) are set in [backend/Dockerfile](backend/Dockerfile); override via `.env.docker` if needed.
- The frontend Dockerfile builds a static bundle served by nginx; the API URL is baked in at build time via `REACT_APP_API_URL` (only relevant when frontend and backend are served via different URLs).

**Docker Deploy (Pre-built Images)**
- Copy `.env.docker.example` to `.env.docker` and configure `JWT_SECRET_KEY`, `FRONTEND_URL`, and optionally `DATA_PATH`/`PHOTOS_PATH` for volume locations.
- Deploy using pre-built images from GHCR: `docker compose up -d`
- Set `IMAGE_TAG` in `.env.docker` to pin a specific version (default: `latest`).

**Testing**
- Backend Go tests (`go test ./...` or `make test`) spin up in-memory SQLite in helpers like [backend/controllers/activity_controller_test.go](backend/controllers/activity_controller_test.go); mirror that pattern for new suites.
- Validation and middleware behavior has dedicated coverage in [backend/middleware/validation_test.go](backend/middleware/validation_test.go) and related files—extend these before touching shared validators.
- Reminder scheduling is covered in [backend/services/reminder_service_test.go](backend/services/reminder_service_test.go) with clock control helpers; keep cron changes tested there.
- Frontend tests run with `yarn test` and rely on React Testing Library setup in [frontend/src/setupTests.ts](frontend/src/setupTests.ts), which already registers jest-dom.

**Data & Integrations**
- SQLite lives at `SQLITE_DB_PATH` (default meerkat.db); migrations in [backend/database/migrations](backend/database/migrations) are embedded into the binary and auto-run on startup.
- JWT expiry, HTTP timeouts, trusted proxies, and Resend email settings are declared in [backend/config/config.go](backend/config/config.go) and loaded based on environment variables; use Config.Validate to catch misconfigurations.
- File uploads stream through [backend/controllers/photo_controller.go](backend/controllers/photo_controller.go) and land in `static/photos`; served through protected routes to enforce auth.
- API consumers expect consistent field casing (e.g., `Firstname` in responses vs. lower-case in queries); follow existing JSON tags in [backend/models/contact.go](backend/models/contact.go).
- Deletions often clean up dependent entities manually (contacts remove reminders, notes, relationships, and activity links); mirror transaction patterns from [backend/controllers/contact_controller.go](backend/controllers/contact_controller.go).

**Dependencies & Updates**

- **Backend (Go modules)**
	1. `cd backend && go mod tidy && go mod verify` to pull new indirect deps, drop unused modules, and confirm checksums.
	2. Use `go get -u ./...` (or target a module) when you intentionally bump versions; commit both go.mod and go.sum together.
	3. Re-run `go test ./...` (or `make test`) plus `make migrate-status` if schema changes shipped with the upgrade.

- **Frontend (Yarn)**
	1. `cd frontend && yarn install --check-files` to sync lockfiles and ensure native binaries rebuild.
	2. For minor bumps run `yarn upgrade` (or `yarn up <pkg>@latest` for a specific lib); keep `yarn.lock` in the PR.
	3. After upgrades, run `yarn build` for production bundles

**Releases (only relevant for maintainers)**
- Ensure all changes are committed and pushed to `main`
- Create a tag using semantic versioning: `git tag v1.5.3`
- Push the tag to GitHub: `git push origin v1.5.3`
- This triggers a GitHub Actions workflow that automatically builds and publishes Docker images to GHCR
- Users can then deploy the new version by setting `IMAGE_TAG=v1.5.3` (or by just using `:latest`) in their `.env.docker` and running `docker compose up -d`