package tasksource

import (
	"context"
	"sync"
)

// CommentRecord records a PostComment call.
type CommentRecord struct {
	ExternalID string
	Body       string
}

// AddLabelsRecord records an AddLabels call.
type AddLabelsRecord struct {
	ExternalID string
	Labels     []string
}

// RemoveLabelRecord records a RemoveLabel call.
type RemoveLabelRecord struct {
	ExternalID string
	Label      string
}

// MemoryProvider is an in-memory Provider implementation for testing.
type MemoryProvider struct {
	// Mu protects all fields during concurrent access.
	// Exported so external test packages can safely mutate provider state mid-test.
	Mu sync.Mutex

	// Tasks is the list returned by FetchCandidateTasks.
	Tasks []NormalizedTask

	// FetchError, if non-nil, is returned by FetchCandidateTasks instead of Tasks.
	FetchError error

	// WriteError, if non-nil, is returned by PostComment/AddLabels/RemoveLabel.
	WriteError error

	// Call history
	Comments      []CommentRecord
	AddedLabels   []AddLabelsRecord
	RemovedLabels []RemoveLabelRecord
	FetchCount    int
}

func (m *MemoryProvider) FetchCandidateTasks(ctx context.Context) ([]NormalizedTask, error) {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	m.FetchCount++
	if m.FetchError != nil {
		return nil, m.FetchError
	}
	return cloneTasks(m.Tasks), nil
}

func (m *MemoryProvider) PostComment(ctx context.Context, externalID string, body string) error {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	if m.WriteError != nil {
		return m.WriteError
	}
	m.Comments = append(m.Comments, CommentRecord{ExternalID: externalID, Body: body})
	return nil
}

func (m *MemoryProvider) AddLabels(ctx context.Context, externalID string, labels []string) error {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	if m.WriteError != nil {
		return m.WriteError
	}
	m.AddedLabels = append(m.AddedLabels, AddLabelsRecord{
		ExternalID: externalID,
		Labels:     cloneStrings(labels),
	})
	return nil
}

func (m *MemoryProvider) RemoveLabel(ctx context.Context, externalID string, label string) error {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	if m.WriteError != nil {
		return m.WriteError
	}
	m.RemovedLabels = append(m.RemovedLabels, RemoveLabelRecord{ExternalID: externalID, Label: label})
	return nil
}

func cloneTasks(tasks []NormalizedTask) []NormalizedTask {
	if tasks == nil {
		return nil
	}

	result := make([]NormalizedTask, len(tasks))
	for i, task := range tasks {
		result[i] = NormalizedTask{
			ExternalID:  task.ExternalID,
			ExternalKey: task.ExternalKey,
			Title:       task.Title,
			Body:        task.Body,
			Labels:      cloneStrings(task.Labels),
			Priority:    task.Priority,
			SourceType:  task.SourceType,
			SourceURL:   task.SourceURL,
			Metadata:    cloneMetadata(task.Metadata),
			CreatedAt:   task.CreatedAt,
			UpdatedAt:   task.UpdatedAt,
		}
	}
	return result
}

func cloneStrings(values []string) []string {
	if values == nil {
		return nil
	}

	result := make([]string, len(values))
	copy(result, values)
	return result
}

func cloneMetadata(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}

	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}
