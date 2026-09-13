package store

import (
	"context"

	"demo-mf-app/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *Store) UpsertAccount(ctx context.Context, doc *models.AccountDoc) error {
	_, err := s.Accounts.ReplaceOne(ctx, bson.M{"_id": doc.ID}, doc, options.Replace().SetUpsert(true))
	return err
}

func (s *Store) GetAccount(ctx context.Context, id string) (*models.AccountDoc, error) {
	var doc models.AccountDoc
	err := s.Accounts.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (s *Store) ListAccounts(ctx context.Context, investorID string) ([]models.AccountDoc, error) {
	filter := bson.M{}
	if investorID != "" {
		filter["investor_id"] = investorID
	}
	cur, err := s.Accounts.Find(ctx, filter, options.Find().SetSort(bson.M{"created_at": -1}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	docs := []models.AccountDoc{}
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}
