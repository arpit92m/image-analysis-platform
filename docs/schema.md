# Database Schema

## ER Diagram

```
┌─────────────────────────────────┐         ┌─────────────────────────────────┐
│            users                │         │            images               │
├─────────────────────────────────┤         ├─────────────────────────────────┤
│ id          INTEGER  PK         │         │ id                INTEGER  PK   │
│ username    TEXT     UNIQUE, NN  │────┐    │ user_id           TEXT     NN   │
│ password    TEXT     NN          │    │    │ original_filename TEXT     NN   │
│ created_at  DATETIME             │    │    │ upload_date       DATETIME      │
│ updated_at  DATETIME             │    └───▶│ width             INTEGER       │
└─────────────────────────────────┘         │ height            INTEGER       │
                                            │ file_size         INTEGER       │
                                            │ file_type         TEXT          │
                                            │ storage_path      TEXT          │
                                            │ analysis_status   TEXT  DEFAULT │
                                            │                   'pending'     │
                                            │ created_at        DATETIME      │
                                            │ updated_at        DATETIME      │
                                            └─────────────────────────────────┘
```

## Table: `users`

| Column     | Type     | Constraints        | Description                    |
|------------|----------|--------------------|--------------------------------|
| id         | INTEGER  | PRIMARY KEY, AUTO  | Unique user identifier         |
| username   | TEXT     | UNIQUE, NOT NULL   | Login username (3-50 chars)    |
| password   | TEXT     | NOT NULL           | bcrypt hashed password         |
| created_at | DATETIME | AUTO               | Account creation timestamp     |
| updated_at | DATETIME | AUTO               | Last update timestamp          |

## Table: `images`

| Column            | Type     | Constraints           | Description                         |
|-------------------|----------|-----------------------|-------------------------------------|
| id                | INTEGER  | PRIMARY KEY, AUTO     | Unique image identifier             |
| user_id           | TEXT     | NOT NULL, INDEX       | Owner's user ID                     |
| original_filename | TEXT     | NOT NULL              | Original name of the uploaded file  |
| upload_date       | DATETIME | AUTO                  | When the image was uploaded         |
| width             | INTEGER  |                       | Image width in pixels               |
| height            | INTEGER  |                       | Image height in pixels              |
| file_size         | INTEGER  |                       | File size in bytes                  |
| file_type         | TEXT     |                       | MIME type (image/jpeg, image/png…)  |
| storage_path      | TEXT     |                       | Path or key in object storage       |
| analysis_status   | TEXT     | DEFAULT 'pending'     | pending, processing, completed      |
| created_at        | DATETIME | AUTO                  | Row creation timestamp              |
| updated_at        | DATETIME | AUTO                  | Last modification timestamp         |

## Indexes

- `idx_images_user_id` on `images(user_id)` — fast lookup of images by user
- `idx_users_username` on `users(username)` — unique constraint + fast login lookup

## Design Decisions

1. **SQLite for development** — Easy setup with no external dependencies. In production, swap to PostgreSQL via GORM driver change.
2. **`user_id` as TEXT** — Allows flexibility to use UUIDs or external identity provider IDs.
3. **`storage_path`** — Stores the object storage key. The API generates pre-signed URLs on the fly rather than storing full URLs.
4. **`analysis_status`** — Tracks the async analysis pipeline state. Updated by workers via message queue consumers.
5. **Soft deletes not used** — GORM soft delete was considered but kept simple with hard deletes for this implementation.
