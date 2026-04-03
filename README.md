# Finance Dashboard API

A production-ready RESTful API for managing financial records with role-based access control, analytics, and dashboard insights. Built with **Go**, **Gin**, **GORM**, and **PostgreSQL**.

**Stack:** Go · Gin · GORM · PostgreSQL · JWT · bcrypt

---

## Features

- **JWT Authentication** with role-based access control (Viewer, Analyst, Admin)
- **Financial Records CRUD** with soft deletes
- **Advanced Filtering** by date range, category, and transaction type
- **Paginated Listing** with configurable page size
- **Dashboard Analytics** — summary totals, monthly trends, category breakdown
- **User Management** — admin-only user administration
- **Consistent JSON Envelope** — every response follows `{ success, message, data }`
- **UUID Primary Keys** — scalable, non-enumerable identifiers
- **IP-Based Rate Limiting** — 100 requests/minute per IP
- **Health Check Endpoint** for liveness probes

---

## Project Structure

```
finance-dashboard/
├── cmd/
│   └── main.go                          # Application entry point
├── config/
│   └── config.go                        # Environment config + DB connection
├── services/
│   ├── auth_service.go                  # Auth business logic
│   ├── auth_service_test.go             # 8 tests
│   ├── user_service.go                  # User business logic
│   ├── user_service_test.go             # 10 tests
│   ├── record_service.go               # Record business logic
│   ├── record_service_test.go           # 21 tests
│   ├── dashboard_service.go             # Dashboard analytics logic
│   ├── dashboard_service_test.go        # 7 tests
│   └── test_helpers_test.go             # Test DB setup + cleanup
├── middleware/
│   ├── auth.go                          # JWT authentication middleware
│   ├── auth_test.go                     # 6 tests
│   ├── rbac.go                          # Role-based access control middleware
│   ├── rbac_test.go                     # 5 tests
│   ├── rate_limiter.go                  # IP-based rate limiting
│   └── rate_limiter_test.go             # 4 tests
├── models/
│   ├── user.go                          # User model with role enum
│   └── financial_record.go              # Financial record model (soft delete)
├── handlers/
│   ├── auth_handler.go                  # Register + Login handlers
│   ├── user_handler.go                  # User management handlers
│   ├── record_handler.go                # Financial record handlers
│   └── dashboard_handler.go             # Analytics handlers
├── routes/
│   └── routes.go                        # Route registration + middleware wiring
├── utils/
│   ├── jwt.go                           # JWT generation + validation
│   ├── jwt_test.go                      # 6 tests
│   └── response.go                      # Standardized response helpers
├── postman/
│   ├── Finance_Dashboard_API.postman_collection.json
│   └── Finance_Dashboard_API.postman_environment.json
├── .env.example                         # Environment variable template
├── .gitignore
├── go.mod
├── go.sum
└── README.md
```

---

## Getting Started

### Prerequisites

- **Go** 1.21 or higher
- **PostgreSQL** 12 or higher

### Installation

1. **Clone the repository**

   ```bash
   git clone https://github.com/Slambot01/Finance_Project.git
   cd Finance_Project
   ```

2. **Configure environment variables**

   ```bash
   cp .env.example .env
   ```

   Edit `.env` with your values:

   | Variable | Description | Default |
   |----------|-------------|---------|
   | `DB_HOST` | PostgreSQL host | `localhost` |
   | `DB_PORT` | PostgreSQL port | `5432` |
   | `DB_USER` | Database username | `postgres` |
   | `DB_PASSWORD` | Database password | — |
   | `DB_NAME` | Database name | `finance_dashboard` |
   | `JWT_SECRET` | Secret key for signing JWTs (required) | — |
   | `JWT_EXPIRY_HOURS` | Token expiry duration in hours | `24` |
   | `PORT` | Server port | `8080` |

3. **Create the database**

   ```sql
   CREATE DATABASE finance_dashboard;
   ```

4. **Run the application**

   ```bash
   go mod tidy
   go run cmd/main.go
   ```

   The server will start and auto-migrate all database tables:
   ```
   Database connection established successfully
   Database migration completed successfully
   Server running on port 8080
   ```

---

## API Documentation

### Endpoints

| Method | Route | Access | Description |
|--------|-------|--------|-------------|
| `GET` | `/health` | Public | Health check |
| `POST` | `/auth/register` | Public | Register a new user |
| `POST` | `/auth/login` | Public | Login and receive JWT |
| `GET` | `/api/users` | Admin | List all users |
| `PUT` | `/api/users/:id` | Admin | Update user role/status |
| `DELETE` | `/api/users/:id` | Admin | Delete user |
| `GET` | `/api/records` | Viewer+ | List records (filtered, paginated) |
| `POST` | `/api/records` | Admin | Create a financial record |
| `GET` | `/api/records/:id` | Viewer+ | Get a single record |
| `PUT` | `/api/records/:id` | Admin | Update a record |
| `DELETE` | `/api/records/:id` | Admin | Soft-delete a record |
| `GET` | `/api/dashboard/summary` | Viewer+ | Income/expense/balance summary |
| `GET` | `/api/dashboard/trends` | Analyst+ | Monthly income vs expense trends |
| `GET` | `/api/dashboard/categories` | Analyst+ | Spending breakdown by category |

> **Viewer+** = Viewer, Analyst, Admin &nbsp;|&nbsp; **Analyst+** = Analyst, Admin

### Authentication

All `/api/*` endpoints require a JWT in the `Authorization` header:

```
Authorization: Bearer <your_jwt_token>
```

---

## Roles & Permissions

