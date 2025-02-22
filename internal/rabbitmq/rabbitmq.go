package rabbitmq

import (
	"context"
	"fmt"
	"github.com/rabbitmq/amqp091-go"
)

type Consumer struct {
	conn           *amqp091.Connection
	channel        *amqp091.Channel
	exchange       string
	exchangeType   string
	routingKey     string
	consumerGroup  string
	prefetchCount  int
}

type ConsumerConfig struct {
	URI           string
	Exchange      string
	ExchangeType  string
	RoutingKey    string
	ConsumerGroup string
	PrefetchCount int
}

func New(config ConsumerConfig) (*Consumer, error) {
	if config.PrefetchCount == 0 {
		config.PrefetchCount = 1 // Default prefetch count
	}

	conn, err := amqp091.Dial(config.URI)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Set QoS/prefetch for fair dispatch
	err = ch.Qos(
		config.PrefetchCount, // prefetch count
		0,                    // prefetch size
		false,               // global
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	// Declare exchange
	err = ch.ExchangeDeclare(
		config.Exchange,     // name
		config.ExchangeType, // type
		true,               // durable
		false,              // auto-deleted
		false,              // internal
		false,              // no-wait
		nil,                // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declare queue for the consumer group
	queue, err := ch.QueueDeclare(
		config.ConsumerGroup, // name - using consumer group as queue name
		true,                // durable
		false,               // delete when unused
		false,               // exclusive
		false,               // no-wait
		nil,                // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange
	err = ch.QueueBind(
		queue.Name,         // queue name
		config.RoutingKey,  // routing key
		config.Exchange,    // exchange
		false,             // no-wait
		nil,               // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to bind queue: %w", err)
	}

	return &Consumer{
		conn:          conn,
		channel:       ch,
		exchange:      config.Exchange,
		exchangeType:  config.ExchangeType,
		routingKey:    config.RoutingKey,
		consumerGroup: config.ConsumerGroup,
		prefetchCount: config.PrefetchCount,
	}, nil
}

func (c *Consumer) Close() {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *Consumer) Consume(ctx context.Context) (<-chan amqp091.Delivery, error) {
	// Each consumer in the group gets its own consumer tag
	consumerTag := fmt.Sprintf("%s-%d", c.consumerGroup, c.prefetchCount)
	
	return c.channel.Consume(
		c.consumerGroup, // queue
		consumerTag,     // consumer
		false,          // auto-ack
		false,          // exclusive
		false,          // no-local
		false,          // no-wait
		nil,            // args
	)
}

// Publish sends a message to the exchange
func (c *Consumer) Publish(ctx context.Context, body []byte) error {
	return c.channel.PublishWithContext(ctx,
		c.exchange,   // exchange
		c.routingKey, // routing key
		false,        // mandatory
		false,        // immediate
		amqp091.Publishing{
			ContentType:  "application/json",
			Body:        body,
			DeliveryMode: amqp091.Persistent,
		},
	)
}
