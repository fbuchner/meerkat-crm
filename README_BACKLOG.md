# Meerkat CRM - TODO List

This document outlines improvement opportunities for the Meerkat CRM codebase, organized by category and priority.

## Priority Legend
- 🔴 **Critical**: Security issues or major bugs
- 🟠 **High**: Important for production readiness
- 🟡 **Medium**: Nice to have, improves quality
- 🟢 **Low**: Future enhancements

---

## 🔴 Critical Priority

### Security

- [ ] **Add input validation middleware**
  - Sanitize all user inputs before database operations
  - Validate email formats, phone numbers, dates
  - Protect against SQL injection (GORM provides protection, but validate input types)
  - **Effort**: Medium (2-3 days)
  - **Files**: Create `backend/middleware/validation.go`, apply to all controllers

- [ ] **Implement rate limiting**
  - Prevent brute force attacks on login endpoint
  - Use `gin-contrib/rate` or similar
  - **Effort**: Small (4-6 hours)
  - **Files**: `backend/middleware/rate_limiter.go`, apply in `routes/routes.go`

- [ ] **Add password strength requirements**
  - Minimum length, complexity rules
  - Implement at registration and password change
  - **Effort**: Small (2-3 hours)
  - **Files**: `backend/services/user_service.go`, `backend/controllers/user_controller.go`

- [ ] **Implement HTTPS/TLS**
  - Add TLS certificate configuration
  - Force HTTPS in production
  - **Effort**: Small (deployment configuration)
  - **Files**: `backend/main.go`, add TLS config

- [ ] **Secure secret key management**
  - Current: Secret key in plain text .env file
  - Use secrets manager (e.g., HashiCorp Vault, AWS Secrets Manager)
  - Or at minimum: Generate strong random keys and document rotation
  - **Effort**: Medium (1 day)
  - **Files**: `backend/config/config.go`

---

## 🟠 High Priority

### Backend Code Quality

- [ ] **Add comprehensive error handling**
  - Current: Some errors return generic messages
  - Create custom error types
  - Add error logging with context
  - Return meaningful error messages to client
  - **Effort**: Medium (3-4 days)
  - **Files**: Create `backend/errors/`, update all controllers

- [ ] **Implement structured logging**
  - Replace `fmt.Println` with proper logger (e.g., zerolog, zap)
  - Add log levels (debug, info, warn, error)
  - Include request IDs for tracing
  - **Effort**: Medium (2-3 days)
  - **Files**: All backend files, create `backend/logger/`

- [ ] **Add request validation**
  - Use `go-playground/validator` for struct validation
  - Validate all incoming request bodies
  - Return clear validation errors
  - **Effort**: Medium (2-3 days)
  - **Files**: All controllers, models

- [ ] **Implement database migrations**
  - Current: Auto-migrate in main.go (not production-ready)
  - Use proper migration tool (golang-migrate, goose)
  - Version control database schema
  - **Effort**: Medium (2-3 days)
  - **Files**: Create `backend/migrations/`, update `main.go`

### Frontend Code Quality

- [ ] **Add proper TypeScript types for all components**
  - Some components use `any` type
  - Create comprehensive interfaces
  - Enable stricter TypeScript config
  - **Effort**: Medium (2-3 days)
  - **Files**: All `.tsx` files, create `src/types/`

- [ ] **Implement error boundaries**
  - Catch React rendering errors gracefully
  - Show user-friendly error messages
  - **Effort**: Small (4-6 hours)
  - **Files**: `src/components/ErrorBoundary.tsx`, wrap in `App.tsx`

- [ ] **Add loading skeletons**
  - Replace basic "Loading..." text with skeleton components
  - Improve perceived performance
  - **Effort**: Small (1 day)
  - **Files**: All page components, use MUI Skeleton

- [ ] **Implement optimistic UI updates**
  - Update UI immediately before API response
  - Rollback on error
  - **Effort**: Medium (2-3 days)
  - **Files**: Custom hooks, API services

### Testing

