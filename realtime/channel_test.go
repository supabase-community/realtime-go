package realtime

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestChannelSubscribe(t *testing.T) {
	client, mockConn := testClient()
	channel := client.Channel("test-channel", &ChannelConfig{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Add a mock response message
	responseMsg, _ := json.Marshal(map[string]interface{}{
		"type":    "phx_reply",
		"topic":   "test-channel",
		"event":   "phx_join",
		"payload": map[string]interface{}{"status": "ok"},
	})
	mockConn.AddReadMessage(responseMsg)

	err := channel.Subscribe(ctx, func(state SubscribeState, err error) {
		assert.NoError(t, err)
		assert.Equal(t, SubscribeStateSubscribed, state)
	})

	assert.NoError(t, err)

	// Verify the subscribe message was sent
	messages := mockConn.GetWriteMessages()
	assert.Len(t, messages, 1)
	assert.Contains(t, messages[0].(string), "phx_join")
}

func TestChannelUnsubscribe(t *testing.T) {
	client, mockConn := testClient()

	// Print initial messages
	initialMessages := mockConn.GetWriteMessages()
	t.Logf("Initial messages: %d", len(initialMessages))

	channel := client.Channel("test-channel", &ChannelConfig{})

	// Print messages after channel creation
	afterChannelMessages := mockConn.GetWriteMessages()
	t.Logf("After channel creation messages: %d", len(afterChannelMessages))
	for i, msg := range afterChannelMessages {
		t.Logf("Message %d: %v", i, msg)
	}

	err := channel.Unsubscribe()
	assert.NoError(t, err)
	assert.Equal(t, ChannelStateLeaving, channel.GetState())

	// Print messages after unsubscribe
	afterUnsubMessages := mockConn.GetWriteMessages()
	t.Logf("After unsubscribe messages: %d", len(afterUnsubMessages))
	for i, msg := range afterUnsubMessages {
		t.Logf("Message %d: %v", i, msg)
	}
}

func TestChannelOnMessage(t *testing.T) {
	client, mockConn := testClient()
	channel := client.Channel("test-channel", &ChannelConfig{})

	messageReceived := false
	channel.OnMessage(func(msg Message) {
		messageReceived = true
		assert.Equal(t, "test-event", msg.Event)
	})

	// Add a mock message
	broadcastMsg, _ := json.Marshal(map[string]interface{}{
		"type":    "broadcast",
		"topic":   "test-channel",
		"event":   "test-event",
		"payload": map[string]interface{}{"test": "data"},
	})
	mockConn.AddReadMessage(broadcastMsg)

	// Process the message
	client.(*RealtimeClient).ProcessMessage(map[string]interface{}{
		"type":    "broadcast",
		"topic":   "test-channel",
		"event":   "test-event",
		"payload": map[string]interface{}{"test": "data"},
	})

	// Wait for message to be processed
	time.Sleep(100 * time.Millisecond)
	assert.True(t, messageReceived)
}

func TestChannelOnPresence(t *testing.T) {
	client, mockConn := testClient()
	channel := client.Channel("test-channel", &ChannelConfig{})

	presenceReceived := false
	channel.OnPresence(func(event PresenceEvent) {
		presenceReceived = true
		assert.Equal(t, "test-key", event.Key)
	})

	// Add a mock presence event
	presenceMsg, _ := json.Marshal(map[string]interface{}{
		"type":  "presence",
		"topic": "test-channel",
		"key":   "test-key",
	})
	mockConn.AddReadMessage(presenceMsg)

	// Process the message
	client.(*RealtimeClient).ProcessMessage(map[string]interface{}{
		"type":  "presence",
		"topic": "test-channel",
		"key":   "test-key",
	})

	// Wait for message to be processed
	time.Sleep(100 * time.Millisecond)
	assert.True(t, presenceReceived)
}

func TestChannelOnBroadcast(t *testing.T) {
	client, mockConn := testClient()
	channel := client.Channel("test-channel", &ChannelConfig{})

	broadcastReceived := false
	err := channel.OnBroadcast("test-event", func(payload json.RawMessage) {
		broadcastReceived = true
	})
	assert.NoError(t, err)

	// Simulate sending a broadcast
	err = channel.SendBroadcast("test-event", map[string]string{"test": "data"})
	assert.NoError(t, err)

	// Verify the broadcast message was sent
	messages := mockConn.GetWriteMessages()
	assert.Len(t, messages, 1)
	assert.Contains(t, messages[0].(string), "broadcast")
	assert.Contains(t, messages[0].(string), "test-event")

	// Add a mock broadcast response
	broadcastResponse, _ := json.Marshal(map[string]interface{}{
		"type":    "broadcast",
		"topic":   "test-channel",
		"event":   "test-event",
		"payload": map[string]string{"test": "data"},
	})
	mockConn.AddReadMessage(broadcastResponse)

	// Process the message
	client.(*RealtimeClient).ProcessMessage(map[string]interface{}{
		"type":    "broadcast",
		"topic":   "test-channel",
		"event":   "test-event",
		"payload": map[string]string{"test": "data"},
	})

	// Wait for message to be processed
	time.Sleep(100 * time.Millisecond)
	assert.True(t, broadcastReceived)
}

func TestChannelOnPostgresChange(t *testing.T) {
	client, mockConn := testClient()
	channel := client.Channel("test-channel", &ChannelConfig{})

	changeReceived := false
	err := channel.OnPostgresChange("INSERT", func(event PostgresChangeEvent) {
		changeReceived = true
		assert.Equal(t, "INSERT", event.Type)
		assert.Equal(t, "test_table", event.Table)
	})
	assert.NoError(t, err)

	// Add a mock postgres change event
	postgresMsg, _ := json.Marshal(map[string]interface{}{
		"type":   "postgres_changes",
		"topic":  "test-channel",
		"table":  "test_table",
		"schema": "public",
	})
	mockConn.AddReadMessage(postgresMsg)

	// Process the message
	client.(*RealtimeClient).ProcessMessage(map[string]interface{}{
		"type":   "postgres_changes",
		"topic":  "test-channel",
		"table":  "test_table",
		"schema": "public",
	})

	// Wait for message to be processed
	time.Sleep(100 * time.Millisecond)
	assert.True(t, changeReceived)
}

func TestChannelTrackUntrack(t *testing.T) {
	client, mockConn := testClient()
	channel := client.Channel("test-channel", &ChannelConfig{})

	// Test tracking
	err := channel.Track(map[string]interface{}{"user_id": 1})
	assert.NoError(t, err)

	// Verify the track message was sent
	messages := mockConn.GetWriteMessages()
	assert.Len(t, messages, 1)
	assert.Contains(t, messages[0].(string), "track")

	// Test untracking
	err = channel.Untrack()
	assert.NoError(t, err)

	// Verify the untrack message was sent
	messages = mockConn.GetWriteMessages()
	assert.Len(t, messages, 2)
	assert.Contains(t, messages[1].(string), "untrack")
}

func TestChannelGetTopic(t *testing.T) {
	client, _ := testClient()
	topic := "test-channel"
	channel := client.Channel(topic, &ChannelConfig{})

	assert.Equal(t, topic, channel.GetTopic())
}

func TestChannelRejoin(t *testing.T) {
	client, mockConn := testClient()
	ch := client.Channel("test-channel", &ChannelConfig{}).(*channel)

	// Initially joinedOnce is false, so rejoin should return nil
	err := ch.rejoin()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(mockConn.GetWriteMessages()))

	// Set joinedOnce to true but don't set state to error
	// Since channel is not subscribed yet, Subscribe will return error
	ch.joinedOnce = true

	// When rejoin is called in this state, it will try to subscribe
	// but we expect an error since we already set joinedOnce = true
	err = ch.rejoin()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "channel already subscribed")
}
