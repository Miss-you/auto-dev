package tasksource

import (
	"testing"
)

// --- Test helper functions ---

// newTask creates a NormalizedTask with the given labels and no metadata.
func newTask(id string, labels ...string) NormalizedTask {
	return NormalizedTask{
		ExternalID: id,
		Title:      "Task " + id,
		Labels:     labels,
	}
}

// newTaskWithMetadata creates a NormalizedTask with the given labels and metadata.
func newTaskWithMetadata(id string, metadata map[string]string, labels ...string) NormalizedTask {
	return NormalizedTask{
		ExternalID: id,
		Title:      "Task " + id,
		Labels:     labels,
		Metadata:   metadata,
	}
}

// newTaskWithState creates a NormalizedTask with the given labels and a state metadata entry.
func newTaskWithState(id string, state string, labels ...string) NormalizedTask {
	return NormalizedTask{
		ExternalID: id,
		Title:      "Task " + id,
		Labels:     labels,
		Metadata:   map[string]string{"state": state},
	}
}

// taskIDs extracts ExternalID values from a slice of NormalizedTask for easy comparison.
func taskIDs(tasks []NormalizedTask) []string {
	ids := make([]string, len(tasks))
	for i, t := range tasks {
		ids[i] = t.ExternalID
	}
	return ids
}

// equalIDs returns true if two string slices contain the same elements in order.
func equalIDs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestFilterConfigApply(t *testing.T) {
	tests := []struct {
		name    string
		filter  FilterConfig
		input   []NormalizedTask
		wantIDs []string
	}{
		{
			name:   "empty config passes all tasks through",
			filter: FilterConfig{},
			input: []NormalizedTask{
				newTask("1", "bug"),
				newTask("2", "feature"),
				newTask("3"),
			},
			wantIDs: []string{"1", "2", "3"},
		},
		{
			name: "include labels: task with required labels passes",
			filter: FilterConfig{
				IncludeLabels: []string{"bug", "urgent"},
			},
			input: []NormalizedTask{
				newTask("1", "bug", "urgent", "backend"),
				newTask("2", "bug"),
				newTask("3", "urgent"),
			},
			wantIDs: []string{"1"},
		},
		{
			name: "include labels: task missing a required label is excluded",
			filter: FilterConfig{
				IncludeLabels: []string{"bug", "urgent"},
			},
			input: []NormalizedTask{
				newTask("1", "bug"),
				newTask("2", "urgent"),
				newTask("3", "feature"),
			},
			wantIDs: []string{},
		},
		{
			name: "exclude labels: task with excluded label is excluded",
			filter: FilterConfig{
				ExcludeLabels: []string{"wontfix"},
			},
			input: []NormalizedTask{
				newTask("1", "bug", "wontfix"),
				newTask("2", "bug"),
				newTask("3", "wontfix"),
			},
			wantIDs: []string{"2"},
		},
		{
			name: "exclude labels: task without excluded labels passes",
			filter: FilterConfig{
				ExcludeLabels: []string{"wontfix", "duplicate"},
			},
			input: []NormalizedTask{
				newTask("1", "bug", "urgent"),
				newTask("2", "feature"),
			},
			wantIDs: []string{"1", "2"},
		},
		{
			name: "states: task with matching state metadata passes",
			filter: FilterConfig{
				States: []string{"open"},
			},
			input: []NormalizedTask{
				newTaskWithState("1", "open", "bug"),
				newTaskWithState("2", "closed", "bug"),
			},
			wantIDs: []string{"1"},
		},
		{
			name: "states: task with non-matching state is excluded",
			filter: FilterConfig{
				States: []string{"open", "in_progress"},
			},
			input: []NormalizedTask{
				newTaskWithState("1", "closed", "bug"),
				newTaskWithState("2", "resolved", "feature"),
			},
			wantIDs: []string{},
		},
		{
			name: "states: task with no state metadata is excluded when states are configured",
			filter: FilterConfig{
				States: []string{"open"},
			},
			input: []NormalizedTask{
				newTask("1", "bug"),
				newTaskWithMetadata("2", map[string]string{"assignee": "alice"}, "bug"),
			},
			wantIDs: []string{},
		},
		{
			name: "combined: include + exclude + states together",
			filter: FilterConfig{
				IncludeLabels: []string{"bug"},
				ExcludeLabels: []string{"wontfix"},
				States:        []string{"open"},
			},
			input: []NormalizedTask{
				// has "bug", no "wontfix", state "open" -> pass
				newTaskWithState("1", "open", "bug", "urgent"),
				// has "bug", has "wontfix" -> excluded by ExcludeLabels
				newTaskWithState("2", "open", "bug", "wontfix"),
				// missing "bug" -> excluded by IncludeLabels
				newTaskWithState("3", "open", "feature"),
				// has "bug", no "wontfix", state "closed" -> excluded by States
				newTaskWithState("4", "closed", "bug"),
				// has "bug", no "wontfix", no state metadata -> excluded by States
				newTask("5", "bug"),
			},
			wantIDs: []string{"1"},
		},
		{
			name:    "empty input slice returns empty slice",
			filter:  FilterConfig{IncludeLabels: []string{"bug"}},
			input:   []NormalizedTask{},
			wantIDs: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.filter.Apply(tc.input)
			gotIDs := taskIDs(got)

			if !equalIDs(gotIDs, tc.wantIDs) {
				t.Errorf("Apply() returned IDs %v, want %v", gotIDs, tc.wantIDs)
			}
		})
	}
}
