package realtime

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"time"
)

type RealtimeChannel struct {
	topic          string
	client         *RealtimeClient
	hasSubscribed  bool

   rwMu                    sync.RWMutex
   numBindings             int
   bindings                map[string][]*binding
   postgresBindingRoute    map[int]*binding
}

// Bind an event with the user's callback function
type binding struct {
   eventType   string
   filter      eventFilter
   callback    func(any)
}

// Initialize a new channel
func CreateRealtimeChannel(client *RealtimeClient, topic string) *RealtimeChannel {
	return &RealtimeChannel{
		client: client,
		topic:  topic,
      numBindings: 0,
      bindings: make(map[string][]*binding),
      postgresBindingRoute: make(map[int]*binding),
      hasSubscribed: false,
	}
}

// Perform callbacks on specific events. Successive calls to On()
// will result in multiple callbacks acting at the event
func (channel *RealtimeChannel) On(eventType string, filter map[string]string, callback func(any)) error {
   if !verifyEventType(eventType) {
      return fmt.Errorf("Invalid event type: %s", eventType)
   }

   eventFilter, err := createEventFilter(eventType, filter)
   if err != nil {
      return fmt.Errorf("Invalid filter criteria for %s event type: %w", eventType, err)
   }

   newBinding := &binding{
      eventType: eventType,
      filter: eventFilter,
      callback: callback,
   }

   channel.numBindings += 1
   channel.bindings[eventType] = append(channel.bindings[eventType], newBinding)

   return nil
}

// Subscribe to the channel and start listening to events
func (channel *RealtimeChannel) Subscribe(ctx context.Context) error {
   if channel.hasSubscribed {
      return fmt.Errorf("Error: Channel %s can only be subscribed once", channel.topic)
   }

   // Do nothing if there are no bindings
   if channel.numBindings == 0 {
      return nil
   }

   // Flatten all type of bindings into one slice
   allBindings := make([]*binding, channel.numBindings)
   for _, eventType := range []string{postgresChangesEventType, broadcastEventType, presenceEventType} {
      copy(allBindings, channel.bindings[eventType])
   }

   respPayload, err := channel.client.subscribe(channel.topic, allBindings, ctx)
   if err != nil {
      return fmt.Errorf("Channel %s failed to subscribe: %v", channel.topic, err)
   }

   // Verify and map postgres events. If there are any mismatch, channel will
   // rollback, and unsubscribe to the events.
   changes := respPayload.Response.PostgresChanges
   postgresBindings := channel.bindings[postgresChangesEventType]
   if len(postgresBindings) != len(changes) {
      channel.Unsubscribe(ctx)
      return fmt.Errorf("Server returns the wrong number of subscribed events: %v events", len(changes))
   }

   for i, change := range changes {
      bindingFilter, ok := postgresBindings[i].filter.(postgresFilter)
      if !ok {
         panic("TYPE ASSERTION FAILED: expecting type postgresFilter")
      }
      if change.Schema != bindingFilter.Schema || change.Event  != bindingFilter.Event ||
         change.Table  != bindingFilter.Table  || change.Filter != bindingFilter.Filter {
         channel.Unsubscribe(ctx)
         return fmt.Errorf("Configuration mismatch between server's event and channel's event")
      }
      channel.postgresBindingRoute[change.ID] = postgresBindings[i]
   }

   channel.hasSubscribed = true

	return nil
}

// Unsubscribe from the channel and stop listening to events
func (channel *RealtimeChannel) Unsubscribe(ctx context.Context) {
	if !channel.hasSubscribed {
      return
	}

   // Refresh all the binding routes
   channel.rwMu.Lock()
   clear(channel.postgresBindingRoute)
   channel.rwMu.Unlock()

   channel.client.unsubscribe(channel.topic, ctx)
   channel.hasSubscribed = false
}

// Send a custom event to the server. Payload must be either:
func (channel *RealtimeChannel) Send(event CustomEvent, ctx context.Context) error {
   if !verifyEventType(event.Type) {
      return fmt.Errorf("Invalid event type: %s", event.Type)
   }

   // Verify that payload is a struct
   if reflect.TypeOf(event.Payload).Kind() != reflect.Struct {
      return fmt.Errorf("Payload must be of kind Struct. Invalid payload: %+v", event.Payload)
   }
   
   metadata := createMsgMetadata(event.Event, channel.topic)
   msg := &Msg{
      Metadata: *metadata,
      Payload: event.Payload,
   }
   msg.Metadata.Ref = strconv.FormatInt(time.Now().Unix(), 10)
   return channel.client.send(msg, channel.hasSubscribed, ctx)
}

// Route the id of triggered event to appropriate callback
func (channel *RealtimeChannel) routePostgresEvent(id int, payload *PostgresCDCPayload) {
   channel.rwMu.RLock()
   binding, ok := channel.postgresBindingRoute[id] 
   channel.rwMu.RUnlock()

   if !ok {
      channel.client.logger.Printf("Error: Unrecognized id %v", id)
      return
   }
   
   bindFilter, ok := binding.filter.(postgresFilter)
   if !ok {
      panic("TYPE ASSERTION FAILED: expecting type postgresFilter")
   }

   // Match * | INSERT | UPDATE | DELETE
   switch bindFilter.Event {
      case "*":
         fallthrough
      case payload.Data.ActionType:
         go binding.callback(payload)
         break
      default:
         return 
   }
}

// Route the broadcast event to the right callback
func (channel *RealtimeChannel) routeBroadcastEvent(payload *BroadcastPayload) {
   hasFound := false
   // TODO: Instead of go through them one by one, use a hashmap to store the 
   // broadcast bindings instead
   for _, bind := range channel.bindings[broadcastEventType] {
      filter, ok := bind.filter.(broadcastFilter)
      if !ok {
         panic("TYPE ASSERTION FAILED: expecting type broadcastFilter")
      }
      if filter.Event == payload.Event {
         bind.callback(payload)
         hasFound = true
      }
   }

   if !hasFound {
      channel.client.logger.Printf("Error: %+v", payload)
      channel.client.logger.Printf("Error: Failed to find the broadcast event %v", payload.Event)
   }
}
