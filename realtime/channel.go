package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"nhooyr.io/websocket"
)

type channel struct {
	topic             string
	state             ChannelState
	config            *ChannelConfig
	client            *RealtimeClient
	broadcastHandlers map[string]func(json.RawMessage)
	presenceHandler   func(PresenceEvent)
	postgresHandlers  map[string]func(PostgresChangeEvent)
	callbacks         map[string][]interface{}
	mu                sync.RWMutex
	joinedOnce        bool
}

func newChannel(topic string, config *ChannelConfig, client *RealtimeClient) *channel {
	return &channel{
		topic:             topic,
		state:             ChannelStateClosed,
		config:            config,
		client:            client,
		broadcastHandlers: make(map[string]func(json.RawMessage)),
		postgresHandlers:  make(map[string]func(PostgresChangeEvent)),
		callbacks:         make(map[string][]interface{}),
	}
}

func (ch *channel) Subscribe(ctx context.Context, callback func(SubscribeState, error)) error {
	if ch.joinedOnce {
		return fmt.Errorf("channel already subscribed")
	}

	ch.mu.Lock()
	ch.state = ChannelStateJoining
	ch.mu.Unlock()

	subscribeMsg := struct {
		Type    string      `json:"type"`
		Topic   string      `json:"topic"`
		Event   string      `json:"event"`
		Payload interface{} `json:"payload"`
	}{
		Type:    "subscribe",
		Topic:   ch.topic,
		Event:   "phx_join",
		Payload: ch.config,
	}

	data, err := json.Marshal(subscribeMsg)
	if err != nil {
		return err
	}

	if err := ch.client.conn.Write(ctx, websocket.MessageText, data); err != nil {
		ch.mu.Lock()
		ch.state = ChannelStateErrored
		ch.mu.Unlock()
		if callback != nil {
			callback(SubscribeStateChannelError, err)
		}
		return err
	}

	ch.joinedOnce = true
	ch.mu.Lock()
	ch.state = ChannelStateJoined
	ch.mu.Unlock()

	if callback != nil {
		callback(SubscribeStateSubscribed, nil)
	}

	return nil
}

func (ch *channel) Unsubscribe() error {
	ch.mu.Lock()
	ch.state = ChannelStateLeaving
	ch.mu.Unlock()

	unsubscribeMsg := struct {
		Type  string `json:"type"`
		Topic string `json:"topic"`
		Event string `json:"event"`
	}{
		Type:  "unsubscribe",
		Topic: ch.topic,
		Event: "phx_leave",
	}

	data, err := json.Marshal(unsubscribeMsg)
	if err != nil {
		return err
	}
	return ch.client.conn.Write(context.Background(), websocket.MessageText, data)
}

func (ch *channel) OnMessage(callback func(Message)) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	ch.callbacks["message"] = append(ch.callbacks["message"], callback)
}

func (ch *channel) OnPresence(callback func(PresenceEvent)) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	ch.callbacks["presence"] = append(ch.callbacks["presence"], callback)
}

func (ch *channel) OnBroadcast(event string, callback func(json.RawMessage)) error {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	ch.callbacks[fmt.Sprintf("broadcast:%s", event)] = append(ch.callbacks[fmt.Sprintf("broadcast:%s", event)], callback)
	return nil
}

func (ch *channel) SendBroadcast(event string, payload interface{}) error {
	broadcastMsg := struct {
		Type    string      `json:"type"`
		Topic   string      `json:"topic"`
		Event   string      `json:"event"`
		Payload interface{} `json:"payload"`
		Ref     int         `json:"ref"`
	}{
		Type:    "broadcast",
		Topic:   ch.topic,
		Event:   event,
		Payload: payload,
		Ref:     ch.client.nextRef(),
	}

	data, err := json.Marshal(broadcastMsg)
	if err != nil {
		return err
	}
	return ch.client.conn.Write(context.Background(), websocket.MessageText, data)
}

func (ch *channel) OnPostgresChange(event string, callback func(PostgresChangeEvent)) error {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	ch.callbacks[fmt.Sprintf("postgres_changes:%s", event)] = append(ch.callbacks[fmt.Sprintf("postgres_changes:%s", event)], callback)
	return nil
}

func (ch *channel) Track(payload interface{}) error {
	trackMsg := struct {
		Type    string      `json:"type"`
		Topic   string      `json:"topic"`
		Event   string      `json:"event"`
		Payload interface{} `json:"payload"`
	}{
		Type:    "track",
		Topic:   ch.topic,
		Event:   "track",
		Payload: payload,
	}

	data, err := json.Marshal(trackMsg)
	if err != nil {
		return err
	}
	return ch.client.conn.Write(context.Background(), websocket.MessageText, data)
}

func (ch *channel) Untrack() error {
	untrackMsg := struct {
		Type  string `json:"type"`
		Topic string `json:"topic"`
		Event string `json:"event"`
	}{
		Type:  "untrack",
		Topic: ch.topic,
		Event: "untrack",
	}

	data, err := json.Marshal(untrackMsg)
	if err != nil {
		return err
	}
	return ch.client.conn.Write(context.Background(), websocket.MessageText, data)
}

func (ch *channel) GetState() ChannelState {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	return ch.state
}

func (ch *channel) updateAuth(token string) {
	// Implementation for updating auth token
}

func (ch *channel) rejoin() error {
	if !ch.joinedOnce {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), ch.client.config.Timeout)
	defer cancel()

	return ch.Subscribe(ctx, nil)
}

// GetTopic returns the channel's topic
func (c *channel) GetTopic() string {
	return c.topic
}
