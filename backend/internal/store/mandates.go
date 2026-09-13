package store

import (
	"context"

	"demo-mf-app/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *Store) UpsertMandate(ctx context.Context, doc *models.MandateDoc) error {
	_, err := s.Mandates.ReplaceOne(ctx, bson.M{"_id": doc.ID}, doc, options.Replace().SetUpsert(true))
	return err
}

func (s *Store) GetMandate(ctx context.Context, id string) (*models.MandateDoc, error) {
	var doc models.MandateDoc
	err := s.Mandates.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (s *Store) ListNonTerminalMandates(ctx context.Context) ([]models.MandateDoc, error) {
	statuses := make([]string, 0, len(models.MandateNonTerminal))
	for st := range models.MandateNonTerminal {
		statuses = append(statuses, st)
	}
	cur, err := s.Mandates.Find(ctx, bson.M{"status": bson.M{"$in": statuses}})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	docs := []models.MandateDoc{}
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}
