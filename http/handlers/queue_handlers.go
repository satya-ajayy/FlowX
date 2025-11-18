package handlers

import (
	// Go Internal Packages
	"context"
	"fmt"
	"net/http"

	// Local Packages
	errors "flowx/errors"
)

type QueueService interface {
	InitWorkflow(ctx context.Context, input map[string]interface{}) (string, error)
}

type QueueHandler struct {
	queueSVC QueueService
}

func NewQueueHandler(queueSVC QueueService) *QueueHandler {
	return &QueueHandler{queueSVC: queueSVC}
}

func (h *QueueHandler) InitWorkflow(_ http.ResponseWriter, r *http.Request) (response any, status int, err error) {
	name := r.URL.Query().Get("name")
	if name == "" {
		return nil, http.StatusBadRequest, errors.E(errors.Invalid, "name can't be empty")
	}

	input := map[string]interface{}{
		"name": name,
	}

	workflowID, err := h.queueSVC.InitWorkflow(r.Context(), input)
	if err == nil {
		return map[string]interface{}{
			"message": fmt.Sprintf("Created Workflow With ID: %s", workflowID),
		}, http.StatusOK, nil
	}
	return
}
