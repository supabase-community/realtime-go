package realtime

import (
	"fmt"
	"reflect"
)

// Events that are used to communicate with the server
const (
   joinEvent string = "phx_join"
   replyEvent string = "phx_reply"

   // DB Subscription Events
   postgresChangesEvent string = "postgres_changes"

   // Broadcast Events
   broadcastEvent string = "broadcast"

// Other Events
const SYS_EVENT = "system"
const HEARTBEAT_EVENT = "heartbeat"
const ACCESS_TOKEN_EVENT = "access_token"
