package realtime;

type TemplateMsg struct {
   Event string `json:"event"`
   Topic string `json:"topic"`
   Ref string `json:"ref"`
}

type ConnectionMsg struct {
   *TemplateMsg

   Payload struct {
      Config struct {
         Broadcast struct {
            Self bool `json:"self"`
         } `json:"broadcast"`

         Presence struct {
            Key string `json:"key"`
         } `json:"presence"`

         PostgresChanges []postgresFilter `json:"postgres_changes"`
      } `json:"config"`
   } `json:"payload"`
}

type PostgresCDCMsg struct {
   *TemplateMsg

   Payload struct {
      Data struct {
         Schema string `json:"schema"`
         Table string `json:"table"`
         CommitTime string `json:"commit_timestamp"`
         EventType string  `json:"eventType"`
         New map[string]string `json:"new"`
         Old map[string]string `json:"old"`
         Errors string `json:"errors"`
      } `json:"data"`
   } `json:"payload"`
}

type HearbeatMsg struct {
   *TemplateMsg

   Payload struct {
   } `json:"payload"`
}

// create a template message
func createTemplateMessage(event string, topic realtimeTopic) *TemplateMsg {
   return &TemplateMsg{
      Event: event,
      Topic: string(topic),
      Ref: "",
   }
}

// create a connection message depending on event type
func createConnectionMessage(topic realtimeTopic, filter eventFilter) *ConnectionMsg {
   msg := &ConnectionMsg{}

   // Common part across the three event type
   msg.TemplateMsg = createTemplateMessage(joinEvent, topic)
   switch filter.(type)  {
   case postgresFilter:
      msg.Payload.Config.PostgresChanges = []postgresFilter{filter.(postgresFilter)}
      break
   case broadcastFilter:
      msg.Payload.Config.Broadcast.Self = true
      break
   case presenceFilter:
      msg.Payload.Config.Presence.Key = ""
      break
   }

   return msg
}
