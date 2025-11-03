package handler

import (
	"encoding/json"
	"log"
	"net/http"
)

// ImageErrorResponse represents the JSON error response for the image endpoint
type ImageErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// ImageHandler handles GET /image requests
// This is a placeholder implementation that returns 501 Not Implemented
// The actual image processing logic will be implemented in Issues #6-#10
func ImageHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)

	response := ImageErrorResponse{
		Error:   "Image processing not yet implemented",
		Message: "Coming soon in Issue #6-#10",
	}

	// Marshal and write response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding image error response: %v", err)
	}
}
