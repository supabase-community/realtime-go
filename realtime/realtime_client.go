package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"nhooyr.io/websocket"
)

// Conn represents a WebSocket connection interface
type Conn interface {
	Close(code websocket.StatusCode, reason string) error
	Read(ctx context.Context) (websocket.MessageType, []byte, error)
	Write(ctx context.Context, messageType websocket.MessageType, data []byte) error
	Ping(ctx context.Context) error
	SetReadLimit(limit int64)
	SetWriteLimit(limit int64)
}

// RealtimeClient represents the main client for Supabase Realtime
type RealtimeClient struct {
	config         *Config
	conn           Conn
	channels       map[string]*channel
	mu             sync.RWMutex
	authToken      string
	ref            int
	refMu          sync.Mutex
	hbTimer        *time.Timer
	hbStop         chan struct{}
	reconnMu       sync.Mutex
	isReconnecting bool
	logger         *log.Logger
}

// NewRealtimeClient creates a new RealtimeClient instance
func NewRealtimeClient(projectRef string, apiKey string) IRealtimeClient {
	config := NewConfig()
	config.URL = fmt.Sprintf(
		"wss://%s.supabase.co/realtime/v1/websocket?apikey=%s&log_level=info&vsn=1.0.0",
		projectRef,
		apiKey,
	)
	config.APIKey = apiKey

	return &RealtimeClient{
		config:   config,
		channels: make(map[string]*channel),
		hbStop:   make(chan struct{}),
		logger:   log.Default(),
	}
}

// Connect establishes a connection to the Supabase Realtime server
func (c *RealtimeClient) Connect(ctx context.Context) error {
	c.reconnMu.Lock()
	if c.isReconnecting {
		c.reconnMu.Unlock()
		return fmt.Errorf("client is already reconnecting")
	}
	c.reconnMu.Unlock()

	opts := &websocket.DialOptions{
		HTTPHeader: make(map[string][]string),
	}

	if c.config.APIKey != "" {
		opts.HTTPHeader["apikey"] = []string{c.config.APIKey}
	}
	if c.authToken != "" {
		opts.HTTPHeader["Authorization"] = []string{fmt.Sprintf("Bearer %s", c.authToken)}
	}

	conn, _, err := websocket.Dial(ctx, c.config.URL, opts)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Wrap the websocket.Conn in our Conn interface
	c.conn = &websocketConnWrapper{conn}
	go c.handleMessages()
	go c.startHeartbeat()

	return nil
}

// websocketConnWrapper wraps *websocket.Conn to implement our Conn interface
type websocketConnWrapper struct {
	*websocket.Conn
}

func (w *websocketConnWrapper) SetWriteLimit(limit int64) {
	// No-op as websocket.Conn doesn't have this method
}

// Disconnect closes the connection to the Supabase Realtime server
func (c *RealtimeClient) Disconnect() error {
	if c.conn != nil {
		close(c.hbStop)
		if c.hbTimer != nil {
			c.hbTimer.Stop()
		}
		return c.conn.Close(websocket.StatusNormalClosure, "Closing the connection")
	}
	return nil
}

// Channel creates a new channel for realtime subscriptions
func (c *RealtimeClient) Channel(topic string, config *ChannelConfig) Channel {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ch, exists := c.channels[topic]; exists {
		return ch
	}

	ch := newChannel(topic, config, c)
	c.channels[topic] = ch
	return ch
}

// SetAuth sets the authentication token for the client
func (c *RealtimeClient) SetAuth(token string) error {
	c.authToken = token
	return nil
}

// GetChannels returns all active channels
func (c *RealtimeClient) GetChannels() map[string]Channel {
	c.mu.RLock()
	defer c.mu.RUnlock()
	channels := make(map[string]Channel)
	for topic, ch := range c.channels {
		channels[topic] = ch
	}
	return channels
}

// RemoveChannel removes a channel from the client
func (c *RealtimeClient) RemoveChannel(ch Channel) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ch, ok := ch.(*channel); ok {
		if _, exists := c.channels[ch.topic]; exists {
			delete(c.channels, ch.topic)
			return ch.Unsubscribe()
		}
	}
	return fmt.Errorf("channel not found")
}

