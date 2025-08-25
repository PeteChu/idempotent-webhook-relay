package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/petechu/idempotent-webhook-relay/internal/config"
	"github.com/petechu/idempotent-webhook-relay/internal/db"
	"github.com/petechu/idempotent-webhook-relay/internal/queue"
	"github.com/petechu/idempotent-webhook-relay/internal/utils"
)

type Producer struct {
	Context context.Context
	DB      *db.Queries
}

func main() {
	ctx := context.Background()

	cfg := utils.Must(config.LoadConfig())
	conn := utils.Must(pgx.Connect(ctx, cfg.DatabaseURL()))
	query := db.New(conn)

	p := Producer{
		Context: ctx,
		DB:      query,
	}

	q, err := queue.NewQueue(ctx, "amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Panicf("Failed to create a new queue: %s", err)
	}
	defer q.Close()

	forever := make(chan os.Signal, 1)
	signal.Notify(forever, syscall.SIGINT, syscall.SIGTERM)

	fn := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		events, err := query.ListUnprocessedEvents(ctx, []string{
			"payment_intent.created",
			"payment_intent.succeeded",
			"payment_intent.canceled",
			"payment_intent.payment_failed",
		})
		if err != nil {
			fmt.Printf(" [!] Error fetching events: %s\n", err)
		}

		fmt.Printf(" [*] Found %d events to process\n", len(events))
		for _, evt := range events {
			payload, err := json.Marshal(evt)
			if err != nil {
				p.failOnError(
					ctx,
					evt.ID,
					fmt.Errorf("failed to marshal event: %w", err),
				)
				continue
			}
			if err := q.Publish(payload); err != nil {
				p.failOnError(
					ctx,
					evt.ID,
					fmt.Errorf("failed to publish a message: %w", err),
				)
			}

			err = query.UpdateOutboxEvent(ctx, db.UpdateOutboxEventParams{
				ID: evt.ID,
				Status: pgtype.Text{
					String: "pending",
					Valid:  true,
				},
			})
			if err != nil {
				fmt.Printf(" [!] Error updating event %d: %s\n", evt.ID, err)
				p.failOnError(ctx, evt.ID, err)
			}
		}
	}

	go StartJobInterval(fn)

	fmt.Println(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

func StartJobInterval(fn func()) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		fn()
		fmt.Println(" [*] Job executed at", time.Now().Format(time.RFC3339))
	}
}

func (p *Producer) failOnError(ctx context.Context, evtID int32, err error) {
	if err := p.DB.UpdateOutboxEvent(ctx, db.UpdateOutboxEventParams{
		ID: evtID,
		Status: pgtype.Text{
			String: "failed",
			Valid:  true,
		},
		LastError: pgtype.Text{
			String: err.Error(),
			Valid:  true,
		},
	}); err != nil {
		log.Panicf(" [!] Error updating event %d to failed: %s\n", evtID, err)
	}
}
