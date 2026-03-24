package steprun

// StepRunID is the composite key for a step run document.
// A step run is uniquely identified by its parent run and step name.
type StepRunID struct {
	RunID    string `json:"run_id" bson:"run_id"`
	StepName string `json:"step_name" bson:"step_name"`
}

// StepEndState captures the final state of a step after execution.
type StepEndState struct {
	EndState string         `json:"end_state" bson:"end_state"` // COMPLETED or FAILED
	Reason   string         `json:"reason" bson:"reason"`
	EndedAt  string         `json:"ended_at" bson:"ended_at"`
	Output   map[string]any `json:"output" bson:"output"`
	Duration int            `json:"duration" bson:"duration"`
}

// StepRun tracks the execution state of a single step within a run.
// It is persisted in MongoDB so that on restart, the service can
// determine which step to resume from.
type StepRun struct {
	ID        StepRunID      `json:"_id" bson:"_id"`
	Version   int            `json:"version" bson:"version"`
	CreatedAt string         `json:"created_at" bson:"created_at"`
	Input     map[string]any `json:"input" bson:"input"`
	Ending    *StepEndState  `json:"ending,omitempty" bson:"ending,omitempty"`
}

// IsEndedSuccessfully returns true if the step completed without errors.
func (s *StepRun) IsEndedSuccessfully() bool {
	return s.Ending != nil && s.Ending.EndState == "COMPLETED"
}
