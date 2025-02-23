package main

import (
	"context"
	"fmt"
	"github.com/zcubbs/sbomer/config"
	"github.com/zcubbs/sbomer/internal/db"
	"github.com/zcubbs/sbomer/internal/gitlab"
	"github.com/zcubbs/sbomer/internal/processor"
	"github.com/zcubbs/sbomer/internal/rabbitmq"
	"github.com/zcubbs/sbomer/internal/syft"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Set up logging
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.LUTC)

	fmt.Println("Starting sbomer...")
	fmt.Println("Loading configuration...")

	// Load configuration
	cfg, err := config.LoadConfig("")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	fmt.Printf("Configuration loaded successfully. Log level: %s\n", cfg.App.LogLevel)

	// Initialize components
	fmt.Println("Initializing components...")

	// Initialize database connection
	ctx := context.Background()
	database, err := db.New(ctx, cfg.GetDatabaseURI())
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Initialize GitLab client
	gitlabClient, err := gitlab.New(
		cfg.GitLab.Token,
		cfg.GitLab.Host,
		cfg.GitLab.Scheme,
		cfg.GitLab.TempDir,
	)
	if err != nil {
		log.Fatalf("Failed to initialize GitLab client: %v", err)
	}

	// Initialize SBOM generator
	sbomGenerator := syft.New(cfg.Syft.Format, cfg.Syft.SyftBinPath)

	// Initialize message processor
	msgProcessor := processor.New(
		database,
		gitlabClient,
		sbomGenerator,
	)

	// Initialize RabbitMQ consumer
	consumer, err := rabbitmq.New(rabbitmq.ConsumerConfig{
		URI:           cfg.AMQP.URI,
		Exchange:      cfg.AMQP.Exchange,
		ExchangeType:  cfg.AMQP.ExchangeType,
		RoutingKey:    cfg.AMQP.RoutingKey,
		ConsumerGroup: cfg.AMQP.ConsumerGroup,
	})
	if err != nil {
		log.Fatalf("Failed to initialize RabbitMQ consumer: %v", err)
	}
	defer consumer.Close()

	// Start consuming messages
	messages, err := consumer.Consume(ctx)
	if err != nil {
		log.Fatalf("Failed to start consuming messages: %v", err)
	}

	fmt.Println("Ready to process messages...")

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Process messages
	go func() {
		for msg := range messages {
			if err := msgProcessor.ProcessMessage(ctx, msg.Body); err != nil {
				log.Printf("Error processing message: %v", err)
			}
			msg.Ack(false)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	fmt.Println("\nShutting down gracefully...")
}
