package mongodb

import (
	// Go Internal Packages
	"context"

	// Local Packages
	models "flowx/models/workflow"
	helpers "flowx/utils/helpers"

	// External Packages
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type WorkflowRepository struct {
	client     *mongo.Client
	database   string
	collection string
}

func NewWorkflowRepository(client *mongo.Client) *WorkflowRepository {
	return &WorkflowRepository{
		client:     client,
		database:   "flowx",
		collection: "workflows",
	}
}

func (r *WorkflowRepository) CreateWorkflow(ctx context.Context, workflow models.WorkflowDBModel) error {
	collection := r.client.Database(r.database).Collection(r.collection)
	_, err := collection.InsertOne(ctx, workflow)
	if err != nil {
		return err
	}
	return nil
}

func (r *WorkflowRepository) GetInCompleted(ctx context.Context) ([]models.WorkflowDBModel, error) {
	collection := r.client.Database(r.database).Collection(r.collection)
	cursor, err := collection.Find(ctx, bson.M{"is_completed": false})
	if err != nil {
		return nil, err
	}

	var result []models.WorkflowDBModel
	for cursor.Next(ctx) {
		var workflow models.WorkflowDBModel
		if err = cursor.Decode(&workflow); err != nil {
			return nil, err
		}
		result = append(result, workflow)
	}

	return result, nil
}

func (r *WorkflowRepository) MarkAsComplete(ctx context.Context, workflowID string) error {
	collection := r.client.Database(r.database).Collection(r.collection)
	curTime := helpers.GetCurrentDateTime()

	updatedFields := bson.M{
		"$set": bson.M{
			"is_completed":     true,
			"completed_at":     curTime,
			"last_task_status": true,
		},
	}

	_, err := collection.UpdateOne(ctx, bson.M{"_id": workflowID}, updatedFields)
	return err
}
