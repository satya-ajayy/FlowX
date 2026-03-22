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

// GetEmptyWorkflow return an empty workflow
func (s *WorkflowService) GetEmptyWorkflow() models.Workflow {
	return models.Workflow{
		Name:  "Empty-Workflow",
		Tasks: []models.Task{},
	}
}
