package monitor

type MonitorID struct {
	WorkflowID string `json:"workflow_id" bson:"workflow_id"`
	TaskName   string `json:"task_name" bson:"task_name"`
}

// TaskEndState represents the state of a task at the end of its execution.
type TaskEndState struct {
	EndState string                 `json:"end_state" bson:"end_state"` // COMPLETED or FAILED
	Reason   string                 `json:"reason" bson:"reason"`
	EndedAt  string                 `json:"ended_at" bson:"ended_at"`
	Output   map[string]interface{} `json:"output" bson:"output"`
	Duration int                    `json:"duration" bson:"duration"`
}

type TaskMonitor struct {
	ID        MonitorID              `json:"_id" bson:"_id"`
	Version   int                    `json:"version" bson:"version"`
	CreatedAt string                 `json:"created_at" bson:"created_at"`
	Input     map[string]interface{} `json:"input" bson:"input"`
	Ending    *TaskEndState          `json:"ending,omitempty" bson:"ending,omitempty"`
}

// IsEndedSuccessfully checks if the task ended with a "COMPLETED" state.
func (t *TaskMonitor) IsEndedSuccessfully() bool {
	return t.Ending != nil && t.Ending.EndState == "COMPLETED"
}
