package executor

import (
	// Go Internal Packages
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	// Local Packages
	config "flowx/config"
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
// It handles step-level persistence, retries with exponential backoff + jitter,
// and resume-from-failure logic.
type Executor struct {
	logger      *zap.Logger
	stepRunRepo StepRunRepo
	flow        flow.Flow
	config      config.Executor
}

// NewService creates an Executor using the flow name from config to resolve
// the flow definition from the registry.
func NewService(logger *zap.Logger, k config.Executor, stepRunRepo StepRunRepo) *Executor {
	f, _ := flow.Get(k.Flow)
	return &Executor{
		logger:      logger,
		stepRunRepo: stepRunRepo,
		flow:        f,
		config:      k,
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
		return e.executeSteps(ctx, runID, workerID, input, allSteps)
	}

	pendingSteps, resumeInput := e.findPendingSteps(lastStep)
	stepNames := make([]string, len(pendingSteps))
	for i, s := range pendingSteps {
		stepNames[i] = s.Name
	}

	e.logger.Info("Resuming Run", zap.String("runId", runID),
		zap.Int("workerId", workerID), zap.Strings("pendingSteps", stepNames))
	return e.executeSteps(ctx, runID, workerID, resumeInput, pendingSteps)
}

// executeSteps runs the given steps in order, chaining output → input between them.
func (e *Executor) executeSteps(ctx context.Context, runID string, workerID int, initialInput map[string]any, steps []flow.Step) error {
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

// executeStepWithRetry attempts a step up to MaxRetries times with
// exponential backoff and jitter between attempts.
func (e *Executor) executeStepWithRetry(ctx context.Context, runID string, workerID int, input map[string]any, step flow.Step) (map[string]any, error) {
	var lastError error

	for attempt := 1; attempt <= e.config.MaxRetries; attempt++ {
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

		lastError = err

		if attempt < e.config.MaxRetries {
			backoff := e.calculateBackoff(attempt)
			e.logger.Warn(fmt.Sprintf("Step [%s] Failed, Retrying in %s", step.Name, backoff),
				zap.Int("workerId", workerID), zap.Int("attempt", attempt), zap.Error(err))

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}
	}

	e.logger.Error(fmt.Sprintf("Max Retries Reached, Step [%s] Failed", step.Name),
		zap.Int("workerId", workerID), zap.Error(lastError))

	if logErr := e.stepRunRepo.RecordStepEnd(ctx, runID, step.Name, "FAILED", lastError.Error(), -1, nil); logErr != nil {
		return nil, fmt.Errorf("step logging failed (failure): %w", logErr)
	}
	return nil, lastError
}

// calculateBackoff computes the wait duration for a given retry attempt using
// exponential backoff capped at MaxBackoff, with random jitter applied.
//
// Example with defaults (initial=30s, factor=2, jitter=0.2):
//
//	attempt 1 → 30s  ± 20%
//	attempt 2 → 60s  ± 20%
//	attempt 3 → 120s ± 20%
func (e *Executor) calculateBackoff(attempt int) time.Duration {
	initialBackoff := float64(e.config.InitialBackoff) * float64(time.Second)
	base := initialBackoff * math.Pow(e.config.BackoffFactor, float64(attempt-1))

	maxBackoff := float64(e.config.MaxBackoff) * float64(time.Second)
	if base > maxBackoff {
		base = maxBackoff
	}

	// Apply jitter: shift by a random amount within ±(jitterFraction * base)
	jitterRange := base * e.config.JitterFraction
	jitter := (rand.Float64()*2 - 1) * jitterRange
	base += jitter

	if base < 0 {
		base = 0
	}

	return time.Duration(base)
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
