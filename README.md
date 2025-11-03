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

The image is available on GitHub Container Registry with multi-architecture support:

```bash
# Pull latest version (recommended)
docker pull ghcr.io/steviee/github-workflow-article:latest

# Pull specific version
docker pull ghcr.io/steviee/github-workflow-article:v1.0.0

# Pull specific minor version (gets latest patch)
docker pull ghcr.io/steviee/github-workflow-article:v1.0

# Pull specific major version (gets latest minor.patch)
docker pull ghcr.io/steviee/github-workflow-article:v1
```

**Multi-Architecture Support:**

The image supports multiple architectures and Docker will automatically pull the correct one:
- `linux/amd64` (x86_64) - Standard Intel/AMD servers
- `linux/arm64` (aarch64) - ARM64 servers, AWS Graviton, Apple Silicon

```bash
# Verify image architecture
docker image inspect ghcr.io/steviee/github-workflow-article:latest \
  | grep Architecture

# Explicitly pull for specific platform (if needed)
docker pull --platform linux/amd64 ghcr.io/steviee/github-workflow-article:latest
docker pull --platform linux/arm64 ghcr.io/steviee/github-workflow-article:latest
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
# 1. Check container is running
docker ps | grep image-api

# 2. Check container logs
docker logs image-api

# 3. Health check
curl http://localhost:8080/health
# Expected: {"status":"healthy"}

# 4. Readiness check
curl http://localhost:8080/ready
# Expected: {"status":"ready"}

# 5. Test basic image fetch
curl "http://localhost:8080/image?url=https://picsum.photos/400/300" -o test-basic.png
file test-basic.png
# Expected: test-basic.png: PNG image data, 400 x 300, 8-bit/color RGBA

# 6. Test image processing (rotation)
curl "http://localhost:8080/image?url=https://picsum.photos/800/600&op=rotate-90" -o test-rotated.png

# 7. Test cache (second request should be faster)
time curl "http://localhost:8080/image?url=https://picsum.photos/800/600&op=rotate-90" -o /dev/null
time curl "http://localhost:8080/image?url=https://picsum.photos/800/600&op=rotate-90" -o /dev/null

# 8. Verify metrics are exposed
curl -s http://localhost:8080/metrics | grep http_requests_total

# 9. Check cache hit/miss headers
curl -v "http://localhost:8080/image?url=https://picsum.photos/800/600&op=rotate-90" 2>&1 | grep X-Cache
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

### Docker Compose Deployment

#### Basic Docker Compose

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  image-api:
    image: ghcr.io/steviee/github-workflow-article:latest
    container_name: image-api
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - CACHE_TTL=5m
      - MAX_IMAGE_SIZE=52428800
      - MAX_OUTPUT_DIMENSION=1400
      - LOG_LEVEL=info
      - CACHE_CLEANUP_INTERVAL=30s
    restart: unless-stopped
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 1G
        reservations:
          cpus: '0.5'
          memory: 512M
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

#### Using Environment File

Create `.env` file:

```bash
# .env
# Server Configuration
PORT=8080
LOG_LEVEL=info

# Cache Configuration
CACHE_TTL=5m
CACHE_CLEANUP_INTERVAL=30s

# Image Processing Limits
MAX_IMAGE_SIZE=52428800
MAX_OUTPUT_DIMENSION=1400
```

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  image-api:
    image: ghcr.io/steviee/github-workflow-article:latest
    container_name: image-api
    ports:
      - "${PORT:-8080}:${PORT:-8080}"
    env_file:
      - .env
    restart: unless-stopped
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 1G
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:${PORT:-8080}/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

**Run with Docker Compose:**

```bash
# Start services
docker compose up -d

# View logs
docker compose logs -f

# Check status
docker compose ps

# Stop services
docker compose down

# Stop and remove volumes (if any)
docker compose down -v
```

#### Multi-Service Setup with Monitoring

Create `docker-compose.monitoring.yml`:

```yaml
version: '3.8'

services:
  image-api:
    image: ghcr.io/steviee/github-workflow-article:latest
    container_name: image-api
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - CACHE_TTL=5m
      - LOG_LEVEL=info
    restart: unless-stopped
    networks:
      - monitoring
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  prometheus:
    image: prom/prometheus:latest
    container_name: prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prometheus-data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'
    restart: unless-stopped
    networks:
      - monitoring

  grafana:
    image: grafana/grafana:latest
    container_name: grafana
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
      - GF_USERS_ALLOW_SIGN_UP=false
    volumes:
      - grafana-data:/var/lib/grafana
    restart: unless-stopped
    networks:
      - monitoring
    depends_on:
      - prometheus

networks:
  monitoring:
    driver: bridge

volumes:
  prometheus-data:
  grafana-data:
```

Create `prometheus.yml`:

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'image-api'
    static_configs:
      - targets: ['image-api:8080']
    scrape_interval: 15s
    scrape_timeout: 10s
```

**Run multi-service stack:**

```bash
# Start all services
docker compose -f docker-compose.monitoring.yml up -d

# Access services:
# - Image API: http://localhost:8080
# - Prometheus: http://localhost:9090
# - Grafana: http://localhost:3000 (admin/admin)
```

### Production Best Practices

#### Scaling Strategies

