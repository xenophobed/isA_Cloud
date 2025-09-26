package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// EventBusClient handles all event-driven communication
type EventBusClient struct {
	nc     *nats.Conn
	js     jetstream.JetStream
	config *Config
}

// Config for EventBus client
type Config struct {
	NATSUrl      string
	Username     string
	Password     string
	ClientID     string
	MaxReconnect int
	ReconnectWait time.Duration
}

// NewEventBusClient creates a new EventBus client
func NewEventBusClient(config *Config) (*EventBusClient, error) {
	// Set defaults
	if config.NATSUrl == "" {
		config.NATSUrl = nats.DefaultURL
	}
	if config.MaxReconnect == 0 {
		config.MaxReconnect = 10
	}
	if config.ReconnectWait == 0 {
		config.ReconnectWait = 2 * time.Second
	}

	// Connection options
	opts := []nats.Option{
		nats.Name(config.ClientID),
		nats.MaxReconnects(config.MaxReconnect),
		nats.ReconnectWait(config.ReconnectWait),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Printf("EventBus disconnected: %v", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("EventBus reconnected to %s", nc.ConnectedUrl())
		}),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			log.Printf("EventBus error: %v", err)
		}),
	}

	// Add authentication if provided
	if config.Username != "" && config.Password != "" {
		opts = append(opts, nats.UserInfo(config.Username, config.Password))
	}

	// Connect to NATS
	nc, err := nats.Connect(config.NATSUrl, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	client := &EventBusClient{
		nc:     nc,
		js:     js,
		config: config,
	}

	// Initialize streams
	if err := client.initializeStreams(context.Background()); err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to initialize streams: %w", err)
	}

	return client, nil
}

// initializeStreams creates the necessary JetStream streams
func (c *EventBusClient) initializeStreams(ctx context.Context) error {
	// Events stream for all domain events
	eventsStream, err := c.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        "EVENTS",
		Description: "Stream for all domain events",
		Subjects:    []string{"events.>"},
		Retention:   jetstream.LimitsPolicy,
		MaxAge:      30 * 24 * time.Hour, // 30 days retention
		MaxBytes:    64 * 1024 * 1024, // 64MB max size
		Replicas:    3,
		Storage:     jetstream.MemoryStorage,
	})
	if err != nil {
		return fmt.Errorf("failed to create EVENTS stream: %w", err)
	}
	log.Printf("Events stream created/updated: %s", eventsStream.CachedInfo().Config.Name)

	// Commands stream for command messages
	commandsStream, err := c.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        "COMMANDS",
		Description: "Stream for command messages",
		Subjects:    []string{"commands.>"},
		Retention:   jetstream.WorkQueuePolicy, // Commands are consumed once
		MaxAge:      24 * time.Hour,
		MaxBytes:    32 * 1024 * 1024, // 32MB max size  
		Replicas:    3,
		Storage:     jetstream.MemoryStorage,
	})
	if err != nil {
		return fmt.Errorf("failed to create COMMANDS stream: %w", err)
	}
	log.Printf("Commands stream created/updated: %s", commandsStream.CachedInfo().Config.Name)

	// Notifications stream
	notificationsStream, err := c.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        "NOTIFICATIONS",
		Description: "Stream for notification events",
		Subjects:    []string{"notifications.>"},
		Retention:   jetstream.InterestPolicy, // Keep until consumed
		MaxAge:      7 * 24 * time.Hour,
		MaxBytes:    32 * 1024 * 1024, // 32MB max size
		Replicas:    3,
		Storage:     jetstream.MemoryStorage,
	})
	if err != nil {
		return fmt.Errorf("failed to create NOTIFICATIONS stream: %w", err)
	}
	log.Printf("Notifications stream created/updated: %s", notificationsStream.CachedInfo().Config.Name)

	return nil
}

// PublishEvent publishes a domain event
func (c *EventBusClient) PublishEvent(ctx context.Context, event Event) error {
	// Set timestamp if not set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	// Set event ID if not set
	if event.ID == "" {
		event.ID = generateEventID()
	}

	// Marshal event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Construct subject
	subject := fmt.Sprintf("events.%s.%s", event.Source, event.Type)

	// Publish to JetStream
	ack, err := c.js.PublishAsync(subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	// Wait for acknowledgment
	select {
	case <-ack.Ok():
		log.Printf("Event published: %s [%s]", event.Type, event.ID)
		return nil
	case err := <-ack.Err():
		return fmt.Errorf("event publish failed: %w", err)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// SubscribeToEvents subscribes to events with a durable consumer
func (c *EventBusClient) SubscribeToEvents(ctx context.Context, pattern string, handler EventHandler, opts ...ConsumerOption) error {
	config := &ConsumerConfig{
		Durable:       fmt.Sprintf("%s-%s", c.config.ClientID, pattern),
		FilterSubject: fmt.Sprintf("events.%s", pattern),
		MaxDeliver:    3,
		AckWait:       30 * time.Second,
		AckPolicy:     jetstream.AckExplicitPolicy,
	}

	// Apply options
	for _, opt := range opts {
		opt(config)
	}

	// Get or create stream
	stream, err := c.js.Stream(ctx, "EVENTS")
	if err != nil {
		return fmt.Errorf("failed to get stream: %w", err)
	}

	// Create or get consumer
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       config.Durable,
		FilterSubject: config.FilterSubject,
		MaxDeliver:    config.MaxDeliver,
		AckWait:       config.AckWait,
		AckPolicy:     config.AckPolicy,
		DeliverPolicy: jetstream.DeliverAllPolicy,
	})
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	// Start consuming messages
	consumeCtx, err := consumer.Consume(func(msg jetstream.Msg) {
		var event Event
		if err := json.Unmarshal(msg.Data(), &event); err != nil {
			log.Printf("Failed to unmarshal event: %v", err)
			msg.Nak()
			return
		}

		// Process event
		if err := handler(ctx, event); err != nil {
			log.Printf("Event handler error: %v", err)
			msg.Nak()
			return
		}

		// Acknowledge message
		if err := msg.Ack(); err != nil {
			log.Printf("Failed to ack message: %v", err)
		}
	})

	if err != nil {
		return fmt.Errorf("failed to start consumer: %w", err)
	}

	// Wait for context cancellation
	<-ctx.Done()
	consumeCtx.Stop()
	return nil
}

// PublishCommand publishes a command message
func (c *EventBusClient) PublishCommand(ctx context.Context, command Command) (*CommandResult, error) {
	// Set command ID if not set
	if command.ID == "" {
		command.ID = generateEventID()
	}

	// Marshal command
	data, err := json.Marshal(command)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal command: %w", err)
	}

	// Construct subject
	subject := fmt.Sprintf("commands.%s.%s", command.Target, command.Type)

	// Request-Reply pattern for commands
	msg, err := c.nc.RequestWithContext(ctx, subject, data)
	if err != nil {
		return nil, fmt.Errorf("command request failed: %w", err)
	}

	// Unmarshal result
	var result CommandResult
	if err := json.Unmarshal(msg.Data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal command result: %w", err)
	}

	return &result, nil
}

// Close closes the EventBus connection
func (c *EventBusClient) Close() {
	if c.nc != nil {
		c.nc.Close()
	}
}

// Health checks the health of the EventBus connection
func (c *EventBusClient) Health() error {
	if c.nc == nil || !c.nc.IsConnected() {
		return fmt.Errorf("not connected to NATS")
	}
	return nil
}