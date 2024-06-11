package realtime

import "fmt"

type RealtimeChannel struct {
	topic     string
	client    *RealtimeClient
	hasJoined bool
}

// Initialize a new channel
func CreateRealtimeChannel(client *RealtimeClient, topic string) *RealtimeChannel {
	return &RealtimeChannel{
		client: client,
		topic:  topic,
	}
}

// Perform callbacks on specific events. Successive calls to On()
// will result in multiple callbacks acting at the event
func (channel *RealtimeChannel) On(eventType string, filter map[string]string, callback func(any)) error {
   if !verifyEventType(eventType) {
      return fmt.Errorf("invalid event type: %s", eventType)
   }
   eventFilter, err := createEventFilter(eventType, filter)
   if err != nil {
      return fmt.Errorf("Invalid filter criteria for %s event type: %w", eventType, err)
   }

   msg := createConnectionMessage(channel.topic, eventFilter)
   newBinding := binding{
      msg: msg,
      callback: callback,
   }

   channel.client.addBinding(newBinding)

   return nil
}

// Subscribe to the channel and start listening to events
func (channel *RealtimeChannel) Subscribe() error {
   if err := channel.client.subscribe(); err != nil {
      return fmt.Errorf("Channel %s failed to subscribe: %w", channel.topic, err)
   }

	return nil
}

func (channel *RealtimeChannel) Unsubscribe() {
	if channel.client.isClientAlive() {

	}
}
