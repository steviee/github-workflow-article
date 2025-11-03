package handler

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/steviee/github-workflow-article/internal/fetcher"
	"github.com/steviee/github-workflow-article/internal/placeholder"
)

const (
	defaultMaxImageSize = 52428800 // 50MB
	defaultTimeout      = 30 * time.Second
)

// ImageHandler handles GET /image requests
// Query parameters:
//   - url: required, the image URL to fetch and process
//   - width: optional, desired output width (default: original)
//   - height: optional, desired output height (default: original)
func ImageHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	imageURL := r.URL.Query().Get("url")
	if imageURL == "" {
		sendPlaceholder(w, http.StatusBadRequest, 400, 300)
		return
	}

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

	// For now, just return the fetched image without processing
	// Processing (rotation, resize, PNG conversion) will be added in Issues #7-#9
	w.Header().Set("Content-Type", result.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(result.Size, 10))
	w.Header().Set("X-Original-URL", result.URL)
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(result.Data); err != nil {
		log.Printf("Error writing image response: %v", err)
	}
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
