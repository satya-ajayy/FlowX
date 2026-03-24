package flow

import (
	"context"
	"fmt"
	"time"
)

// Dummy is a sample flow for testing the execution pipeline end-to-end.
// It simulates three sequential steps: validation, processing, and notification.
var Dummy = Flow{
	Name: "dummy_flow",
	Steps: []Step{
		{
			Name:        "validate_input",
			Description: "Validates the incoming input payload",
			Execute: func(ctx context.Context, input map[string]any) (map[string]any, error) {
				time.Sleep(2 * time.Second)

				name, ok := input["name"]
				if !ok {
					return nil, fmt.Errorf("missing required field: name")
				}

				return map[string]any{
					"name":         name,
					"validated_at": time.Now().UTC().Format(time.RFC3339),
				}, nil
			},
		},
		{
			Name:        "process_data",
			Description: "Processes the validated data",
			Execute: func(ctx context.Context, input map[string]any) (map[string]any, error) {
				time.Sleep(3 * time.Second)

				return map[string]any{
					"name":         input["name"],
					"processed":    true,
					"processed_at": time.Now().UTC().Format(time.RFC3339),
				}, nil
			},
		},
		{
			Name:        "send_notification",
			Description: "Sends a completion notification",
			Execute: func(ctx context.Context, input map[string]any) (map[string]any, error) {
				time.Sleep(1 * time.Second)

				fmt.Printf("[send_notification] Notification sent for: %v\n", input["name"])
				return map[string]any{
					"name":      input["name"],
					"notified":  true,
					"completed": true,
				}, nil
			},
		},
	},
}
