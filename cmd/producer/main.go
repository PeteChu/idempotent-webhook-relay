package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/petechu/idempotent-webhook-relay/internal/queue"
	"github.com/rabbitmq/amqp091-go"
)

func main() {
	q, err := queue.NewQueue("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Panicf("Failed to create a new queue: %s", err)
	}
	defer q.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	forever := make(chan os.Signal, 1)
	signal.Notify(forever, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for i := 1; ; i++ {
			body := fmt.Sprintf("Hello World! %d", i)

			if err := q.Channel.PublishWithContext(ctx, "", q.Name, false, false, amqp091.Publishing{
				DeliveryMode: amqp091.Persistent,
				ContentType:  "text/plain",
				Body:         []byte(body),
			}); err != nil {
				fmt.Printf("Failed to publish a message: %s\n", err)
			}
			time.Sleep(time.Duration(rand.Intn(1e3)) * time.Millisecond)

		}
	}()

	fmt.Println(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
