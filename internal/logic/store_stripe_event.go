package logic

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/petechu/idempotent-webhook-relay/internal/db"
	"github.com/petechu/idempotent-webhook-relay/internal/svc"
	"github.com/stripe/stripe-go/v82"
)

type StoreStripeEventLogic struct {
	ctx context.Context
	svc *svc.ServiceContext
}

func NewStoreStripeEventLogic(ctx context.Context, svc *svc.ServiceContext) *StoreStripeEventLogic {
	return &StoreStripeEventLogic{
		ctx: ctx,
		svc: svc,
	}
}

func (l *StoreStripeEventLogic) StoreStripeEvent(event stripe.Event) error {
	item, err := l.svc.OutboxDB.GetOutBoxEvent(l.ctx, event.ID)
	if !errors.Is(err, sql.ErrNoRows) || item.ID > 0 {
		return fmt.Errorf("event with ID %s already exists in the outbox: %w", event.ID, err)
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	_, err = l.svc.OutboxDB.InsertOutboxEvent(l.ctx, db.InsertOutboxEventParams{
		EventID:  event.ID,
		Type:     string(event.Type),
		Payload:  payload,
		Provider: "stripe",
	})
	if err != nil {
		return fmt.Errorf("failed to insert outbox event: %w", err)
	}

	return nil
}
