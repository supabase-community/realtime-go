package realtime

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDisconnect(t *testing.T) {
	client, mockConn := testClient()
	err := client.Disconnect()
	assert.NoError(t, err)
	assert.True(t, mockConn.IsClosed())
}

func TestNextRef(t *testing.T) {
	client, _ := testClient()
	ref1 := client.(*RealtimeClient).NextRef()
	ref2 := client.(*RealtimeClient).NextRef()
	assert.Equal(t, ref1+1, ref2)
}

func TestHandleBroadcast(t *testing.T) {
	client, _ := testClient()
	channel := client.Channel("test-channel", &ChannelConfig{})

	broadcastReceived := false
	channel.OnBroadcast("test-event", func(payload json.RawMessage) {
		broadcastReceived = true
	})

	// Create a broadcast message
	msg := map[string]interface{}{
		"type":    "broadcast",
		"topic":   "test-channel",
		"event":   "test-event",
		"payload": map[string]string{"test": "data"},
	}

	// Process the message
	client.ProcessMessage(msg)

	// Wait for message to be processed
	time.Sleep(100 * time.Millisecond)
	assert.True(t, broadcastReceived)
}

func TestHandlePresence(t *testing.T) {
	client, _ := testClient()
	channel := client.Channel("test-channel", &ChannelConfig{})

	presenceReceived := false
	channel.OnPresence(func(event PresenceEvent) {
		presenceReceived = true
	})

	// Create a presence message
	msg := map[string]interface{}{
		"type":  "presence",
		"topic": "test-channel",
		"key":   "test-key",
	}

	// Process the message
	client.ProcessMessage(msg)

	// Wait for message to be processed
	time.Sleep(100 * time.Millisecond)
	assert.True(t, presenceReceived)
}

func TestHandlePostgresChanges(t *testing.T) {
	client, _ := testClient()
	channel := client.Channel("test-channel", &ChannelConfig{})

	changeReceived := false
	channel.OnPostgresChange("INSERT", func(event PostgresChangeEvent) {
		changeReceived = true
	})

	// Create a postgres changes message
	msg := map[string]interface{}{
		"type":   "postgres_changes",
		"topic":  "test-channel",
		"table":  "test_table",
		"schema": "public",
	}

	// Process the message
	client.ProcessMessage(msg)

	// Wait for message to be processed
	time.Sleep(100 * time.Millisecond)
	assert.True(t, changeReceived)
}

func TestProcessMessage(t *testing.T) {
	client, _ := testClient()

	// Create a message
	msg := map[string]interface{}{
		"type":    "message",
		"topic":   "test-channel",
		"event":   "test-event",
		"payload": map[string]string{"test": "data"},
	}

	// Process the message (this just tests that it doesn't panic)
	client.(*RealtimeClient).ProcessMessage(msg)
}

func TestSendHeartbeat(t *testing.T) {
	client, mockConn := testClient()

	// Send a heartbeat
	err := client.(*RealtimeClient).SendHeartbeat()
	assert.NoError(t, err)

	// Verify the heartbeat message was sent
	messages := mockConn.GetWriteMessages()
	assert.Len(t, messages, 1)
	assert.Contains(t, messages[0].(string), "heartbeat")
}

func TestNewConfig(t *testing.T) {
	config := NewConfig()
	assert.NotNil(t, config)
	assert.Equal(t, "wss://realtime.supabase.com", config.URL)
	assert.True(t, config.AutoReconnect)
	assert.Equal(t, 30*time.Second, config.HBInterval)
	assert.Equal(t, 5, config.MaxRetries)
	assert.Equal(t, time.Second, config.InitialBackoff)
	assert.Equal(t, 30*time.Second, config.Timeout)
}

func TestHandleBroadcastDirect(t *testing.T) {
	client, _ := testClient()
	realtimeClient := client.(*RealtimeClient)

	channel := client.Channel("test-channel", &ChannelConfig{})
	broadcastReceived := false

	// Register broadcast handler
	channel.OnBroadcast("test-event", func(payload json.RawMessage) {
		broadcastReceived = true
	})

	// Create message payload
	payload := json.RawMessage(`{"data":"test"}`)

	// Create and handle broadcast message directly
	msg := Message{
		Type:    "broadcast",
		Topic:   "test-channel",
		Event:   "test-event",
		Payload: payload,
	}

	realtimeClient.handleBroadcast(msg)
	assert.True(t, broadcastReceived)

	// Test with non-existing channel
	nonExistingMsg := Message{
		Type:    "broadcast",
		Topic:   "non-existing-channel",
		Event:   "test-event",
		Payload: payload,
	}
	// This should not crash
	realtimeClient.handleBroadcast(nonExistingMsg)

	// Test with non-existing event
	nonExistingEvent := Message{
		Type:    "broadcast",
		Topic:   "test-channel",
		Event:   "non-existing-event",
		Payload: payload,
	}
	// This should not crash
	realtimeClient.handleBroadcast(nonExistingEvent)
}

