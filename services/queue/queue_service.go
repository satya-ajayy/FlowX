package queue

import (
	// Go Internal Packages
	"context"
	"fmt"
	"sync"

	// Local Packages
	config "flowx/config"
	models "flowx/models/workflow"
	slack "flowx/utils/slack"

	// External Packages
	"go.uber.org/zap"
)

type WorkflowRepository interface {
	GetInCompleted(ctx context.Context) ([]models.WorkflowDBModel, error)
	MarkAsComplete(ctx context.Context, workflowID string) error
}

type Processor interface {
	StartWorkflow(ctx context.Context, workerID int, workflow models.WorkflowDBModel) error
}

type QueueService struct {
	logger       *zap.Logger
	workflowRepo WorkflowRepository
	processor    Processor
	queue        chan models.WorkflowDBModel
	workers      int
	wg           sync.WaitGroup
	slack        slack.Sender
}

func NewQueueService(logger *zap.Logger, conf config.Queue, workflowRepo WorkflowRepository, processor Processor, slack slack.Sender) *QueueService {
	queue := make(chan models.WorkflowDBModel, conf.Size)
	return &QueueService{
		logger:       logger,
		workflowRepo: workflowRepo,
		queue:        queue,
		workers:      conf.Workers,
		processor:    processor,
		slack:        slack,
	}
}

func (s *QueueService) Start(ctx context.Context) error {
	// Start workers
	for i := 0; i < s.workers; i++ {
		s.wg.Add(1)
		go s.worker(ctx, i)
	}

	// Load incomplete workflows
	workflows, err := s.workflowRepo.GetInCompleted(ctx)
	if err != nil {
		return fmt.Errorf("failed to load incomplete workflows: %v", err)
	}

	for _, wf := range workflows {
		s.Enqueue(ctx, wf)
	}
	return nil
}

func (s *QueueService) Enqueue(_ context.Context, workflow models.WorkflowDBModel) {
	s.queue <- workflow
	s.logger.Info("Workflow Added To Queue Successfully", zap.String("workflowId", workflow.ID))
}

func (s *QueueService) worker(ctx context.Context, workerID int) {
	defer s.wg.Done()
	s.logger.Info("Worker started", zap.Int("workerId", workerID))

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Worker received shutdown signal", zap.Int("workerId", workerID))
			return
		case workflow := <-s.queue:
			// Process the workflow
			processErr := s.processor.StartWorkflow(ctx, workerID, workflow)
			if processErr != nil {
				slackErr := s.slack.SendAlert(workflow, processErr)
				if slackErr != nil {
					s.logger.Error("Failed to send slack alert", zap.String("workflowId", workflow.ID),
						zap.Int("workedId", workerID), zap.Error(slackErr))
				}
				continue
			}

			// Mark the workflow as completed
			updateErr := s.workflowRepo.MarkAsComplete(ctx, workflow.ID)
			if updateErr != nil {
				s.logger.Error("Failed to mark workflow as completed", zap.String("workflowId", workflow.ID),
					zap.Int("workedId", workerID), zap.Error(updateErr))
				continue
			}

			s.logger.Info("Workflow Completed Successfully", zap.String("workflowId", workflow.ID),
				zap.Int("workerId", workerID))
		}
	}
}
