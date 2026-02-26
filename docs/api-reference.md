---
title: API Reference
nav_order: 8
has_children: false
---

# API Reference

## Base URL

```
/api/v1
```

## Authentication

Auth endpoints (`/login`, `/register`, `/logout`, `/password-reset/*`) are public. All other endpoints require authentication.

Login sets an httpOnly JWT cookie. Subsequent requests must include it (i.e. send with `credentials: 'include'`). The cookie name and domain are configured via `COOKIE_DOMAIN` and `COOKIE_SECURE`.

Admin endpoints (`/admin/*`) additionally require the user to have the admin flag set.

## Error Responses

All errors follow the same structure:

```json
{
  "code": "NOT_FOUND",
  "message": "Contact not found",
  "details": {}
}
```

Common error codes:

| Code | HTTP status |
|---|---|
| `UNAUTHORIZED` | 401 |
| `INVALID_CREDENTIALS` | 401 |
| `TOKEN_EXPIRED` | 401 |
| `FORBIDDEN` | 403 |
| `NOT_FOUND` | 404 |
| `VALIDATION_ERROR` | 400 |
| `INVALID_INPUT` | 400 |
| `ALREADY_EXISTS` | 409 |
| `RATE_LIMIT_EXCEEDED` | 429 |
| `INTERNAL_ERROR` | 500 |

## Request IDs

Every request and response carries an `X-Request-ID` header created by the middleware.

---

## Endpoints

### Auth

| Method | Path | Description |
|---|---|---|
| `POST` | `/register` | Create a new user account |
| `POST` | `/login` | Authenticate and set session cookie |
| `POST` | `/logout` | Clear session cookie |
| `POST` | `/check-password-strength` | Validate a password without registering |
| `POST` | `/password-reset/request` | Send a password reset email |
| `POST` | `/password-reset/confirm` | Apply a password reset token |

### Users

| Method | Path | Description |
|---|---|---|
| `GET` | `/users/me` | Get the current user |
| `POST` | `/users/change-password` | Change password |
| `PATCH` | `/users/language` | Update UI language preference |
| `PATCH` | `/users/date-format` | Update date format preference |
| `GET` | `/users/custom-fields` | Get custom field names |
| `PATCH` | `/users/custom-fields` | Update custom field names |

### Contacts

| Method | Path | Description |
|---|---|---|
| `GET` | `/contacts` | List contacts (supports search and circle filter) |
| `POST` | `/contacts` | Create a contact |
| `GET` | `/contacts/:id` | Get a contact (supports filtering the returned fields) |
| `PUT` | `/contacts/:id` | Update a contact |
| `DELETE` | `/contacts/:id` | Delete a contact |
| `POST` | `/contacts/:id/archive` | Archive a contact |
| `POST` | `/contacts/:id/unarchive` | Unarchive a contact |
| `GET` | `/contacts/circles` | List all circles in use |
| `GET` | `/contacts/random` | Get five random contacts |
| `GET` | `/contacts/birthdays` | Get upcoming birthdays |
| `POST` | `/contacts/:id/profile_picture` | Upload a profile picture (multipart) |
| `GET` | `/contacts/:id/profile_picture` | Get a contact's profile picture |
| `GET` | `/proxy/image` | Proxy an external image URL for upload preview |

### Relationships

| Method | Path | Description |
|---|---|---|
| `GET` | `/contacts/:id/relationships` | List outgoing relationships |
| `GET` | `/contacts/:id/incoming-relationships` | List incoming relationships |
| `POST` | `/contacts/:id/relationships` | Create a relationship |
| `PUT` | `/contacts/:id/relationships/:rid` | Update a relationship |
| `DELETE` | `/contacts/:id/relationships/:rid` | Delete a relationship |

### Notes

| Method | Path | Description |
|---|---|---|
| `GET` | `/contacts/:id/notes` | List notes for a contact |
| `POST` | `/contacts/:id/notes` | Create a note for a contact |
| `GET` | `/notes` | List unassigned notes |
| `POST` | `/notes` | Create an unassigned note |
| `GET` | `/notes/:id` | Get a note |
| `PUT` | `/notes/:id` | Update a note |
| `DELETE` | `/notes/:id` | Delete a note |

### Activities

| Method | Path | Description |
|---|---|---|
| `GET` | `/activities` | List all activities |
| `POST` | `/activities` | Create an activity |
| `GET` | `/activities/:id` | Get an activity |
| `PUT` | `/activities/:id` | Update an activity |
| `DELETE` | `/activities/:id` | Delete an activity |
| `GET` | `/contacts/:id/activities` | List activities for a contact |

### Reminders

| Method | Path | Description |
|---|---|---|
| `GET` | `/reminders` | List all reminders |
| `GET` | `/reminders/upcoming` | List upcoming reminders (used by dashboard) |
| `GET` | `/reminders/:id` | Get a reminder |
| `PUT` | `/reminders/:id` | Update a reminder |
| `DELETE` | `/reminders/:id` | Delete a reminder |
| `POST` | `/reminders/:id/complete` | Mark a reminder complete (creates timeline entry) |
| `GET` | `/contacts/:id/reminders` | List reminders for a contact |
| `POST` | `/contacts/:id/reminders` | Create a reminder for a contact |
| `GET` | `/contacts/:id/reminder-completions` | List completion history for a contact (timeline entries) |
| `DELETE` | `/reminder-completions/:id` | Delete a completion entry |

### Import

| Method | Path | Description |
|---|---|---|
| `POST` | `/contacts/import/upload` | Upload a CSV file, returns parsed preview data |
| `POST` | `/contacts/import/preview` | Apply column mapping, returns contacts with duplicate detection |
| `POST` | `/contacts/import/confirm` | Execute the import with per-row decisions |
| `POST` | `/contacts/import/vcf/upload` | Upload a VCF file, returns contacts with duplicate detection |
| `POST` | `/contacts/import/vcf/confirm` | Execute the VCF import |

### Export

| Method | Path | Description |
|---|---|---|
| `GET` | `/export` | Download all data as CSV |
| `GET` | `/export/vcf` | Download all contacts as VCF (includes photos) |

### Network

| Method | Path | Description |
|---|---|---|
| `GET` | `/graph` | Get contact network graph data |

### Admin

| Method | Path | Description |
|---|---|---|
| `GET` | `/admin/users` | List all users |
| `GET` | `/admin/users/:id` | Get a user |
| `PATCH` | `/admin/users/:id` | Update a user (e.g. set admin flag) |
| `DELETE` | `/admin/users/:id` | Delete a user |

### Health

| Method | Path | Description |
|---|---|---|
| `GET` | `/health` | Health check (no auth, no versioning) |
