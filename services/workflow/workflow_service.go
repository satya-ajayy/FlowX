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

// GetSquareWorkflow return the sequence of tasks required to Square Numbers
func (s *WorkflowService) GetSquareWorkflow() models.Workflow {
	// Tasks always need to be executed in a synchronous way for this workflow
	return models.Workflow{
		Name: "Square-Workflow",
		Tasks: []models.Task{
			{
				Name:        "UpdatePublishAndMutationIDs",
				Description: "Update the publishID and mutationID",
				Cleanup:     nil,
				Execute:     s.SquareTask,
			},
			{
				Name:        "UpdateConfigsInCache",
				Description: "Config Updates in the Redis Cache",
				Cleanup:     nil,
				Execute:     s.SquareTask,
			},
			{
				Name:        "AppendNewPublishIdInRedis",
				Description: "Creates new publishID in Redis",
				Cleanup:     nil,
				Execute:     s.SquareTask,
			},
		},
	}
}
