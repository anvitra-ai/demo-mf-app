package store

import (
	"context"
	"time"

	"demo-mf-app/internal/models"

	"go.mongodb.org/mongo-driver/mongo"
)

// MarkDeliveryProcessed records a webhook delivery id. It returns
// (alreadyProcessed=true, nil) if this delivery was seen before, so the
// caller can skip re-applying it (guide §5.3: at-least-once delivery).
func (s *Store) MarkDeliveryProcessed(ctx context.Context, deliveryID, event string) (bool, error) {
	_, err := s.Deliveries.InsertOne(ctx, &models.WebhookDeliveryDoc{
		ID:         deliveryID,
		Event:      event,
		ReceivedAt: time.Now().UTC(),
	})
	if mongo.IsDuplicateKeyError(err) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return false, nil
}
