# Image Analysis Platform Lite

A RESTful API service for image upload and analysis built with Go and the Gin framework. Supports image metadata CRUD operations, JWT authentication with refresh tokens, and is designed for a high-traffic environment (1M DAU, 10K peak RPS).

## Prerequisites

- Go 1.21+
- GCC (required by SQLite CGo driver)

## Getting Started

```bash
git clone <repo-url>
cd image-analysis-platform
go mod download
go run main.go
```

The server starts on `http://localhost:8081`.

## Configuration

Configuration is done via environment variables:

| Variable     | Default                          | Description              |
|--------------|----------------------------------|--------------------------|
| `PORT`       | `8081`                           | Server port              |
| `DB_PATH`    | `images.db`                      | SQLite database file     |
| `UPLOAD_DIR` | `./uploads`                      | Upload directory path    |
| `JWT_SECRET` | `dev-secret-change-in-production`| JWT signing secret       |

## Running Tests

```bash
go test ./... -v
```

## API Verification and DB Seeding

With the service running, execute the Go smoke-test/seeding script:

```bash
go run ./scripts/api_verify_seed.go
```

By default, the script targets `http://localhost:8081`, verifies every documented endpoint, checks common error paths, and creates seed users plus image metadata rows through the API. The throwaway image used to verify delete behavior is removed, while seed data remains in the DB.

Optional flags:

```bash
go run ./scripts/api_verify_seed.go -base-url http://localhost:8081 -seed-users 3 -seed-images 4
```

## API Endpoints

### Public

| Method | Endpoint              | Description            |
|--------|-----------------------|------------------------|
| GET    | `/health`             | Health check           |
| POST   | `/api/v1/auth/register` | Register a new user  |
| POST   | `/api/v1/auth/login`    | Login, get tokens    |
| POST   | `/api/v1/auth/refresh`  | Refresh access token |

### Protected (requires `Authorization: Bearer <token>`)

| Method | Endpoint                       | Description              |
|--------|--------------------------------|--------------------------|
| POST   | `/api/v1/images`               | Upload image metadata    |
| GET    | `/api/v1/images?user_id=X`     | List images for a user   |
| GET    | `/api/v1/images/:id`           | Get image details        |
| PUT    | `/api/v1/images/:id`           | Update image metadata    |
| DELETE | `/api/v1/images/:id`           | Delete an image          |
| GET    | `/api/v1/images/:id/download`  | Get download URL         |

## Quick Start Example

```bash
# Register
curl -X POST http://localhost:8081/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username": "demo", "password": "demo123"}'

# Login
TOKEN=$(curl -s -X POST http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "demo", "password": "demo123"}' | jq -r '.access_token')

# Upload image metadata
curl -X POST http://localhost:8081/api/v1/images \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "user_id": "user-1",
    "original_filename": "photo.jpg",
    "width": 1920, "height": 1080,
    "file_size": 2048000,
    "file_type": "image/jpeg"
  }'

# List images
curl http://localhost:8081/api/v1/images?user_id=user-1 \
  -H "Authorization: Bearer $TOKEN"
```

## Project Structure

```
.
├── main.go              # Application entry point
├── config/
│   └── config.go        # Environment-based configuration
├── database/
│   └── db.go            # GORM + SQLite initialization
├── models/
│   ├── image.go         # Image model and request structs
│   └── user.go          # User model and auth request structs
├── handlers/
│   ├── image.go         # Image CRUD handlers
│   ├── auth.go          # Auth handlers (register, login, refresh)
│   ├── image_test.go    # Image handler tests
│   ├── auth_test.go     # Auth handler tests
│   └── testutil_test.go # Test helpers
├── middleware/
│   └── auth.go          # JWT middleware
├── routes/
│   └── routes.go        # Route definitions
└── docs/
    ├── api.md           # Full API documentation
    ├── architecture.md  # System architecture & diagrams
    └── schema.md        # Database schema & ER diagram
```

## Documentation

- **[API Reference](docs/api.md)** — Full endpoint documentation with examples
- **[Architecture](docs/architecture.md)** — System design, component diagrams, scaling strategy
- **[Database Schema](docs/schema.md)** — ER diagram, table definitions, design decisions

## Design Decisions & Assumptions

1. **Metadata-only upload** — The API stores image metadata. Actual file uploads would go directly to S3/GCS via pre-signed URLs in production.
2. **SQLite for development** — Swap to PostgreSQL in production by changing the GORM driver. No code changes needed.
3. **JWT auth** — Access tokens expire in 15 minutes; refresh tokens in 7 days. Secret is configurable via env var.
4. **Pagination** — List endpoint supports `page` and `per_page` query params (default 20, max 100).
5. **Analysis status** — Set to `pending` on upload. In production, updated by async workers via a message queue.
6. **No actual file handling** — The download endpoint returns a simulated URL. In production, it would generate a pre-signed S3 URL.

## Tech Stack

- **Language:** Go
- **Framework:** Gin
- **ORM:** GORM
- **Database:** SQLite (dev) / PostgreSQL (prod)
- **Auth:** JWT (golang-jwt/jwt)
- **Password hashing:** bcrypt
