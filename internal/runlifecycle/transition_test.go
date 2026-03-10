package runlifecycle_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/lihui/auto-dev/internal/runlifecycle"
)

// TestAllowedTransitions verifies every named transition in the spec succeeds
// with the correct actor.
func TestAllowedTransitions(t *testing.T) {
	tests := []struct {
		from  runlifecycle.RunState
		to    runlifecycle.RunState
		actor runlifecycle.Actor
	}{
		{runlifecycle.StateSpecDrafting, runlifecycle.StateSpecInReview, runlifecycle.ActorSupervisor},
		{runlifecycle.StateSpecInReview, runlifecycle.StateSpecDrafting, runlifecycle.ActorSupervisor},
		{runlifecycle.StateSpecInReview, runlifecycle.StateSpecApproved, runlifecycle.ActorSupervisor},
		{runlifecycle.StateSpecApproved, runlifecycle.StateImplQueued, runlifecycle.ActorDispatcher},
		{runlifecycle.StateImplQueued, runlifecycle.StateImplementing, runlifecycle.ActorSupervisor},
		{runlifecycle.StateImplementing, runlifecycle.StateCodeInReview, runlifecycle.ActorSupervisor},
		{runlifecycle.StateCodeInReview, runlifecycle.StateFixingReview, runlifecycle.ActorSupervisor},
		{runlifecycle.StateFixingReview, runlifecycle.StateCodeInReview, runlifecycle.ActorSupervisor},
		{runlifecycle.StateCodeInReview, runlifecycle.StateVerified, runlifecycle.ActorSupervisor},
		{runlifecycle.StateVerified, runlifecycle.StateCompleted, runlifecycle.ActorDispatcher},
	}

	for _, tt := range tests {
		name := fmt.Sprintf("%s->%s_by_%s", tt.from, tt.to, tt.actor)
		t.Run(name, func(t *testing.T) {
			err := runlifecycle.ValidateTransition(tt.from, tt.to, tt.actor)
			if err != nil {
				t.Errorf("expected transition %s -> %s by %s to be allowed, got error: %v",
					tt.from, tt.to, tt.actor, err)
			}
		})
	}
}

// nonTerminalStates returns all states that are not terminal.
func nonTerminalStates() []runlifecycle.RunState {
	var states []runlifecycle.RunState
	states = append(states, runlifecycle.PhaseAStates()...)
	states = append(states, runlifecycle.PhaseBStates()...)
	return states
}

// TestFailedTransitionFromAnyNonTerminal verifies that every non-terminal state
// can transition to failed via supervisor or dispatcher.
func TestFailedTransitionFromAnyNonTerminal(t *testing.T) {
	actors := []runlifecycle.Actor{
		runlifecycle.ActorSupervisor,
		runlifecycle.ActorDispatcher,
	}

	for _, from := range nonTerminalStates() {
		for _, actor := range actors {
			name := fmt.Sprintf("%s->failed_by_%s", from, actor)
			t.Run(name, func(t *testing.T) {
				err := runlifecycle.ValidateTransition(from, runlifecycle.StateFailed, actor)
				if err != nil {
					t.Errorf("expected %s -> failed by %s to succeed, got: %v", from, actor, err)
				}
			})
		}
	}
}

// TestAbortedTransitionFromAnyNonTerminal verifies that every non-terminal state
// can transition to aborted via operator only.
func TestAbortedTransitionFromAnyNonTerminal(t *testing.T) {
	for _, from := range nonTerminalStates() {
		name := fmt.Sprintf("%s->aborted_by_operator", from)
		t.Run(name, func(t *testing.T) {
			err := runlifecycle.ValidateTransition(from, runlifecycle.StateAborted, runlifecycle.ActorOperator)
			if err != nil {
				t.Errorf("expected %s -> aborted by operator to succeed, got: %v", from, err)
			}
		})
	}
}

// TestTerminalStateLocked verifies that all three terminal states reject any
// outbound transition with ErrTerminalState.
func TestTerminalStateLocked(t *testing.T) {
	terminalStates := []runlifecycle.RunState{
		runlifecycle.StateCompleted,
		runlifecycle.StateFailed,
		runlifecycle.StateAborted,
	}
	targets := []runlifecycle.RunState{
		runlifecycle.StateSpecDrafting,
		runlifecycle.StateImplementing,
		runlifecycle.StateCompleted,
		runlifecycle.StateFailed,
		runlifecycle.StateAborted,
	}
	actors := []runlifecycle.Actor{
		runlifecycle.ActorSupervisor,
		runlifecycle.ActorDispatcher,
		runlifecycle.ActorOperator,
	}

	for _, from := range terminalStates {
		for _, to := range targets {
			for _, actor := range actors {
				name := fmt.Sprintf("%s->%s_by_%s", from, to, actor)
				t.Run(name, func(t *testing.T) {
					err := runlifecycle.ValidateTransition(from, to, actor)
					if err == nil {
						t.Fatalf("expected error transitioning from terminal state %s, got nil", from)
					}
					if !errors.Is(err, runlifecycle.ErrTerminalState) {
						t.Errorf("expected ErrTerminalState, got: %v", err)
					}
				})
			}
		}
	}
}

