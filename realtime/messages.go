package realtime;

type ConnectionMsg struct {
   Event string `json:"event"`
   Topic string `json:"topic"`
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
   Ref string `json:"ref"`
}

type HearbeatMsg struct {
   Event string `json:"event"`
   Topic string `json:"topic"`
   Payload struct {
   } `json:"payload"`
   Ref string `json:"ref"`
}
