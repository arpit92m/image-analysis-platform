# Image Analysis Platform Lite

A RESTful API service for image upload and analysis built with Go and the Gin framework.

## Prerequisites

- Go 1.21+
- SQLite3

## Getting Started

```bash
git clone <repo-url>
cd image-analysis-platform
go mod download
go run main.go
```

The server starts on `http://localhost:8080`.

## Health Check

```bash
curl http://localhost:8080/health
```
