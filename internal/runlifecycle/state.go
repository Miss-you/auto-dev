package runlifecycle

import "slices"

// RunState represents a canonical task state in the run lifecycle.
type RunState string

const (
	StateSpecDrafting RunState = "spec_drafting"
	StateSpecInReview RunState = "spec_in_review"
	StateSpecApproved RunState = "spec_approved"
	StateImplQueued   RunState = "impl_queued"
	StateImplementing RunState = "implementing"
	StateCodeInReview RunState = "code_in_review"
	StateFixingReview RunState = "fixing_review"
	StateVerified     RunState = "verified"
	StateCompleted    RunState = "completed"
	StateFailed       RunState = "failed"
	StateAborted      RunState = "aborted"
)

// Actor represents an authorized trigger for state transitions.
type Actor string

const (
	ActorDispatcher Actor = "dispatcher"
	ActorSupervisor Actor = "supervisor"
	ActorOperator   Actor = "operator"
)

// Phase groupings (unexported to prevent mutation).
var (
	phaseAStates = []RunState{
		StateSpecDrafting,
		StateSpecInReview,
		StateSpecApproved,
	}

	phaseBStates = []RunState{
		StateImplQueued,
		StateImplementing,
		StateCodeInReview,
		StateFixingReview,
		StateVerified,
	}

	terminalStates = []RunState{
		StateCompleted,
		StateFailed,
		StateAborted,
	}

	allStates = func() []RunState {
		all := make([]RunState, 0, len(phaseAStates)+len(phaseBStates)+len(terminalStates))
		all = append(all, phaseAStates...)
		all = append(all, phaseBStates...)
		all = append(all, terminalStates...)
		return all
	}()
)

var terminalSet = func() map[RunState]bool {
	m := make(map[RunState]bool, len(terminalStates))
	for _, s := range terminalStates {
		m[s] = true
	}
	return m
}()

// IsTerminal returns true if the state is a terminal state.
func IsTerminal(s RunState) bool {
	return terminalSet[s]
}

// PhaseOf returns "A", "B", or "terminal" for a given state.
func PhaseOf(s RunState) string {
	if slices.Contains(phaseAStates, s) {
		return "A"
	}
	if slices.Contains(phaseBStates, s) {
		return "B"
	}
	if IsTerminal(s) {
		return "terminal"
	}
	return "unknown"
}

// PhaseAStates returns a copy of the Phase-A state slice.
func PhaseAStates() []RunState { return append([]RunState{}, phaseAStates...) }

// PhaseBStates returns a copy of the Phase-B state slice.
func PhaseBStates() []RunState { return append([]RunState{}, phaseBStates...) }

// TerminalStates returns a copy of the terminal state slice.
func TerminalStates() []RunState { return append([]RunState{}, terminalStates...) }

// AllStates returns a copy of all defined states.
func AllStates() []RunState { return append([]RunState{}, allStates...) }
