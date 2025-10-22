package processor

import (
	// Go Internal Packages
	"context"
	"fmt"
	"time"

	// Local Packages
	tmodels "flowx/models/monitor"
	wmodels "flowx/models/workflow"
	helpers "flowx/utils/helpers"

	// External Packages
	"go.uber.org/zap"
)

type MonitorRepository interface {
	GetLastRecordedTask(ctx context.Context, workflowID string) (*tmodels.TaskMonitor, error)
	RecordTaskStart(ctx context.Context, workflowID, taskName string, input map[string]interface{}) error
	RecordTaskEnd(ctx context.Context, workflowID, taskName, state, reason string, duration int, output map[string]interface{}) error
}

type ProcessorService struct {
	logger   *zap.Logger
	monitor  MonitorRepository
	workflow wmodels.Workflow
}

func NewProcessor(logger *zap.Logger, monitor MonitorRepository, workflow wmodels.Workflow) *ProcessorService {
	return &ProcessorService{
		logger:   logger,
		monitor:  monitor,
		workflow: workflow,
	}
}

// StartWorkflow checks the last executed task and starts the workflow from there
// if none found it start as fresh workflow and run all the tasks
func (s *ProcessorService) StartWorkflow(ctx context.Context, workerID int, workflow wmodels.WorkflowDBModel) error {
	lastExecutedTask, err := s.monitor.GetLastRecordedTask(ctx, workflow.ID)
	if err != nil {
		return err
	}

	if lastExecutedTask == nil {
		allTasks := s.workflow.GetAllTasks()

		s.logger.Info("Processing New Workflow", zap.String("workflowId", workflow.ID),
			zap.Int("workerId", workerID), zap.Strings("tasks", allTasks.GetNames()))

		processErr := s.ProcessWorkflow(ctx, workflow.ID, workerID, workflow.Input, allTasks)
		return processErr
	}

	pendingTasks, input := s.FindPendingTasks(lastExecutedTask)
	s.logger.Info("Processing Old Workflow", zap.String("workflowId", workflow.ID),
		zap.Int("workerId", workerID), zap.Strings("tasks", pendingTasks.GetNames()))
	processErr := s.ProcessWorkflow(ctx, workflow.ID, workerID, input, pendingTasks)
	return processErr
}

func (s *ProcessorService) ProcessWorkflow(ctx context.Context, workflowID string, workerID int,
	initialInput map[string]interface{}, tasks []wmodels.Task) error {
	input := initialInput
	for _, task := range tasks {
		mongoErr := s.monitor.RecordTaskStart(ctx, workflowID, task.Name, input)
		if mongoErr != nil {
			return mongoErr
		}

		s.logger.Info(fmt.Sprintf("Processing Task [%s]", task.Name),
			zap.String("workflowId", workflowID), zap.Int("workerId", workerID))

		output, processErr := s.ProcessTaskWithRetry(ctx, workflowID, workerID, input, task)
		if processErr != nil {
			return processErr
		}

		input = output
	}
	return nil
}

func (s *ProcessorService) ProcessTaskWithRetry(ctx context.Context, workflowID string, workerID int,
	input map[string]interface{}, task wmodels.Task) (map[string]interface{}, error) {
	var lastError error
	for attempt := 1; attempt <= 3; attempt++ {
		output, sec, processErr := s.ProcessTask(ctx, task, input)
		duration := time.Duration(sec) * time.Second
		if processErr == nil {
			s.logger.Info(fmt.Sprintf("Task [%s] Executed Successfully", task.Name), zap.Int("workerId", workerID),
				zap.Duration("duration", duration), zap.Int("attempt", attempt))

			err := s.monitor.RecordTaskEnd(ctx, workflowID, task.Name, "COMPLETED", "", sec, output)
			if err != nil {
				return nil, fmt.Errorf("Task(S) Monitoring Failed: %w", err)
			}
			return output, nil
		}

		s.logger.Warn(fmt.Sprintf("Task [%s] Execution Failed, Retrying", task.Name),
			zap.Int("workerId", workerID), zap.Int("attempt", attempt), zap.Error(processErr))

		time.Sleep(1 * time.Minute)
		lastError = processErr
	}

	s.logger.Error(fmt.Sprintf("Retry Attempts Reached, Task [%s] Failed", task.Name),
		zap.Int("workerId", workerID), zap.Error(lastError))
	err := s.monitor.RecordTaskEnd(ctx, workflowID, task.Name, "FAILED", lastError.Error(), -1, nil)
	if err != nil {
		return nil, fmt.Errorf("Task(F) Monitoring Failed: %w", err)
	}
	return nil, lastError
}

func (s *ProcessorService) ProcessTask(ctx context.Context, task wmodels.Task, input map[string]interface{}) (map[string]interface{}, int, error) {
	startTime := time.Now()

	// Phase 1: Cleanup
	if task.Cleanup != nil {
		err := task.Cleanup(ctx, input)
		if err != nil {
			return nil, helpers.SecondsSince(startTime), err
		}
	}

	// Phase 2: Execution
	if task.Execute != nil {
		output, err := task.Execute(ctx, input)
		if err != nil {
			return nil, helpers.SecondsSince(startTime), err
		}
		return output, helpers.SecondsSince(startTime), nil
	}

	return input, helpers.SecondsSince(startTime), nil
}

func (s *ProcessorService) FindPendingTasks(lastExecutedTask *tmodels.TaskMonitor) (wmodels.TaskList, map[string]interface{}) {
	// Check weather the last task is ended successfully or not
	isLastTaskCompleted := lastExecutedTask.IsEndedSuccessfully()
	lastTaskName := lastExecutedTask.ID.TaskName

	// Get all Pending tasks and return them
	pendingTasks := s.workflow.GetPendingTasks(lastTaskName, isLastTaskCompleted)
	if isLastTaskCompleted {
		return pendingTasks, lastExecutedTask.Ending.Output
	}

	return pendingTasks, lastExecutedTask.Input
}
