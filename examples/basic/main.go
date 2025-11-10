package basic_connection

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

	// Create a channel
	channel := client.Channel("room:123", &realtime.ChannelConfig{})

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

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down...")
}
