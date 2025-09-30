package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
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
	"github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	Context context.Context
	DB      *db.Queries
}

func main() {
	ctx := context.Background()
	q, err := queue.NewQueue(ctx, "amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Panicf("Failed to create a new queue: %s", err)
	}
	defer q.Close()

	cfg := utils.Must(config.LoadConfig())
	conn := utils.Must(pgx.Connect(ctx, cfg.DatabaseURL()))
	query := db.New(conn)

	consumer := Consumer{
		Context: ctx,
		DB:      query,
	}

	messages, err := q.Channel.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		log.Panicf("Failed to register a consumer: %s", err)
	}

	forever := make(chan os.Signal, 1)
	signal.Notify(forever, syscall.SIGINT, syscall.SIGTERM)

	go consumer.readMessages(messages)

	fmt.Println(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

func (c Consumer) readMessages(messages <-chan amqp091.Delivery) {
	const workerCount = 10
	jobs := make(chan db.Outbox)

	for range workerCount {
		go func() {
			for event := range jobs {
				if err := utils.Backoff(
					func() error { return processEvent(event) },
					5,
				); err != nil {
					c.processFailed(event.ID, err)
				} else {
					c.processSucceeded(event.ID)
				}
			}
		}()
	}

	for msg := range messages {
		log.Printf("Received a message: %s", msg.Body)

		event := db.Outbox{}
		if err := json.Unmarshal(msg.Body, &event); err != nil {
			log.Printf("Failed to unmarshal message body: %s", err)
			continue
		}

		idempotencyKey := event.EventID
		event, err := c.DB.GetOutBoxEvent(c.Context, idempotencyKey)
		if err != nil {
			c.processFailed(event.ID, err)
			continue
		}

		jobs <- event
	}
	close(jobs)
}

// simulate event processing; sometimes it fails
func processEvent(event db.Outbox) error {
	duration := rand.Intn(451) + 50
	time.Sleep(time.Duration(duration) * time.Millisecond)

	num := time.Now().UnixNano() % 3
	if num == 0 {
		// simulate failure
		return fmt.Errorf("failed to process event: %s", event.EventID)
	}
	return nil
}

func (c Consumer) processFailed(eventID int32, err error) {
	updateErr := c.DB.UpdateOutboxEvent(c.Context, db.UpdateOutboxEventParams{
		ID: eventID,
		Status: pgtype.Text{
			Valid:  true,
			String: "process_failed",
		},
		LastError: pgtype.Text{
			Valid:  true,
			String: err.Error(),
		},
		LastAttemptAt: pgtype.Timestamptz{
			Valid: true,
			Time:  time.Now(),
		},
	})
	if updateErr != nil {
		log.Printf("Failed to update outbox event (processFailed): %v", updateErr)
	}
}

func (c Consumer) processSucceeded(eventID int32) {
	updateErr := c.DB.UpdateOutboxEvent(c.Context, db.UpdateOutboxEventParams{
		ID: eventID,
		Status: pgtype.Text{
			Valid:  true,
			String: "processed",
		},
		LastError: pgtype.Text{
			Valid:  true,
			String: "",
		},
		LastAttemptAt: pgtype.Timestamptz{
			Valid: true,
			Time:  time.Now(),
		},
	})
	if updateErr != nil {
		log.Printf("Failed to update outbox event (processSucceeded): %v", updateErr)
	}
}
