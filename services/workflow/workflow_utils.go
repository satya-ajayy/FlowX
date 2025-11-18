package workflow

import (
	// Go Internal Packages
	"context"

	// Local Packages
	helpers "flowx/utils/helpers"

	// External Packages
	"go.uber.org/zap"
)

func (s *WorkflowService) BasicTask(_ context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	// Extract the required fields from the input
	name, _ := input["name"].(string)
	jumbledName := helpers.JumbleName(name)

	s.logger.Info("Successfully Jumbled Name", zap.String("name", jumbledName))

	output := map[string]interface{}{
		"name": jumbledName,
	}

	return output, nil
}
