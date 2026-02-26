---
title: Contributing
parent: Development
nav_order: 5
---

# Contributing

## Getting Started

1. Fork the repository and clone your fork locally.
2. Follow the setup steps in [Backend](backend.md) and [Frontend](frontend.md).
3. Create a feature branch from `main`.

## Development Workflow

1. Make your changes with tests.
2. Run `go test ./...` (backend) and the Playwright E2E suite (frontend)
3. Open a pull request against `main`.

## Code Style

**Backend:** Standard Go formatting (`gofmt`). Follow existing controller and service patterns with thin controllers, logic in services, always scope queries by `user_id`.

**Frontend:** TypeScript strict mode and MUI for all UI components. All user-facing strings through `i18next`. Custom hooks for data fetching, not inline `useEffect` + `fetch`.

## Pull Requests

- Keep PRs focused with one feature or fix per PR.
- AI tools may assist coding but you are responsible for the code quality. Do not open hands-off vibe-coded PRs. In those cases rather open a feature request instead.
- Describe what changed and why, not how.

## Reporting Issues

Open an issue on [GitHub](https://github.com/fbuchner/meerkat-crm/issues/new/choose). Try to include steps to reproduce for bugs. Use the feature request template for new ideas.
