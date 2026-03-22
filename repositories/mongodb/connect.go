package mongodb

import (
	// Go Internal Packages
	"context"
	"time"

	// External Packages
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Connect connects to the mongodb server and returns the client.
func Connect(ctx context.Context, uri string) (*mongo.Client, error) {
	// Set the server selection timeout to 5 seconds.
	timeout := time.Second * 5
	opts := options.Client().ApplyURI(uri).SetServerSelectionTimeout(timeout)

	// Create a new MongoDB client with the provided URI and options.
	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, err
	}

	// Ping the MongoDB server to verify the connection.
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	// Return the connected client.
	return client, nil
}
