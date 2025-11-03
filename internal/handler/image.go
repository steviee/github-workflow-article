package handler

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"  // Register GIF decoder
	_ "image/jpeg" // Register JPEG decoder
	"image/png"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/steviee/github-workflow-article/internal/fetcher"
	"github.com/steviee/github-workflow-article/internal/placeholder"
	"github.com/steviee/github-workflow-article/internal/processor"
	_ "golang.org/x/image/webp" // Register WebP decoder
)

const (
	defaultMaxImageSize = 52428800 // 50MB
	defaultTimeout      = 30 * time.Second
)

// ImageHandler handles GET /image requests
// Query parameters:
//   - url: required, the image URL to fetch and process
//   - op: optional, comma-separated list of operations (e.g., "rotate-90,resize-800x600")
//
// Supported operations:
//   - rotate-90: Rotate image 90 degrees clockwise
//   - rotate-180: Rotate image 180 degrees clockwise
//   - rotate-270: Rotate image 270 degrees clockwise
//   - resize-WxH: Resize to exact WxH dimensions with aspect-ratio preserving crop
func ImageHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	imageURL := r.URL.Query().Get("url")
	if imageURL == "" {
		sendPlaceholder(w, http.StatusBadRequest, 400, 300)
		return
	}

	operations := r.URL.Query().Get("op")

	// Create fetcher
	f := fetcher.NewFetcher(defaultMaxImageSize, defaultTimeout)

	// Fetch image
	result, err := f.Fetch(imageURL)
	if err != nil {
		log.Printf("Error fetching image from %s: %v", imageURL, err)
		// Determine status code based on error type
		statusCode := http.StatusBadGateway
		sendPlaceholder(w, statusCode, 400, 300)
		return
	}

	// If no operations, return the fetched image as-is
	if operations == "" {
		w.Header().Set("Content-Type", result.ContentType)
		w.Header().Set("Content-Length", strconv.FormatInt(result.Size, 10))
		w.Header().Set("X-Original-URL", result.URL)
		w.WriteHeader(http.StatusOK)

		if _, err := w.Write(result.Data); err != nil {
			log.Printf("Error writing image response: %v", err)
		}
		return
	}

	// Decode the image
	img, format, err := image.Decode(bytes.NewReader(result.Data))
	if err != nil {
		log.Printf("Error decoding image: %v", err)
		sendPlaceholder(w, http.StatusBadGateway, 400, 300)
		return
	}

	log.Printf("Decoded image format: %s, size: %dx%d", format, img.Bounds().Dx(), img.Bounds().Dy())

	// Apply operations
	processedImg, err := applyOperations(img, operations)
	if err != nil {
		log.Printf("Error applying operations: %v", err)
		sendPlaceholder(w, http.StatusBadRequest, 400, 300)
		return
	}

	// Encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, processedImg); err != nil {
		log.Printf("Error encoding PNG: %v", err)
		sendPlaceholder(w, http.StatusInternalServerError, 400, 300)
		return
	}

	// Return processed image
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	w.Header().Set("X-Original-URL", result.URL)
	w.Header().Set("X-Original-Format", format)
	w.Header().Set("X-Operations-Applied", operations)
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(buf.Bytes()); err != nil {
		log.Printf("Error writing processed image response: %v", err)
	}
}

// applyOperations applies a sequence of operations to an image
func applyOperations(img image.Image, operations string) (image.Image, error) {
	if operations == "" {
		return img, nil
	}

	ops := strings.Split(operations, ",")
	result := img

	for _, op := range ops {
		op = strings.TrimSpace(op)
		if op == "" {
			continue
		}

		var err error
		result, err = applyOperation(result, op)
		if err != nil {
			return nil, fmt.Errorf("operation %q failed: %w", op, err)
		}
	}

	return result, nil
}

// applyOperation applies a single operation to an image
func applyOperation(img image.Image, operation string) (image.Image, error) {
	switch {
	case operation == "rotate-90":
		return processor.Rotate90(img), nil
	case operation == "rotate-180":
		return processor.Rotate180(img), nil
	case operation == "rotate-270":
		return processor.Rotate270(img), nil
	case strings.HasPrefix(operation, "resize-"):
		return applyResize(img, operation)
	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}
}

// applyResize parses and applies a resize operation
func applyResize(img image.Image, operation string) (image.Image, error) {
	// Parse "resize-WxH" format
	parts := strings.TrimPrefix(operation, "resize-")
	dimensions := strings.Split(parts, "x")

	if len(dimensions) != 2 {
		return nil, fmt.Errorf("invalid resize format, expected resize-WxH, got: %s", operation)
	}

	width, err := strconv.Atoi(dimensions[0])
	if err != nil || width <= 0 {
		return nil, fmt.Errorf("invalid width in resize operation: %s", dimensions[0])
	}

	height, err := strconv.Atoi(dimensions[1])
	if err != nil || height <= 0 {
		return nil, fmt.Errorf("invalid height in resize operation: %s", dimensions[1])
	}

	return processor.Resize(img, width, height), nil
}

// sendPlaceholder generates and sends a placeholder image with the given status code
func sendPlaceholder(w http.ResponseWriter, statusCode, width, height int) {
	data, err := placeholder.Generate(statusCode, width, height)
	if err != nil {
		log.Printf("Error generating placeholder: %v", err)
		// Fallback to simple error response
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte("Error generating placeholder"))
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("X-Placeholder", "true")
	w.WriteHeader(statusCode)

	if _, err := w.Write(data); err != nil {
		log.Printf("Error writing placeholder response: %v", err)
	}
}
