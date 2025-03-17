package tests

import (
	"context"
	"fmt"
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
	readError     error
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

	// If there's an error set, return it
	if c.readError != nil {
		return 0, nil, c.readError
	}

	if len(c.readMessages) == 0 {
		return 0, nil, fmt.Errorf("no messages to read")
	}

	msg := c.readMessages[0]
	c.readMessages = c.readMessages[1:]

	// Return the message data directly
	return websocket.MessageText, msg.([]byte), nil
}

// Write implements websocket.Conn.Write
func (c *MockConn) Write(ctx context.Context, messageType websocket.MessageType, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Store the actual string
	c.writeMessages = append(c.writeMessages, string(data))
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
func (c *MockConn) AddReadMessage(msg []byte) {
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

// CloseWithError closes the connection with an error, used for testing error conditions
func (m *MockConn) CloseWithError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readError = err
	m.closed = true
}
