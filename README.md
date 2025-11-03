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
- [Usage Examples](#usage-examples)
- [Architecture](#architecture)
- [Deployment](#deployment)
- [Development](#development)
- [Monitoring](#monitoring)
- [Configuration](#configuration)
- [Troubleshooting](#troubleshooting)
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

## Usage Examples

### Comprehensive cURL Examples

#### Basic Image Fetching

```bash
# Fetch and convert to PNG (no operations)
curl "http://localhost:8080/image?url=https://picsum.photos/800/600" \
  -o image.png

# Verify cache MISS on first request
curl -v "http://localhost:8080/image?url=https://picsum.photos/800/600&op=rotate-90" \
  -o rotated.png 2>&1 | grep "X-Cache: MISS"

# Verify cache HIT on second request
curl -v "http://localhost:8080/image?url=https://picsum.photos/800/600&op=rotate-90" \
  -o rotated.png 2>&1 | grep "X-Cache: HIT"
```

#### Rotation Operations

```bash
# Rotate 90 degrees clockwise
curl "http://localhost:8080/image?url=https://picsum.photos/1200/800&op=rotate-90" \
  -o rotate-90.png

# Rotate 180 degrees
curl "http://localhost:8080/image?url=https://picsum.photos/1000/1000&op=rotate-180" \
  -o rotate-180.png

# Rotate 270 degrees (90 degrees counter-clockwise)
curl "http://localhost:8080/image?url=https://picsum.photos/600/400&op=rotate-270" \
  -o rotate-270.png
```

#### Resize Operations

```bash
# Resize to 800x600 (aspect-ratio preserving crop/zoom)
curl "http://localhost:8080/image?url=https://picsum.photos/1920/1080&op=resize-800x600" \
  -o resized-800x600.png

# Resize to square dimensions
curl "http://localhost:8080/image?url=https://picsum.photos/1600/900&op=resize-500x500" \
  -o square-500.png

# Resize to maximum allowed dimensions
curl "http://localhost:8080/image?url=https://picsum.photos/3000/2000&op=resize-1400x1400" \
  -o max-size.png
```

#### Complex Operation Chains

```bash
# Rotate then resize
curl "http://localhost:8080/image?url=https://picsum.photos/1920/1080&op=rotate-90,resize-800x600" \
  -o rotated-then-resized.png

# Multiple rotations (net 180 degrees)
curl "http://localhost:8080/image?url=https://picsum.photos/1000/800&op=rotate-90,rotate-90" \
  -o double-rotated.png

# Resize then rotate (portrait from landscape)
curl "http://localhost:8080/image?url=https://picsum.photos/1600/900&op=resize-600x800,rotate-270" \
  -o portrait.png

# Complex chain: resize, rotate, resize again
curl "http://localhost:8080/image?url=https://picsum.photos/2000/1500&op=resize-1000x1000,rotate-90,resize-800x600" \
  -o complex-chain.png
```

#### Error Scenarios

```bash
# Invalid URL (400 Bad Request - Orange placeholder)
curl "http://localhost:8080/image?url=not-a-valid-url" \
  -o error-400.png

# Missing URL parameter (400 Bad Request)
curl "http://localhost:8080/image" \
  -o error-missing-url.png

# Invalid operation (400 Bad Request)
curl "http://localhost:8080/image?url=https://picsum.photos/800/600&op=invalid-op" \
  -o error-invalid-op.png

# Image too large (400 Bad Request)
curl "http://localhost:8080/image?url=https://example.com/huge-100mb-image.jpg" \
  -o error-too-large.png

# Unreachable URL (500 Internal Server Error - Red placeholder)
curl "http://localhost:8080/image?url=https://nonexistent-domain-12345.com/image.jpg" \
  -o error-500.png
```

### JavaScript/Fetch Example

```javascript
/**
 * Fetch and process an image using the Image API
 * @param {string} imageUrl - Source image URL
 * @param {string[]} operations - Array of operations to apply
 * @returns {Promise<Blob>} Processed image as PNG blob
 */
async function fetchProcessedImage(imageUrl, operations = []) {
  const apiUrl = 'http://localhost:8080/image';
  const params = new URLSearchParams({
    url: imageUrl
  });

  if (operations.length > 0) {
    params.append('op', operations.join(','));
  }

  const response = await fetch(`${apiUrl}?${params}`);

  if (!response.ok) {
    // API returns error placeholders as PNG images
    console.warn(`API returned status ${response.status}`);
  }

  const cacheStatus = response.headers.get('X-Cache') || 'SKIP';
  console.log(`Cache status: ${cacheStatus}`);

  return await response.blob();
}

// Usage examples
async function examples() {
  // Simple fetch
  const simple = await fetchProcessedImage('https://picsum.photos/800/600');

  // With rotation
  const rotated = await fetchProcessedImage(
    'https://picsum.photos/1200/800',
    ['rotate-90']
  );

  // Complex operations
  const complex = await fetchProcessedImage(
    'https://picsum.photos/1920/1080',
    ['rotate-180', 'resize-800x600']
  );

  // Display in browser
  const imgElement = document.createElement('img');
  imgElement.src = URL.createObjectURL(complex);
  document.body.appendChild(imgElement);
}

// Download processed image
async function downloadImage(imageUrl, operations, filename) {
  const blob = await fetchProcessedImage(imageUrl, operations);
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

// Example: Download rotated and resized image
downloadImage(
  'https://picsum.photos/1600/900',
  ['rotate-90', 'resize-800x600'],
  'processed-image.png'
);
```

### Python/Requests Example

```python
#!/usr/bin/env python3
"""
Image Processing API Client - Python Example

Usage:
    python image_client.py
"""

import requests
from typing import List, Optional
from pathlib import Path


class ImageAPIClient:
    """Client for the Image Processing REST API"""

    def __init__(self, base_url: str = "http://localhost:8080"):
        self.base_url = base_url
        self.session = requests.Session()
        self.session.headers.update({
            'User-Agent': 'ImageAPIClient/1.0 (Python)'
        })

    def fetch_image(
        self,
        url: str,
        operations: Optional[List[str]] = None,
        output_path: Optional[Path] = None
    ) -> bytes:
        """
        Fetch and process an image

        Args:
            url: Source image URL
            operations: List of operations to apply (e.g., ['rotate-90', 'resize-800x600'])
            output_path: Optional path to save the image

        Returns:
            Image data as bytes (PNG format)

        Raises:
            requests.HTTPError: If the request fails
        """
        params = {'url': url}
        if operations:
            params['op'] = ','.join(operations)

        response = self.session.get(
            f"{self.base_url}/image",
            params=params,
            timeout=30
        )

        # Log cache status
        cache_status = response.headers.get('X-Cache', 'SKIP')
        print(f"Cache status: {cache_status}")

        # API returns error placeholders as images, check status
        if response.status_code >= 400:
            print(f"Warning: API returned status {response.status_code}")

        response.raise_for_status()

        image_data = response.content

        # Save to file if path provided
        if output_path:
            output_path.write_bytes(image_data)
            print(f"Image saved to: {output_path}")

        return image_data

    def health_check(self) -> bool:
        """Check if the API is healthy"""
        try:
            response = self.session.get(f"{self.base_url}/health", timeout=5)
            return response.status_code == 200
        except requests.RequestException:
            return False


def main():
    """Example usage"""
    client = ImageAPIClient()

    # Check API health
    if not client.health_check():
        print("ERROR: API is not healthy!")
        return

    print("API is healthy ✓\n")

    # Example 1: Simple fetch
    print("Example 1: Fetching image...")
    client.fetch_image(
        url="https://picsum.photos/800/600",
        output_path=Path("simple.png")
    )

    # Example 2: Rotate 90 degrees
    print("\nExample 2: Rotating image 90°...")
    client.fetch_image(
        url="https://picsum.photos/1200/800",
        operations=["rotate-90"],
        output_path=Path("rotated-90.png")
    )

    # Example 3: Resize
    print("\nExample 3: Resizing image...")
    client.fetch_image(
        url="https://picsum.photos/1920/1080",
        operations=["resize-800x600"],
        output_path=Path("resized.png")
    )

    # Example 4: Complex operation chain
    print("\nExample 4: Complex operations (rotate + resize)...")
    client.fetch_image(
        url="https://picsum.photos/1600/900",
        operations=["rotate-180", "resize-1000x1000"],
        output_path=Path("complex.png")
    )

    # Example 5: Cache test (second request should be cached)
    print("\nExample 5: Testing cache (making same request twice)...")
    client.fetch_image(
        url="https://picsum.photos/800/600",
        operations=["rotate-90"]
    )
    print("Making same request again...")
    client.fetch_image(
        url="https://picsum.photos/800/600",
        operations=["rotate-90"]
    )

    print("\n✓ All examples completed successfully!")


if __name__ == "__main__":
    main()
```

### Go/net/http Example

```go
package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// ImageAPIClient is a client for the Image Processing REST API
type ImageAPIClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewImageAPIClient creates a new API client
func NewImageAPIClient(baseURL string) *ImageAPIClient {
	return &ImageAPIClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchImage fetches and processes an image
func (c *ImageAPIClient) FetchImage(imageURL string, operations []string, outputPath string) error {
	// Build request URL
	reqURL := fmt.Sprintf("%s/image?url=%s", c.BaseURL, url.QueryEscape(imageURL))
	if len(operations) > 0 {
		reqURL += "&op=" + url.QueryEscape(strings.Join(operations, ","))
	}

	// Make request
	resp, err := c.HTTPClient.Get(reqURL)
	if err != nil {
		return fmt.Errorf("failed to fetch image: %w", err)
	}
	defer resp.Body.Close()

	// Log cache status
	cacheStatus := resp.Header.Get("X-Cache")
	if cacheStatus == "" {
		cacheStatus = "SKIP"
	}
	fmt.Printf("Cache status: %s\n", cacheStatus)

	// Check status code (API returns error placeholders as images)
	if resp.StatusCode >= 400 {
		fmt.Printf("Warning: API returned status %d\n", resp.StatusCode)
	}

	// Save to file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write image data: %w", err)
	}

	fmt.Printf("Image saved to: %s\n", outputPath)
	return nil
}

// HealthCheck checks if the API is healthy
func (c *ImageAPIClient) HealthCheck() (bool, error) {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/health")
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200, nil
}

func main() {
	client := NewImageAPIClient("http://localhost:8080")

	// Check API health
	healthy, err := client.HealthCheck()
	if err != nil || !healthy {
		fmt.Println("ERROR: API is not healthy!")
		os.Exit(1)
	}
	fmt.Println("API is healthy ✓\n")

	// Example 1: Simple fetch
	fmt.Println("Example 1: Fetching image...")
	err = client.FetchImage(
		"https://picsum.photos/800/600",
		nil,
		"simple.png",
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// Example 2: Rotate 90 degrees
	fmt.Println("\nExample 2: Rotating image 90°...")
	err = client.FetchImage(
		"https://picsum.photos/1200/800",
		[]string{"rotate-90"},
		"rotated-90.png",
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// Example 3: Resize
	fmt.Println("\nExample 3: Resizing image...")
	err = client.FetchImage(
		"https://picsum.photos/1920/1080",
		[]string{"resize-800x600"},
		"resized.png",
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// Example 4: Complex operation chain
	fmt.Println("\nExample 4: Complex operations (rotate + resize)...")
	err = client.FetchImage(
		"https://picsum.photos/1600/900",
		[]string{"rotate-180", "resize-1000x1000"},
		"complex.png",
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// Example 5: Cache test
	fmt.Println("\nExample 5: Testing cache (making same request twice)...")
	_ = client.FetchImage(
		"https://picsum.photos/800/600",
		[]string{"rotate-90"},
		"cache-test-1.png",
	)
	fmt.Println("Making same request again...")
	_ = client.FetchImage(
		"https://picsum.photos/800/600",
		[]string{"rotate-90"},
		"cache-test-2.png",
	)

	fmt.Println("\n✓ All examples completed successfully!")
}
```

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
# prometheus.yml - Basic configuration
scrape_configs:
  - job_name: 'image-api'
    static_configs:
      - targets: ['localhost:8080']
    scrape_interval: 15s
    scrape_timeout: 10s
```

**Docker Compose with Prometheus:**

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
    restart: unless-stopped

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prometheus-data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
    restart: unless-stopped

volumes:
  prometheus-data:
```

### Key Metrics to Monitor

#### Available Metrics

| Metric Name | Type | Description |
|-------------|------|-------------|
| `http_requests_total` | Counter | Total HTTP requests by status code and endpoint |
| `http_request_duration_seconds` | Histogram | HTTP request latency distribution |
| `image_cache_hits_total` | Counter | Number of cache hits |
| `image_cache_misses_total` | Counter | Number of cache misses |
| `image_cache_size` | Gauge | Current number of cached images |
| `image_processing_duration_seconds` | Histogram | Image processing time distribution |
| `image_fetch_duration_seconds` | Histogram | Image download time distribution |

#### Essential Prometheus Queries

**1. Request Rate (requests per second):**
```promql
rate(http_requests_total[5m])
```

**2. Error Rate (5xx errors per second):**
```promql
rate(http_requests_total{status=~"5.."}[5m])
```

**3. Error Percentage:**
```promql
100 * (
  sum(rate(http_requests_total{status=~"5.."}[5m]))
  /
  sum(rate(http_requests_total[5m]))
)
```

**4. Request Latency P50, P95, P99:**
```promql
# P50 (median)
histogram_quantile(0.50, rate(http_request_duration_seconds_bucket[5m]))

# P95
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# P99
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))
```

**5. Cache Hit Rate (percentage):**
```promql
100 * (
  rate(image_cache_hits_total[5m])
  /
  (rate(image_cache_hits_total[5m]) + rate(image_cache_misses_total[5m]))
)
```

**6. Cache Size (current number of entries):**
```promql
image_cache_size
```

**7. Image Processing Performance:**
```promql
# Average processing time
rate(image_processing_duration_seconds_sum[5m]) / rate(image_processing_duration_seconds_count[5m])

# P95 processing time
histogram_quantile(0.95, rate(image_processing_duration_seconds_bucket[5m]))
```

**8. Image Fetch Performance:**
```promql
# Average fetch time
rate(image_fetch_duration_seconds_sum[5m]) / rate(image_fetch_duration_seconds_count[5m])

# P95 fetch time
histogram_quantile(0.95, rate(image_fetch_duration_seconds_bucket[5m]))
```

**9. Requests by Endpoint:**
```promql
sum by (endpoint) (rate(http_requests_total[5m]))
```

**10. Requests by Status Code:**
```promql
sum by (status) (rate(http_requests_total[5m]))
```

### Alerting Rules

Example Prometheus alerting rules:

```yaml
# alerts.yml
groups:
  - name: image-api-alerts
    interval: 30s
    rules:
      - alert: HighErrorRate
        expr: |
          100 * (
            sum(rate(http_requests_total{status=~"5.."}[5m]))
            /
            sum(rate(http_requests_total[5m]))
          ) > 5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value }}% (threshold: 5%)"

      - alert: HighLatency
        expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 2
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High request latency detected"
          description: "P95 latency is {{ $value }}s (threshold: 2s)"

      - alert: LowCacheHitRate
        expr: |
          100 * (
            rate(image_cache_hits_total[5m])
            /
            (rate(image_cache_hits_total[5m]) + rate(image_cache_misses_total[5m]))
          ) < 50
        for: 10m
        labels:
          severity: info
        annotations:
          summary: "Low cache hit rate"
          description: "Cache hit rate is {{ $value }}% (threshold: 50%)"

      - alert: ServiceDown
        expr: up{job="image-api"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Image API service is down"
          description: "The image-api service has been down for more than 1 minute"

      - alert: HighMemoryUsage
        expr: image_cache_size > 1000
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High cache memory usage"
          description: "Cache has {{ $value }} entries (threshold: 1000)"
```

### Grafana Dashboard

Import this JSON to create a comprehensive dashboard:

```json
{
  "dashboard": {
    "title": "Image Processing API",
    "panels": [
      {
        "title": "Request Rate",
        "targets": [
          {
            "expr": "rate(http_requests_total[5m])"
          }
        ],
        "type": "graph"
      },
      {
        "title": "Error Rate",
        "targets": [
          {
            "expr": "rate(http_requests_total{status=~\"5..\"}[5m])"
          }
        ],
        "type": "graph"
      },
      {
        "title": "Cache Hit Rate %",
        "targets": [
          {
            "expr": "100 * (rate(image_cache_hits_total[5m]) / (rate(image_cache_hits_total[5m]) + rate(image_cache_misses_total[5m])))"
          }
        ],
        "type": "graph"
      },
      {
        "title": "Latency Percentiles",
        "targets": [
          {
            "expr": "histogram_quantile(0.50, rate(http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "P50"
          },
          {
            "expr": "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "P95"
          },
          {
            "expr": "histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "P99"
          }
        ],
        "type": "graph"
      }
    ]
  }
}
```

### Testing Metrics

Verify metrics are being exposed:

```bash
# Fetch raw metrics
curl http://localhost:8080/metrics

# Check specific metric
curl -s http://localhost:8080/metrics | grep http_requests_total

# Query Prometheus API
curl 'http://localhost:9090/api/v1/query?query=rate(http_requests_total[5m])'
```

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

## Troubleshooting

### Common Errors

#### 1. "400 Bad Request" - Invalid URL

**Symptom**: Orange placeholder image with "400" text

**Causes:**
- Missing `url` parameter
- Invalid URL format
- URL not properly encoded

**Solutions:**
```bash
# ❌ Wrong - missing URL parameter
curl "http://localhost:8080/image"

# ✅ Correct - URL parameter provided
curl "http://localhost:8080/image?url=https://picsum.photos/800/600"

# ❌ Wrong - URL not encoded (contains special characters)
curl "http://localhost:8080/image?url=https://example.com/image?size=large"

# ✅ Correct - URL properly encoded
curl "http://localhost:8080/image?url=$(python3 -c 'import urllib.parse; print(urllib.parse.quote("https://example.com/image?size=large", safe=""))')"
```

#### 2. "400 Bad Request" - Invalid Operation

**Symptom**: Orange placeholder image with error message

**Causes:**
- Typo in operation name
- Invalid operation syntax
- Unsupported operation

**Solutions:**
```bash
# ❌ Wrong - typo in operation
curl "http://localhost:8080/image?url=https://picsum.photos/800/600&op=rotate-45"

# ✅ Correct - valid rotation values
curl "http://localhost:8080/image?url=https://picsum.photos/800/600&op=rotate-90"

# ❌ Wrong - invalid resize format
curl "http://localhost:8080/image?url=https://picsum.photos/800/600&op=resize-800"

# ✅ Correct - resize format is WIDTHxHEIGHT
curl "http://localhost:8080/image?url=https://picsum.photos/800/600&op=resize-800x600"
```

**Supported operations:**
- `rotate-90`, `rotate-180`, `rotate-270`
- `resize-WxH` (e.g., `resize-800x600`, `resize-1400x1400`)

#### 3. "400 Bad Request" - Image Too Large

**Symptom**: Orange placeholder with error message

**Cause**: Source image exceeds 50MB limit

**Solutions:**
```bash
# Option 1: Use a smaller source image
curl "http://localhost:8080/image?url=https://example.com/smaller-image.jpg"

# Option 2: Increase limit via environment variable (if you control the server)
docker run -d -p 8080:8080 \
  -e MAX_IMAGE_SIZE=104857600 \
  ghcr.io/steviee/github-workflow-article:latest
```

#### 4. "500 Internal Server Error" - Cannot Fetch Image

**Symptom**: Red placeholder image with "500" text

**Causes:**
- Source URL is unreachable
- Network timeout
- DNS resolution failure
- SSL/TLS certificate errors

**Solutions:**
```bash
# Verify URL is accessible
curl -I https://example.com/image.jpg

# Check DNS resolution
nslookup example.com

# Test with a known-good URL
curl "http://localhost:8080/image?url=https://picsum.photos/800/600"
```

#### 5. Service Not Starting

**Symptom**: Container exits immediately or "connection refused"

**Debug steps:**
```bash
# Check container logs
docker logs image-api

# Check if port is already in use
lsof -i :8080

# Run in foreground to see errors
docker run --rm -p 8080:8080 ghcr.io/steviee/github-workflow-article:latest

# Check health endpoint
curl http://localhost:8080/health
```

#### 6. High Memory Usage

**Symptom**: Container using excessive RAM

**Causes:**
- Cache growing too large
- Cache TTL too long
- Processing very large images

**Solutions:**
```bash
# Reduce cache TTL
docker run -d -p 8080:8080 \
  -e CACHE_TTL=2m \
  -e CACHE_CLEANUP_INTERVAL=15s \
  ghcr.io/steviee/github-workflow-article:latest

# Set memory limit
docker run -d -p 8080:8080 \
  --memory=512m \
  --memory-swap=512m \
  ghcr.io/steviee/github-workflow-article:latest

# Monitor cache size
curl -s http://localhost:8080/metrics | grep image_cache_size
```

#### 7. Slow Response Times

**Symptom**: Requests taking several seconds

**Debug steps:**
```bash
# Check processing time metrics
curl -s http://localhost:8080/metrics | grep image_processing_duration

# Check fetch time metrics (slow network?)
curl -s http://localhost:8080/metrics | grep image_fetch_duration

# Test with local/fast image source
curl "http://localhost:8080/image?url=https://picsum.photos/800/600&op=rotate-90"

# Check cache hit rate
curl -s http://localhost:8080/metrics | grep image_cache
```

**Solutions:**
```bash
# Increase cache TTL to improve hit rate
docker run -d -p 8080:8080 \
  -e CACHE_TTL=10m \
  ghcr.io/steviee/github-workflow-article:latest

# Allocate more CPU
docker run -d -p 8080:8080 \
  --cpus=2 \
  ghcr.io/steviee/github-workflow-article:latest

# Use faster image sources
# Slow: Large images from slow servers
# Fast: CDN-hosted images, smaller dimensions
```

#### 8. Cache Not Working

**Symptom**: `X-Cache: SKIP` on all requests

**Causes:**
- No operations specified (cache only applies to processed images)
- Cache disabled or not initialized

**Verification:**
```bash
# ❌ No cache (no operations)
curl -v "http://localhost:8080/image?url=https://picsum.photos/800/600" 2>&1 | grep X-Cache
# Output: X-Cache: SKIP

# ✅ Cache applies (with operations)
curl -v "http://localhost:8080/image?url=https://picsum.photos/800/600&op=rotate-90" 2>&1 | grep X-Cache
# First request: X-Cache: MISS
# Second request: X-Cache: HIT
```

### Performance Tuning

#### For High Throughput

```bash
docker run -d -p 8080:8080 \
  --name image-api-high-perf \
  --cpus=4 \
  --memory=4g \
  -e CACHE_TTL=15m \
  -e MAX_IMAGE_SIZE=104857600 \
  -e LOG_LEVEL=warn \
  ghcr.io/steviee/github-workflow-article:latest
```

#### For Low Memory Environments

```bash
docker run -d -p 8080:8080 \
  --name image-api-low-mem \
  --cpus=1 \
  --memory=256m \
  --memory-swap=256m \
  -e CACHE_TTL=1m \
  -e CACHE_CLEANUP_INTERVAL=10s \
  -e MAX_IMAGE_SIZE=26214400 \
  -e LOG_LEVEL=error \
  ghcr.io/steviee/github-workflow-article:latest
```

#### For Fast Cache Rotation

```bash
docker run -d -p 8080:8080 \
  --name image-api-fast-rotation \
  -e CACHE_TTL=2m \
  -e CACHE_CLEANUP_INTERVAL=10s \
  ghcr.io/steviee/github-workflow-article:latest
```

### Debug Mode

Enable verbose logging for troubleshooting:

```bash
docker run -d -p 8080:8080 \
  --name image-api-debug \
  -e LOG_LEVEL=debug \
  ghcr.io/steviee/github-workflow-article:latest

# Watch logs in real-time
docker logs -f image-api-debug
```

### Health Checks

Verify service health:

```bash
# Basic health check
curl http://localhost:8080/health
# Expected: {"status":"healthy"}

# Readiness check
curl http://localhost:8080/ready
# Expected: {"status":"ready"}

# Test image processing end-to-end
curl "http://localhost:8080/image?url=https://picsum.photos/100/100" -o test.png
file test.png
# Expected: test.png: PNG image data, 100 x 100, 8-bit/color RGBA, non-interlaced

# Verify PNG format
xxd test.png | head -n 1
# Expected to start with: 89 50 4e 47 (PNG magic bytes)
```

### Common Integration Issues

#### CORS Errors (Browser)

**Symptom**: Browser console shows CORS policy error

**Solution**: The API already enables CORS for all origins. If you're still seeing errors:

```javascript
// Ensure you're not sending credentials
fetch('http://localhost:8080/image?url=https://picsum.photos/800/600', {
  credentials: 'omit'  // Don't send cookies
})
```

#### URL Encoding Issues

**Problem**: URLs with query parameters or special characters fail

**Solutions:**

```bash
# Bash - use Python
IMAGE_URL="https://example.com/image.jpg?size=large&format=png"
ENCODED=$(python3 -c "import urllib.parse; print(urllib.parse.quote('$IMAGE_URL', safe=''))")
curl "http://localhost:8080/image?url=$ENCODED"

# JavaScript
const imageUrl = 'https://example.com/image.jpg?size=large';
const encoded = encodeURIComponent(imageUrl);
fetch(`http://localhost:8080/image?url=${encoded}`);

# Python
import urllib.parse
image_url = 'https://example.com/image.jpg?size=large'
encoded = urllib.parse.quote(image_url, safe='')
requests.get(f'http://localhost:8080/image?url={encoded}')
```

### Getting Help

If you encounter issues not covered here:

1. **Check logs**: `docker logs <container-name>`
2. **Check metrics**: `curl http://localhost:8080/metrics`
3. **Enable debug logging**: Set `LOG_LEVEL=debug`
4. **Test with known-good URLs**: Use `https://picsum.photos/800/600`
5. **Open an issue**: [GitHub Issues](https://github.com/steviee/github-workflow-article/issues) with:
   - Exact curl command used
   - Error message or placeholder screenshot
   - Relevant logs
   - Environment details (Docker version, OS, etc.)

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
