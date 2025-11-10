package realtime

type TemplateMsg struct {
	Event string `json:"event"`
	Topic string `json:"topic"`
	Ref   string `json:"ref"`
}

type ConnectionMsg struct {
	TemplateMsg

	Payload struct {
		Data struct {
			Schema     string            `json:"schema"`
			Table      string            `json:"table"`
			CommitTime string            `json:"commit_timestamp"`
			EventType  string            `json:"eventType"`
			New        map[string]string `json:"new"`
			Old        map[string]string `json:"old"`
			Errors     string            `json:"errors"`
		} `json:"data"`
	} `json:"payload"`
}

type HearbeatMsg struct {
	TemplateMsg

	Payload struct {
	} `json:"payload"`
}
