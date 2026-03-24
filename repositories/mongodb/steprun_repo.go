package mongodb

import (
	// Go Internal Packages
	"context"
	"fmt"

	// Local Packages
	errors "flowx/errors"
	models "flowx/models/steprun"
	helpers "flowx/utils/helpers"

	// External Packages
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// StepRunRepository handles all MongoDB operations for the "step_runs" collection.
type StepRunRepository struct {
	collection *mongo.Collection
}

// NewStepRunRepository creates a new StepRunRepository backed by the "step_runs" collection.
func NewStepRunRepository(client *mongo.Client) *StepRunRepository {
	return &StepRunRepository{
		collection: client.Database("flowx").Collection("step_runs"),
	}
}

// RecordStepStart upserts a step run document when execution begins.
// Uses upsert to handle idempotent restarts safely.
func (r *StepRunRepository) RecordStepStart(ctx context.Context, runID, stepName string, input map[string]any) error {
	stepID := models.StepRunID{
		RunID:    runID,
		StepName: stepName,
	}

	curTime := helpers.GetCurrentDateTime()
	stepRun := models.StepRun{
		Version:   1,
		ID:        stepID,
		CreatedAt: curTime,
		Input:     input,
		Ending:    nil,
	}

	filter := bson.M{"_id": stepID}
	update := bson.M{"$set": stepRun}
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

// RecordStepEnd updates a step run with its final execution state (COMPLETED or FAILED).
func (r *StepRunRepository) RecordStepEnd(ctx context.Context, runID, stepName, state, reason string, duration int, output map[string]any) error {
	stepID := models.StepRunID{
		RunID:    runID,
		StepName: stepName,
	}

	curTime := helpers.GetCurrentDateTime()
	update := bson.M{
		"$set": bson.M{
			"ending": models.StepEndState{
				EndState: state,
				Reason:   reason,
				EndedAt:  curTime,
				Output:   output,
				Duration: duration,
			},
		},
	}

	filter := bson.M{"_id": stepID}
	res, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if res.ModifiedCount == 0 {
		return fmt.Errorf("document not modified")
	}
	return nil
}

// GetLastRecordedStep returns the most recently created step run for a given run.
// Returns nil if no steps have been recorded yet (fresh run).
func (r *StepRunRepository) GetLastRecordedStep(ctx context.Context, runID string) (*models.StepRun, error) {
	filter := bson.M{"_id.run_id": runID}
	opts := options.FindOne().SetSort(bson.M{"created_at": -1})

	var stepRun models.StepRun
	err := r.collection.FindOne(ctx, filter, opts).Decode(&stepRun)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	return &stepRun, nil
}
