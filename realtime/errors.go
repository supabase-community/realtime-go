package realtime

import "fmt"

// NotConnectedError is returned when operations requiring a connection are executed when socket is not connected
type NotConnectedError struct {
	FuncName string
}

func (e *NotConnectedError) Error() string {
	return fmt.Sprintf("A WS connection has not been established. Ensure you call RealtimeClient.Connect() before calling RealtimeClient.%s()", e.FuncName)
}

// ChannelError is returned when there is an error with a channel operation
type ChannelError struct {
	Message string
	Err     error
}

func (e *ChannelError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("Channel error: %s: %v", e.Message, e.Err)
	}
	return fmt.Sprintf("Channel error: %s", e.Message)
}

// SubscriptionError is returned when there is an error with a subscription
type SubscriptionError struct {
	State SubscribeState
	Err   error
}

func (e *SubscriptionError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("Subscription error (state: %d): %v", e.State, e.Err)
	}
	return fmt.Sprintf("Subscription error (state: %d)", e.State)
}

// AuthenticationError is returned when there is an error with authentication
type AuthenticationError struct {
	Message string
}

func (e *AuthenticationError) Error() string {
	return fmt.Sprintf("Authentication error: %s", e.Message)
}

// ReconnectionError is returned when there is an error during reconnection
type ReconnectionError struct {
	Attempts int
	Err      error
}

func (e *ReconnectionError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("Failed to reconnect after %d attempts: %v", e.Attempts, e.Err)
	}
	return fmt.Sprintf("Failed to reconnect after %d attempts", e.Attempts)
}
