package handler

import (
	"encoding/json"
	"net/http"
)

// HealthResponse represents the JSON response for health check endpoints
type HealthResponse struct {
	Status string `json:"status"`
}

// HealthHandler handles GET /health requests
// Always returns 200 OK to indicate the service is running
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := HealthResponse{
		Status: "healthy",
	}

	// Marshal and write response
	// In production code, we'd log errors, but for now keep it simple
	_ = json.NewEncoder(w).Encode(response)
}

// ReadyHandler handles GET /ready requests
// Returns 200 OK when the service is ready to accept traffic
// In the future, this could check dependencies like cache, external services, etc.
func ReadyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := HealthResponse{
		Status: "ready",
	}

	// Marshal and write response
	_ = json.NewEncoder(w).Encode(response)
}
