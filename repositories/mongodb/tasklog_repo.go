package mongodb

import (
	// Go Internal Packages
	"context"
	"fmt"

	// Local Packages
	errors "flowx/errors"
	models "flowx/models/tasklog"
	helpers "flowx/utils/helpers"

	// External Packages
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type TaskLogRepository struct {
	collection *mongo.Collection
}

func NewTaskLogRepository(client *mongo.Client) *TaskLogRepository {
	return &TaskLogRepository{
		collection: client.Database("flowx").Collection("tasklogs"),
	}
}

func (r *TaskLogRepository) RecordTaskStart(ctx context.Context, workflowID, taskName string, input map[string]any) error {
	taskID := models.TaskLogID{
		WorkflowID: workflowID,
		TaskName:   taskName,
	}

	curTime := helpers.GetCurrentDateTime()
	taskLog := models.TaskLog{
		Version:   1,
		ID:        taskID,
		CreatedAt: curTime,
		Input:     input,
		Ending:    nil,
	}

	filter := bson.M{"_id": taskID}
	update := bson.M{"$set": taskLog}
	opts := options.UpdateOne().SetUpsert(true)

	res, err := r.collection.UpdateOne(ctx, filter, update, opts)
	if mongo.IsDuplicateKeyError(err) {
		return errors.E(errors.Conflict, "duplicate entry")
	}
	if err != nil {
		return err
	}

	if res.MatchedCount == 0 && res.UpsertedCount == 0 {
		return fmt.Errorf("update failed or no change")
	}
	return nil
}

func (r *TaskLogRepository) RecordTaskEnd(ctx context.Context, workflowID, taskName, state, reason string, duration int, output map[string]any) error {
	taskID := models.TaskLogID{
		WorkflowID: workflowID,
		TaskName:   taskName,
	}

	curTime := helpers.GetCurrentDateTime()
	update := bson.M{
		"$set": bson.M{
			"ending": models.TaskEndState{
				EndState: state,
				Reason:   reason,
				EndedAt:  curTime,
				Output:   output,
				Duration: duration,
			},
		},
	}

	filter := bson.M{"_id": taskID}
	res, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if res.ModifiedCount == 0 {
		return fmt.Errorf("document not modified")
	}
	return nil
}

func (r *TaskLogRepository) GetLastRecordedTask(ctx context.Context, workflowID string) (*models.TaskLog, error) {
	filter := bson.M{"_id.workflow_id": workflowID}
	opts := options.FindOne().SetSort(bson.M{"created_at": -1})

	var task models.TaskLog
	err := r.collection.FindOne(ctx, filter, opts).Decode(&task)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	return &task, nil
}
