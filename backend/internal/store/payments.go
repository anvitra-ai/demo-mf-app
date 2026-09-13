package store

import (
	"context"

	"demo-mf-app/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *Store) UpsertPayment(ctx context.Context, doc *models.PaymentDoc) error {
	_, err := s.Payments.ReplaceOne(ctx, bson.M{"_id": doc.ID}, doc, options.Replace().SetUpsert(true))
	return err
}

func (s *Store) GetPayment(ctx context.Context, id string) (*models.PaymentDoc, error) {
	var doc models.PaymentDoc
	err := s.Payments.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (s *Store) ListPayments(ctx context.Context, investorID string) ([]models.PaymentDoc, error) {
	filter := bson.M{}
	if investorID != "" {
		filter["investor_id"] = investorID
	}
	cur, err := s.Payments.Find(ctx, filter, options.Find().SetSort(bson.M{"created_at": -1}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	docs := []models.PaymentDoc{}
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}

func (s *Store) ListNonTerminalPayments(ctx context.Context) ([]models.PaymentDoc, error) {
	statuses := make([]string, 0, len(models.PaymentNonTerminal))
	for st := range models.PaymentNonTerminal {
		statuses = append(statuses, st)
	}
	cur, err := s.Payments.Find(ctx, bson.M{"status": bson.M{"$in": statuses}})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	docs := []models.PaymentDoc{}
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}
