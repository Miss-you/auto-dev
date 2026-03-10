package tasksource_test

import (
	"context"
	"errors"
	"testing"

	"github.com/lihui/auto-dev/internal/tasksource"
)

// Compile-time check: MemoryProvider must implement Provider.
var _ tasksource.Provider = (*tasksource.MemoryProvider)(nil)

func TestMemoryProviderFetch(t *testing.T) {
	tasks := []tasksource.NormalizedTask{
		{ExternalID: "1", Title: "First task"},
		{ExternalID: "2", Title: "Second task"},
	}
	mp := &tasksource.MemoryProvider{Tasks: tasks}

	got, err := mp.FetchCandidateTasks(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != len(tasks) {
		t.Fatalf("expected %d tasks, got %d", len(tasks), len(got))
	}
	for i, task := range got {
		if task.ExternalID != tasks[i].ExternalID {
			t.Errorf("task[%d].ExternalID = %q, want %q", i, task.ExternalID, tasks[i].ExternalID)
		}
		if task.Title != tasks[i].Title {
			t.Errorf("task[%d].Title = %q, want %q", i, task.Title, tasks[i].Title)
		}
	}
}

func TestMemoryProviderFetchReturnsDeepCopies(t *testing.T) {
	tasks := []tasksource.NormalizedTask{
		{
			ExternalID: "1",
			Title:      "First task",
			Labels:     []string{"bug"},
			Metadata:   map[string]string{"state": "open"},
		},
	}
	mp := &tasksource.MemoryProvider{Tasks: tasks}

	got, err := mp.FetchCandidateTasks(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 task, got %d", len(got))
	}

	got[0].Labels[0] = "changed"
	got[0].Metadata["state"] = "closed"

	if mp.Tasks[0].Labels[0] != "bug" {
		t.Fatalf("provider labels mutated through fetched task: got %q", mp.Tasks[0].Labels[0])
	}
	if mp.Tasks[0].Metadata["state"] != "open" {
		t.Fatalf("provider metadata mutated through fetched task: got %q", mp.Tasks[0].Metadata["state"])
	}
}

func TestMemoryProviderFetchError(t *testing.T) {
	want := errors.New("fetch failed")
	mp := &tasksource.MemoryProvider{FetchError: want}

	_, err := mp.FetchCandidateTasks(context.Background())
	if !errors.Is(err, want) {
		t.Fatalf("expected error %v, got %v", want, err)
	}
}

func TestMemoryProviderPostComment(t *testing.T) {
	mp := &tasksource.MemoryProvider{}

	if err := mp.PostComment(context.Background(), "issue-42", "looks good"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mp.Comments) != 1 {
		t.Fatalf("expected 1 comment record, got %d", len(mp.Comments))
	}
	rec := mp.Comments[0]
	if rec.ExternalID != "issue-42" {
		t.Errorf("ExternalID = %q, want %q", rec.ExternalID, "issue-42")
	}
	if rec.Body != "looks good" {
		t.Errorf("Body = %q, want %q", rec.Body, "looks good")
	}
}

func TestMemoryProviderAddLabels(t *testing.T) {
	mp := &tasksource.MemoryProvider{}
	labels := []string{"bug", "urgent"}

	if err := mp.AddLabels(context.Background(), "issue-7", labels); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mp.AddedLabels) != 1 {
		t.Fatalf("expected 1 AddLabels record, got %d", len(mp.AddedLabels))
	}
	rec := mp.AddedLabels[0]
	if rec.ExternalID != "issue-7" {
		t.Errorf("ExternalID = %q, want %q", rec.ExternalID, "issue-7")
	}
	if len(rec.Labels) != len(labels) {
		t.Fatalf("expected %d labels, got %d", len(labels), len(rec.Labels))
	}
	for i, l := range rec.Labels {
		if l != labels[i] {
			t.Errorf("label[%d] = %q, want %q", i, l, labels[i])
		}
	}

	labels[0] = "changed"
	if rec.Labels[0] != "bug" {
		t.Fatalf("recorded labels should not be mutated by caller changes, got %q", rec.Labels[0])
	}
}

func TestMemoryProviderRemoveLabel(t *testing.T) {
	mp := &tasksource.MemoryProvider{}

	if err := mp.RemoveLabel(context.Background(), "issue-99", "wontfix"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(mp.RemovedLabels) != 1 {
		t.Fatalf("expected 1 RemoveLabel record, got %d", len(mp.RemovedLabels))
	}
	rec := mp.RemovedLabels[0]
	if rec.ExternalID != "issue-99" {
		t.Errorf("ExternalID = %q, want %q", rec.ExternalID, "issue-99")
	}
	if rec.Label != "wontfix" {
		t.Errorf("Label = %q, want %q", rec.Label, "wontfix")
	}
}

func TestMemoryProviderWriteError(t *testing.T) {
	want := errors.New("write failed")
	mp := &tasksource.MemoryProvider{WriteError: want}

	if err := mp.PostComment(context.Background(), "id", "body"); !errors.Is(err, want) {
		t.Errorf("PostComment: expected error %v, got %v", want, err)
	}
	if err := mp.AddLabels(context.Background(), "id", []string{"a"}); !errors.Is(err, want) {
		t.Errorf("AddLabels: expected error %v, got %v", want, err)
	}
	if err := mp.RemoveLabel(context.Background(), "id", "a"); !errors.Is(err, want) {
		t.Errorf("RemoveLabel: expected error %v, got %v", want, err)
	}

	// Verify no records were stored when WriteError is set.
	if len(mp.Comments) != 0 {
		t.Errorf("expected 0 Comments, got %d", len(mp.Comments))
	}
	if len(mp.AddedLabels) != 0 {
		t.Errorf("expected 0 AddedLabels, got %d", len(mp.AddedLabels))
	}
	if len(mp.RemovedLabels) != 0 {
		t.Errorf("expected 0 RemovedLabels, got %d", len(mp.RemovedLabels))
	}
}

func TestMemoryProviderFetchCount(t *testing.T) {
	mp := &tasksource.MemoryProvider{
		Tasks: []tasksource.NormalizedTask{{ExternalID: "1"}},
	}

	for i := 1; i <= 3; i++ {
		if _, err := mp.FetchCandidateTasks(context.Background()); err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
		if mp.FetchCount != i {
			t.Errorf("after call %d: FetchCount = %d, want %d", i, mp.FetchCount, i)
		}
	}
}
