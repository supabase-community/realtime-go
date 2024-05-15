package realtime

import (
	"fmt"
	"reflect"
	"strings"
)

// Events that are used to communicate with the server
const (
   joinEvent string = "phx_join"
   replyEvent string = "phx_reply"

   // DB Subscription Events
   postgresChangesEvent string = "postgres_changes"

   // Broadcast Events
   broadcastEvent string = "broadcast"

   // Presence Events
   presenceStateEvent string = "presence_state"
   presenceDiffEvent string ="presence_diff"

   // Other Events
   systemEvent string = "system"
   heartbeatEvent string = "heartbeat"
   accessTokennEvent string = "access_token"
)

// Event "type" that the user can specify for channel to listen to
const (
   presenceEventType        string = "presence"
   broadcastEventType       string = "broadcast"
   postgresChangesEventType string = "postgres_changes"
)

type postgresFilter struct {
   Event    string   `supabase:"required"`
   Schema   string   `supabase:"required"`
   Table    string   `supabase:"optional"`
   Filter   string   `supabase:"optional"`
}

type broadcastFilter struct {
   event string   `supabase:"required"`
}

type presenceFilter struct {
   event string   `supabase:"required"`
}

// Verify if the given event type is supported
func verifyEventType(eventType string) bool {
   switch eventType {
      case presenceEventType:
      case broadcastEventType:
      case postgresChangesEventType:
         return true
   }

   return false
}


// Enforce client's filter object to follow a specific message
// structure of certain events. Check messages.go for more
// information on the struct of each event. By default,
// non-supported events will return an error
// Only the following events are currently supported:
//    + postgres_changes, broadcast, presence
func verifyFilter(eventType string, filter map[string]string) error {
   var filterType reflect.Type
   var missingFields []string

   switch eventType {
      case postgresChangesEvent:
         filterType = reflect.TypeOf(postgresFilter{})
         break
      case broadcastEvent:
         filterType = reflect.TypeOf(broadcastFilter{})
         break
      case presenceEventType:
         filterType = reflect.TypeOf(presenceFilter{})
      default:
         return fmt.Errorf("Unsupported event type: %s", eventType)
   }

   missingFields = make([]string, 0, filterType.NumField())
   for i := 0; i < filterType.NumField(); i++ {
      currField      := filterType.Field(i)
      currFieldName  := strings.ToLower(currField.Name)
      isRequired     := currField.Tag.Get("supabase") == "required"

      if _, ok := filter[currFieldName]; !ok && isRequired {
         missingFields = append(missingFields, currFieldName)
      }
   }

   if len(missingFields) != 0 {
      return fmt.Errorf("Criteria for %s is missing: %+v", eventType, missingFields)
   }

   return nil
}
