package realtime

import (
	"encoding/json"
	"strconv"
	"time"
)

// This is a general message strucutre. It follows the message protocol
// of the phoenix server:
/*
   {
      event: string,
      topic: string,
      payload: [key: string]: boolean | int | string | any,
      ref: string
   }
*/
type Msg struct {
   Metadata
   Payload any `json:"payload"`
}

// Generic message that contains raw payload. It can be used
// as a tagged union, where the event field can be used to
// determine the structure of the payload.
type RawMsg struct {
	Metadata
	Payload json.RawMessage `json:"payload"`
}

// The other fields besides the payload that make up a message.
// It describes other information about a message such as type of event,
// the topic the message belongs to, and its reference.
type Metadata struct {
	Event string `json:"event"`
	Topic string `json:"topic"`
	Ref   string `json:"ref"`
}

// Payload for the conection message for when client first joins the channel. 
// More info: https://supabase.com/docs/guides/realtime/protocol#connection
type ConnectionPayload struct {
   Config struct {
      Broadcast struct {
         Self bool `json:"self"`
      } `json:"broadcast,omitempty"`

      Presence struct {
         Key string `json:"key"`
      } `json:"presence,omitempty"`

      PostgresChanges []postgresFilter `json:"postgres_changes,omitempty"`
   } `json:"config"`
}

// Payload of the server's first response of three upon joining channel. 
// It contains details about subscribed postgres events.
// More info: https://supabase.com/docs/guides/realtime/protocol#connection
type ReplyPayload struct {
   Response struct {
      PostgresChanges []struct{
         ID int `json:"id"`
         postgresFilter
      } `json:"postgres_changes"`
   } `json:"response"`
   Status string `json:"status"`
}

// Payload of the server's second response of three upon joining channel. 
// It contains details about the status of subscribing to PostgresSQL.
// More info: https://supabase.com/docs/guides/realtime/protocol#system-messages
type SystemPayload struct {
   Channel     string `json:"channel"`
   Extension   string `json:"extension"`
   Message     string `json:"message"`
   Status      string `json:"status"`
}

// Payload of the server's third response of three upon joining channel. 
// It contains details about the Presence feature of Supabase.
// More info: https://supabase.com/docs/guides/realtime/protocol#state-update
type PresenceStatePayload map[string]struct{
   Metas []struct{
      Ref   string   `json:"phx_ref"` 
      Name  string   `json:"name"` 
      T     float64  `json:"t"` 
   } `json:"metas,omitempty"`
}

// Payload of the server's response when there is a postgres_changes event. 
// More info: https://supabase.com/docs/guides/realtime/protocol#system-messages
type PostgresCDCPayload struct {
   Data struct {
      Schema     string          `json:"schema"`
      Table      string          `json:"table"`
      CommitTime string          `json:"commit_timestamp"`
      Record     map[string]any  `json:"record"`
      Columns    []struct{
         Name string `json:"name"`
         Type string `json:"type"`
      } `json:"columns"`
      ActionType  string          `json:"type"`
      Old        map[string]any  `json:"old_record"`
      Errors     string          `json:"errors"`
   } `json:"data"`
   IDs []int `json:"ids"`
}

// Payload of the server's response when there is a broadcast event. 
// More info: https://supabase.com/docs/guides/realtime/protocol#broadcast-message
type BroadcastPayload struct {
   Event    string `json:"event"`
   Payload  any    `json:"payload"`
   Type     string `json:"type"`
}

// create a template message
func createMsgMetadata(event string, topic string) *Metadata {
	return &Metadata{
		Event: event,
		Topic: topic,
		Ref:   "",
	}
}

// create a connection message depending on event type
func createConnectionMessage(topic string, bindings []*binding) *Msg {
	msg := &Msg{}

   // Fill out the message template
   msg.Metadata = *createMsgMetadata(joinEvent, topic)
   msg.Metadata.Ref = strconv.FormatInt(time.Now().Unix(), 10)

   // Fill out the payload
   payload := &ConnectionPayload{}
   for _, bind := range bindings {
      filter := bind.filter
      switch filter.(type) {
         case postgresFilter:
            if payload.Config.PostgresChanges == nil {
               payload.Config.PostgresChanges = make([]postgresFilter, 0, 1)
            }
            payload.Config.PostgresChanges = append(payload.Config.PostgresChanges, filter.(postgresFilter))
            break
         case broadcastFilter:
            payload.Config.Broadcast.Self = true
            break
         case presenceFilter:
            payload.Config.Presence.Key = ""
            break
         default:
            panic("TYPE ASSERTION FAILED: expecting one of postgresFilter, broadcastFilter, or presenceFilter")
      }
   }

   msg.Payload = payload

	return msg
}
