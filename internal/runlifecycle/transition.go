package runlifecycle

import "fmt"

type transitionKey struct {
	From RunState
	To   RunState
}

// allowedTransitions maps each valid (from, to) pair to the authorized actor.
// "any non-terminal → failed" and "any non-terminal → aborted" are handled
// separately in ValidateTransition to avoid enumerating all combinations.
var allowedTransitions = map[transitionKey]Actor{
	{StateSpecDrafting, StateSpecInReview}:  ActorSupervisor,
	{StateSpecInReview, StateSpecDrafting}:  ActorSupervisor,
	{StateSpecInReview, StateSpecApproved}:  ActorSupervisor,
	{StateSpecApproved, StateImplQueued}:    ActorDispatcher,
	{StateImplQueued, StateImplementing}:    ActorSupervisor,
	{StateImplementing, StateCodeInReview}:  ActorSupervisor,
	{StateCodeInReview, StateFixingReview}:  ActorSupervisor,
	{StateFixingReview, StateCodeInReview}:  ActorSupervisor,
	{StateCodeInReview, StateVerified}:      ActorSupervisor,
	{StateVerified, StateCompleted}:         ActorDispatcher,
}

// ValidateTransition checks whether a transition from → to by actor is allowed.
// Returns nil if valid, ErrTerminalState if from is terminal,
// or ErrIllegalTransition if the transition is not in the allowed set.
func ValidateTransition(from, to RunState, actor Actor) error {
	if IsTerminal(from) {
		return fmt.Errorf("cannot transition from %s: %w", from, ErrTerminalState)
	}

	// any non-terminal → failed (supervisor or dispatcher)
	if to == StateFailed {
		if actor == ActorSupervisor || actor == ActorDispatcher {
			return nil
		}
		return fmt.Errorf("transition %s → %s requires supervisor or dispatcher, got %s: %w",
			from, to, actor, ErrIllegalTransition)
	}

	// any non-terminal → aborted (operator only)
	if to == StateAborted {
		if actor == ActorOperator {
			return nil
		}
		return fmt.Errorf("transition %s → %s requires operator, got %s: %w",
			from, to, actor, ErrIllegalTransition)
	}

	// named transitions
	key := transitionKey{from, to}
	requiredActor, ok := allowedTransitions[key]
	if !ok {
		return fmt.Errorf("transition %s → %s is not allowed: %w", from, to, ErrIllegalTransition)
	}
	if actor != requiredActor {
		return fmt.Errorf("transition %s → %s requires %s, got %s: %w",
			from, to, requiredActor, actor, ErrIllegalTransition)
	}
	return nil
}
