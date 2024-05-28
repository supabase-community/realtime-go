package realtime

import (
	"context"
	"errors"
	"fmt"
	"io"
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
      reconnectInterval: 500 * time.Millisecond,
   }
}

// Connect the client with the realtime server
func (client *RealtimeClient) Connect() error {
   if client.isClientAlive() {
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

   if !client.isClientAlive() {
      return nil
   }

   err := client.conn.Close(websocket.StatusNormalClosure, "Closing the connection")
   if err != nil {
      if !client.isConnectionAlive(err) {
         client.logger.Println("Connection has already been terminated")
         close(client.closed)
      } else {
         return fmt.Errorf("Failed to close the connection: %w", err)
      }
   } else {
      close(client.closed)
   }
   
   return nil
}

// Start sending heartbeats to the server to maintain connection
func (client *RealtimeClient) startHeartbeats() {
   for client.isClientAlive() {
      err := client.sendHeartbeat()

      if err != nil {
         if client.isConnectionAlive(err) {
            client.logger.Println(err) 
         } else {
            client.logger.Println("Error: lost connection with the server")
            client.logger.Println("Attempting to to send hearbeat again")

            ctx, cancel := context.WithCancel(context.Background())
            defer cancel()

            // there should never be an error returned, since it'll keep trying
            _ = client.reconnect(ctx)
         }
      }

      // in case where the client needs to reconnect with the server,
      // the interval between heartbeats be however long it takes to
      // reconnect plus the number of heartbeatInterval has gone by
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

   client.logger.Print("Sending heartbeat")

   err := wsjson.Write(ctx, client.conn, msg)
   if err != nil {
      return fmt.Errorf("Failed to send hearbeat in %f seconds: %w", client.heartbeatDuration.Seconds(), err)
   }

   return nil
}

// Dial the server with a certain timeout in seconds
func (client *RealtimeClient) dialServer() error {
   client.mu.Lock()
   defer client.mu.Unlock()

   if client.isClientAlive() {
      return nil
   }

   ctx, cancel := context.WithTimeout(context.Background(), client.dialTimeout)
   defer cancel()

   conn, _, err := websocket.Dial(ctx, client.Url, nil)
   if err != nil {
      return fmt.Errorf("Failed to dial the server: %w", err)
   }

   client.conn = conn

   return nil
}

// Keep trying to reconnect every 0.5 seconds until ctx is done/invalidated
func (client *RealtimeClient) reconnect(ctx context.Context) error {
   for client.isClientAlive() {
      client.logger.Println("Attempt to reconnect to the server")

      select {
         case <-ctx.Done():
            return fmt.Errorf("Failed to reconnect to the server within time limit")
         default:
            err := client.dialServer()
            if err == nil {
               return nil
            }

            client.logger.Printf("Failed to reconnect to the server: %s", err)
            time.Sleep(client.reconnectInterval)
      }
   }

   return nil
}

// Check if the realtime client has been killed
func (client *RealtimeClient) isClientAlive() bool {
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

// The underlying package of websocket returns an error if the connection is
// terminated on the server side. Therefore, the state of the connection can 
// be achieved by investigating the error
// Constraints: err must be returned from interacting with the connection
func (client *RealtimeClient) isConnectionAlive(err error) bool {
   return !errors.Is(err, io.EOF)
}
