package tasksource

import (
	"log/slog"
	"slices"
)

// FilterConfig defines source-side filtering rules.
type FilterConfig struct {
	// IncludeLabels requires tasks to have ALL of these labels (AND semantics).
	// Empty means no inclusion filter.
	IncludeLabels []string

	// ExcludeLabels excludes tasks that have ANY of these labels.
	// Empty means no exclusion filter.
	ExcludeLabels []string

	// States restricts tasks to these states (e.g., "open", "closed").
	// Empty means no state filter (all states pass).
	// Note: This is for adapter-layer filtering; GitHub API-level state filtering
	// is handled in the provider.
	States []string
}

// Apply filters a slice of NormalizedTasks according to the config.
// Tasks that do not match the filter criteria are excluded.
func (f FilterConfig) Apply(tasks []NormalizedTask) []NormalizedTask {
	if len(f.IncludeLabels) == 0 && len(f.ExcludeLabels) == 0 && len(f.States) == 0 {
		return tasks
	}

	result := make([]NormalizedTask, 0, len(tasks))
	for _, t := range tasks {
		if f.match(t) {
			result = append(result, t)
		}
	}
	return result
}

func (f FilterConfig) match(t NormalizedTask) bool {
	// Check include labels: task must have ALL include labels.
	for _, req := range f.IncludeLabels {
		if !slices.Contains(t.Labels, req) {
			slog.Debug("task excluded: missing required label",
				"external_id", t.ExternalID, "missing_label", req)
			return false
		}
	}

	// Check exclude labels: task must NOT have ANY exclude label.
	for _, excl := range f.ExcludeLabels {
		if slices.Contains(t.Labels, excl) {
			slog.Debug("task excluded: has excluded label",
				"external_id", t.ExternalID, "excluded_label", excl)
			return false
		}
	}

	// Check states: if States is set, task's Metadata["state"] must be in the list.
	// NormalizedTask doesn't have a State field; state filtering at the adapter layer
	// uses Metadata["state"] if present. If States is configured but task has no state
	// metadata, the task is excluded.
	if len(f.States) > 0 {
		state, ok := t.Metadata["state"]
		if !ok || !slices.Contains(f.States, state) {
			slog.Debug("task excluded: state not in whitelist",
				"external_id", t.ExternalID, "state", state)
			return false
		}
	}

	return true
}
