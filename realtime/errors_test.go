package realtime

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNotConnectedError(t *testing.T) {
	err := &NotConnectedError{FuncName: "TestFunc"}
	assert.Equal(t, "A WS connection has not been established. Ensure you call RealtimeClient.Connect() before calling RealtimeClient.TestFunc()", err.Error())
}

func TestChannelError(t *testing.T) {
	// Test with inner error
	innerErr := errors.New("inner error")
	err := &ChannelError{Message: "test message", Err: innerErr}
	assert.Equal(t, "Channel error: test message: inner error", err.Error())

	// Test without inner error
	err = &ChannelError{Message: "test message"}
	assert.Equal(t, "Channel error: test message", err.Error())
}

func TestSubscriptionError(t *testing.T) {
	// Test with inner error
	innerErr := errors.New("inner error")
	err := &SubscriptionError{State: SubscribeStateChannelError, Err: innerErr}
	assert.Equal(t, "Subscription error (state: 1): inner error", err.Error())

	// Test without inner error
	err = &SubscriptionError{State: SubscribeStateChannelError}
	assert.Equal(t, "Subscription error (state: 1)", err.Error())
}

func TestAuthenticationError(t *testing.T) {
	err := &AuthenticationError{Message: "test message"}
	assert.Equal(t, "Authentication error: test message", err.Error())
}

func TestReconnectionError(t *testing.T) {
	// Test with inner error
	innerErr := errors.New("inner error")
	err := &ReconnectionError{Attempts: 3, Err: innerErr}
	assert.Equal(t, "Failed to reconnect after 3 attempts: inner error", err.Error())

	// Test without inner error
	err = &ReconnectionError{Attempts: 3}
	assert.Equal(t, "Failed to reconnect after 3 attempts", err.Error())
}
