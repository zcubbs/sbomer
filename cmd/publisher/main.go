package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Message struct {
	ProjectID int `json:"project_id"`
}

func main() {
	// Command line flags
	projectID := flag.Int("project", 1234, "GitLab project ID")
	count := flag.Int("count", 1, "Number of messages to send")
	uri := flag.String("uri", "amqp://guest:guest@localhost:5672/", "RabbitMQ URI")
	queue := flag.String("queue", "sbomer", "Queue name")
	flag.Parse()

	// Connect to RabbitMQ
	conn, err := amqp.Dial(*uri)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	// Create a channel
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open channel: %v", err)
	}
	defer ch.Close()

	// Declare the queue
	q, err := ch.QueueDeclare(
		*queue, // name
		true,   // durable
		false,  // delete when unused
		false,  // exclusive
		false,  // no-wait
		nil,    // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}

	// Create context for publishing
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Publish messages
	for i := 0; i < *count; i++ {
		msg := Message{
			ProjectID: *projectID,
		}

		body, err := json.Marshal(msg)
		if err != nil {
			log.Fatalf("Failed to marshal message: %v", err)
		}

		err = ch.PublishWithContext(ctx,
			"",     // exchange
			q.Name, // routing key
			false,  // mandatory
			false,  // immediate
			amqp.Publishing{
				ContentType: "application/json",
				Body:        body,
			})
		if err != nil {
			log.Fatalf("Failed to publish message: %v", err)
		}

		fmt.Printf("Published message %d with project ID: %d\n", i+1, *projectID)
	}
}