- [ ] **Increase backend test coverage**
  - Current: Basic controller tests exist
  - Add service tests
  - Add middleware tests
  - Aim for >80% coverage
  - **Effort**: Large (1 week)
  - **Files**: All `*_test.go` files

- [ ] **Add frontend unit tests**
  - Current: Minimal tests
  - Test all custom hooks
  - Test critical component logic
  - Test API services
  - **Effort**: Large (1 week)
  - **Files**: Create `*.test.tsx` for all components/hooks

- [ ] **Add integration tests**
  - End-to-end API testing
  - Use Cypress or Playwright
  - Test critical user flows
  - **Effort**: Large (1-2 weeks)
  - **Files**: Create `e2e/` directory

### DevOps

- [ ] **Create Docker setup**
  - Multi-stage Dockerfile
  - Docker Compose for dev environment
  - Optimize image size
  - **Effort**: Medium (1-2 days)
  - **Files**: `Dockerfile`, `docker-compose.yml`, `.dockerignore`

- [ ] **Add CI/CD pipeline**
  - GitHub Actions or similar
  - Run tests on PR
  - Automated builds
  - Deployment automation
  - **Effort**: Medium (2-3 days)
  - **Files**: `.github/workflows/`

- [ ] **Environment-specific configs**
  - Development, staging, production configs
  - Validate required env vars on startup
  - **Effort**: Small (1 day)
  - **Files**: `backend/config/`, `frontend/.env.*`

---

## 🟡 Medium Priority

### Features

- [ ] **Add email verification for registration**
  - Send verification email with token
  - Verify email before allowing login
  - **Effort**: Medium (2-3 days)
  - **Files**: `user_controller.go`, `user_service.go`, email templates

- [ ] **Implement password reset flow**
  - Forgot password functionality
  - Email with reset token
  - **Effort**: Medium (2-3 days)
  - **Files**: `user_controller.go`, `user_service.go`, add routes

- [ ] **Add contact import/export**
  - CSV import for bulk contact addition
  - Export contacts to CSV/VCF
  - **Effort**: Medium (3-4 days)
  - **Files**: New controller, frontend upload component

- [ ] **Implement advanced search**
  - Full-text search across multiple fields
  - Search in notes, activities
  - Filters and sorting options
  - **Effort**: Large (1 week)
  - **Files**: Controllers, add search indexes

- [ ] **Add contact tagging system**
  - More flexible than circles
  - Multiple tags per contact
  - Tag management UI
  - **Effort**: Medium (3-4 days)
  - **Files**: New model, migration, controllers, frontend

- [ ] **Timeline view improvements**
  - Visual timeline for contact history
  - Filter by type (notes, activities, reminders)
  - Infinite scroll
  - **Effort**: Medium (2-3 days)
  - **Files**: `ContactDetailPage.tsx`, backend pagination

### Performance

- [ ] **Add database indexes**
  - Index frequently queried fields (email, name, birthday)
  - Composite indexes for common queries
  - **Effort**: Small (4-6 hours)
  - **Files**: Database migrations or GORM index tags

- [ ] **Implement API response caching**
  - Cache GET requests for contacts, notes
  - Redis or in-memory cache
  - Cache invalidation on updates
  - **Effort**: Medium (3-4 days)
  - **Files**: New cache layer, middleware

- [ ] **Optimize image storage**
  - Compress images on upload
  - Serve optimized formats (WebP)
  - Consider CDN for static assets
  - **Effort**: Medium (2-3 days)
  - **Files**: `photo_controller.go`, image processing

- [ ] **Add lazy loading for lists**
  - Virtual scrolling for large contact lists
  - Load images on demand
  - **Effort**: Small (1-2 days)
  - **Files**: Frontend list components

### Code Organization

- [ ] **Refactor backend to clean architecture**
  - Separate domain, application, infrastructure layers
  - Dependency injection
  - Interface-based design
  - **Effort**: Large (2 weeks)
  - **Files**: Major restructuring

- [ ] **Extract reusable backend utilities**
  - Common validation functions
  - Response helpers
  - Query builders
  - **Effort**: Small (1-2 days)
  - **Files**: Create `backend/utils/` or `backend/pkg/`

