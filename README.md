# Image Processing REST API

A high-performance REST API service built with Go 1.25.x for fetching, caching, and transforming images via RESTful endpoints.

[![Go Version](https://img.shields.io/badge/Go-1.25.x-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker)](https://github.com/steviee/github-workflow-article/pkgs/container/github-workflow-article)

## Features

- **Image Fetching**: Download images from any publicly accessible URL
- **Format Conversion**: Automatic conversion to PNG format with transparency support
- **Image Transformations**:
  - Rotation: 90°, 180°, 270° clockwise
  - Resize: Aspect-ratio-preserving crop/zoom to exact dimensions
  - **Multi-operation chaining**: Apply multiple operations in sequence
- **Smart Caching**: In-memory cache with 5-minute idle expiration
- **Error Handling**: Graceful error responses with color-coded placeholder images
- **Observability**: Health checks, readiness probes, and Prometheus metrics
- **Security**: Input validation, size limits, and vulnerability scanning
- **Production Ready**: Dockerized with all dependencies included

## Table of Contents

- [Quick Start](#quick-start)
- [API Documentation](#api-documentation)
- [Architecture](#architecture)
- [Deployment](#deployment)
- [Development](#development)
- [Monitoring](#monitoring)
- [Configuration](#configuration)
- [Contributing](#contributing)

## Quick Start

### Using Docker (Recommended)

```bash
# Pull the latest image
docker pull ghcr.io/steviee/github-workflow-article:latest

# Run the service
docker run -d \
  -p 8080:8080 \
  --name image-api \
  ghcr.io/steviee/github-workflow-article:latest

# Test it
curl "http://localhost:8080/image?url=https://picsum.photos/800/600&op=rotate-90" -o output.png
```

### Using Docker Compose

```yaml
version: '3.8'
services:
  image-api:
    image: ghcr.io/steviee/github-workflow-article:latest
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - CACHE_TTL=5m
      - MAX_IMAGE_SIZE=52428800  # 50MB in bytes
      - MAX_OUTPUT_DIMENSION=1400
    restart: unless-stopped
```

## API Documentation

### Base URL

```
http://localhost:8080
```

### Endpoints

#### `GET /image`

Fetch and transform an image from a URL.

**Query Parameters:**

| Parameter | Type   | Required | Description                                    | Example                      |
|-----------|--------|----------|------------------------------------------------|------------------------------|
| `url`     | string | Yes      | URL of the source image (max 50MB)             | `https://example.com/pic.jpg`|
| `op`      | string | No       | Comma-separated list of operations to apply    | `rotate-90,resize-800x600`   |

**Supported Operations:**

| Operation           | Description                                                                     |
|---------------------|---------------------------------------------------------------------------------|
| `rotate-90`         | Rotate image 90° clockwise                                                      |
| `rotate-180`        | Rotate image 180° clockwise                                                     |
| `rotate-270`        | Rotate image 270° clockwise                                                     |
| `resize-WxH`        | Resize to exact WxH dimensions, maintaining aspect ratio with crop/zoom         |

**Examples:**

```bash
# Simple image fetch (converts to PNG)
curl "http://localhost:8080/image?url=https://picsum.photos/800/600" -o image.png

# Rotate 90 degrees
curl "http://localhost:8080/image?url=https://example.com/photo.jpg&op=rotate-90" -o rotated.png

# Resize to 800x600 (aspect-ratio preserving crop)
curl "http://localhost:8080/image?url=https://example.com/photo.jpg&op=resize-800x600" -o resized.png

# Chain multiple operations (rotate then resize)
curl "http://localhost:8080/image?url=https://example.com/photo.jpg&op=rotate-180,resize-1200x800" -o transformed.png
```

**Response:**
- **Success (200)**: PNG image binary data
- **Error (4xx/5xx)**: PNG placeholder image with error code displayed

#### `GET /health`

Health check endpoint for load balancers.

**Response:**
```json
{
  "status": "healthy"
}
```

#### `GET /ready`

Readiness probe for orchestration systems.

**Response:**
```json
{
  "status": "ready"
}
```

#### `GET /metrics`

Prometheus metrics endpoint for monitoring.

**Exposed Metrics:**
- `http_requests_total`: Total HTTP requests by status code and endpoint
- `http_request_duration_seconds`: HTTP request latency histogram
- `image_cache_hits_total`: Cache hit counter
- `image_cache_misses_total`: Cache miss counter
- `image_cache_size`: Current number of cached images
- `image_processing_duration_seconds`: Image processing time histogram
- `image_fetch_duration_seconds`: Image download time histogram

## Architecture

### System Design

```
┌─────────┐      ┌──────────────────────────────────────┐
│ Client  │─────▶│  Image Processing REST API (Go)      │
└─────────┘      │                                      │
                 │  ┌────────────────────────────────┐  │
                 │  │  HTTP Router (chi)             │  │
                 │  │  - /image   (main endpoint)    │  │
                 │  │  - /health  (health check)     │  │
                 │  │  - /ready   (readiness probe)  │  │
                 │  │  - /metrics (prometheus)       │  │
                 │  └────────────────────────────────┘  │
                 │                 │                     │
                 │                 ▼                     │
                 │  ┌────────────────────────────────┐  │
                 │  │  Cache Layer (sync.Map)        │  │
                 │  │  - 5-minute idle TTL           │  │
                 │  │  - In-memory storage           │  │
                 │  │  - Background cleanup          │  │
                 │  └────────────────────────────────┘  │
                 │                 │                     │
                 │                 ▼                     │
                 │  ┌────────────────────────────────┐  │
                 │  │  Image Processor               │  │
                 │  │  - Fetch (50MB limit)          │  │
                 │  │  - Rotate (90/180/270)         │  │
                 │  │  - Resize (crop/zoom)          │  │
                 │  │  - PNG conversion              │  │
                 │  │  - Error placeholders          │  │
                 │  └────────────────────────────────┘  │
                 └──────────────────────────────────────┘
```

### Technology Stack

- **Language**: Go 1.25.x
- **Image Processing**: ImageMagick (via Go bindings)
- **HTTP Router**: chi/gorilla mux
- **Metrics**: Prometheus client
- **Container**: Docker with Alpine Linux
- **Registry**: GitHub Container Registry (ghcr.io)

### Caching Strategy

The service uses an intelligent in-memory cache:

1. **Cache Key**: `SHA256(url + operations)`
2. **TTL**: 5 minutes idle time (resets on each access)
3. **Eviction**: Background goroutine checks every 30 seconds
4. **Storage**: `sync.Map` for concurrent access
5. **Memory Safety**: No external storage required

### Error Handling

All errors generate placeholder PNG images:

- **4xx Errors** (Client errors):
  - Background: Orange (#FF8C00)
  - Text: White error code and message
  - Examples: 400 Bad Request, 404 Not Found

- **5xx Errors** (Server errors):
  - Background: Red (#DC143C)
  - Text: White error code and message
  - Examples: 500 Internal Server Error, 503 Service Unavailable

Placeholders respect requested dimensions (or default to 400x300).

## Deployment

### Prerequisites

- Docker 20.10+ (for running the container)
- 512MB+ RAM (recommended: 1GB+)
- Network access to fetch external images

### Production Deployment

#### 1. Pull the Image

```bash
docker pull ghcr.io/steviee/github-workflow-article:latest
```

#### 2. Run the Container

**Basic:**
```bash
docker run -d \
  -p 8080:8080 \
  --name image-api \
  --restart unless-stopped \
  ghcr.io/steviee/github-workflow-article:latest
```

**With Custom Configuration:**
```bash
docker run -d \
  -p 8080:8080 \
  --name image-api \
  --restart unless-stopped \
  -e PORT=8080 \
  -e CACHE_TTL=300s \
  -e MAX_IMAGE_SIZE=52428800 \
  -e MAX_OUTPUT_DIMENSION=1400 \
  -e LOG_LEVEL=info \
  --memory=1g \
  --cpus=2 \
  ghcr.io/steviee/github-workflow-article:latest
```

#### 3. Verify Deployment

```bash
# Check health
curl http://localhost:8080/health

# Test image processing
curl "http://localhost:8080/image?url=https://picsum.photos/800/600&op=rotate-90" -o test.png
```

### Environment Variables

| Variable              | Default   | Description                                 |
|-----------------------|-----------|---------------------------------------------|
| `PORT`                | `8080`    | HTTP server port                            |
| `CACHE_TTL`           | `5m`      | Cache idle expiration time                  |
| `MAX_IMAGE_SIZE`      | `52428800`| Max source image size in bytes (50MB)       |
| `MAX_OUTPUT_DIMENSION`| `1400`    | Max output width or height in pixels        |
| `LOG_LEVEL`           | `info`    | Log level (debug, info, warn, error)        |
| `CACHE_CLEANUP_INTERVAL` | `30s` | How often to check for expired cache items  |

### Resource Recommendations

| Deployment Size | CPU    | Memory | Max Concurrent Requests |
|-----------------|--------|--------|-------------------------|
| Small           | 0.5    | 512MB  | ~10                     |
| Medium          | 1.0    | 1GB    | ~50                     |
| Large           | 2.0    | 2GB    | ~100                    |

## Development

### Prerequisites

- Go 1.25.x or later
- Docker 20.10+
- ImageMagick development libraries
- make (optional)

### Setup

1. **Clone the repository:**
```bash
git clone https://github.com/steviee/github-workflow-article.git
cd github-workflow-article
```

2. **Install dependencies:**
```bash
# On Ubuntu/Debian
sudo apt-get update
sudo apt-get install -y libmagickwand-dev

# On macOS
brew install imagemagick

# Install Go dependencies
go mod download
```

3. **Run locally:**
```bash
go run cmd/api/main.go
```

4. **Run tests:**
```bash
# Unit tests
go test ./... -v

# With coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Integration tests
go test ./tests/integration/... -v
```

5. **Build Docker image:**
```bash
docker build -t image-api:dev .
```

### Project Structure

```
.
├── cmd/
│   └── api/
│       └── main.go              # Application entrypoint
├── internal/
│   ├── cache/
│   │   ├── cache.go             # Cache implementation
│   │   └── cache_test.go
│   ├── handler/
│   │   ├── image.go             # Image endpoint handler
│   │   ├── health.go            # Health/ready handlers
│   │   ├── metrics.go           # Metrics handler
│   │   └── handler_test.go
│   ├── processor/
│   │   ├── processor.go         # Image processing logic
│   │   ├── fetcher.go           # Image fetching
│   │   ├── operations.go        # Rotate/resize operations
│   │   ├── placeholder.go       # Error placeholder generator
│   │   └── processor_test.go
│   ├── middleware/
│   │   ├── logging.go           # Request logging
│   │   ├── cors.go              # CORS middleware
│   │   └── metrics.go           # Metrics collection
│   └── config/
│       └── config.go            # Configuration management
├── tests/
│   └── integration/
│       └── api_test.go          # End-to-end tests
├── .github/
│   └── workflows/
│       ├── ci.yml               # CI pipeline
│       └── cd.yml               # CD pipeline
├── Dockerfile                   # Container definition
├── go.mod                       # Go module definition
├── go.sum                       # Go dependencies lock
├── README.md                    # This file
├── CLAUDE.md                    # AI assistant workflow rules
└── LICENSE                      # MIT License
```

### Workflow

See [CLAUDE.md](CLAUDE.md) for detailed development workflow and contribution guidelines.

**Summary:**
1. Work on feature branches: `feature/issue-N-description`
2. Create PRs with comprehensive descriptions
3. All CI checks must pass (tests, lint, security, coverage)
4. Keep documentation updated with every commit
5. Reference GitHub Issues in commits

## Monitoring

### Prometheus Integration

The service exposes Prometheus metrics on `/metrics`:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'image-api'
    static_configs:
      - targets: ['localhost:8080']
    scrape_interval: 15s
```

### Key Metrics to Monitor

1. **Request Rate**: `rate(http_requests_total[5m])`
2. **Error Rate**: `rate(http_requests_total{status=~"5.."}[5m])`
3. **Latency P95**: `histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))`
4. **Cache Hit Rate**: `rate(image_cache_hits_total[5m]) / (rate(image_cache_hits_total[5m]) + rate(image_cache_misses_total[5m]))`
5. **Cache Size**: `image_cache_size`

### Grafana Dashboard

(TODO: Add Grafana dashboard JSON once deployed)

## Configuration

### Performance Tuning

**High Throughput:**
```bash
docker run -d \
  -p 8080:8080 \
  -e CACHE_TTL=10m \
  -e MAX_IMAGE_SIZE=104857600 \
  --memory=4g \
  --cpus=4 \
  ghcr.io/steviee/github-workflow-article:latest
```

**Low Memory:**
```bash
docker run -d \
  -p 8080:8080 \
  -e CACHE_TTL=2m \
  -e MAX_IMAGE_SIZE=26214400 \
  --memory=512m \
  --cpus=1 \
  ghcr.io/steviee/github-workflow-article:latest
```

### Security Considerations

1. **Input Validation**: All URLs are validated before fetching
2. **Size Limits**: Max 50MB source images, 1400x1400 output
3. **Timeout Protection**: Network requests timeout after 30 seconds
4. **Dependency Scanning**: Automated Trivy/Snyk scans on every build
5. **No External Storage**: Everything in-memory, no persistent data

## Contributing

We use GitHub Issues and Pull Requests for all contributions.

1. Check existing [Issues](https://github.com/steviee/github-workflow-article/issues)
2. Create a new issue for your feature/bug
3. Fork and create a feature branch
4. Follow the workflow in [CLAUDE.md](CLAUDE.md)
5. Submit a PR referencing the issue

### Code Quality Requirements

- ✅ 85%+ test coverage
- ✅ Zero golangci-lint errors
- ✅ No high/critical security vulnerabilities
- ✅ All tests passing
- ✅ Documentation updated

## Roadmap

- [ ] WebP output format support
- [ ] Additional operations (blur, sharpen, grayscale)
- [ ] Redis-backed distributed cache
- [ ] Rate limiting per IP
- [ ] Signed URLs for authorized access
- [ ] Batch processing endpoint

## License

MIT License - See [LICENSE](LICENSE) file for details

## Support

- **Issues**: [GitHub Issues](https://github.com/steviee/github-workflow-article/issues)
- **Discussions**: [GitHub Discussions](https://github.com/steviee/github-workflow-article/discussions)

---

**Made with Go and love** ❤️
