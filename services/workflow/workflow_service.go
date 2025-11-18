package workflow

import (
	// Local Packages
	models "flowx/models/workflow"

	// External Packages
	"go.uber.org/zap"
)

type WorkflowService struct {
	logger *zap.Logger
}

func NewWorkflowService(logger *zap.Logger) *WorkflowService {
	return &WorkflowService{logger: logger}
}

// GetBasicWorkflow return the sequence of tasks required to Square Numbers
func (s *WorkflowService) GetBasicWorkflow() models.Workflow {
	// Tasks always need to be executed in a synchronous way for this workflow
	return models.Workflow{
		Name: "Basic-Workflow",
		Tasks: []models.Task{
			{
				Name:        "Step 1",
				Description: "Step 1 in Basic Workflow",
				Cleanup:     nil,
				Execute:     s.BasicTask,
			},
			{
				Name:        "Step 2",
				Description: "Step 2 in Basic Workflow",
				Cleanup:     nil,
				Execute:     s.BasicTask,
			},
			{
				Name:        "Step 3",
				Description: "Step 3 in Basic Workflow",
				Cleanup:     nil,
				Execute:     s.BasicTask,
			},
		},
	}
}