- [ ] **Create shared TypeScript types**
  - Backend and frontend share same type definitions
  - Code generation from Go structs
  - **Effort**: Medium (2-3 days)
  - **Files**: Generate `frontend/src/types/generated/`

- [ ] **Component library documentation**
  - Storybook for component showcase
  - Document component props and usage
  - **Effort**: Medium (3-4 days)
  - **Files**: Setup Storybook, create stories

### UI/UX

- [ ] **Add dark mode**
  - Theme toggle in settings
  - Persist preference
  - **Effort**: Small (1-2 days)
  - **Files**: MUI theme config, context provider

- [ ] **Improve mobile responsiveness**
  - Test on various screen sizes
  - Optimize touch targets
  - Mobile-friendly navigation
  - **Effort**: Medium (3-4 days)
  - **Files**: All frontend components, CSS

- [ ] **Add keyboard shortcuts**
  - Quick actions (create contact, search)
  - Navigation shortcuts
  - **Effort**: Small (1-2 days)
  - **Files**: Add keyboard event listeners

- [ ] **Implement drag-and-drop**
  - Reorder items in lists
  - Drag contacts to activities
  - **Effort**: Medium (2-3 days)
  - **Files**: List components, DnD library

---

## 🟢 Low Priority

### Documentation

- [ ] **API documentation with OpenAPI/Swagger**
  - Auto-generated API docs
  - Interactive API explorer
  - **Effort**: Medium (2-3 days)
  - **Files**: Add Swagger annotations, setup swagger-ui

- [ ] **User manual/wiki**
  - How to use the CRM
  - Feature documentation
  - Screenshots and tutorials
  - **Effort**: Large (ongoing)
  - **Files**: Create `docs/` directory or wiki

- [ ] **Developer onboarding guide**
  - Setup instructions
  - Architecture overview (expand current doc)
  - Contributing guidelines
  - **Effort**: Small (1-2 days)
  - **Files**: `CONTRIBUTING.md`, `SETUP.md`

- [ ] **Code comments and documentation**
  - Add godoc comments to all exported functions
  - JSDoc for TypeScript functions
  - **Effort**: Medium (ongoing)
  - **Files**: All code files

### Features (Future)

- [ ] **Multi-user support**
  - User roles and permissions
  - Shared contacts between users
  - Privacy controls
  - **Effort**: Very Large (3-4 weeks)
  - **Files**: Major feature, affects all layers

- [ ] **Calendar integration**
  - Sync activities with Google Calendar, Outlook
  - iCal export
  - **Effort**: Large (1-2 weeks)
  - **Files**: New service, OAuth integration

- [ ] **Email integration**
  - Link emails to contacts
  - Email history timeline
  - **Effort**: Very Large (4+ weeks)
  - **Files**: New service, IMAP/API integration

- [ ] **Mobile app**
  - React Native or native iOS/Android
  - Push notifications
  - **Effort**: Very Large (2-3 months)
  - **Files**: New repository

- [ ] **Social media integration**
  - Import contacts from LinkedIn, Facebook
  - Show social profiles
  - **Effort**: Large (2-3 weeks)
  - **Files**: OAuth integration, new services

- [ ] **Advanced analytics**
  - Contact interaction frequency
  - Relationship insights
  - Network visualization
  - **Effort**: Large (2-3 weeks)
  - **Files**: Analytics service, visualization components

### Infrastructure

- [ ] **Database migration to PostgreSQL**
  - For multi-user scenarios
  - Better performance for large datasets
  - **Effort**: Medium (3-4 days)
  - **Files**: Database driver, connection config

- [ ] **Implement backup system**
  - Automated database backups
  - Restore functionality
  - **Effort**: Small (1-2 days)
  - **Files**: Backup script, cron job

- [ ] **Add monitoring and alerting**
  - Application performance monitoring (APM)
  - Error tracking (Sentry)
  - Uptime monitoring
  - **Effort**: Medium (2-3 days)
  - **Files**: Integration config, dashboards

