package workflow

import (
	// Go Internal Packages
	"context"
)

type Task struct {
	Name        string
	Description string
	Cleanup     func(ctx context.Context, input map[string]interface{}) error
	Execute     func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)
}

type TaskList []Task

type Workflow struct {
	Name  string   `json:"name" bson:"name"`
	Tasks TaskList `json:"tasks" bson:"tasks"`
}

func (tasks TaskList) GetNames() []string {
	var names []string
	for _, task := range tasks {
		names = append(names, task.Name)
	}
	return names
}

func (w Workflow) GetAllTasks() TaskList {
	return w.Tasks
}

func (w Workflow) GetPendingTasks(lastExecutedTask string, lastTaskStatus bool) []Task {
	var remaining []Task
	found := false

	for _, task := range w.Tasks {
		if task.Name == lastExecutedTask {
			found = true
			if !lastTaskStatus {
				remaining = append(remaining, task)
			}
			continue
		}

		if found {
			remaining = append(remaining, task)
		}
	}

	return remaining
}

type WorkflowDBModel struct {
	ID             string                 `json:"_id" bson:"_id"`
	CreatedAt      string                 `json:"created_at" bson:"created_at"`
	Input          map[string]interface{} `json:"input" bson:"input"`
	IsCompleted    bool                   `json:"is_completed" bson:"is_completed"`
	CompletedAt    string                 `json:"completed_at" bson:"completed_at"`
	LastTaskStatus bool                   `json:"last_task_status" bson:"last_task_status"`
}
