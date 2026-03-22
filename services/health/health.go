package health

import (
	// Go Internal Packages
	"context"

	// External Packages
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
)

// HealthCheckService is the service for checking the health of the database connections.
type HealthCheckService struct {
	logger      *zap.Logger
	mongoClient *mongo.Client
}

// NewService creates a new HealthCheckService instance and returns the instance.
func NewService(logger *zap.Logger, mongoClient *mongo.Client) *HealthCheckService {
	return &HealthCheckService{
		logger:      logger,
		mongoClient: mongoClient,
	}
}

// Health checks the health of the database connections and returns true if all the connections are healthy.
func (h *HealthCheckService) HealthCheck(ctx context.Context) bool {
	// Check MongoDB Ping
	if mongoPingErr := h.mongoClient.Ping(ctx, nil); mongoPingErr != nil {
		h.logger.Error("Mongo ping failed", zap.Error(mongoPingErr))
		return false
	}

	// Return true if all the connections are healthy.
	return true
}
