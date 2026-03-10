package tasksource

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNormalizedTaskJSONRoundTrip(t *testing.T) {
	original := NormalizedTask{
		ExternalID:  "ext-123",
		ExternalKey: "PROJ-456",
		Title:       "Implement feature X",
		Body:        "Detailed description of feature X.",
		Labels:      []string{"enhancement", "high-priority"},
		Priority:    3,
		SourceType:  "github",
		SourceURL:   "https://github.com/org/repo/issues/456",
		Metadata: map[string]string{
			"state":    "open",
			"assignee": "alice",
		},
		CreatedAt: time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC),
		UpdatedAt: time.Date(2025, 7, 20, 14, 45, 0, 0, time.UTC),
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var restored NormalizedTask
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	// Verify scalar fields.
	if restored.ExternalID != original.ExternalID {
		t.Errorf("ExternalID: got %q, want %q", restored.ExternalID, original.ExternalID)
	}
	if restored.ExternalKey != original.ExternalKey {
		t.Errorf("ExternalKey: got %q, want %q", restored.ExternalKey, original.ExternalKey)
	}
	if restored.Title != original.Title {
		t.Errorf("Title: got %q, want %q", restored.Title, original.Title)
	}
	if restored.Body != original.Body {
		t.Errorf("Body: got %q, want %q", restored.Body, original.Body)
	}
	if restored.Priority != original.Priority {
		t.Errorf("Priority: got %d, want %d", restored.Priority, original.Priority)
	}
	if restored.SourceType != original.SourceType {
		t.Errorf("SourceType: got %q, want %q", restored.SourceType, original.SourceType)
	}
	if restored.SourceURL != original.SourceURL {
		t.Errorf("SourceURL: got %q, want %q", restored.SourceURL, original.SourceURL)
	}

	// Verify Labels slice.
	if len(restored.Labels) != len(original.Labels) {
		t.Fatalf("Labels length: got %d, want %d", len(restored.Labels), len(original.Labels))
	}
	for i, l := range original.Labels {
		if restored.Labels[i] != l {
			t.Errorf("Labels[%d]: got %q, want %q", i, restored.Labels[i], l)
		}
	}

	// Verify Metadata map.
	if len(restored.Metadata) != len(original.Metadata) {
		t.Fatalf("Metadata length: got %d, want %d", len(restored.Metadata), len(original.Metadata))
	}
	for k, v := range original.Metadata {
		if restored.Metadata[k] != v {
			t.Errorf("Metadata[%q]: got %q, want %q", k, restored.Metadata[k], v)
		}
	}

	// Verify timestamps with tolerance to account for any serialization rounding.
	const tolerance = time.Second
	if diff := restored.CreatedAt.Sub(original.CreatedAt); diff < -tolerance || diff > tolerance {
		t.Errorf("CreatedAt: got %v, want %v (diff %v exceeds tolerance %v)",
			restored.CreatedAt, original.CreatedAt, diff, tolerance)
	}
	if diff := restored.UpdatedAt.Sub(original.UpdatedAt); diff < -tolerance || diff > tolerance {
		t.Errorf("UpdatedAt: got %v, want %v (diff %v exceeds tolerance %v)",
			restored.UpdatedAt, original.UpdatedAt, diff, tolerance)
	}
}

func TestNormalizedTaskZeroValues(t *testing.T) {
	var task NormalizedTask

	if task.ExternalID != "" {
		t.Errorf("ExternalID: got %q, want empty string", task.ExternalID)
	}
	if task.ExternalKey != "" {
		t.Errorf("ExternalKey: got %q, want empty string", task.ExternalKey)
	}
	if task.Title != "" {
		t.Errorf("Title: got %q, want empty string", task.Title)
	}
	if task.Body != "" {
		t.Errorf("Body: got %q, want empty string", task.Body)
	}
	if task.SourceType != "" {
		t.Errorf("SourceType: got %q, want empty string", task.SourceType)
	}
	if task.SourceURL != "" {
		t.Errorf("SourceURL: got %q, want empty string", task.SourceURL)
	}
	if task.Labels != nil {
		t.Errorf("Labels: got %v, want nil", task.Labels)
	}
	if task.Metadata != nil {
		t.Errorf("Metadata: got %v, want nil", task.Metadata)
	}
	if task.Priority != 0 {
		t.Errorf("Priority: got %d, want 0", task.Priority)
	}
	if !task.CreatedAt.IsZero() {
		t.Errorf("CreatedAt: got %v, want zero time", task.CreatedAt)
	}
	if !task.UpdatedAt.IsZero() {
		t.Errorf("UpdatedAt: got %v, want zero time", task.UpdatedAt)
	}
}
