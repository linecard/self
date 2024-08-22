package config

type Event struct {
	Source     string      `json:"source"`
	DetailType string      `json:"detail-type"`
	Detail     EventDetail `json:"detail"`
}

type EventDetail struct {
	Traceparent    string   `json:"traceparent"`
	Tracestate     string   `json:"tracestate"`
	Action         string   `json:"action"`
	Sha            string   `json:"sha"`
	Branch         string   `json:"branch"`
	Origin         string   `json:"origin"`
	RepositoryName string   `json:"repository-name"`
	ResourceName   string   `json:"resource-name"`
	ExceptAccounts []string `json:"except-accounts"`
}
