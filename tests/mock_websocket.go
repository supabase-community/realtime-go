package tests

import (
	"context"
	"encoding/json"
	"sync"

	"nhooyr.io/websocket"
)

// MockConn implements websocket.Conn for testing
type MockConn struct {
	closed        bool
	readMessages  []interface{}
	writeMessages []interface{}
	mu            sync.Mutex
	readLimit     int64
	writeLimit    int64
}

// Close implements websocket.Conn.Close
func (c *MockConn) Close(code websocket.StatusCode, reason string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}

// Read implements websocket.Conn.Read
func (c *MockConn) Read(ctx context.Context) (websocket.MessageType, []byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.readMessages) == 0 {
		return 0, nil, websocket.CloseError{
			Code:   websocket.StatusNormalClosure,
			Reason: "no more messages",
		}
	}

	msg := c.readMessages[0]
	c.readMessages = c.readMessages[1:]

	data, err := json.Marshal(msg)
	if err != nil {
		return 0, nil, err
	}

	return websocket.MessageText, data, nil
}

// Write implements websocket.Conn.Write
func (c *MockConn) Write(ctx context.Context, messageType websocket.MessageType, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var msg interface{}
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}

	c.writeMessages = append(c.writeMessages, msg)
	return nil
}

// Ping implements websocket.Conn.Ping
func (c *MockConn) Ping(ctx context.Context) error {
	return nil
}

// SetReadLimit implements websocket.Conn.SetReadLimit
func (c *MockConn) SetReadLimit(limit int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.readLimit = limit
}

// SetWriteLimit implements websocket.Conn.SetWriteLimit
func (c *MockConn) SetWriteLimit(limit int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.writeLimit = limit
}

// AddReadMessage adds a message to be read by the connection
func (c *MockConn) AddReadMessage(msg interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.readMessages = append(c.readMessages, msg)
}

// GetWriteMessages returns all messages that have been written
func (c *MockConn) GetWriteMessages() []interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.writeMessages
}

// ClearWriteMessages clears all written messages
func (c *MockConn) ClearWriteMessages() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.writeMessages = []interface{}{}
}

// GetReadMessages returns all messages that have been added for reading
func (c *MockConn) GetReadMessages() []interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.readMessages
}

// IsClosed returns whether the connection is closed
func (c *MockConn) IsClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}
