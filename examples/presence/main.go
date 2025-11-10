package main

import (
	"context"
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

	// Create a channel with presence configuration
	channelConfig := &realtime.ChannelConfig{}
	channelConfig.Presence.Key = "user_id" // Key to identify users
	channel := client.Channel("room:123", channelConfig)

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

	// Listen for presence changes
	channel.OnPresence(func(presenceEvent realtime.PresenceEvent) {
		fmt.Printf("Presence event type: %s\n", presenceEvent.Type)
		fmt.Printf("Presence key: %s\n", presenceEvent.Key)

		if len(presenceEvent.NewPresence) > 0 {
			fmt.Println("New presence:")
			for key, value := range presenceEvent.NewPresence {
				fmt.Printf("  %s: %v\n", key, value)
			}
		}

		if len(presenceEvent.CurrentPresence) > 0 {
			fmt.Println("Current presence:")
			for key, value := range presenceEvent.CurrentPresence {
				fmt.Printf("  %s: %v\n", key, value)
			}
		}
	})

	// Track user presence
	userId := fmt.Sprintf("user-%d", time.Now().Unix())
	presenceData := map[string]interface{}{
		"user_id":      userId,
		"status":       "online",
		"last_seen_at": time.Now().Format(time.RFC3339),
	}

	if err := channel.Track(presenceData); err != nil {
		log.Fatalf("Failed to track presence: %v", err)
	}
	fmt.Printf("Tracking presence for user: %s\n", userId)

	// Update presence data every 30 seconds
	go func() {
		for {
			time.Sleep(30 * time.Second)

			presenceData["last_seen_at"] = time.Now().Format(time.RFC3339)
			if err := channel.Track(presenceData); err != nil {
				log.Printf("Failed to update presence: %v", err)
			} else {
				fmt.Printf("Updated presence for user: %s\n", userId)
			}
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Untrack presence before shutting down
	if err := channel.Untrack(); err != nil {
		log.Printf("Failed to untrack presence: %v", err)
	}

	fmt.Println("Shutting down...")
}
