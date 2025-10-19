package handlers

import (
	// Go Internal Packages
	"context"
	"fmt"
	"net/http"

	// Local Packages
	errors "flowx/errors"
	models "flowx/models/square"
	helpers "flowx/utils/helpers"
)

type SquareService interface {
	InitWorkflow(ctx context.Context, qp models.SquareQP) (string, error)
}

type SquareHandler struct {
	squareSVC SquareService
}

func NewSquareHandler(squareSVC SquareService) *SquareHandler {
	return &SquareHandler{squareSVC: squareSVC}
}

func (h *SquareHandler) InitWorkflow(_ http.ResponseWriter, r *http.Request) (response any, status int, err error) {
	var queryParams models.SquareQP
	if dErr := helpers.GetSchemaDecoder().Decode(&queryParams, r.URL.Query()); dErr != nil {
		return nil, http.StatusBadRequest, errors.InvalidParamsErr(dErr)
	}
	if err = queryParams.Validate(); err != nil {
		return nil, http.StatusBadRequest, errors.ValidationFailedErr(err)
	}

	workflowID, err := h.squareSVC.InitWorkflow(r.Context(), queryParams)
	if err == nil {
		return map[string]interface{}{
			"message": fmt.Sprintf("Created Workflow With ID: %s", workflowID),
		}, http.StatusOK, nil
	}
	return
}
