package realtime

import (
	"context"
	"fmt"
	"strings"
	"sync"
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
   eventType = strings.ToLower(eventType)
   if !verifyEventType(eventType) {
      return fmt.Errorf("invalid event type: %s", eventType)
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
   startIdx    := 0
   for _, eventType := range []string{postgresChangesEventType, broadcastEventType, presenceEventType} {
      if startIdx >= channel.numBindings {
         break
      }

      copy(allBindings[startIdx:], channel.bindings[eventType])
      startIdx += len(channel.bindings[eventType])
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
      if strings.ToLower(change.Schema) != strings.ToLower(bindingFilter.Schema) || 
         strings.ToUpper(change.Event)  != strings.ToUpper(bindingFilter.Event)  ||
         strings.ToLower(change.Table)  != strings.ToLower(bindingFilter.Table)  || 
         strings.ToLower(change.Filter) != strings.ToLower(bindingFilter.Filter) {
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
   switch strings.ToUpper(bindFilter.Event) {
      case "*":
         fallthrough
      case payload.Data.ActionType:
         go binding.callback(payload)
         break
      default:
         return 
   }
}
