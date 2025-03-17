package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/supabase-community/realtime-go/realtime"
)

// testClient creates a RealtimeClient with a mock connection for testing
func testClient() (*realtime.RealtimeClient, *MockConn) {
	client := realtime.NewRealtimeClient("test-project", "test-key").(*realtime.RealtimeClient)
	mockConn := &MockConn{}
	client.SetConn(mockConn)
	return client, mockConn
}

func TestNewRealtimeClient(t *testing.T) {
	projectRef := "test-project"
	apiKey := "test-api-key"

	client := realtime.NewRealtimeClient(projectRef, apiKey)
	assert.NotNil(t, client)
}

func TestClientConnect(t *testing.T) {
	client := realtime.NewRealtimeClient("test-project", "test-api-key")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Connect(ctx)
	assert.Error(t, err) // Should fail because we're not connecting to a real server
}

func TestClientChannel(t *testing.T) {
	client, _ := testClient()
	channel := client.Channel("test-channel", &realtime.ChannelConfig{})
	assert.NotNil(t, channel)
	channels := client.GetChannels()
	assert.Equal(t, channel, channels["test-channel"])
}

func TestClientSetAuth(t *testing.T) {
	client, _ := testClient()
	err := client.SetAuth("test-token")
	assert.NoError(t, err)
}

func TestClientGetChannels(t *testing.T) {
	client, _ := testClient()
	channel := client.Channel("test-channel", &realtime.ChannelConfig{})
	channels := client.GetChannels()
	assert.Len(t, channels, 1)
	assert.Equal(t, channel, channels["test-channel"])
}

func TestClientRemoveChannel(t *testing.T) {
	client, _ := testClient()
	channel := client.Channel("test-channel", &realtime.ChannelConfig{})
	err := client.RemoveChannel(channel)
	assert.NoError(t, err)
	channels := client.GetChannels()
	assert.Len(t, channels, 0)
}

func TestClientRemoveAllChannels(t *testing.T) {
	client, _ := testClient()
	client.Channel("test-channel-1", &realtime.ChannelConfig{})
	client.Channel("test-channel-2", &realtime.ChannelConfig{})
	err := client.RemoveAllChannels()
	assert.NoError(t, err)
	channels := client.GetChannels()
	assert.Len(t, channels, 0)
}
