package flow

import "context"

// Step represents a single unit of work within a flow.
// Each step has a name, an optional cleanup phase, and an execution phase.
// The output of one step becomes the input of the next.
type Step struct {
	Name        string
	Description string
	Cleanup     func(ctx context.Context, input map[string]any) error
	Execute     func(ctx context.Context, input map[string]any) (map[string]any, error)
}

// Flow is a static, code-defined blueprint containing an ordered list of steps.
// Flows are NOT persisted — they exist only in code. When a flow is triggered,
// a Run (persisted) is created along with StepRuns for tracking execution state.
type Flow struct {
	Name  string
	Steps []Step
}

// GetAllSteps returns the complete ordered list of steps in the flow.
func (f *Flow) GetAllSteps() []Step {
	return f.Steps
}

// StepNames returns the names of all steps in order, useful for logging.
func (f *Flow) StepNames() []string {
	names := make([]string, len(f.Steps))
	for i, s := range f.Steps {
		names[i] = s.Name
	}
	return names
}

// GetPendingSteps returns all steps that still need to be executed,
// starting from the last executed step. If the last step failed,
// it is included in the pending list for retry.
func (f *Flow) GetPendingSteps(lastStep string, lastSucceeded bool) []Step {
	var remaining []Step
	found := false

	for _, step := range f.Steps {
		if step.Name == lastStep {
			found = true
			if !lastSucceeded {
				remaining = append(remaining, step)
			}
			continue
		}

		if found {
			remaining = append(remaining, step)
		}
	}

	return remaining
}
