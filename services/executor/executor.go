package executor

import (
	// Go Internal Packages
	"context"
	"fmt"
	"time"

	// Local Packages
	flow "flowx/flow"
	srmodels "flowx/models/steprun"
	helpers "flowx/utils/helpers"

	// External Packages
	"go.uber.org/zap"
)

// StepRunRepo defines the persistence operations needed by the executor.
type StepRunRepo interface {
	GetLastRecordedStep(ctx context.Context, runID string) (*srmodels.StepRun, error)
	RecordStepStart(ctx context.Context, runID, stepName string, input map[string]any) error
	RecordStepEnd(ctx context.Context, runID, stepName, state, reason string, duration int, output map[string]any) error
}

// Executor is responsible for running the steps of a flow sequentially.
// It handles step-level persistence, retries, and resume-from-failure logic.
type Executor struct {
	logger      *zap.Logger
	stepRunRepo StepRunRepo
	flow        flow.Flow
}

// NewExecutor creates an Executor wired to a specific flow definition.
func NewExecutor(logger *zap.Logger, stepRunRepo StepRunRepo, f flow.Flow) *Executor {
	return &Executor{
		logger:      logger,
		stepRunRepo: stepRunRepo,
		flow:        f,
	}
}

// StartRun determines where to begin execution for a run. If a previous step
// was recorded (e.g. after a crash), it resumes from there; otherwise it
// starts fresh with all steps.
func (e *Executor) StartRun(ctx context.Context, workerID int, runID string, input map[string]any) error {
	lastStep, err := e.stepRunRepo.GetLastRecordedStep(ctx, runID)
	if err != nil {
		return err
	}

	if lastStep == nil {
		allSteps := e.flow.GetAllSteps()
		e.logger.Info("Executing New Run", zap.String("runId", runID),
			zap.Int("workerId", workerID), zap.Strings("steps", e.flow.StepNames()))
		return e.ExecuteSteps(ctx, runID, workerID, input, allSteps)
	}

	pendingSteps, resumeInput := e.findPendingSteps(lastStep)
	stepNames := make([]string, len(pendingSteps))
	for i, s := range pendingSteps {
		stepNames[i] = s.Name
	}

	e.logger.Info("Resuming Run", zap.String("runId", runID),
		zap.Int("workerId", workerID), zap.Strings("pendingSteps", stepNames))
	return e.ExecuteSteps(ctx, runID, workerID, resumeInput, pendingSteps)
}

// ExecuteSteps runs the given steps in order, chaining output → input between them.
func (e *Executor) ExecuteSteps(ctx context.Context, runID string, workerID int, initialInput map[string]any, steps []flow.Step) error {
	input := initialInput
	for _, step := range steps {
		if err := e.stepRunRepo.RecordStepStart(ctx, runID, step.Name, input); err != nil {
			return err
		}

		e.logger.Info(fmt.Sprintf("Executing Step [%s]", step.Name),
			zap.String("runId", runID), zap.Int("workerId", workerID))

		output, err := e.executeStepWithRetry(ctx, runID, workerID, input, step)
		if err != nil {
			return err
		}

		input = output
	}
	return nil
}

// executeStepWithRetry attempts a step up to 3 times with 1-minute backoff between retries.
func (e *Executor) executeStepWithRetry(ctx context.Context, runID string, workerID int, input map[string]any, step flow.Step) (map[string]any, error) {
	var lastError error

	for attempt := 1; attempt <= 3; attempt++ {
		output, sec, err := e.executeStep(ctx, step, input)
		duration := time.Duration(sec) * time.Second

		if err == nil {
			e.logger.Info(fmt.Sprintf("Step [%s] Executed Successfully", step.Name), zap.Int("workerId", workerID),
				zap.Duration("duration", duration), zap.Int("attempt", attempt))

			if logErr := e.stepRunRepo.RecordStepEnd(ctx, runID, step.Name, "COMPLETED", "", sec, output); logErr != nil {
				return nil, fmt.Errorf("step logging failed (success): %w", logErr)
			}
			return output, nil
		}

		e.logger.Warn(fmt.Sprintf("Step [%s] Failed, Retrying", step.Name),
			zap.Int("workerId", workerID), zap.Int("attempt", attempt), zap.Error(err))

		time.Sleep(1 * time.Minute)
		lastError = err
	}

	e.logger.Error(fmt.Sprintf("Max Retries Reached, Step [%s] Failed", step.Name),
		zap.Int("workerId", workerID), zap.Error(lastError))

	if logErr := e.stepRunRepo.RecordStepEnd(ctx, runID, step.Name, "FAILED", lastError.Error(), -1, nil); logErr != nil {
		return nil, fmt.Errorf("step logging failed (failure): %w", logErr)
	}
	return nil, lastError
}

// executeStep runs cleanup (if defined) followed by execution, tracking elapsed time.
func (e *Executor) executeStep(ctx context.Context, step flow.Step, input map[string]any) (map[string]any, int, error) {
	startTime := time.Now()

	if step.Cleanup != nil {
		if err := step.Cleanup(ctx, input); err != nil {
			return nil, helpers.SecondsSince(startTime), err
		}
	}

	if step.Execute != nil {
		output, err := step.Execute(ctx, input)
		if err != nil {
			return nil, helpers.SecondsSince(startTime), err
		}
		return output, helpers.SecondsSince(startTime), nil
	}

	return input, helpers.SecondsSince(startTime), nil
}

// findPendingSteps determines which steps remain based on the last recorded step.
// If the last step succeeded, resume from the next one using its output.
// If it failed, re-run it using its original input.
func (e *Executor) findPendingSteps(lastStep *srmodels.StepRun) ([]flow.Step, map[string]any) {
	succeeded := lastStep.IsEndedSuccessfully()
	lastStepName := lastStep.ID.StepName

	pendingSteps := e.flow.GetPendingSteps(lastStepName, succeeded)
	if succeeded {
		return pendingSteps, lastStep.Ending.Output
	}

	return pendingSteps, lastStep.Input
}
