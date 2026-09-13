package store

import (
	"context"

	"demo-mf-app/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *Store) UpsertInvestor(ctx context.Context, doc *models.InvestorDoc) error {
	_, err := s.Investors.ReplaceOne(ctx, bson.M{"_id": doc.ID}, doc, options.Replace().SetUpsert(true))
	return err
}

func (s *Store) GetInvestor(ctx context.Context, id string) (*models.InvestorDoc, error) {
	var doc models.InvestorDoc
	err := s.Investors.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (s *Store) ListNonTerminalInvestors(ctx context.Context) ([]models.InvestorDoc, error) {
	statuses := make([]string, 0, len(models.InvestorNonTerminal))
	for st := range models.InvestorNonTerminal {
		statuses = append(statuses, st)
	}
	cur, err := s.Investors.Find(ctx, bson.M{"status": bson.M{"$in": statuses}})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	docs := []models.InvestorDoc{}
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}
