package run

import (
	// Go Internal Packages
	"context"
	"fmt"
	"sync"

	// Local Packages
	config "flowx/config"
	models "flowx/models/run"
	helpers "flowx/utils/helpers"
	slack "flowx/utils/slack"

	// External Packages
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// RunRepository defines the persistence operations the service needs for runs.
type RunRepository interface {
	Create(ctx context.Context, run models.Run) error
	GetIncomplete(ctx context.Context) ([]models.Run, error)
	MarkComplete(ctx context.Context, runID string) error
}

// Executor defines the step-execution contract used by the run service.
type Executor interface {
	StartRun(ctx context.Context, workerID int, runID string, input map[string]any) error
}

// RunService is the core orchestrator. It creates runs, manages the
// buffered queue, spawns workers, and handles recovery on startup.
type RunService struct {
	logger   *zap.Logger
	runRepo  RunRepository
	executor Executor
	queue    chan models.Run
	workers  int
	wg       sync.WaitGroup
	slack    slack.Sender
}

// NewService creates a RunService with the given queue configuration.
func NewService(logger *zap.Logger, conf config.Queue, runRepo RunRepository, executor Executor, slack slack.Sender) *RunService {
	queue := make(chan models.Run, conf.Size)
	return &RunService{
		logger:   logger,
		runRepo:  runRepo,
		executor: executor,
		queue:    queue,
		workers:  conf.Workers,
		slack:    slack,
	}
}

// Create persists a new run and enqueues it for processing.
func (s *RunService) Create(ctx context.Context, input map[string]any) (string, error) {
	run := models.Run{
		ID:             uuid.New().String(),
		CreatedAt:      helpers.GetCurrentDateTime(),
		Input:          input,
		IsCompleted:    false,
		CompletedAt:    helpers.GetNotEndedTime(),
		LastStepStatus: false,
	}

	if err := s.runRepo.Create(ctx, run); err != nil {
		s.logger.Error("Failed To Create Run", zap.Error(err))
		return "", err
	}

	s.enqueue(run)
	return run.ID, nil
}

// Start spawns workers and re-enqueues any incomplete runs from the database.
// Call this once during server initialization.
func (s *RunService) Start(ctx context.Context) error {
	for i := 0; i < s.workers; i++ {
		s.wg.Add(1)
		go s.worker(ctx, i)
		helpers.Sleep100MS()
	}

	incomplete, err := s.runRepo.GetIncomplete(ctx)
	if err != nil {
		return fmt.Errorf("failed to load incomplete runs: %w", err)
	}

	for _, run := range incomplete {
		s.enqueue(run)
	}

	return nil
}

// enqueue pushes a run onto the buffered queue for worker consumption.
func (s *RunService) enqueue(run models.Run) {
	s.queue <- run
	s.logger.Info("Run Enqueued", zap.String("runId", run.ID))
}

// worker is a long-running goroutine that pulls runs from the queue
// and processes them sequentially (steps within a run are sequential,
// but multiple runs across workers execute in parallel).
func (s *RunService) worker(ctx context.Context, workerID int) {
	defer s.wg.Done()
	s.logger.Info("Worker Started", zap.Int("workerId", workerID))

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Worker Shutting Down", zap.Int("workerId", workerID))
			return
		case run := <-s.queue:
			err := s.executor.StartRun(ctx, workerID, run.ID, run.Input)
			if err != nil {
				alert := slack.Alert{
					Title: "Exception In FlowX Service",
					Fields: map[string]string{
						"Message": "Run Execution Failed",
						"RunID":   run.ID,
						"Error":   err.Error(),
					},
				}
				if alertErr := s.slack.Send(ctx, alert); alertErr != nil {
					s.logger.Error("Failed To Send Alert", zap.String("runId", run.ID),
						zap.Int("workerId", workerID), zap.Error(alertErr))
				}
				continue
			}

			if err := s.runRepo.MarkComplete(ctx, run.ID); err != nil {
				s.logger.Error("Failed To Mark Run As Complete", zap.String("runId", run.ID),
					zap.Int("workerId", workerID), zap.Error(err))
				continue
			}

			s.logger.Info("Run Completed", zap.String("runId", run.ID),
				zap.Int("workerId", workerID))
		}
	}
}
