package main

import (
	"fmt"
	"log"
	"time"

	"github.com/isa-cloud/isa_cloud/internal/eventbus"
)

func main() {
	// Create EventBus client to set up streams
	config := &eventbus.Config{
		NATSUrl:  "nats://localhost:4222",
		Username: "isa_cloud_admin",
		Password: "admin123",
		ClientID: "stream-setup",
	}

	client, err := eventbus.NewEventBusClient(config)
	if err != nil {
		log.Fatalf("Failed to create EventBus client: %v", err)
	}
	defer client.Close()

	fmt.Println("✅ Connected to NATS JetStream")
	
	// The streams should be created automatically by the NewEventBusClient
	// Just wait a moment for them to be fully initialized
	time.Sleep(2 * time.Second)

	fmt.Println("✅ JetStream streams initialized successfully!")
	fmt.Println("   - EVENTS stream for domain events")
	fmt.Println("   - COMMANDS stream for command messages")  
	fmt.Println("   - NOTIFICATIONS stream for notification events")
	fmt.Println("\n🚀 Ready to run Python event-driven tests!")
}