// RemoveAllChannels removes all channels from the client
func (c *RealtimeClient) RemoveAllChannels() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, ch := range c.channels {
		if err := ch.Unsubscribe(); err != nil {
			c.logger.Printf("Error unsubscribing from channel %s: %v", ch.topic, err)
		}
	}
	c.channels = make(map[string]*channel)
	return nil
}

func (c *RealtimeClient) handleMessages() {
	for {
		ctx := context.Background()
		_, data, err := c.conn.Read(ctx)
		if err != nil {
			c.logger.Printf("WebSocket read error: %v", err)
			if c.config.AutoReconnect {
				go c.reconnect()
			}
			return
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			c.logger.Printf("Error unmarshaling message: %v", err)
			continue
		}

		// Handle different message types
		switch msg.Type {
		case "broadcast":
			c.handleBroadcast(msg)
		case "presence":
			c.handlePresence(msg)
		case "postgres_changes":
			c.handlePostgresChanges(msg)
		}
	}
}

func (c *RealtimeClient) startHeartbeat() {
	c.hbTimer = time.NewTimer(c.config.HBInterval)
	defer c.hbTimer.Stop()

	for {
		select {
		case <-c.hbTimer.C:
			if err := c.sendHeartbeat(); err != nil {
				c.logger.Printf("Error sending heartbeat: %v", err)
				if c.config.AutoReconnect {
					go c.reconnect()
				}
			}
		case <-c.hbStop:
			return
		}
	}
}

func (c *RealtimeClient) sendHeartbeat() error {
	heartbeat := struct {
		Type  string `json:"type"`
		Topic string `json:"topic"`
		Event string `json:"event"`
	}{
		Type:  "heartbeat",
		Topic: "phoenix",
		Event: "heartbeat",
	}
	data, err := json.Marshal(heartbeat)
	if err != nil {
		return err
	}
	ctx := context.Background()
	return c.conn.Write(ctx, websocket.MessageText, data)
}

func (c *RealtimeClient) reconnect() {
	c.reconnMu.Lock()
	if c.isReconnecting {
		c.reconnMu.Unlock()
		return
	}
	c.isReconnecting = true
	c.reconnMu.Unlock()

	defer func() {
		c.reconnMu.Lock()
		c.isReconnecting = false
		c.reconnMu.Unlock()
	}()

	backoff := c.config.InitialBackoff
	for i := 0; i < c.config.MaxRetries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
		err := c.Connect(ctx)
		cancel()

		if err == nil {
			// Rejoin all channels
			c.mu.RLock()
			for _, ch := range c.channels {
				go ch.rejoin()
			}
			c.mu.RUnlock()
			return
		}

		c.logger.Printf("Reconnection attempt %d failed: %v", i+1, err)
		if i < c.config.MaxRetries-1 {
			time.Sleep(backoff)
			backoff *= 2
			if backoff > 60*time.Second {
				backoff = 60 * time.Second
			}
		}
	}

	c.logger.Printf("Failed to reconnect after %d attempts", c.config.MaxRetries)
}

func (c *RealtimeClient) nextRef() int {
	c.refMu.Lock()
	defer c.refMu.Unlock()
	c.ref++
	return c.ref
}

func (c *RealtimeClient) handleBroadcast(msg Message) {
	c.mu.RLock()
	ch, exists := c.channels[msg.Topic]
	c.mu.RUnlock()

	if !exists {
		return
	}

	ch.mu.RLock()
	callbacks, exists := ch.callbacks["broadcast:"+msg.Event]
	ch.mu.RUnlock()

	if !exists {
		return
	}

	for _, callback := range callbacks {
		if cb, ok := callback.(func(json.RawMessage)); ok {
			cb(msg.Payload)
		}
	}
}

