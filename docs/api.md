# API Documentation

Base URL: `http://localhost:8081/api/v1`

---

## Authentication

### Register

```
POST /auth/register
```

**Request Body:**
```json
{
  "username": "john_doe",
  "password": "securepass123"
}
```

**Response (201):**
```json
{
  "id": 1,
  "username": "john_doe",
  "message": "User registered successfully"
}
```

**Errors:**
- `400` - Validation error (username min 3 chars, password min 6 chars)
- `409` - Username already taken

---

### Login

```
POST /auth/login
```

**Request Body:**
```json
{
  "username": "john_doe",
  "password": "securepass123"
}
```

**Response (200):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_in": 900
}
```

**Errors:**
- `401` - Invalid username or password

---

### Refresh Token

```
POST /auth/refresh
```

**Request Body:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Response (200):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_in": 900
}
```

**Errors:**
- `401` - Invalid or expired refresh token

---

## Images

> All image endpoints require authentication.
> Include header: `Authorization: Bearer <access_token>`

### Upload Image (Metadata)

```
POST /images
```

**Request Body:**
```json
{
  "user_id": "user-123",
  "original_filename": "vacation.jpg",
  "width": 1920,
  "height": 1080,
  "file_size": 2048000,
  "file_type": "image/jpeg"
}
```

**Supported file types:** `image/jpeg`, `image/png`, `image/gif`, `image/webp`

**Response (201):**
```json
{
  "id": 1,
  "user_id": "user-123",
  "original_filename": "vacation.jpg",
  "upload_date": "2024-01-15T10:30:00Z",
  "width": 1920,
  "height": 1080,
  "file_size": 2048000,
  "file_type": "image/jpeg",
  "storage_path": "uploads/user-123/uuid-string",
  "analysis_status": "pending",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

**Errors:**
- `400` - Validation error
- `401` - Unauthorized

---

### List Images

```
GET /images?user_id=user-123&page=1&per_page=20
```

**Query Parameters:**
| Parameter  | Required | Default | Description                    |
|------------|----------|---------|--------------------------------|
| `user_id`  | Yes      | -       | Filter by user ID              |
| `page`     | No       | 1       | Page number                    |
| `per_page` | No       | 20      | Items per page (max 100)       |

**Response (200):**
```json
{
  "images": [...],
  "total": 42,
  "page": 1,
  "per_page": 20
}
```

---

### Get Image Details

```
GET /images/:id
```

**Response (200):** Returns the full image metadata object.

**Errors:**
- `404` - Image not found

---

### Update Image Metadata

```
PUT /images/:id
```

**Request Body (all fields optional):**
```json
{
  "original_filename": "renamed.jpg",
  "width": 1024,
  "height": 768
}
```

**Response (200):** Returns the updated image metadata.

**Errors:**
- `400` - No fields to update or validation error
- `404` - Image not found

---

### Delete Image

```
DELETE /images/:id
```

**Response (200):**
```json
{
  "message": "Image deleted successfully"
}
```

**Errors:**
- `404` - Image not found

---

### Download Image

```
GET /images/:id/download
```

**Response (200):**
```json
{
  "image_id": 1,
  "filename": "vacation.jpg",
  "download_url": "/files/uploads/user-123/uuid-string",
  "file_size": 2048000,
  "file_type": "image/jpeg"
}
```

> **Note:** In production, `download_url` would be a pre-signed URL from S3/GCS.

**Errors:**
- `404` - Image not found