func TestHandlePresenceDirect(t *testing.T) {
	client, _ := testClient()
	realtimeClient := client.(*RealtimeClient)

	channel := client.Channel("test-channel", &ChannelConfig{})
	presenceReceived := false

	// Register presence handler
	channel.OnPresence(func(event PresenceEvent) {
		presenceReceived = true
		assert.Equal(t, "test-key", event.Key)
	})

	// Create presence payload
	presencePayload := `{"key":"test-key","type":"join"}`

	// Create and handle presence message directly
	msg := Message{
		Type:    "presence",
		Topic:   "test-channel",
		Event:   "presence",
		Payload: json.RawMessage(presencePayload),
	}

	realtimeClient.handlePresence(msg)
	assert.True(t, presenceReceived)

	// Test with non-existing channel
	nonExistingMsg := Message{
		Type:    "presence",
		Topic:   "non-existing-channel",
		Event:   "presence",
		Payload: json.RawMessage(presencePayload),
	}
	// This should not crash
	realtimeClient.handlePresence(nonExistingMsg)

	// Test with invalid payload
	invalidPayload := Message{
		Type:    "presence",
		Topic:   "test-channel",
		Event:   "presence",
		Payload: json.RawMessage(`{invalid-json}`),
	}
	// This should not crash
	realtimeClient.handlePresence(invalidPayload)
}

func TestHandlePostgresChangesDirect(t *testing.T) {
	client, _ := testClient()
	realtimeClient := client.(*RealtimeClient)

	channel := client.Channel("test-channel", &ChannelConfig{})
	changeReceived := false

	// Register postgres change handler
	channel.OnPostgresChange("INSERT", func(event PostgresChangeEvent) {
		changeReceived = true
		assert.Equal(t, "test_table", event.Table)
		assert.Equal(t, "public", event.Schema)
	})

	// Create postgres changes payload
	postgresPayload := `{
		"type": "INSERT",
		"table": "test_table",
		"schema": "public",
		"commit_timestamp": "2021-01-01T00:00:00Z",
		"errors": null
	}`

	// Create and handle postgres changes message directly
	msg := Message{
		Type:    "postgres_changes",
		Topic:   "test-channel",
		Event:   "postgres_changes",
		Payload: json.RawMessage(postgresPayload),
	}

	realtimeClient.handlePostgresChanges(msg)
	assert.True(t, changeReceived)

	// Test with non-existing channel
	nonExistingMsg := Message{
		Type:    "postgres_changes",
		Topic:   "non-existing-channel",
		Event:   "postgres_changes",
		Payload: json.RawMessage(postgresPayload),
	}
	// This should not crash
	realtimeClient.handlePostgresChanges(nonExistingMsg)

	// Test with invalid payload
	invalidPayload := Message{
		Type:    "postgres_changes",
		Topic:   "test-channel",
		Event:   "postgres_changes",
		Payload: json.RawMessage(`{invalid-json}`),
	}
	// This should not crash
	realtimeClient.handlePostgresChanges(invalidPayload)
}

func TestWebsocketConnWrapperSetWriteLimit(t *testing.T) {
	// Create a websocketConnWrapper
	wrapper := &websocketConnWrapper{
		// We don't need a real websocket.Conn for this test
		// as we're just testing the no-op method
		Conn: nil,
	}

	// Call SetWriteLimit
	// This should not panic
	wrapper.SetWriteLimit(100)
}

func TestReconnect(t *testing.T) {
	// Create a client with custom config for testing
	client, _ := testClient()
	realtimeClient := client.(*RealtimeClient)

	// Modify config for faster test
	realtimeClient.config.MaxRetries = 2
	realtimeClient.config.InitialBackoff = time.Millisecond
	realtimeClient.config.Timeout = time.Millisecond * 10

	// Setup channel that should be rejoined after reconnect
	channel := realtimeClient.Channel("test-channel", &ChannelConfig{}).(*channel)
	channel.joinedOnce = true // Simulate that the channel has been joined

	// Call reconnect
	realtimeClient.reconnect()

	// Test reentry prevention
	realtimeClient.reconnMu.Lock()
	realtimeClient.isReconnecting = true
	realtimeClient.reconnMu.Unlock()

	// This should return early and not panic
	realtimeClient.reconnect()
}

func TestStartHeartbeat(t *testing.T) {
	// Create a client with custom config for testing
	client, mockConn := testClient()
	realtimeClient := client.(*RealtimeClient)

	// Set very short heartbeat interval for testing
	realtimeClient.config.HBInterval = 10 * time.Millisecond

	// Start goroutine for heartbeat
	go realtimeClient.startHeartbeat()

	// Wait for at least one heartbeat to be sent
	time.Sleep(15 * time.Millisecond)

	// Stop the heartbeat
	close(realtimeClient.hbStop)

	// Allow for final cleanup
	time.Sleep(5 * time.Millisecond)

	// Verify heartbeat was sent
	messages := mockConn.GetWriteMessages()
	assert.GreaterOrEqual(t, len(messages), 1)

	foundHeartbeat := false
	for _, msg := range messages {
		msgStr, ok := msg.(string)
		if ok && strings.Contains(msgStr, "heartbeat") {
			foundHeartbeat = true
			break
		}
	}
	assert.True(t, foundHeartbeat, "Heartbeat message not found")
}

func TestHandleMessages(t *testing.T) {
	// Create a client with custom config for testing
	client, _ := testClient()
	realtimeClient := client.(*RealtimeClient)

	// Setup broadcast channel and handler
	broadcastCh := client.Channel("broadcast-channel", &ChannelConfig{})

	// Create flag to track received message
	broadcastReceived := false

	// Register handler
	broadcastCh.OnBroadcast("test-event", func(payload json.RawMessage) {
		broadcastReceived = true
	})

	// Create a message payload
	rawPayload := json.RawMessage(`{"data":"test"}`)

	// Create and handle message directly using handler functions
	broadcastMsg := Message{
		Type:    "broadcast",
		Topic:   "broadcast-channel",
		Event:   "test-event",
		Payload: rawPayload,
	}

	// Process the message directly
	realtimeClient.handleBroadcast(broadcastMsg)

	// Check result
	assert.True(t, broadcastReceived, "Broadcast message was not received")
}
