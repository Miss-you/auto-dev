package tasksource

import "context"

// Provider abstracts a task source (e.g., GitHub Issues, Linear).
// All provider-specific API details are isolated behind this interface.
type Provider interface {
	// FetchCandidateTasks returns all candidate tasks from the source.
	// The provider handles pagination internally.
	FetchCandidateTasks(ctx context.Context) ([]NormalizedTask, error)

	// PostComment posts a comment on the specified external task.
	PostComment(ctx context.Context, externalID string, body string) error

	// AddLabels adds labels to the specified external task without affecting existing labels.
	AddLabels(ctx context.Context, externalID string, labels []string) error

	// RemoveLabel removes a single label from the specified external task.
	RemoveLabel(ctx context.Context, externalID string, label string) error
}
