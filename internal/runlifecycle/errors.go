package runlifecycle

import "errors"

var (
	// ErrIllegalTransition is returned when a state transition is not in the allowed set.
	ErrIllegalTransition = errors.New("illegal state transition")

	// ErrTerminalState is returned when attempting to transition from a terminal state.
	ErrTerminalState = errors.New("run is in terminal state")

	// ErrReviewFixLimitExceeded is returned when the review fix limit is exceeded
	// and the run is auto-transitioned to failed.
	ErrReviewFixLimitExceeded = errors.New("review fix limit exceeded")
)
