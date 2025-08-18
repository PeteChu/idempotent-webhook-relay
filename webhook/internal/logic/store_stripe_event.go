package logic

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/petechu/idempotent-webhook-relay/db"
	"github.com/petechu/idempotent-webhook-relay/webhook/internal/svc"
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
		return err
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// Store the event in the outbox
	if _, err := l.svc.OutboxDB.InsertOutboxEvent(l.ctx, db.InsertOutboxEventParams{
		EventID:  event.ID,
		Type:     string(event.Type),
		Payload:  payload,
		Provider: "stripe",
	}); err != nil {
		return err
	}

	return nil
}
