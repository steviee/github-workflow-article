package handler

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "image/gif"  // Register GIF format
	_ "image/jpeg" // Register JPEG format

	"github.com/steviee/github-workflow-article/internal/fetcher"
	"github.com/steviee/github-workflow-article/internal/placeholder"
	"github.com/steviee/github-workflow-article/internal/processor"
)

const (
	defaultMaxImageSize = 52428800 // 50MB
	defaultTimeout      = 30 * time.Second
)

// ImageHandler handles GET /image requests
// Query parameters:
//   - url: required, the image URL to fetch and process
//   - op: optional, comma-separated operations to apply (e.g., "rotate-90,rotate-180")
//   - width: optional, desired output width (default: original)
//   - height: optional, desired output height (default: original)
func ImageHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	imageURL := r.URL.Query().Get("url")
	if imageURL == "" {
		sendPlaceholder(w, http.StatusBadRequest, 400, 300)
		return
	}

	operations := r.URL.Query().Get("op")

	// Parse dimensions if provided
	width, _ := strconv.Atoi(r.URL.Query().Get("width"))
	height, _ := strconv.Atoi(r.URL.Query().Get("height"))

	// Create fetcher
	f := fetcher.NewFetcher(defaultMaxImageSize, defaultTimeout)

	// Fetch image
	result, err := f.Fetch(imageURL)
	if err != nil {
		log.Printf("Error fetching image from %s: %v", imageURL, err)
		// Determine status code based on error type
		statusCode := http.StatusBadGateway
		sendPlaceholder(w, statusCode, width, height)
		return
	}

	// If no operations requested, return original image
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

	// Decode image
	img, format, err := image.Decode(bytes.NewReader(result.Data))
	if err != nil {
		log.Printf("Error decoding image: %v", err)
		sendPlaceholder(w, http.StatusInternalServerError, width, height)
		return
	}

	log.Printf("Decoded image format: %s, dimensions: %dx%d",
		format, img.Bounds().Dx(), img.Bounds().Dy())

	// Apply operations
	processedImg, err := applyOperations(img, operations)
	if err != nil {
		log.Printf("Error applying operations: %v", err)
		sendPlaceholder(w, http.StatusBadRequest, width, height)
		return
	}

	// Encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, processedImg); err != nil {
		log.Printf("Error encoding PNG: %v", err)
		sendPlaceholder(w, http.StatusInternalServerError, width, height)
		return
	}

	// Send response
	pngData := buf.Bytes()
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", strconv.Itoa(len(pngData)))
	w.Header().Set("X-Original-URL", result.URL)
	w.Header().Set("X-Original-Format", format)
	w.Header().Set("X-Operations-Applied", operations)
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(pngData); err != nil {
		log.Printf("Error writing PNG response: %v", err)
	}
}

// applyOperations applies a comma-separated list of operations to an image
func applyOperations(img image.Image, operations string) (image.Image, error) {
	ops := strings.Split(operations, ",")
	result := img

	for _, op := range ops {
		op = strings.TrimSpace(op)
		if op == "" {
			continue
		}

		switch op {
		case "rotate-90":
			result = processor.Rotate90(result)
			log.Printf("Applied rotate-90, new dimensions: %dx%d",
				result.Bounds().Dx(), result.Bounds().Dy())

		case "rotate-180":
			result = processor.Rotate180(result)
			log.Printf("Applied rotate-180, dimensions: %dx%d",
				result.Bounds().Dx(), result.Bounds().Dy())

		case "rotate-270":
			result = processor.Rotate270(result)
			log.Printf("Applied rotate-270, new dimensions: %dx%d",
				result.Bounds().Dx(), result.Bounds().Dy())

		default:
			return nil, fmt.Errorf("unsupported operation: %s", op)
		}
	}

	return result, nil
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
