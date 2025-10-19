package square

import (
	// Local Packages
	errors "flowx/errors"
	wmodels "flowx/models/workflow"
	helpers "flowx/utils/helpers"
)

type SquareQP struct {
	Number int `json:"number" bson:"number"`
}

func (p *SquareQP) Validate() error {
	ve := errors.ValidationErrorBuilder{}

	if p.Number <= 0 {
		ve.Add("number", "cannot be less than zero")
	}

	return ve.Err()
}

func (p *SquareQP) ToWorkflowDBModel(workflowID string) wmodels.WorkflowDBModel {
	return wmodels.WorkflowDBModel{
		ID:        workflowID,
		CreatedAt: helpers.GetCurrentDateTime(),
		Input: map[string]interface{}{
			"number": p.Number,
		},
		IsCompleted:    false,
		CompletedAt:    helpers.GetNotEndedTime(),
		LastTaskStatus: false,
	}
}
