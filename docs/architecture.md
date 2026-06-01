# System Architecture

## High-Level Architecture Diagram

```
┌─────────────┐     ┌──────────────────────────────────────────────────────┐
│   Mobile /  │     │                   Load Balancer                     │
│   Web App   │────▶│              (AWS ALB / Nginx)                      │
│   Clients   │     │         Round-robin / least-connections             │
└─────────────┘     └──────────────┬────────────────┬──────────────────────┘
                                   │                │
                    ┌──────────────▼──┐   ┌────────▼───────────┐
                    │  API Server #1  │   │  API Server #N     │
                    │  (Go + Gin)     │   │  (Go + Gin)        │
                    │  - Auth (JWT)   │   │  - Auth (JWT)      │
                    │  - CRUD APIs    │   │  - CRUD APIs       │
                    │  - Validation   │   │  - Validation      │
                    └──┬────┬────┬───┘   └──┬────┬────┬───────┘
                       │    │    │           │    │    │
          ┌────────────▼┐   │   ┌▼──────────▼┐   │   │
          │  Database    │   │   │  Object     │   │   │
          │  (PostgreSQL │   │   │  Storage    │   │   │
          │   Primary +  │   │   │  (S3/GCS)   │   │   │
          │   Replicas)  │   │   │             │   │   │
          └──────────────┘   │   └─────────────┘   │   │
                             │                     │   │
                    ┌────────▼─────────────────────▼┐  │
                    │       Message Queue            │  │
                    │     (SQS / RabbitMQ)           │  │
                    └────────┬──────────────────────┘  │
                             │                         │
                    ┌────────▼──────────┐    ┌────────▼────────┐
                    │  Image Analysis   │    │   Cache Layer   │
                    │  Workers          │    │   (Redis)       │
                    │  (Auto-scaling)   │    │                 │
                    └───────────────────┘    └─────────────────┘
```

## Component Descriptions

### 1. API Layer (Go + Gin)
- Stateless HTTP REST API servers behind a load balancer
- Handles authentication via JWT (access + refresh tokens)
- Validates requests, manages image metadata CRUD
- Horizontally scalable — add more instances to handle higher RPS
- Target: 10K peak RPS spread across N instances

### 2. Database (PostgreSQL)
- Stores image metadata (dimensions, file type, upload date, user info)
- Stores user accounts and credentials
- Primary-replica setup for read scaling (10:1 read-write ratio)
  - Writes go to primary
  - Reads distributed across replicas
- Indexed on `user_id` and `upload_date` for fast queries
- Connection pooling via PgBouncer

### 3. Object Storage (S3 / GCS)
- Stores actual image files (binary data)
- Pre-signed URLs for secure, direct client uploads and downloads
- Lifecycle policies for cost optimization
- CDN (CloudFront/Cloud CDN) in front for low-latency downloads
- Decouples file storage from the API layer

### 4. Cache Layer (Redis)
- Caches frequently accessed image metadata and user sessions
- Reduces database load for the heavy read workload
- TTL-based eviction (e.g., 5-minute TTL for image details)
- Helps meet P99 < 100ms latency target

### 5. Message Queue (SQS / RabbitMQ)
- Decouples image upload from analysis processing
- Upload API publishes a message; workers consume asynchronously
- Provides resilience — if analysis workers are down, messages queue up
- Enables notification to users when analysis completes

### 6. Image Analysis Workers
- Consume messages from the queue
- Perform image analysis (classification, tagging, OCR, etc.)
- Update analysis results in the database
- Auto-scale based on queue depth
- Push notification (WebSocket/SSE/Push) to users on completion

## Request Flow

### Upload Flow
```
Client → Load Balancer → API Server → Validate & Save metadata to DB
                                    → Return pre-signed upload URL
Client → Upload file directly to S3 using pre-signed URL
S3 Event → Message Queue → Analysis Worker → Update DB with results
                                           → Notify user via WebSocket/Push
```

### Read Flow
```
Client → Load Balancer → API Server → Check Redis cache
                                    → Cache miss: Query DB replica
                                    → Return metadata
```

### Download Flow
```
Client → Load Balancer → API Server → Generate pre-signed S3 URL
                                    → Return URL to client
Client → Download directly from S3/CDN
```

## Scaling Considerations

| Metric          | Strategy                                                |
|-----------------|---------------------------------------------------------|
| 1M DAU          | Horizontal scaling of API servers behind ALB            |
| 10K peak RPS    | 5-10 API instances + Redis cache for reads              |
| 10:1 read/write | DB read replicas + Redis cache layer                    |
| P99 < 100ms     | Redis caching, DB indexing, CDN for file downloads      |
| Availability    | Multi-AZ deployment, auto-scaling groups, health checks |
| No SPOF         | Replicated DB, clustered Redis, multi-AZ load balancer  |

## Notifying Users on Analysis Completion

When image analysis completes, the worker can notify users via:

1. **WebSocket** — Persistent connection for real-time updates
2. **Server-Sent Events (SSE)** — Simpler one-way push from server
3. **Push Notifications** — For mobile clients via FCM/APNs
4. **Polling** — Client periodically checks `GET /images/:id` for `analysis_status` change

The recommended approach for a mobile-first platform is **push notifications** via FCM/APNs, with a WebSocket fallback for web clients.
