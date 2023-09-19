package handler

import (
	"fmt"
	"net/http"
)

// HealthHandler is the handler for the health check.
type HealthHandler struct {
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Healthz is the handler for the health check.
func (h *HealthHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Healthy")
}
