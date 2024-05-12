package realtime

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type RealtimeClient struct {
   Url               string
   ApiKey            string

   mu                sync.Mutex
   conn              *websocket.Conn
   closed            chan struct{}
   logger            *log.Logger
   dialTimeout       time.Duration
   reconnectInterval time.Duration
   reconnectDuration time.Duration
   heartbeatDuration time.Duration
   heartbeatInterval time.Duration
}

// Create a new RealtimeClient with user's speicfications
func CreateRealtimeClient(projectRef string, apiKey string) *RealtimeClient {
   realtimeUrl := fmt.Sprintf(
      "wss://%s.supabase.co/realtime/v1/websocket?apikey=%s&log_level=info&vsn=1.0.0",
      projectRef,
      apiKey,
   )
   newLogger := log.Default()

   return &RealtimeClient{
      Url: realtimeUrl,
      ApiKey: apiKey,
      logger: newLogger,
      dialTimeout: 10 * time.Second,
      heartbeatDuration: 5   * time.Second,
      heartbeatInterval: 20  * time.Second,
      reconnectDuration: 60  * time.Second,
      reconnectInterval: 500 * time.Millisecond,
   }
}

// Connect the client with the realtime server
func (client *RealtimeClient) Connect() error {
   if client.isAlive() {
      return nil
   }

   // Attempt to dial the server
   err := client.dialServer()
   if err != nil {
      return fmt.Errorf("Cannot connect to the server: %w", err)
   }

   // client is only alive after the connection has been made
   client.mu.Lock()
   client.closed = make(chan struct{})
   client.mu.Unlock()

   go client.startHeartbeats()

   return nil
}

// Disconnect the client from the realtime server
func (client *RealtimeClient) Disconnect() error {
   client.mu.Lock()
   defer client.mu.Unlock()

   if !client.isAlive() {
      return nil
   }

   err := client.conn.Close(websocket.StatusNormalClosure, "Closing the connection")
   if err != nil {
      client.logger.Println("Failed to close the connection")
      client.logger.Printf("%v", err)
   } else {
      close(client.closed)
   }

   return fmt.Errorf("Failed to close the connection")
}

// Start sending heartbeats to the server to maintain connection
func (client *RealtimeClient) startHeartbeats() {
   for client.isAlive() {
      err1 := client.sendHeartbeat()

      if err1 != nil {
         client.logger.Println("Attempting to to send hearbeat again")

         ctx, cancel := context.WithTimeout(context.Background(), client.reconnectDuration)
         defer cancel()

         err2 := client.reConnect(ctx)
         if err2 != nil {
            client.logger.Printf("Error: %v", err2)
         }
      }
      time.Sleep(client.heartbeatInterval)
   }
}

// Send the heartbeat to the realtime server
func (client *RealtimeClient) sendHeartbeat() error {
   msg := HearbeatMsg{
      TemplateMsg: TemplateMsg{
         Event: HEARTBEAT_EVENT,
         Topic: "phoenix",
         Ref: "",
      },
      Payload: struct{}{},
   }

   ctx, cancel := context.WithTimeout(context.Background(), client.heartbeatDuration)
   defer cancel()

   client.logger.Print(msg)

   err := wsjson.Write(ctx, client.conn, msg)
   if err != nil {
      client.logger.Printf("Error: %v", err)
      client.logger.Printf("Failed to send heartbeat in %f seconds", client.heartbeatDuration.Seconds())

      return err
   }

   return nil
}

// Dial the server with a certain timeout in seconds
func (client *RealtimeClient) dialServer() error {
   client.mu.Lock()
   defer client.mu.Unlock()

   if client.isAlive() {
      return nil
   }

   ctx, cancel := context.WithTimeout(context.Background(), client.dialTimeout)
   defer cancel()

   conn, _, err := websocket.Dial(ctx, client.Url, nil)
   if err != nil {
      return err
   }

   client.conn = conn

   return nil
}

// Keep trying to reconnect every 0.5 seconds until ctx is done/invalidated
func (client *RealtimeClient) reConnect(ctx context.Context) error {
   for client.isAlive() {
      select {
         case <-ctx.Done():
            return fmt.Errorf("Failed to reconnect to the server")
         default:
            err := client.dialServer()
            if err == nil {
               return nil
            }

            time.Sleep(client.reconnectInterval)
      }
   }

   return nil
}

// Check if there's a connection with the realtime server
func (client *RealtimeClient) isAlive() bool {
   if client.closed == nil {
      return false
   }

   select {
      case <-client.closed:
         return false
      default:
         break
   }

   return true
}
