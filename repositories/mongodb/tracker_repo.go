package mongodb

import (
	// Go Internal Packages
	"context"
	"fmt"

	// Local Packages
	errors "flowx/errors"
	models "flowx/models/tracker"
	helpers "flowx/utils/helpers"

	// External Packages
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type TrackerRepository struct {
	client     *mongo.Client
	database   string
	collection string
}

func NewTrackerRepository(client *mongo.Client) *TrackerRepository {
	return &TrackerRepository{
		client:     client,
		database:   "flowx",
		collection: "tracker",
	}
}

func (r *TrackerRepository) RecordTaskStart(ctx context.Context, workflowID, taskName string, input map[string]interface{}) error {
	collection := r.client.Database(r.database).Collection(r.collection)

	taskID := models.TaskTrackerID{
		WorkflowID: workflowID,
		TaskName:   taskName,
	}

	curTime := helpers.GetCurrentDateTime()
	taskLog := models.TaskTracker{
		Version:   1,
		ID:        taskID,
		CreatedAt: curTime,
		Input:     input,
		Ending:    nil,
	}

	filter := bson.M{"_id": taskID}
	update := bson.M{"$set": taskLog}
	opts := options.Update().SetUpsert(true)

	res, err := collection.UpdateOne(ctx, filter, update, opts)
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

func (r *TrackerRepository) RecordTaskEnd(ctx context.Context, workflowID, taskName, state, reason string, duration int, output map[string]interface{}) error {
	collection := r.client.Database(r.database).Collection(r.collection)

	taskID := models.TaskTrackerID{
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

	res, err := collection.UpdateOne(ctx, bson.M{"_id": taskID}, update)
	if err != nil {
		return err
	}

	if res.ModifiedCount == 0 {
		return fmt.Errorf("document not modified")
	}
	return nil
}

func (r *TrackerRepository) GetLastRecordedTask(ctx context.Context, workflowID string) (*models.TaskTracker, error) {
	collection := r.client.Database(r.database).Collection(r.collection)
	filter := bson.M{"_id.workflow_id": workflowID}
	opts := options.FindOne().SetSort(bson.M{"created_at": -1})

	var task models.TaskTracker
	err := collection.FindOne(ctx, filter, opts).Decode(&task)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &task, nil
}
