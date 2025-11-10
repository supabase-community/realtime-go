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

	"github.com/supabase-community/realtime-go/realtime"
)

func main() {
	// Replace with your Supabase project reference and API key
	projectRef := "your-project-ref"
	apiKey := "your-api-key"

	// Create a new Realtime client
	client := realtime.NewRealtimeClient(projectRef, apiKey)

	// Connect to the Realtime server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	fmt.Println("Connected to Supabase Realtime server")

	// Create a channel for Postgres changes
	// The channel name format is: realtime:{schema}:{table}
	channel := client.Channel("realtime:public:users", &realtime.ChannelConfig{})

	// Subscribe to the channel
	err := channel.Subscribe(context.Background(), func(state realtime.SubscribeState, err error) {
		if err != nil {
			log.Printf("Subscription error: %v", err)
			return
		}

		if state == realtime.SubscribeStateSubscribed {
			fmt.Println("Successfully subscribed to channel")
		}
	})

	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	// Listen for all changes (INSERT, UPDATE, DELETE)
	channel.OnPostgresChange("*", func(change realtime.PostgresChangeEvent) {
		fmt.Printf("Postgres change event: %s\n", change.Type)
		fmt.Printf("Table: %s, Schema: %s\n", change.Table, change.Schema)

		var payload map[string]interface{}
		if err := json.Unmarshal(change.Payload, &payload); err != nil {
			log.Printf("Failed to parse payload: %v", err)
			return
		}

		fmt.Printf("Payload: %v\n", payload)
	})

	// Listen for specific changes
	channel.OnPostgresChange("INSERT", func(change realtime.PostgresChangeEvent) {
		fmt.Println("New record inserted!")

		var payload map[string]interface{}
		if err := json.Unmarshal(change.Payload, &payload); err != nil {
			log.Printf("Failed to parse payload: %v", err)
			return
		}

		if record, ok := payload["record"].(map[string]interface{}); ok {
			fmt.Printf("New record: %v\n", record)
		}
	})

	channel.OnPostgresChange("UPDATE", func(change realtime.PostgresChangeEvent) {
		fmt.Println("Record updated!")

		var payload map[string]interface{}
		if err := json.Unmarshal(change.Payload, &payload); err != nil {
			log.Printf("Failed to parse payload: %v", err)
			return
		}

		if oldRecord, ok := payload["old_record"].(map[string]interface{}); ok {
			fmt.Printf("Old record: %v\n", oldRecord)
		}

		if newRecord, ok := payload["record"].(map[string]interface{}); ok {
			fmt.Printf("New record: %v\n", newRecord)
		}
	})

	channel.OnPostgresChange("DELETE", func(change realtime.PostgresChangeEvent) {
		fmt.Println("Record deleted!")

		var payload map[string]interface{}
		if err := json.Unmarshal(change.Payload, &payload); err != nil {
			log.Printf("Failed to parse payload: %v", err)
			return
		}

		if oldRecord, ok := payload["old_record"].(map[string]interface{}); ok {
			fmt.Printf("Deleted record: %v\n", oldRecord)
		}
	})

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down...")
}