func (c *RealtimeClient) handlePresence(msg Message) {
	c.mu.RLock()
	ch, exists := c.channels[msg.Topic]
	c.mu.RUnlock()

	if !exists {
		return
	}

	var presenceEvent PresenceEvent
	if err := json.Unmarshal(msg.Payload, &presenceEvent); err != nil {
		c.logger.Printf("Error unmarshaling presence event: %v", err)
		return
	}

	ch.mu.RLock()
	callbacks, exists := ch.callbacks["presence"]
	ch.mu.RUnlock()

	if !exists {
		return
	}

	for _, callback := range callbacks {
		if cb, ok := callback.(func(PresenceEvent)); ok {
			cb(presenceEvent)
		}
	}
}

func (c *RealtimeClient) handlePostgresChanges(msg Message) {
	c.mu.RLock()
	ch, exists := c.channels[msg.Topic]
	c.mu.RUnlock()

	if !exists {
		return
	}

	var changeEvent PostgresChangeEvent
	if err := json.Unmarshal(msg.Payload, &changeEvent); err != nil {
		c.logger.Printf("Error unmarshaling postgres change event: %v", err)
		return
	}

	ch.mu.RLock()
	callbacks, exists := ch.callbacks["postgres_changes:"+changeEvent.Type]
	ch.mu.RUnlock()

	if !exists {
		return
	}

	for _, callback := range callbacks {
		if cb, ok := callback.(func(PostgresChangeEvent)); ok {
			cb(changeEvent)
		}
	}
}

// SetConn sets the WebSocket connection for testing purposes
func (c *RealtimeClient) SetConn(conn Conn) {
	c.conn = conn
}

// ProcessMessage processes a single message from the WebSocket connection
func (c *RealtimeClient) ProcessMessage(msg interface{}) {
	// Convert the message to a map
	msgMap, ok := msg.(map[string]interface{})
	if !ok {
		return
	}

	// Get the message type
	msgType, ok := msgMap["type"].(string)
	if !ok {
		return
	}

	// Get the topic
	topic, ok := msgMap["topic"].(string)
	if !ok {
		return
	}

	// Get the channel
	channel, ok := c.channels[topic]
	if !ok {
		return
	}

	// Process the message based on type
	switch msgType {
	case "broadcast":
		event, ok := msgMap["event"].(string)
		if !ok {
			return
		}
		payload, err := json.Marshal(msgMap["payload"])
		if err != nil {
			return
		}

		// Handle both broadcast and message callbacks
		channel.mu.RLock()
		callbacks, exists := channel.callbacks["broadcast:"+event]
		if exists {
			for _, callback := range callbacks {
				if cb, ok := callback.(func(json.RawMessage)); ok {
					cb(payload)
				}
			}
		}

		// Also handle as a general message
		messageCallbacks, exists := channel.callbacks["message"]
		if exists {
			message := Message{
				Type:    msgType,
				Topic:   topic,
				Event:   event,
				Payload: payload,
			}
			for _, callback := range messageCallbacks {
				if cb, ok := callback.(func(Message)); ok {
					cb(message)
				}
			}
		}
		channel.mu.RUnlock()

	case "presence":
		key, ok := msgMap["key"].(string)
		if !ok {
			return
		}
		channel.mu.RLock()
		callbacks, exists := channel.callbacks["presence"]
		channel.mu.RUnlock()
		if exists {
			for _, callback := range callbacks {
				if cb, ok := callback.(func(PresenceEvent)); ok {
					cb(PresenceEvent{Key: key})
				}
			}
		}
	case "postgres_changes":
		table, ok := msgMap["table"].(string)
		if !ok {
			return
		}
		schema, ok := msgMap["schema"].(string)
		if !ok {
			return
		}
		channel.mu.RLock()
		callbacks, exists := channel.callbacks["postgres_changes:INSERT"]
		channel.mu.RUnlock()
		if exists {
			for _, callback := range callbacks {
				if cb, ok := callback.(func(PostgresChangeEvent)); ok {
					cb(PostgresChangeEvent{
						Type:   "INSERT",
						Table:  table,
						Schema: schema,
					})
				}
			}
		}
	}
}