**Horizontal Scaling with Docker Swarm:**

```bash
# Initialize swarm
docker swarm init

# Deploy stack with replicas
docker service create \
  --name image-api \
  --replicas 3 \
  --publish 8080:8080 \
  --env CACHE_TTL=5m \
  --limit-cpu 2 \
  --limit-memory 1g \
  ghcr.io/steviee/github-workflow-article:latest

# Scale up/down
docker service scale image-api=5

# Check service status
docker service ps image-api
```

**Horizontal Scaling with Kubernetes:**

```yaml
# k8s-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: image-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: image-api
  template:
    metadata:
      labels:
        app: image-api
    spec:
      containers:
      - name: image-api
        image: ghcr.io/steviee/github-workflow-article:latest
        ports:
        - containerPort: 8080
        env:
        - name: CACHE_TTL
          value: "5m"
        - name: LOG_LEVEL
          value: "info"
        resources:
          limits:
            cpu: "2"
            memory: "1Gi"
          requests:
            cpu: "500m"
            memory: "512Mi"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: image-api
spec:
  selector:
    app: image-api
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

**Load Balancing with Nginx:**

Create `nginx.conf`:

```nginx
upstream image_api {
    least_conn;
    server localhost:8081;
    server localhost:8082;
    server localhost:8083;
}

server {
    listen 80;
    server_name api.example.com;

    location / {
        proxy_pass http://image_api;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_http_version 1.1;
        proxy_read_timeout 90s;
    }

    location /health {
        access_log off;
        proxy_pass http://image_api;
    }
}
```

Run multiple instances:

```bash
docker run -d -p 8081:8080 --name image-api-1 ghcr.io/steviee/github-workflow-article:latest
docker run -d -p 8082:8080 --name image-api-2 ghcr.io/steviee/github-workflow-article:latest
docker run -d -p 8083:8080 --name image-api-3 ghcr.io/steviee/github-workflow-article:latest

# Run Nginx
docker run -d -p 80:80 -v $(pwd)/nginx.conf:/etc/nginx/nginx.conf:ro nginx:alpine
```

#### Monitoring Integration

**Centralized Logging with Syslog:**

```bash
docker run -d \
  -p 8080:8080 \
  --name image-api \
  --log-driver=syslog \
  --log-opt syslog-address=udp://logs.example.com:514 \
  --log-opt tag="image-api" \
  ghcr.io/steviee/github-workflow-article:latest
```

**JSON Logging for Aggregation:**

```bash
docker run -d \
  -p 8080:8080 \
  --name image-api \
  --log-driver=json-file \
  --log-opt max-size=10m \
  --log-opt max-file=3 \
  ghcr.io/steviee/github-workflow-article:latest
```

**Integration with ELK Stack:**

```yaml
# docker-compose.elk.yml
version: '3.8'

services:
  image-api:
    image: ghcr.io/steviee/github-workflow-article:latest
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
    labels:
      - "co.elastic.logs/enabled=true"
      - "co.elastic.logs/json.keys_under_root=true"
```

#### Security Hardening

```bash
# Run as non-root user (image already configured)
# Read-only root filesystem
# Drop all capabilities except necessary ones
docker run -d \
  -p 8080:8080 \
  --name image-api \
  --read-only \
  --tmpfs /tmp:rw,noexec,nosuid,size=100m \
  --security-opt=no-new-privileges:true \
  --cap-drop=ALL \
  --cap-add=NET_BIND_SERVICE \
  ghcr.io/steviee/github-workflow-article:latest
```

#### Health Monitoring

**Docker Health Checks:**

```bash
docker run -d \
  -p 8080:8080 \
  --name image-api \
  --health-cmd="wget --quiet --tries=1 --spider http://localhost:8080/health || exit 1" \
  --health-interval=30s \
  --health-timeout=10s \
  --health-retries=3 \
  --health-start-period=40s \
  ghcr.io/steviee/github-workflow-article:latest

# Check health status
docker inspect --format='{{.State.Health.Status}}' image-api
```

**Automated Restarts on Unhealthy:**

```yaml
# docker-compose.yml with auto-restart
version: '3.8'

services:
  image-api:
    image: ghcr.io/steviee/github-workflow-article:latest
    restart: on-failure:3
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

#### Backup and Disaster Recovery

Since the service is stateless (all data in-memory cache):

**No backup required** - Service can be restarted anytime without data loss

**Recovery Procedure:**

```bash
# 1. Pull latest image
docker pull ghcr.io/steviee/github-workflow-article:latest

# 2. Stop old container
docker stop image-api

# 3. Remove old container
docker rm image-api

# 4. Start new container
docker run -d -p 8080:8080 --name image-api ghcr.io/steviee/github-workflow-article:latest

# 5. Verify health
curl http://localhost:8080/health
```

**Zero-Downtime Updates:**

```bash
# Run new version on different port
docker run -d -p 8081:8080 --name image-api-new ghcr.io/steviee/github-workflow-article:latest

# Verify new version
curl http://localhost:8081/health

# Update load balancer to point to new instance
# (or use Docker Swarm/Kubernetes rolling updates)

# Stop old version
docker stop image-api
docker rm image-api

# Rename new version
docker rename image-api-new image-api
```

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
