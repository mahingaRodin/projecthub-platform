package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Connect retries until MongoDB answers a ping or the timeout expires.
func Connect(uri, dbName string, timeout time.Duration) (*mongo.Client, *mongo.Database, error) {
	// Every operation gets a 10s deadline unless the caller sets a shorter one.
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri).SetTimeout(10*time.Second))
	if err != nil {
		return nil, nil, err
	}

	deadline := time.Now().Add(timeout)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err = client.Ping(ctx, nil)
		cancel()
		if err == nil {
			log.Printf("connected to MongoDB database %q", dbName)
			return client, client.Database(dbName), nil
		}
		if time.Now().After(deadline) {
			return nil, nil, fmt.Errorf("could not reach MongoDB within %s: %w", timeout, err)
		}
		log.Printf("waiting for MongoDB: %v", err)
		time.Sleep(2 * time.Second)
	}
}

func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	if _, err := db.Collection("users").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}
	if _, err := db.Collection("projects").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "ownerId", Value: 1}}},
		{Keys: bson.D{{Key: "memberIds", Value: 1}}},
	}); err != nil {
		return err
	}
	_, err := db.Collection("tasks").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "projectId", Value: 1}}},
		{Keys: bson.D{{Key: "assigneeId", Value: 1}}},
	})
	return err
}
