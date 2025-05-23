package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"strings"

	"github.com/zcubbs/sbomer/config"
	"github.com/zcubbs/sbomer/internal/db"
	"github.com/zcubbs/sbomer/internal/fetcher"
	"github.com/zcubbs/sbomer/internal/rabbitmq"
)

func encodeAMQPVhost(uri string) string {
	parts := strings.SplitN(uri, "/", 4)
	if len(parts) == 4 {
		// If the vhost doesn't start with %2F, prepend it to encode the leading slash
		vhost := parts[3]
		if !strings.HasPrefix(vhost, "%2F") {
			vhost = "%2F" + vhost
		}
		// No need for url.PathEscape here since we're only encoding the leading slash
		parts[3] = vhost
		return strings.Join(parts, "/")
	}
	return uri
}

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.LUTC)

	log.Println("Starting sbomer fetcher...")
	log.Println("Loading configuration...")

	// Load configuration
	cfg, err := config.LoadConfig("")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create context that will be canceled on SIGINT or SIGTERM
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Printf("Received signal %v, shutting down...", sig)
		cancel()
	}()

	// Initialize database connection
	database, err := db.New(ctx, cfg.GetDatabaseURI())
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Initialize RabbitMQ publisher
	rabbitConfig := rabbitmq.ConsumerConfig{
		URI:           encodeAMQPVhost(cfg.AMQP.URI),
		Exchange:      cfg.AMQP.Exchange,
		ExchangeType:  cfg.AMQP.ExchangeType,
		RoutingKey:    cfg.AMQP.RoutingKey,
		ConsumerGroup: cfg.AMQP.ConsumerGroup,
	}

	publisher, err := rabbitmq.New(rabbitConfig)
	if err != nil {
		log.Fatalf("Failed to create RabbitMQ publisher: %v", err)
	}
	defer publisher.Close()

	// Create fetcher service
	fetcherConfig := fetcher.Config{
		GitLabToken:   cfg.GitLab.Token,
		GitLabURL:     fmt.Sprintf("%s://%s", cfg.GitLab.Scheme, cfg.GitLab.Host),
		Schedule:      cfg.Fetcher.Schedule,
		BatchSize:     cfg.Fetcher.BatchSize,
		CoolOffSecs:   cfg.Fetcher.CoolOffSecs,
		GroupIDs:      cfg.Fetcher.GroupIDs,
		ExcludeTopics: cfg.Fetcher.ExcludeTopics,
		IncludeTopics: cfg.Fetcher.IncludeTopics,
		Publisher:     publisher,
		DB:            database,
	}

	service, err := fetcher.New(fetcherConfig)
	if err != nil {
		log.Fatalf("Failed to create fetcher service: %v", err)
	}
	defer service.Stop()

	// Start the service
	if err := service.Start(ctx); err != nil {
		log.Fatalf("Failed to start fetcher service: %v", err)
	}

	// For "once" mode, we're done after the service completes
	if cfg.Fetcher.Schedule == "once" {
		return
	}

	// For scheduled mode, wait for context cancellation
	<-ctx.Done()
	log.Println("Shutting down fetcher service...")
}
