package mongodb

import (
	// Go Internal Packages
	"context"

	// Local Packages
	models "flowx/models/workflow"
	helpers "flowx/utils/helpers"

	// External Packages
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type WorkflowRepository struct {
	collection *mongo.Collection
}

func NewWorkflowRepository(client *mongo.Client) *WorkflowRepository {
	return &WorkflowRepository{
		collection: client.Database("flowx").Collection("workflows"),
	}
}

func (r *WorkflowRepository) CreateWorkflow(ctx context.Context, workflow models.WorkflowDBModel) error {
	_, err := r.collection.InsertOne(ctx, workflow)
	return err
}

func (r *WorkflowRepository) GetInCompleted(ctx context.Context) ([]models.WorkflowDBModel, error) {
	filter := bson.M{"is_completed": false}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}

	var results []models.WorkflowDBModel
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	return results, nil
}

func (r *WorkflowRepository) MarkAsComplete(ctx context.Context, workflowID string) error {
	curTime := helpers.GetCurrentDateTime()
	update := bson.M{
		"$set": bson.M{
			"is_completed":     true,
			"completed_at":     curTime,
			"last_task_status": true,
		},
	}

	filter := bson.M{"_id": workflowID}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}
