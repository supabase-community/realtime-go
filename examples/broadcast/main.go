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

	// Create a channel with broadcast configuration
	channelConfig := &realtime.ChannelConfig{}
	channelConfig.Broadcast.Self = true // Receive your own broadcasts
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

	// Listen for broadcast messages
	channel.OnBroadcast("cursor", func(payload json.RawMessage) {
		var cursorPosition struct {
			X int `json:"x"`
			Y int `json:"y"`
		}
		if err := json.Unmarshal(payload, &cursorPosition); err != nil {
			log.Printf("Failed to parse cursor position: %v", err)
			return
		}
		fmt.Printf("Received cursor position: x=%d, y=%d\n", cursorPosition.X, cursorPosition.Y)
	})

	// Send a broadcast message every 2 seconds
	go func() {
		for {
			cursorPosition := struct {
				X int `json:"x"`
				Y int `json:"y"`
			}{
				X: int(time.Now().Unix() % 100),
				Y: int(time.Now().Unix() % 100),
			}

			if err := channel.SendBroadcast("cursor", cursorPosition); err != nil {
				log.Printf("Failed to broadcast: %v", err)
			} else {
				fmt.Printf("Sent cursor position: x=%d, y=%d\n", cursorPosition.X, cursorPosition.Y)
			}

			time.Sleep(2 * time.Second)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down...")
}
