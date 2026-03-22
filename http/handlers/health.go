package handlers

import (
	// Go Internal Packages
	"context"
	"fmt"
	"net/http"
)

type HealthCheckService interface {
	HealthCheck(ctx context.Context) bool
}

type HealthCheckHandler struct {
	svc HealthCheckService
}

func NewHealthCheckHandler(svc HealthCheckService) *HealthCheckHandler {
	return &HealthCheckHandler{svc: svc}
}

func (h *HealthCheckHandler) HealthCheck(w http.ResponseWriter, r *http.Request) (response any, status int, err error) {
	ok := h.svc.HealthCheck(r.Context())
	if !ok {
		return nil, http.StatusServiceUnavailable, fmt.Errorf("health check failed")
	}
	return map[string]any{
		"message": "!!! We are RunninGoo !!!",
	}, http.StatusOK, nil
}
