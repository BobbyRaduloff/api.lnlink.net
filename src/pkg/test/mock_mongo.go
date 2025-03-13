package test

import (
	"context"
	"fmt"
	"log"

	"api.codprotect.app/src/pkg/global"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MockMongoDB creates a mocked MongoDB instance for testing.
func MockMongoDB(ctx context.Context) (*mongo.Client, string, func(), error) {
	mongoC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "mongo:8",
			ExposedPorts: []string{"27017/tcp"},
			WaitingFor:   wait.ForListeningPort("27017/tcp"),
		},
		Started: true,
	})
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to start MongoDB container: %w", err)
	}

	// Get connection details
	host, err := mongoC.Host(ctx)
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to get MongoDB container host: %w", err)
	}
	port, err := mongoC.MappedPort(ctx, "27017")
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to get MongoDB container port: %w", err)
	}

	// Build MongoDB URI
	mongoURI := fmt.Sprintf("mongodb://%s:%s", host, port.Port())

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		mongoC.Terminate(ctx)
		return nil, "", nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Cleanup function to terminate the container and disconnect
	cleanup := func() {
		if err := client.Disconnect(ctx); err != nil {
			log.Printf("Failed to disconnect MongoDB client: %v", err)
		}
		if err := mongoC.Terminate(ctx); err != nil {
			log.Printf("Failed to terminate MongoDB container: %v", err)
		}
	}

	global.MONGO_CLIENT = client
	global.MONGO_DB_NAME = "api.codprotect.app"

	return client, global.MONGO_DB_NAME, cleanup, nil
}