- [ ] **Implement feature flags**
  - Gradual feature rollout
  - A/B testing capability
  - **Effort**: Small (1-2 days)
  - **Files**: Feature flag service, UI toggles

---

## Quick Wins (Easy Improvements)

These are small tasks that provide immediate value:

1. **Add `.gitignore` improvements** (30 min)
   - Exclude `perema.db`, `*.env` files
   - Add OS-specific ignores (`.DS_Store`, etc.)

2. **Add proper README badges** (30 min)
   - Build status, test coverage, license
   - Technology stack badges

3. **Fix inconsistent naming** (1 hour)
   - Backend: Some functions use camelCase, others PascalCase
   - Standardize to Go conventions

4. **Add healthcheck endpoint** (1 hour)
   - `GET /health` for monitoring
   - Return DB connection status

5. **Implement API versioning** (2 hours)
   - Prefix all routes with `/api/v1`
   - Future-proof for breaking changes

6. **Add CORS preflight cache** (30 min)
   - Set `MaxAge` in CORS config
   - Reduce OPTIONS requests

7. **Add loading indicators everywhere** (2 hours)
   - Replace missing loading states
   - Consistent spinner/skeleton pattern

8. **Fix TypeScript `any` types** (3 hours)
   - Replace with proper interfaces
   - Enable `noImplicitAny` in tsconfig

9. **Add environment variable validation** (1 hour)
   - Check required vars on startup
   - Fail fast with clear error messages

10. **Add request timeout configuration** (1 hour)
    - Set reasonable timeouts for API calls
    - Prevent hanging requests

---

## Technical Debt

### Current Known Issues

1. **Profile photo storage**
   - Currently stored in `static/photos/`
   - No cleanup on contact deletion
   - No size limits enforced
   - **Recommendation**: Implement cleanup job, add max file size

2. **Circles implementation**
   - Stored as JSON array in text column
   - Not normalized (can't query efficiently)
   - **Recommendation**: Create separate `circles` table with many-to-many

3. **Reminder service reliability**
   - No error recovery if SendGrid fails
   - No logging of sent reminders
   - **Recommendation**: Add retry logic, persistent log

4. **Frontend state management**
   - No centralized state (using local state everywhere)
   - Some prop drilling occurring
   - **Recommendation**: Consider Zustand or Redux for complex state

5. **No soft delete UI**
   - Soft deleted items invisible to user
   - No way to restore deleted contacts
   - **Recommendation**: Add "Trash" view with restore functionality

6. **Birthday field is text, not date**
   - Stored as string, not proper date type
   - Makes date queries complex
   - **Recommendation**: Migration to proper date field

7. **No API pagination on all endpoints**
   - Some endpoints return all items
   - Could cause performance issues with large datasets
   - **Recommendation**: Add pagination everywhere

---

## Dependencies to Update

Check for outdated dependencies periodically:

```bash
# Backend
cd backend
go list -u -m all

# Frontend
cd frontend
npm outdated
```

---

## Suggested Implementation Order

For a production-ready system, tackle in this order:

### Phase 1: Security & Stability (2-3 weeks)
1. Rate limiting
2. Input validation
3. Password requirements
4. Structured logging
5. Error handling
6. Environment variable validation

### Phase 2: Testing & DevOps (2-3 weeks)
7. Backend test coverage
8. Frontend test coverage
9. Docker setup
10. CI/CD pipeline

### Phase 3: Code Quality (1-2 weeks)
11. TypeScript improvements
12. Database migrations
13. Error boundaries
14. API response improvements

### Phase 4: Features & Polish (ongoing)
15. User requested features
16. Performance optimizations
17. UI/UX improvements
18. Documentation

---

## Contributing

When implementing any of these TODOs:
1. Create a new branch
2. Write tests first (TDD)
3. Update documentation
4. Add migration if needed
5. Create PR with clear description
6. Update this TODO list

---

**Last Updated**: 2025-01-26  
**Current Version**: 0.1.0  
**Next Milestone**: 1.0.0 (Production Ready)
