// Package store wraps MongoDB access for the local mirror documents defined
// in internal/models.
package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Store struct {
	client *mongo.Client

	Investors  *mongo.Collection
	Accounts   *mongo.Collection
	Mandates   *mongo.Collection
	Orders     *mongo.Collection
	Payments   *mongo.Collection
	Deliveries *mongo.Collection
}

func Connect(ctx context.Context, uri, dbName string) (*Store, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	db := client.Database(dbName)
	return &Store{
		client:     client,
		Investors:  db.Collection("investors"),
		Accounts:   db.Collection("investment_accounts"),
		Mandates:   db.Collection("mandates"),
		Orders:     db.Collection("orders"),
		Payments:   db.Collection("payments"),
		Deliveries: db.Collection("webhook_deliveries"),
	}, nil
}

func (s *Store) Disconnect(ctx context.Context) error {
	return s.client.Disconnect(ctx)
}
