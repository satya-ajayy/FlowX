package run

// Run represents a single execution instance of a flow, persisted in MongoDB.
// Each API request creates one Run, which is then enqueued for processing.
// On service restart, incomplete runs are re-enqueued automatically.
type Run struct {
	ID             string         `json:"_id" bson:"_id"`
	CreatedAt      string         `json:"created_at" bson:"created_at"`
	Input          map[string]any `json:"input" bson:"input"`
	IsCompleted    bool           `json:"is_completed" bson:"is_completed"`
	CompletedAt    string         `json:"completed_at" bson:"completed_at"`
	LastStepStatus bool           `json:"last_step_status" bson:"last_step_status"`
}
