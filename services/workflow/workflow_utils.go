package workflow

import (
	// Go Internal Packages
	"context"

	// External Packages
	"go.uber.org/zap"
)

func (s *WorkflowService) SquareTask(_ context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	// Extract the required fields from the input
	number, _ := input["number"].(int)
	square := number * number

	s.logger.Info("Successfully Squared Number", zap.Int("number", number),
		zap.Int("square", square))

	output := map[string]interface{}{
		"number": square,
	}

	return output, nil
}
