package square

import (
	// Go Internal Packages
	"context"

	// Local Packages
	smodels "flowx/models/square"
	wmodels "flowx/models/workflow"

	// External Packages
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type WorkflowRepository interface {
	CreateWorkflow(ctx context.Context, workflow wmodels.WorkflowDBModel) error
}

type QueueService interface {
	Enqueue(ctx context.Context, workflow wmodels.WorkflowDBModel)
}

type SquareService struct {
	logger       *zap.Logger
	workflowRepo WorkflowRepository
	queueService QueueService
}

func NewSquareService(logger *zap.Logger, workflowRepo WorkflowRepository, queueService QueueService) *SquareService {
	return &SquareService{
		logger:       logger,
		workflowRepo: workflowRepo,
		queueService: queueService,
	}
}

func (s *SquareService) InitWorkflow(ctx context.Context, qp smodels.SquareQP) (string, error) {
	workflowID := uuid.New().String()
	workflow := qp.ToWorkflowDBModel(workflowID)

	err := s.workflowRepo.CreateWorkflow(ctx, workflow)
	if err != nil {
		return "", err
	}

	// Enqueue the workflow to the queue
	s.queueService.Enqueue(ctx, workflow)
	return workflowID, nil
}
