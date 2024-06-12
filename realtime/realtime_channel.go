package realtime

import (
	"container/list"
	"context"
	"fmt"
)

type RealtimeChannel struct {
	topic     string
	client    *RealtimeClient

   bindings       *list.List 
   bindingsMap    map[int]binding
	hasSubscribed  bool
}

// Initialize a new channel
func CreateRealtimeChannel(client *RealtimeClient, topic string) *RealtimeChannel {
	return &RealtimeChannel{
		client: client,
		topic:  topic,
      bindings: list.New(),
      bindingsMap: make(map[int]binding),
      hasSubscribed: false,
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

   newBinding := binding{
      eventType: eventType,
      filter: eventFilter,
      callback: callback,
   }
   channel.bindings.PushBack(newBinding)

   return nil
}

// Subscribe to the channel and start listening to events
func (channel *RealtimeChannel) Subscribe(ctx context.Context) error {
   if channel.hasSubscribed {
      return fmt.Errorf("Error: Channel %s can only be subscribed once", channel.topic)
   }

   // Do nothing if there are no bindings
   if channel.bindings.Len() == 0 {
      return nil
   }

   ids, err := channel.client.subscribe(channel.topic, channel.bindings, ctx)
   if err != nil {
      return fmt.Errorf("Channel %s failed to subscribe: %v", channel.topic, err)
   }
   
   bindNode := channel.bindings.Front()
   for _, id := range ids {
      if bindNode == nil {
         channel.Unsubscribe()
         return fmt.Errorf("Error: the number subscribed events are not equal to the total events")
      }

      binding, ok := bindNode.Value.(binding)
      if !ok {
         panic("TYPE ASSERTION FAILED: expecting type binding")
      }

      if binding.eventType == postgresChangesEvent {
         channel.bindingsMap[id] = binding;
      }

      bindNode = bindNode.Next()
   }

   channel.hasSubscribed = true

	return nil
}

func (channel *RealtimeChannel) Unsubscribe() {
	if channel.client.isClientAlive() {

	}
   channel.hasSubscribed = false
}

// Route the id of triggered event to appropriate callback
func (channel *RealtimeChannel) routePostgresEvent(id int, payload *PostgresCDCPayload) {
   binding, ok := channel.bindingsMap[id] 
   if !ok {
      channel.client.logger.Printf("Error: Unrecognized id %v", id)
   }

   go binding.callback(payload)
}