| Resource | Viewer | Analyst | Admin |
|----------|--------|---------|-------|
| Auth (register, login) | ✅ | ✅ | ✅ |
| View own records | ✅ | ✅ | ✅ |
| View all records | ❌ | ✅ | ✅ |
| Create/Update/Delete records | ❌ | ❌ | ✅ |
| Dashboard summary | ✅ | ✅ | ✅ |
| Dashboard trends | ❌ | ✅ | ✅ |
| Dashboard categories | ❌ | ✅ | ✅ |
| User management | ❌ | ❌ | ✅ |

---

## Request/Response Examples

All responses use a consistent JSON envelope:

```json
{
  "success": true,
  "message": "descriptive message",
  "data": {}
}
```

### Register

```http
POST /auth/register
Content-Type: application/json

{
  "name": "Jane Doe",
  "email": "jane@example.com",
  "password": "securepassword123",
  "role": "admin"
}
```

**Response** `201 Created`

```json
{
  "success": true,
  "message": "user registered successfully",
  "data": {
    "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "name": "Jane Doe",
    "email": "jane@example.com",
    "role": "admin",
    "is_active": true,
    "created_at": "2026-04-03T10:00:00Z",
    "updated_at": "2026-04-03T10:00:00Z"
  }
}
```

### Login

```http
POST /auth/login
Content-Type: application/json

{
  "email": "jane@example.com",
  "password": "securepassword123"
}
```

**Response** `200 OK`

```json
{
  "success": true,
  "message": "login successful",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "name": "Jane Doe",
      "email": "jane@example.com",
      "role": "admin",
      "is_active": true,
      "created_at": "2026-04-03T10:00:00Z",
      "updated_at": "2026-04-03T10:00:00Z"
    }
  }
}
```

### Create Record

```http
POST /api/records
Authorization: Bearer <token>
Content-Type: application/json

{
  "amount": 4500.00,
  "type": "income",
  "category": "salary",
  "date": "2026-04-01T00:00:00Z",
  "notes": "April salary"
}
```

**Response** `201 Created`

```json
{
  "success": true,
  "message": "financial record created successfully",
  "data": {
    "id": "f7e6d5c4-b3a2-1098-7654-321fedcba098",
    "user_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "amount": 4500.00,
    "type": "income",
    "category": "salary",
    "date": "2026-04-01T00:00:00Z",
    "notes": "April salary",
    "created_at": "2026-04-03T10:05:00Z",
    "updated_at": "2026-04-03T10:05:00Z"
  }
}
```

### Dashboard Summary

```http
GET /api/dashboard/summary
Authorization: Bearer <token>
```

**Response** `200 OK`

```json
{
  "success": true,
  "message": "dashboard summary retrieved successfully",
  "data": {
    "total_income": 15000.00,
    "total_expenses": 8500.00,
    "net_balance": 6500.00,
    "total_records": 42
  }
}
```

---

## Query Parameters for `GET /api/records`

| Parameter | Type | Description | Example |
|-----------|------|-------------|---------|
| `type` | string | Filter by transaction type | `income` or `expense` |
| `category` | string | Filter by category | `food`, `rent`, `salary` |
| `start_date` | string | Filter records from this date | `2026-01-01` |
| `end_date` | string | Filter records up to this date | `2026-03-31` |
| `page` | integer | Page number (default: 1) | `2` |
| `page_size` | integer | Records per page (default: 10, max: 100) | `25` |

**Example:**

```
GET /api/records?type=expense&category=food&start_date=2026-01-01&end_date=2026-03-31&page=1&page_size=20
```

---

## Running Tests

The project includes 66 tests (46 service layer + 20 middleware/utility) covering authentication, authorization, CRUD operations, analytics, and rate limiting.

```bash
# Service layer tests (requires PostgreSQL test database)
go test ./services/ -v

# Middleware and utility tests (no database required)
go test ./middleware/ -v
go test ./utils/ -v

# Run all tests
go test ./... -v
```

> **Note:** Service tests require a running PostgreSQL instance and a `finance_dashboard_test` database. Copy `.env.test.example` to `.env.test` and configure your test database credentials.

---

## Postman Collection

A complete Postman collection and environment are included for testing:

1. Import `postman/Finance_Dashboard_API.postman_collection.json` into Postman
2. Import `postman/Finance_Dashboard_API.postman_environment.json` as an environment
3. Set the environment as active
4. Register a user, then login to get a JWT
5. Copy the token value and set it as the `token` environment variable
6. All protected requests will automatically use the token

---

## Assumptions & Design Decisions

1. **Role Assignment at Registration** — Users select their role during registration. Only admins can subsequently change a user's role.

2. **Viewer Data Scoping** — Viewers can only see their own financial records. Analysts and admins see all records across all users. This scoping is enforced at the handler level by injecting the user's ID into query filters.

3. **Soft Delete for Financial Records** — Deleted financial records are excluded from all queries and dashboard analytics but remain in the database with a `deleted_at` timestamp. This preserves audit trails. Users are hard-deleted when removed by an admin, and their associated records are cascade-deleted.

4. **UUID Primary Keys** — All tables use UUIDs instead of auto-incrementing integers. This prevents ID enumeration attacks, supports distributed systems, and eliminates the need for sequential ID generation.

---

## Architecture Highlights

- **Clean Layer Separation** — Handlers parse HTTP, services contain business logic, models define data structures. No database queries in handlers.
- **Middleware-Level Security** — JWT validation and RBAC are enforced at the middleware layer before any handler code executes.
- **Password Safety** — Passwords are hashed with bcrypt and never returned in API responses (`json:"-"` tag).
- **Parameterized Queries** — All database queries use GORM's parameterized interface, preventing SQL injection.
- **Rate Limiting** — IP-based throttling at 100 requests/minute using `x/time/rate`, applied globally before all routes.
- **No Circular Dependencies** — One-directional dependency graph: `cmd → routes → handlers → services → models`.

---

## License

MIT