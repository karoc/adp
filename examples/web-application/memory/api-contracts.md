# API Contracts

This document defines the contract between backend and frontend for all API endpoints.

## Base URL

- Development: `http://localhost:8080`
- Production: (to be configured)

## Common Response Format

### Success Response
```json
{
  "data": { ... },
  "meta": { ... }  // Optional metadata
}
```

### Error Response
```json
{
  "error": "Human-readable error message"
}
```

## Endpoints

### GET /api/health

Health check endpoint for monitoring server status.

**Request:** None

**Response (200 OK):**
```json
{
  "status": "healthy",
  "timestamp": 1234567890,
  "service": "web-example-api"
}
```

**Errors:** None (always returns 200 if server is running)

---

### GET /api/users

Retrieve list of all users.

**Request:** None (pagination support planned in T7)

**Response (200 OK):**
```json
{
  "users": [
    {
      "id": 1,
      "username": "alice",
      "email": "alice@example.com"
    },
    {
      "id": 2,
      "username": "bob",
      "email": "bob@example.com"
    }
  ],
  "count": 2
}
```

**Errors:** None

---

### POST /api/auth/login

Authenticate user and receive token.

**Request:**
```json
{
  "username": "string (required)",
  "password": "string (required)"
}
```

**Response (200 OK):**
```json
{
  "token": "demo_token_alice_20260614120000",
  "expires_at": "2026-06-15T12:00:00Z"
}
```

**Errors:**
- **400 Bad Request**: Invalid request body or missing fields
  ```json
  {"error": "Username and password required"}
  ```
- **401 Unauthorized**: Invalid credentials
  ```json
  {"error": "Invalid credentials"}
  ```

---

## Planned Endpoints (Future)

### POST /api/auth/register
User registration (Task T10 - future)

### GET /api/users?page=1&page_size=10
Paginated users list (Task T7)

### GET /api/users/search?q=alice
Search users (Task T8 - frontend only, client-side filtering)

---

## CORS Policy

Development: All origins allowed (`Access-Control-Allow-Origin: *`)
Production: Restrict to specific frontend domain

## Authentication

Current: Demo tokens (not secure)
Future: JWT tokens with proper validation

## Rate Limiting

Not implemented (consider for production)

## Versioning

Current: No versioning (v1 implied)
Future: Consider `/api/v1/` prefix for API versioning

---

## Change Log

- **2026-06-14**: Initial API contract
  - Health check endpoint
  - Users list endpoint
  - Login endpoint
