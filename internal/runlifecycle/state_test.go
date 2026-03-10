package runlifecycle_test

import (
	"testing"

	"github.com/lihui/auto-dev/internal/runlifecycle"
)

func TestAllStatesCount(t *testing.T) {
	const want = 11
	if got := len(runlifecycle.AllStates()); got != want {
		t.Errorf("len(AllStates) = %d, want %d", got, want)
	}
}

func TestPhaseGroupsAreMutuallyExclusive(t *testing.T) {
	seen := make(map[runlifecycle.RunState]string)

	groups := []struct {
		name   string
		states []runlifecycle.RunState
	}{
		{"PhaseAStates", runlifecycle.PhaseAStates()},
		{"PhaseBStates", runlifecycle.PhaseBStates()},
		{"TerminalStates", runlifecycle.TerminalStates()},
	}

	for _, g := range groups {
		for _, s := range g.states {
			if prev, ok := seen[s]; ok {
				t.Errorf("state %q appears in both %s and %s", s, prev, g.name)
			}
			seen[s] = g.name
		}
	}
}

func TestPhaseGroupsCoverAllStates(t *testing.T) {
	union := make(map[runlifecycle.RunState]bool)
	for _, s := range runlifecycle.PhaseAStates() {
		union[s] = true
	}
	for _, s := range runlifecycle.PhaseBStates() {
		union[s] = true
	}
	for _, s := range runlifecycle.TerminalStates() {
		union[s] = true
	}

	allSet := make(map[runlifecycle.RunState]bool)
	for _, s := range runlifecycle.AllStates() {
		allSet[s] = true
	}

	// Check that every state in AllStates is covered by a group.
	for _, s := range runlifecycle.AllStates() {
		if !union[s] {
			t.Errorf("state %q is in AllStates but not in any phase group", s)
		}
	}

	// Check that every state in the union is present in AllStates.
	for s := range union {
		if !allSet[s] {
			t.Errorf("state %q is in a phase group but not in AllStates", s)
		}
	}

	if len(union) != len(allSet) {
		t.Errorf("union of groups has %d states, AllStates has %d", len(union), len(allSet))
	}
}

func TestIsTerminal(t *testing.T) {
	tests := []struct {
		state runlifecycle.RunState
		want  bool
	}{
		// Terminal states.
		{runlifecycle.StateCompleted, true},
		{runlifecycle.StateFailed, true},
		{runlifecycle.StateAborted, true},
		// Non-terminal states.
		{runlifecycle.StateSpecDrafting, false},
		{runlifecycle.StateSpecInReview, false},
		{runlifecycle.StateSpecApproved, false},
		{runlifecycle.StateImplQueued, false},
		{runlifecycle.StateImplementing, false},
		{runlifecycle.StateCodeInReview, false},
		{runlifecycle.StateFixingReview, false},
		{runlifecycle.StateVerified, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			if got := runlifecycle.IsTerminal(tt.state); got != tt.want {
				t.Errorf("IsTerminal(%q) = %v, want %v", tt.state, got, tt.want)
			}
		})
	}
}

func TestPhaseOf(t *testing.T) {
	tests := []struct {
		state runlifecycle.RunState
		want  string
	}{
		// Phase A states.
		{runlifecycle.StateSpecDrafting, "A"},
		{runlifecycle.StateSpecInReview, "A"},
		{runlifecycle.StateSpecApproved, "A"},
		// Phase B states.
		{runlifecycle.StateImplQueued, "B"},
		{runlifecycle.StateImplementing, "B"},
		{runlifecycle.StateCodeInReview, "B"},
		{runlifecycle.StateFixingReview, "B"},
		{runlifecycle.StateVerified, "B"},
		// Terminal states.
		{runlifecycle.StateCompleted, "terminal"},
		{runlifecycle.StateFailed, "terminal"},
		{runlifecycle.StateAborted, "terminal"},
		// Unknown state.
		{runlifecycle.RunState("bogus"), "unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			if got := runlifecycle.PhaseOf(tt.state); got != tt.want {
				t.Errorf("PhaseOf(%q) = %q, want %q", tt.state, got, tt.want)
			}
		})
	}
}
