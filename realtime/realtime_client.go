package realtime

import (
	"context"
	"fmt"
	"time"

	"nhooyr.io/websocket"
)

type RealtimeClient struct {
   Url               string
   ApiKey            string

   conn              *websocket.Conn
   closed            chan struct{}
   dialTimeout       time.Duration
   heartbeatDuration time.Duration
}

// Create a new RealtimeClient with user's speicfications
func CreateRealtimeClient(projectRef string, apiKey string) *RealtimeClient {
   realtimeUrl := fmt.Sprintf(
      "wss://%s.supabase.co/realtime/v1/websocket?apikey=%s&log_level=info&vsn=1.0.0",
      projectRef,
      apiKey,
   )

   return &RealtimeClient{
      Url: realtimeUrl,
      ApiKey: apiKey,
      dialTimeout: 10,
      heartbeatDuration: 20,
   }
}

// Connect the client with the realtie server
func (client *RealtimeClient) Connect() error {
   if client.closed != nil {
      return nil
   }
   
   client.closed = make(chan struct{})

   // Attempt to dial the server
   err := client.dialServer()
   if err != nil {
      return fmt.Errorf("Cannot connect to the server: %w", err)
   }

   // Start sending heartbeat to keep the connection alive
   go client.startHeartbeats() 

   return nil
}

// Disconnect the client from the realtime server
func (client *RealtimeClient) Disconnect() error {
   err := client.conn.CloseNow()

   return err
}

// Start sending heartbeats to the server to maintain connection
func (client *RealtimeClient) startHeartbeats() {
   isBeating := true
   for isBeating {
      select {
         case <-client.closed:
            isBeating = false
            break

         default:
            client.sendHeartbeat()
            time.Sleep(client.heartbeatDuration * time.Second)
      }
   }
}

func (client *RealtimeClient) sendHeartbeat() {
   // Send the heartbeat
}

// Dial the server with a certain timeout in seconds
func (client *RealtimeClient) dialServer() error {
   if client.conn != nil {
      return nil
   }

   ctx, cancel := context.WithTimeout(context.Background(), client.dialTimeout * time.Second) 
   defer cancel()

   conn, _, err := websocket.Dial(ctx, client.Url, nil)
   if err != nil {
      return err
   }

   client.conn = conn 

   return nil
}
