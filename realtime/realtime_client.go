package realtime

import (
	"context"
	"fmt"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
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

   return nil
}

// Disconnect the client from the realtime server
func (client *RealtimeClient) Disconnect() error {
   if client.closed == nil {
      return nil
   }

   close(client.closed)
   err := client.conn.Close(websocket.StatusNormalClosure, "Closing the connection")

   return err
}

// Start sending heartbeats to the server to maintain connection
func (client *RealtimeClient) startHeartbeats() {
Loop:
   for {
      select {
         case <-client.closed:
            client.closed = nil
            break Loop

         default:
            client.sendHeartbeat()
            time.Sleep(client.heartbeatDuration * time.Second)
      }
   }
}

func (client *RealtimeClient) sendHeartbeat() {
   // Send the heartbeat
   msg := HearbeatMsg{
      TemplateMsg: TemplateMsg{
         Event: HEARTBEAT_EVENT,
         Topic: "phoenix",
         Ref: "",
      },
      Payload: struct{}{},
   }

   ctx, cancel := context.WithCancel(context.Background())
   defer cancel()

   err := wsjson.Write(ctx, client.conn, msg)
   if err != nil {
      fmt.Println(websocket.CloseStatus(err))
   }

   fmt.Println(msg)
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
