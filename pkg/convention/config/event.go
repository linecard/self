package config

type Event struct {
	Source     string      `json:"source"`
	DetailType string      `json:"detail-type"`
	Detail     EventDetail `json:"detail"`
}

// Traceparent and Tracestate are placed internally to the event is to avoid conflict with x-ray and tracing provided by AWS infrastructure itself.
type EventDetail struct {
	Traceparent    string   `json:"traceparent"`
	Tracestate     string   `json:"tracestate"`
	Sha            string   `json:"sha"`
	Branch         string   `json:"branch"`
	Origin         string   `json:"origin"`
	RepositoryName string   `json:"repository-name"`
	ResourceName   string   `json:"resource-name"`
	ExceptAccounts []string `json:"except-accounts"`
}
