package tests

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"nhooyr.io/websocket"
)

func TestMockConn(t *testing.T) {
	mockConn := &MockConn{}

	// Test SetReadLimit
	mockConn.SetReadLimit(100)

	// Test SetWriteLimit
	mockConn.SetWriteLimit(100)

	// Test Write and ClearWriteMessages
	testData := []byte("test message")
	err := mockConn.Write(context.Background(), websocket.MessageText, testData)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(mockConn.GetWriteMessages()))
	assert.Equal(t, "test message", mockConn.GetWriteMessages()[0])

	mockConn.ClearWriteMessages()
	assert.Equal(t, 0, len(mockConn.GetWriteMessages()))

	// Test Ping
	err = mockConn.Ping(context.Background())
	assert.NoError(t, err)

	// Test AddReadMessage and GetReadMessages
	readData := []byte("read test")
	mockConn.AddReadMessage(readData)

	readMessages := mockConn.GetReadMessages()
	assert.Equal(t, 1, len(readMessages))
	assert.Equal(t, readData, readMessages[0])

	// Test Read
	msgType, msg, err := mockConn.Read(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, websocket.MessageText, msgType)
	assert.Equal(t, readData, msg)

	// Verify the message was consumed
	assert.Equal(t, 0, len(mockConn.GetReadMessages()))

	// Test Close
	err = mockConn.Close(websocket.StatusNormalClosure, "test close")
	assert.NoError(t, err)
	assert.True(t, mockConn.IsClosed())
}

func TestMockConnCloseWithError(t *testing.T) {
	mockConn := &MockConn{}

	// Set an error
	testError := fmt.Errorf("test error")
	mockConn.CloseWithError(testError)

	// Verify the connection is closed
	assert.True(t, mockConn.IsClosed())

	// Verify that Read returns the error
	_, _, err := mockConn.Read(context.Background())
	assert.Error(t, err)
	assert.Equal(t, testError, err)
}
