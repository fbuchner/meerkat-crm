# Meerkat CRM - Architecture Documentation

## Table of Contents
1. [Project Overview](#project-overview)
2. [System Architecture](#system-architecture)
3. [Backend Architecture](#backend-architecture)
4. [Frontend Architecture](#frontend-architecture)
5. [Database Schema](#database-schema)
6. [API Documentation](#api-documentation)
7. [Authentication & Security](#authentication--security)
8. [Data Flow](#data-flow)
9. [Technology Stack](#technology-stack)
10. [Deployment](#deployment)

---

## Project Overview

**Meerkat CRM** is a self-hosted personal CRM (Customer Relationship Management) system designed to help individuals manage their personal contacts, notes, activities, and reminders. Think of it as a "digital Rolodex" for your personal life.

### Core Features
- **Contact Management**: Store and organize contacts with detailed information
- **Circles**: Group contacts by categories (friends, family, work)
- **Notes**: Personal journaling and contact-specific notes
- **Activities**: Track events and meetings with multiple contacts
- **Reminders**: Birthday notifications and keep-in-touch reminders
- **Relationships**: Define how contacts relate to each other
- **Profile Photos**: Upload and manage contact profile pictures
- **Multi-language Support**: i18n with language detection

---

## System Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                 Client Layer (Web frontend)                 │
│  (React SPA - TypeScript, Material-UI, React Router)        │
└────────────────┬────────────────────────────────────────────┘
                 │
                 │ HTTP/REST API (JSON)
                 │ JWT Authentication
                 │
┌────────────────▼────────────────────────────────────────────┐
│                 Application Layer (Backend)                 │
│   (Go - Gin Framework, GORM, JWT, Bcrypt)                   │
│                                                             │
│    ┌──────────────────────────────────────────────────┐     │
│    │  Routes → Middleware → Controllers → Services    │     │
│    └──────────────────────────────────────────────────┘     │
└────────────────┬────────────────────────────────────────────┘
                 │
                 │ GORM ORM
                 │
┌────────────────▼────────────────────────────────────────────┐
│                    Data Layer (Database)                    │
│              (SQLite Database)                              │
│   • Contacts    • Notes      • Activities                   │
│   • Users       • Reminders  • Relationships                │
└─────────────────────────────────────────────────────────────┘
```

### Architecture Pattern
- **Backend**: MVC-inspired (Model-View-Controller) pattern
  - **Models**: Data structures and database entities
  - **Controllers**: Request handlers and business logic
  - **Services**: Reusable business logic (reminders)
  - **Middleware**: Authentication, CORS
  
- **Frontend**: Modern React architecture
  - **Components**: Reusable UI components
  - **Pages**: Route-level components
  - **Hooks**: Custom data-fetching hooks
  - **API Services**: Centralized API calls with TypeScript types
  - **State Management**: React hooks (useState, useEffect)

---

## Backend Architecture

### Directory Structure

```
backend/
├── main.go              # Application entry point
├── config/              # Configuration management
│   └── config.go        # Environment variables loader
├── models/              # Data models (GORM entities)
│   ├── user.go
│   ├── contact.go
│   ├── note.go
│   ├── activity.go
│   ├── reminder.go
│   └── relationship.go
├── controllers/         # Request handlers
│   ├── user_controller.go
│   ├── contact_controller.go
│   ├── note_controller.go
│   ├── activity_controller.go
│   ├── reminder_controller.go
│   ├── relationship_controller.go
│   ├── photo_controller.go
│   └── *_test.go        # Unit tests for each controller
├── middleware/          # HTTP middleware
│   └── auth.go          # JWT authentication middleware
├── services/            # Business logic services
│   ├── reminder_service.go
│   ├── user_service.go
│   └── *_test.go
└── routes/
    └── routes.go        # Route definitions
```

### Core Components

#### 1. **Models (Data Layer)**
Each model represents a database table and defines:
- Fields with GORM tags for database mapping
- JSON serialization tags
- Relationships between entities

**Key Models:**
- **User**: Authentication and user management
- **Contact**: Core entity with personal information
- **Note**: Text entries linked to contacts or standalone
- **Activity**: Events involving one or more contacts
- **Reminder**: Scheduled notifications for contacts
- **Relationship**: Defines how contacts relate to each other

#### 2. **Controllers (Request Handlers)**
Handle HTTP requests and responses:
- Parse request data
- Validate input
- Call database operations
- Return JSON responses
- Error handling

**Pattern Example:**
```go
func GetContact(c *gin.Context) {
    // 1. Get ID from URL params
    id := c.Param("id")
    
    // 2. Get database from context
    db := c.MustGet("db").(*gorm.DB)
    
    // 3. Query database
    var contact models.Contact
    if err := db.First(&contact, id).Error; err != nil {
        c.JSON(404, gin.H{"error": "Contact not found"})
        return
    }
    
    // 4. Return response
    c.JSON(200, contact)
}
```

#### 3. **Middleware**
- **AuthMiddleware**: JWT token validation for protected routes
- **CORS Middleware**: Cross-origin request handling
- **Database Injection**: Injects DB instance into request context

#### 4. **Services**
Reusable business logic:
- **ReminderService**: 
  - Sends email reminders via SendGrid
  - Handles birthday reminders
  - Manages recurring reminders
  - Scheduled daily execution via gocron

#### 5. **Routes**
Centralized route registration:
- Public routes: `/register`, `/login`
- Protected routes: Everything else (requires JWT)
- RESTful API design

### Database Layer (GORM)

**ORM Features Used:**
- Auto-migrations
- Relationships (One-to-Many, Many-to-Many)
- Preloading for eager loading
- Soft deletes (built into gorm.Model)
- JSON serialization for arrays (Circles)

**Relationships:**
```
Contact ─┬─> has many Notes
         ├─> has many Activities (many-to-many)
         ├─> has many Reminders
         └─> has many Relationships

Activity ──> has many Contacts (many-to-many)

Relationship ─> references Contact (self-referential)
```

---

## Frontend Architecture

### Directory Structure

```
frontend/src/
├── App.tsx                    # Main application component
├── index.tsx                  # React entry point
├── auth.ts                    # Authentication utilities
├── api.ts                     # Legacy API helpers
├── pages/
│   ├── ContactsPage.tsx       # Contact list with search
│   ├── ContactDetailPage.tsx  # Detailed contact view
│   ├── NotesPage.tsx          # Standalone notes
│   ├── ActivitiesPage.tsx     # Activities timeline
│   ├── LoginPage.tsx          # Login form
│   └── RegisterPage.tsx       # User registration
├── api/                       # **NEW** API Service Layer
│   ├── client.ts              # API client with auth
│   ├── contacts.ts            # Contact API functions
│   ├── notes.ts               # Note API functions
│   ├── activities.ts          # Activity API functions
│   └── index.ts               # Barrel exports
├── hooks/                     # **NEW** Custom Hooks
│   ├── useContacts.ts         # Contact data fetching
│   ├── useNotes.ts            # Note data fetching
│   ├── useActivities.ts       # Activity data fetching
│   └── index.ts               # Barrel exports
├── components/                # Reusable UI components
│   ├── AddNoteDialog.tsx
│   └── AddActivityDialog.tsx
└── i18n/                      # Internationalization
    ├── index.ts
    ├── en.json
    └── de.json
```

### Modern Architecture (Recently Refactored)

#### **API Service Layer** (`src/api/`)
Centralized API calls with full TypeScript support:

**Benefits:**
- Type-safe API calls
- Single source of truth for endpoints
- Centralized error handling
- Automatic token expiration handling (401 → logout)
- Reusable across components

**Example:**
```typescript
// api/contacts.ts
export interface Contact {
  ID: number;
  firstname: string;
  lastname: string;
  // ... more fields
}

export async function getContacts(
  params: GetContactsParams,
  token: string
): Promise<ContactsResponse> {
  const response = await apiFetch(
    `${API_BASE_URL}/contacts?...`,
    { headers: getAuthHeaders(token) }
  );
  return response.json();
}
```

#### **Custom Hooks** (`src/hooks/`)
Reusable data-fetching logic with state management:

**Benefits:**
- Clean separation of concerns
- Automatic loading/error states
- Consistent data fetching patterns
- Easy to test and mock
- Reduces component complexity

**Example:**
```typescript
// hooks/useContacts.ts
export function useContacts(params: GetContactsParams) {
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchContacts = useCallback(async () => {
    // ... fetching logic
  }, [params]);

  useEffect(() => {
    fetchContacts();
  }, [fetchContacts]);

  return { contacts, loading, error, refetch: fetchContacts };
}
```

**Usage in Components:**
```typescript
// Clean component code
const { contacts, loading, error } = useContacts({ 
  page: 1, 
  limit: 10 
});
```

### Component Architecture

#### **Page Components**
- **ContactsPage**: List view with search, pagination, circle filtering
- **ContactDetailPage**: Complex view with timeline, inline editing
- **NotesPage**: Journal-style notes with timeline UI
- **ActivitiesPage**: Event list with contact chips

#### **UI Framework**
- **Material-UI (MUI)**: Component library
- **React Router**: Client-side routing
- **i18next**: Internationalization
- **Timeline Components**: From @mui/lab

### State Management
- **No Redux**: Uses React hooks for simplicity
- **Local State**: useState for UI state
- **Server State**: Custom hooks for data fetching
- **Context**: Used for auth state (via localStorage)

---

## Database Schema

### Entity Relationship Diagram

```
┌─────────────┐
│    User     │
│─────────────│
│ ID          │
│ Username    │
│ Email       │
│ Password    │
└─────────────┘

┌──────────────────────┐        ┌─────────────────┐
│      Contact         │◄────-──┤   Relationship  │
│──────────────────────│        │─────────────────│
│ ID                   │        │ ID              │
│ Firstname            │        │ ContactID       │
│ Lastname             │        │ RelatedContactID│
│ Nickname             │        │ Type            │
│ Gender               │        │ Description     │
│ Email                │        └─────────────────┘
│ Phone                │
│ Birthday             │
│ Photo                │        ┌─────────────────┐
│ Address              │◄────-──┤      Note       │
│ HowWeMet             │        │─────────────────│
│ FoodPreference       │        │ ID              │
│ WorkInformation      │        │ Content         │
│ ContactInformation   │        │ Date            │
│ Circles (JSON array) │        │ ContactID       │
└───────┬──────────────┘        └─────────────────┘
        │
        │ Many-to-Many          ┌─────────────────┐
        ├───────────────────────┤    Activity     │
        │                       │─────────────────│
        │                       │ ID              │
        │                       │ Title           │
        │                       │ Description     │
        │                       │ Location        │
        │                       │ Date            │
        │                       └─────────────────┘
        │
        │                       ┌─────────────────┐
        └───────────────────────┤    Reminder     │
                                │─────────────────│
                                │ ID              │
                                │ Message         │
                                │ RemindAt        │
                                │ Recurrence      │
                                │ ContactID       │
                                │ IsSent          │
                                └─────────────────┘
```

### Table Schemas

#### **contacts**
```sql
CREATE TABLE contacts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,
    firstname TEXT NOT NULL COLLATE NOCASE,
    lastname TEXT COLLATE NOCASE,
    nickname TEXT COLLATE NOCASE,
    gender TEXT,
    email TEXT COLLATE NOCASE,
    phone TEXT,
    birthday TEXT,
    photo TEXT,
    photo_thumbnail TEXT,
    address TEXT,
    how_we_met TEXT,
    food_preference TEXT,
    work_information TEXT,
    contact_information TEXT,
    circles TEXT  -- JSON array serialized
);
```

#### **activities**
```sql
CREATE TABLE activities (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,
    title TEXT,
    description TEXT,
    location TEXT,
    date DATETIME
);

CREATE TABLE activity_contacts (
    contact_id INTEGER,
    activity_id INTEGER,
    PRIMARY KEY (contact_id, activity_id)
);
```

#### **notes**
```sql
CREATE TABLE notes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,
    content TEXT,
    date DATETIME,
    contact_id INTEGER,
    FOREIGN KEY (contact_id) REFERENCES contacts(id)
);
```

#### **reminders**
```sql
CREATE TABLE reminders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,
    message TEXT NOT NULL,
    remind_at DATETIME,
    recurrence TEXT,  -- "daily", "weekly", "monthly", "yearly"
    contact_id INTEGER,
    is_sent BOOLEAN,
    FOREIGN KEY (contact_id) REFERENCES contacts(id)
);
```

#### **relationships**
```sql
CREATE TABLE relationships (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,
    contact_id INTEGER NOT NULL,
    related_contact_id INTEGER NOT NULL,
    type TEXT,  -- "Child", "Parent", "Sibling", etc.
    description TEXT,
    FOREIGN KEY (contact_id) REFERENCES contacts(id),
    FOREIGN KEY (related_contact_id) REFERENCES contacts(id)
);
```

#### **users**
```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME,
    username TEXT UNIQUE,
    email TEXT UNIQUE NOT NULL,
    password TEXT NOT NULL  -- Bcrypt hashed
);
```

---

## API Documentation

### Health Check Endpoint

#### GET `/health`
Check the health status of the API and its dependencies. This endpoint does not require authentication.

**Response:** `200 OK` (when healthy)
```json
{
  "status": "healthy",
  "timestamp": "2025-10-27T22:30:00Z",
  "database": {
    "status": "healthy",
    "response_time_ms": 1.5
  },
  "version": "0.1.0"
}
```

**Response:** `503 Service Unavailable` (when unhealthy)
```json
{
  "status": "unhealthy",
  "timestamp": "2025-10-27T22:30:00Z",
  "database": {
    "status": "unhealthy",
    "response_time_ms": 0
  },
  "version": "0.1.0"
}
```

**Use Cases:**
- Container orchestration health checks (Docker, Kubernetes)
- Load balancer health monitoring
- Uptime monitoring services
- CI/CD deployment verification

---

## API Documentation

### API Versioning

All API endpoints (except `/health`) are versioned and prefixed with `/api/v1`.

**Base URL**: `http://localhost:8080/api/v1`

**Example Endpoints:**
- `POST /api/v1/register`
- `POST /api/v1/login`
- `GET /api/v1/contacts`
- `GET /api/v1/notes`

**Health Check** (no versioning):
- `GET /health` - Available at root level for monitoring tools

**Future Versions:**
- When breaking changes are needed, we'll introduce `/api/v2`
- Previous versions will be maintained for backward compatibility
- Deprecation notices will be provided in advance

---

### Authentication Endpoints

All authentication endpoints are under `/api/v1`.

#### POST `/api/v1/register`
Register a new user.

**Request:**
```json
{
  "username": "john_doe",
  "email": "john@example.com",
  "password": "securepassword"
}
```

**Response:** `201 Created`
```json
{
  "message": "User registered successfully"
}
```

#### POST `/api/v1/login`
Authenticate and receive JWT token.

**Request:**
```json
{
  "email": "john@example.com",
  "password": "securepassword"
}
```

**Response:** `200 OK`
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 1,
    "username": "john_doe",
    "email": "john@example.com"
  }
}
```

### Contact Endpoints (Protected)

All endpoints require `Authorization: Bearer <token>` header.

#### GET `/api/v1/contacts`
Get paginated list of contacts with optional search and filtering.

**Query Parameters:**
- `page` (default: 1)
- `limit` (default: 25)
- `search` (optional): Search in name, email, phone
- `circle` (optional): Filter by circle name

**Response:** `200 OK`
```json
{
  "contacts": [...],
  "total": 50,
  "page": 1,
  "limit": 25
}
```

#### GET `/api/v1/contacts/:id`
Get a single contact by ID.

#### POST `/contacts`
Create a new contact.

#### PUT `/contacts/:id`
Update a contact.

#### DELETE `/contacts/:id`
Soft delete a contact.

#### GET `/contacts/circles`
Get all unique circle names.

**Response:** `200 OK`
```json
["Friends", "Family", "Work", "Gym"]
```

### Note Endpoints (Protected)

#### GET `/contacts/:id/notes`
Get all notes for a contact.

#### POST `/contacts/:id/notes`
Create a note for a contact.

#### GET `/notes`
Get all unassigned notes (journal entries).

#### POST `/notes`
Create an unassigned note.

#### GET `/notes/:id`
Get a specific note.

#### PUT `/notes/:id`
Update a note.

#### DELETE `/notes/:id`
Delete a note.

### Activity Endpoints (Protected)

#### GET `/activities`
Get all activities with pagination.

**Query Parameters:**
- `page`, `limit`
- `include=contacts`: Include related contacts

#### GET `/contacts/:id/activities`
Get activities for a specific contact.

#### POST `/activities`
Create a new activity.

**Request:**
```json
{
  "title": "Dinner",
  "description": "Italian restaurant",
  "location": "Downtown",
  "date": "2025-10-26T19:00:00Z",
  "contact_ids": [1, 2, 3]
}
```

#### GET `/activities/:id`
Get a specific activity.

#### PUT `/activities/:id`
Update an activity.

#### DELETE `/activities/:id`
Delete an activity.

### Reminder Endpoints (Protected)

#### GET `/contacts/:id/reminders`
Get reminders for a contact.

#### POST `/contacts/:id/reminders`
Create a reminder.

**Request:**
```json
{
  "message": "Call about birthday",
  "remind_at": "2025-11-15T10:00:00Z",
  "recurrence": "yearly"
}
```

#### GET `/reminders/:id`
Get a specific reminder.

#### PUT `/reminders/:id`
Update a reminder.

#### DELETE `/reminders/:id`
Delete a reminder.

### Relationship Endpoints (Protected)

#### GET `/contacts/:id/relationships`
Get all relationships for a contact.

#### POST `/contacts/:id/relationships`
Create a relationship.

**Request:**
```json
{
  "related_contact_id": 5,
  "type": "Sibling",
  "description": "Younger sister"
}
```

#### PUT `/contacts/:id/relationships/:rid`
Update a relationship.

#### DELETE `/contacts/:id/relationships/:rid`
Delete a relationship.

### Photo Endpoints (Protected)

#### POST `/contacts/:id/profile_picture`
Upload a profile picture.

**Request:** `multipart/form-data`
- Field: `photo` (image file)

**Processing:**
- Accepts JPEG and PNG
- Creates thumbnail (200x200)
- Stores in `static/photos/`

#### GET `/contacts/:id/profile_picture`
Get profile picture.

**Response:** Image binary (JPEG/PNG)

---

## Authentication & Security

### JWT Authentication

**Token Generation:**
```go
// services/user_service.go
func GenerateToken(user *models.User, cfg *config.Config) (string, error) {
    claims := jwt.MapClaims{
        "user_id": user.ID,
        "exp":     time.Now().Add(24 * time.Hour).Unix(),
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(cfg.SecretKey))
}
```

**Token Validation:**
```go
// middleware/auth.go
func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
    return func(c *gin.Context) {
        tokenString := c.GetHeader("Authorization")
        token, err := jwt.Parse(tokenString, ...)
        // Validate token
        c.Set("user_id", claims["user_id"])
        c.Next()
    }
}
```

**Frontend Token Storage:**
```typescript
// auth.ts
export function setToken(token: string) {
  localStorage.setItem('jwt_token', token);
}

export function getToken(): string | null {
  return localStorage.getItem('jwt_token');
}
```

**Automatic Logout on Expiration:**
```typescript
// api/client.ts
export async function apiFetch(url: string, options: RequestInit) {
  const response = await fetch(url, options);
  
  if (response.status === 401) {
    localStorage.removeItem('jwt_token');
    window.location.href = '/login';
    throw new Error('Session expired');
  }
  
  return response;
}
```

### Password Security

- **Hashing**: Bcrypt with default cost (10)
- **No plain text storage**
- **Password validation**: Done at registration

### CORS Configuration

```go
r.Use(cors.New(cors.Config{
    AllowOrigins:     []string{cfg.FrontendURL},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
    AllowCredentials: true,
}))
```

---

## Data Flow

### Typical Request Flow

```
1. User Action (Frontend)
   ↓
2. React Component Event Handler
   ↓
3. Custom Hook or API Service Function
   ↓
4. HTTP Request with JWT Token
   ↓
5. CORS Middleware (Backend)
   ↓
6. Auth Middleware (Validate JWT)
   ↓
7. Router → Controller
   ↓
8. Controller validates input
   ↓
9. Database Operation (GORM)
   ↓
10. JSON Response
    ↓
11. API Service parses response
    ↓
12. Custom Hook updates state
    ↓
13. Component re-renders
```

### Example: Creating a Note

```
Frontend (NotesPage.tsx):
  → handleNoteSave() called
  → api/notes.createUnassignedNote({ content, date }, token)
  
API Service (api/notes.ts):
  → POST /notes with auth headers
  
Backend (note_controller.go):
  → CreateUnassignedNote(c)
  → Parse JSON body
  → db.Create(&note)
  → return JSON(note)
  
Frontend:
  → Response received
  → refetch() called (custom hook)
  → useNotes hook updates state
  → Component shows new note
```

---

## Technology Stack

### Backend
- **Language**: Go 1.21+
- **Framework**: Gin (HTTP web framework)
- **ORM**: GORM v2
- **Database**: SQLite 3
- **Authentication**: JWT (golang-jwt/jwt)
- **Password**: bcrypt
- **Scheduling**: gocron (for reminders)
- **Email**: SendGrid API
- **Image Processing**: Go standard library (image, image/jpeg, image/png)
- **CORS**: gin-contrib/cors

### Frontend
- **Language**: TypeScript 4.9+
- **Framework**: React 18.2
- **UI Library**: Material-UI (MUI) v7
- **Routing**: React Router v6
- **i18n**: i18next, react-i18next
- **HTTP Client**: Fetch API (native)
- **Build Tool**: Create React App (react-scripts)
- **Testing**: Jest, React Testing Library

### DevOps
- **Version Control**: Git
- **Database**: SQLite (file-based, no server needed)
- **Environment Config**: .env files
- **Process Management**: systemd or Docker (recommended)

---

## Deployment

### Environment Configuration

**Backend** (`backend/environment.env`):
```bash
PORT=8080
FRONTEND_URL=http://localhost:3000
DB_PATH=./perema.db
SECRET_KEY=your-secret-key-here
SENDGRID_API_KEY=your-sendgrid-key
SENDGRID_FROM_EMAIL=noreply@example.com
REMINDER_TIME=09:00
TRUSTED_PROXIES=127.0.0.1
```

**Frontend** (`frontend/.env`):
```bash
REACT_APP_API_URL=http://localhost:8080
```

### Development Setup

1. **Backend**:
```bash
cd backend
go mod tidy
source my_environment.env
go run main.go
```

2. **Frontend**:
```bash
cd frontend
npm install
npm start
```

### Production Deployment

**Option 1: Binary + Static Files**
```bash
# Backend
cd backend
go build -o perema-server main.go
./perema-server

# Frontend
cd frontend
npm run build
# Serve build/ with nginx or similar
```

**Option 2: Docker** (Recommended)
```dockerfile
# Multi-stage Dockerfile
FROM golang:1.21 AS backend-build
WORKDIR /app
COPY backend/ .
RUN go build -o server main.go

FROM node:18 AS frontend-build
WORKDIR /app
COPY frontend/ .
RUN npm install && npm run build

FROM debian:bookworm-slim
COPY --from=backend-build /app/server /server
COPY --from=frontend-build /app/build /static
CMD ["/server"]
```

### Systemd Service Example

```ini
[Unit]
Description=Meerkat CRM Service
After=network.target

[Service]
Type=simple
User=perema
WorkingDirectory=/opt/perema/backend
EnvironmentFile=/opt/perema/backend/environment.env
ExecStart=/opt/perema/backend/perema-server
Restart=always

[Install]
WantedBy=multi-user.target
```

---

## Performance Considerations

### Backend Optimizations
- **Connection Pooling**: GORM handles SQLite connection pooling
- **Eager Loading**: Use `Preload()` for related entities
- **Pagination**: Default limit of 25 items per page
- **Indexes**: SQLite auto-indexes primary keys and foreign keys
- **Soft Deletes**: Enables data recovery without losing history

### Frontend Optimizations
- **Code Splitting**: React Router lazy loading (potential)
- **Memoization**: useMemo for expensive computations
- **Debouncing**: Search inputs debounced (400ms)
- **Image Optimization**: Thumbnails generated for profile pics
- **PWA**: Service worker for offline capability

### Scaling Considerations
- **Database**: SQLite suitable for single-user or small teams
  - For larger deployments, consider PostgreSQL or MySQL
- **File Storage**: Profile photos stored locally
  - Consider S3 or similar for larger scale
- **Caching**: Currently no caching layer
  - Add Redis for session/data caching if needed

---

## Testing Strategy

### Backend Tests
- **Unit Tests**: Controller tests using httptest
- **Coverage**: Core CRUD operations covered
- **Database**: In-memory SQLite for tests
- **Mocking**: Gin test context

**Run tests:**
```bash
cd backend
go test ./... -v
```

### Frontend Tests
- **Unit Tests**: Component tests with React Testing Library
- **Setup**: Jest configured via CRA
- **Mocking**: API mocks with jest.mock()

**Run tests:**
```bash
cd frontend
npm test
```

### Integration Testing
- **Manual**: No automated E2E tests currently

