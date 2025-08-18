package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/petechu/idempotent-webhook-relay/internal/queue"
)

func main() {
	q, err := queue.NewQueue("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Panicf("Failed to create a new queue: %s", err)
	}
	defer q.Close()

	messages, err := q.Channel.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		log.Panicf("Failed to register a consumer: %s", err)
	}

	forever := make(chan os.Signal, 1)
	signal.Notify(forever, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for msg := range messages {
			log.Printf("Received a message: %s", msg.Body)
		}
	}()

	fmt.Println(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
