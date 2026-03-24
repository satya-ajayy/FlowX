package mongodb

import (
	// Go Internal Packages
	"context"

	// Local Packages
	models "flowx/models/run"
	helpers "flowx/utils/helpers"

	// External Packages
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// RunRepository handles all MongoDB operations for the "runs" collection.
type RunRepository struct {
	collection *mongo.Collection
}

// NewRunRepository creates a new RunRepository backed by the "runs" collection.
func NewRunRepository(client *mongo.Client) *RunRepository {
	return &RunRepository{
		collection: client.Database("flowx").Collection("runs"),
	}
}

// Create inserts a new run document into MongoDB.
func (r *RunRepository) Create(ctx context.Context, run models.Run) error {
	_, err := r.collection.InsertOne(ctx, run)
	return err
}

// GetIncomplete returns all runs that have not yet finished executing.
// These are re-enqueued on service startup for recovery.
func (r *RunRepository) GetIncomplete(ctx context.Context) ([]models.Run, error) {
	filter := bson.M{"is_completed": false}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}

	var results []models.Run
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	return results, nil
}

// MarkComplete updates a run as completed with a timestamp and success status.
func (r *RunRepository) MarkComplete(ctx context.Context, runID string) error {
	curTime := helpers.GetCurrentDateTime()
	update := bson.M{
		"$set": bson.M{
			"is_completed":     true,
			"completed_at":     curTime,
			"last_step_status": true,
		},
	}

	filter := bson.M{"_id": runID}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}
