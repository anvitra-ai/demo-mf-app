package store

import (
	"context"

	"demo-mf-app/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *Store) UpsertOrder(ctx context.Context, doc *models.OrderDoc) error {
	_, err := s.Orders.ReplaceOne(ctx, bson.M{"_id": doc.ID}, doc, options.Replace().SetUpsert(true))
	return err
}

func (s *Store) GetOrder(ctx context.Context, id string) (*models.OrderDoc, error) {
	var doc models.OrderDoc
	err := s.Orders.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (s *Store) ListOrders(ctx context.Context, investorID string) ([]models.OrderDoc, error) {
	filter := bson.M{}
	if investorID != "" {
		filter["investor_id"] = investorID
	}
	cur, err := s.Orders.Find(ctx, filter, options.Find().SetSort(bson.M{"created_at": -1}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	docs := []models.OrderDoc{}
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}

func (s *Store) ListNonTerminalOrders(ctx context.Context) ([]models.OrderDoc, error) {
	statuses := make([]string, 0, len(models.OrderNonTerminal))
	for st := range models.OrderNonTerminal {
		statuses = append(statuses, st)
	}
	cur, err := s.Orders.Find(ctx, bson.M{"status": bson.M{"$in": statuses}})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	docs := []models.OrderDoc{}
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}
