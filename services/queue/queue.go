package queue

import (
	// Go Internal Packages
	"context"
	"fmt"
	"sync"

	// Local Packages
	config "flowx/config"
	models "flowx/models/workflow"
	notifications "flowx/notifications"
	helpers "flowx/utils/helpers"

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
	alerter      notifications.Alerter
}

func NewQueueService(logger *zap.Logger, conf config.Queue, workflowRepo WorkflowRepository, processor Processor, alerter notifications.Alerter) *QueueService {
	queue := make(chan models.WorkflowDBModel, conf.Size)
	return &QueueService{
		logger:       logger,
		workflowRepo: workflowRepo,
		queue:        queue,
		workers:      conf.Workers,
		processor:    processor,
		alerter:      alerter,
	}
}

func (s *QueueService) Start(ctx context.Context) error {
	// Start All The Workers
	for i := 0; i < s.workers; i++ {
		s.wg.Add(1)
		go s.worker(ctx, i)
		helpers.SleepOneSecond()
	}

	// Load All The Incomplete Workflows
	workflows, err := s.workflowRepo.GetInCompleted(ctx)
	if err != nil {
		return fmt.Errorf("failed to load incomplete workflows: %w", err)
	}

	// Enqueue All The Incomplete Workflows
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
	s.logger.Info("Worker Started [Polling For Workflows]", zap.Int("workerId", workerID))

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Worker Received Shutdown Signal", zap.Int("workerId", workerID))
			return
		case workflow := <-s.queue:
			// Process the workflow
			err := s.processor.StartWorkflow(ctx, workerID, workflow)
			if err != nil {
				s.SendAlert(ctx, workflow.ID, workerID, err.Error())
				continue
			}

			// Mark the workflow as completed
			err = s.workflowRepo.MarkAsComplete(ctx, workflow.ID)
			if err != nil {
				s.logger.Error("Failed To Mark Workflow As Completed", zap.String("workflowId", workflow.ID),
					zap.Int("workedId", workerID), zap.Error(err))
				continue
			}

			s.logger.Info("Workflow Completed Successfully", zap.String("workflowId", workflow.ID),
				zap.Int("workerId", workerID))
		}
	}
}

func (s *QueueService) SendAlert(ctx context.Context, workflowID string, workerID int, errorMessage string) {
	alert := notifications.Alert{
		Title: "Exception In FlowX Service",
		Fields: map[string]string{
			"Worker":   fmt.Sprintf("Worker-%d", workerID),
			"Message":  "Workflow Execution Failed!",
			"Workflow": workflowID,
			"Error":    errorMessage,
		},
	}

	if err := s.alerter.Send(ctx, alert); err != nil {
		s.logger.Error("Failed To Send Alert", zap.String("workflowId", workflowID),
			zap.Int("workerId", workerID), zap.Error(err))
	}
}
