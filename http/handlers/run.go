package handlers

import (
	// Go Internal Packages
	"context"
	"encoding/json"
	"net/http"

	// Local Packages
	errors "flowx/errors"
)

// RunService defines the contract the handler needs from the run service layer.
type RunService interface {
	Create(ctx context.Context, input map[string]any) (string, error)
}

// RunHandler exposes HTTP endpoints for run operations.
type RunHandler struct {
	svc RunService
}

// NewRunHandler creates a new RunHandler backed by the given service.
func NewRunHandler(svc RunService) *RunHandler {
	return &RunHandler{svc: svc}
}

// Create handles POST /runs — decodes the input payload, creates a new run,
// and returns the generated run ID.
func (h *RunHandler) Create(w http.ResponseWriter, r *http.Request) (response any, status int, err error) {
	var input map[string]any
	if err = json.NewDecoder(r.Body).Decode(&input); err != nil {
		return nil, http.StatusBadRequest, errors.InvalidBodyErr(err)
	}

	runID, err := h.svc.Create(r.Context(), input)
	if err == nil {
		return map[string]any{
			"message": "Run Created Successfully!",
			"run_id":  runID,
		}, http.StatusCreated, nil
	}
	return
}