// TestIllegalTransitions verifies that transitions not in the allowed set
// return ErrIllegalTransition.
func TestIllegalTransitions(t *testing.T) {
	tests := []struct {
		from  runlifecycle.RunState
		to    runlifecycle.RunState
		actor runlifecycle.Actor
	}{
		// skip states in the pipeline
		{runlifecycle.StateSpecDrafting, runlifecycle.StateImplementing, runlifecycle.ActorSupervisor},
		// jump from approved straight to completed
		{runlifecycle.StateSpecApproved, runlifecycle.StateCompleted, runlifecycle.ActorDispatcher},
		// reverse direction not allowed
		{runlifecycle.StateImplementing, runlifecycle.StateImplQueued, runlifecycle.ActorSupervisor},
		// jump from drafting to verified
		{runlifecycle.StateSpecDrafting, runlifecycle.StateVerified, runlifecycle.ActorSupervisor},
		// impl_queued directly to code_in_review
		{runlifecycle.StateImplQueued, runlifecycle.StateCodeInReview, runlifecycle.ActorSupervisor},
		// verified back to implementing
		{runlifecycle.StateVerified, runlifecycle.StateImplementing, runlifecycle.ActorSupervisor},
	}

	for _, tt := range tests {
		name := fmt.Sprintf("%s->%s_by_%s", tt.from, tt.to, tt.actor)
		t.Run(name, func(t *testing.T) {
			err := runlifecycle.ValidateTransition(tt.from, tt.to, tt.actor)
			if err == nil {
				t.Fatalf("expected error for illegal transition %s -> %s, got nil", tt.from, tt.to)
			}
			if !errors.Is(err, runlifecycle.ErrIllegalTransition) {
				t.Errorf("expected ErrIllegalTransition, got: %v", err)
			}
		})
	}
}

// TestActorMismatch verifies that valid transitions with the wrong actor
// return ErrIllegalTransition.
func TestActorMismatch(t *testing.T) {
	tests := []struct {
		from       runlifecycle.RunState
		to         runlifecycle.RunState
		wrongActor runlifecycle.Actor
	}{
		// spec_drafting -> spec_in_review requires supervisor, not dispatcher
		{runlifecycle.StateSpecDrafting, runlifecycle.StateSpecInReview, runlifecycle.ActorDispatcher},
		// spec_drafting -> spec_in_review requires supervisor, not operator
		{runlifecycle.StateSpecDrafting, runlifecycle.StateSpecInReview, runlifecycle.ActorOperator},
		// spec_approved -> impl_queued requires dispatcher, not supervisor
		{runlifecycle.StateSpecApproved, runlifecycle.StateImplQueued, runlifecycle.ActorSupervisor},
		// spec_approved -> impl_queued requires dispatcher, not operator
		{runlifecycle.StateSpecApproved, runlifecycle.StateImplQueued, runlifecycle.ActorOperator},
		// verified -> completed requires dispatcher, not supervisor
		{runlifecycle.StateVerified, runlifecycle.StateCompleted, runlifecycle.ActorSupervisor},
		// impl_queued -> implementing requires supervisor, not dispatcher
		{runlifecycle.StateImplQueued, runlifecycle.StateImplementing, runlifecycle.ActorDispatcher},
		// code_in_review -> verified requires supervisor, not dispatcher
		{runlifecycle.StateCodeInReview, runlifecycle.StateVerified, runlifecycle.ActorDispatcher},
		// any non-terminal -> aborted requires operator, not supervisor
		{runlifecycle.StateImplementing, runlifecycle.StateAborted, runlifecycle.ActorSupervisor},
		// any non-terminal -> aborted requires operator, not dispatcher
		{runlifecycle.StateImplementing, runlifecycle.StateAborted, runlifecycle.ActorDispatcher},
		// any non-terminal -> failed requires supervisor or dispatcher, not operator
		{runlifecycle.StateImplementing, runlifecycle.StateFailed, runlifecycle.ActorOperator},
	}

	for _, tt := range tests {
		name := fmt.Sprintf("%s->%s_by_%s", tt.from, tt.to, tt.wrongActor)
		t.Run(name, func(t *testing.T) {
			err := runlifecycle.ValidateTransition(tt.from, tt.to, tt.wrongActor)
			if err == nil {
				t.Fatalf("expected error for actor mismatch %s -> %s by %s, got nil",
					tt.from, tt.to, tt.wrongActor)
			}
			if !errors.Is(err, runlifecycle.ErrIllegalTransition) {
				t.Errorf("expected ErrIllegalTransition for actor mismatch, got: %v", err)
			}
		})
	}
}

func TestSpecApprovedCannotRevertToSpecDrafting(t *testing.T) {
	err := runlifecycle.ValidateTransition(
		runlifecycle.StateSpecApproved,
		runlifecycle.StateSpecDrafting,
		runlifecycle.ActorSupervisor,
	)
	if err == nil {
		t.Fatal("expected error for spec_approved -> spec_drafting, got nil")
	}
	if !errors.Is(err, runlifecycle.ErrIllegalTransition) {
		t.Fatalf("expected ErrIllegalTransition, got: %v", err)
	}
}
