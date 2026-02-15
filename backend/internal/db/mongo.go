package db

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Mongo struct {
	Client *mongo.Client
	DB     *mongo.Database
}

func ConnectMongo(ctx context.Context, mongoURI string) (*Mongo, error) {
	// timeout
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, err
	}

	// ping when connect
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, err
	}

	// db name from URI( on .env we use /cinema)
	db := client.Database("cinema")

	return &Mongo{
		Client: client,
		DB:     db,
	}, nil
}
