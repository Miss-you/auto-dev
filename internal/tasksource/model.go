package tasksource

import "time"

// NormalizedTask is the canonical task representation shared by all components.
type NormalizedTask struct {
	ExternalID  string            `json:"external_id"`
	ExternalKey string            `json:"external_key"`
	Title       string            `json:"title"`
	Body        string            `json:"body"`
	Labels      []string          `json:"labels"`
	Priority    int               `json:"priority"`
	SourceType  string            `json:"source_type"`
	SourceURL   string            `json:"source_url"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}
