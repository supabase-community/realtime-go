package realtime

import (
	"fmt"
	"reflect"
	"strings"
)

// Events that are used to communicate with the server
const (
	joinEvent  string = "phx_join"
	replyEvent string = "phx_reply"
	leaveEvent string = "phx_leave"
	closeEvent string = "phx_close"

	// DB Subscription Events
	postgresChangesEvent string = "postgres_changes"

	// Broadcast Events
	broadcastEvent string = "broadcast"

	// Presence Events
	presenceStateEvent string = "presence_state"
	presenceDiffEvent  string = "presence_diff"

	// Other Events
	systemEvent       string = "system"
	heartbeatEvent    string = "heartbeat"
	accessTokennEvent string = "access_token"
)

// Event "type" that the user can specify for channel to listen to
const (
	presenceEventType        string = "presence"
	broadcastEventType       string = "broadcast"
	postgresChangesEventType string = "postgres_changes"
)

// type eventFilter struct {}
type eventFilter interface {}

type postgresFilter struct {
   Event  string `supabase:"required" json:"event"`
	Schema string `supabase:"required" json:"schema"`
	Table  string `supabase:"optional" json:"table,omitempty"`
	Filter string `supabase:"optional" json:"filter,omitempty"`
}

type broadcastFilter struct {
	Event string `supabase:"required"`
}

type presenceFilter struct {
	Event string `supabase:"required"`
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
// information on the struct of each event.
// Only the following events are currently supported:
//   - postgres_changes, broadcast, presence
func createEventFilter(eventType string, filter map[string]string) (eventFilter, error) {
	var filterType reflect.Type      // Type for filter
	var filterConValue reflect.Value // Concrete value
	var filterPtrValue reflect.Value // Pointer value to the concrete value
	var missingFields []string

	switch eventType {
	case postgresChangesEvent:
		filterPtrValue = reflect.ValueOf(&postgresFilter{})
		break
	case broadcastEvent:
		filterPtrValue = reflect.ValueOf(&broadcastFilter{})
		break
	case presenceEventType:
		filterPtrValue = reflect.ValueOf(&presenceFilter{})
	default:
		return nil, fmt.Errorf("Unsupported event type: %s", eventType)
	}

	// Get the underlying filter type to identify missing fields
	filterConValue = filterPtrValue.Elem()
	filterType = filterConValue.Type()
	missingFields = make([]string, 0, filterType.NumField())

	for i := 0; i < filterType.NumField(); i++ {
		currField := filterType.Field(i)
		currFieldName := strings.ToLower(currField.Name)
		isRequired := currField.Tag.Get("supabase") == "required"

		val, ok := filter[currFieldName]
		if !ok && isRequired {
			missingFields = append(missingFields, currFieldName)
		}

		// Set field to empty string when value for currFieldName is missing
		filterConValue.Field(i).SetString(val)
	}

	if len(missingFields) != 0 {
		return nil, fmt.Errorf("Criteria for %s is missing: %+v", eventType, missingFields)
	}

	filterFinal, ok := filterConValue.Interface().(eventFilter)
	if !ok {
		return nil, fmt.Errorf("Unexpected Error: cannot create event filter")
	}

	return filterFinal, nil
}
