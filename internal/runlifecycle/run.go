package runlifecycle

import (
	"fmt"
	"time"
)

const (
	// DefaultReviewFixLimit is the default maximum number of review fix cycles.
	DefaultReviewFixLimit = 3

	// FailureReasonPostVerificationCleanup indicates sync/archive failure after verification.
	FailureReasonPostVerificationCleanup = "post_verification_cleanup_failed"
)

// TransitionRecord captures a single state transition event.
type TransitionRecord struct {
	From      RunState  `json:"from"`
	To        RunState  `json:"to"`
	Actor     Actor     `json:"actor"`
	Reason    string    `json:"reason,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Run represents a single task run instance with its lifecycle state.
type Run struct {
	ID             string             `json:"id"`
	State          RunState           `json:"state"`
	History        []TransitionRecord `json:"history"`
	ReviewFixCount int                `json:"review_fix_count"`
	ReviewFixLimit int                `json:"review_fix_limit"`
	FailureReason  string             `json:"failure_reason,omitempty"`
}

// RunOption configures a Run at construction time.
type RunOption func(*Run)

// WithReviewFixLimit sets the maximum number of review fix cycles.
// A limit of 0 means no fix cycles are allowed.
func WithReviewFixLimit(limit int) RunOption {
	return func(r *Run) {
		if limit < 0 {
			limit = 0
		}
		r.ReviewFixLimit = limit
	}
}

// NewRun creates a new Run in the spec_drafting initial state.
func NewRun(id string, opts ...RunOption) *Run {
	r := &Run{
		ID:             id,
		State:          StateSpecDrafting,
		History:        make([]TransitionRecord, 0),
		ReviewFixLimit: DefaultReviewFixLimit,
	}
	for _, opt := range opts {
		opt(r)
	}
	r.History = []TransitionRecord{{
		From:      "",
		To:        StateSpecDrafting,
		Actor:     ActorDispatcher,
		Reason:    "run created",
		Timestamp: time.Now(),
	}}
	return r
}

// CompletedEvidence captures the evidence required before transitioning to completed.
type CompletedEvidence struct {
	VerifyPassed         bool
	SpecsSynced          bool
	ChangeArchived       bool
	AllPRsMergedOrClosed bool
}

// TransitionOption configures a single transition call.
type TransitionOption func(*transitionConfig)

type transitionConfig struct {
	reason   string
	evidence *CompletedEvidence
}

// WithReason attaches a reason string to the transition record.
func WithReason(reason string) TransitionOption {
	return func(c *transitionConfig) {
		c.reason = reason
	}
}

// WithCompletedEvidence attaches completion evidence to the transition.
// Evidence is required when transitioning to the completed state.
func WithCompletedEvidence(ev CompletedEvidence) TransitionOption {
	return func(c *transitionConfig) {
		c.evidence = &ev
	}
}

// Transition attempts to move the run to a new state.
// It validates the transition, applies special rules (failure reason, review fix counting),
// records the transition in history, and updates the state.
func (r *Run) Transition(to RunState, actor Actor, opts ...TransitionOption) error {
	cfg := &transitionConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	// Completed evidence check: transitioning to completed requires all evidence fields.
	if to == StateCompleted {
		if cfg.evidence == nil {
			return fmt.Errorf("transition to %s requires completed evidence: %w", StateCompleted, ErrIllegalTransition)
		}
		if !cfg.evidence.VerifyPassed {
			return fmt.Errorf("transition to %s: VerifyPassed is false: %w", StateCompleted, ErrIllegalTransition)
		}
		if !cfg.evidence.SpecsSynced {
			return fmt.Errorf("transition to %s: SpecsSynced is false: %w", StateCompleted, ErrIllegalTransition)
		}
		if !cfg.evidence.ChangeArchived {
			return fmt.Errorf("transition to %s: ChangeArchived is false: %w", StateCompleted, ErrIllegalTransition)
		}
		if !cfg.evidence.AllPRsMergedOrClosed {
			return fmt.Errorf("transition to %s: AllPRsMergedOrClosed is false: %w", StateCompleted, ErrIllegalTransition)
		}
	}

	// Review fix cycle limit check: if transitioning to fixing_review and limit reached,
	// auto-transition to failed instead.
	if r.State == StateCodeInReview && to == StateFixingReview {
		// Validate the original transition first — reject unauthorized actors
		// before checking the limit, so no side-effects occur on illegal requests.
		if err := ValidateTransition(r.State, to, actor); err != nil {
			return err
		}
		if r.ReviewFixCount >= r.ReviewFixLimit {
			if err := r.doTransition(StateFailed, actor, "review fix limit exceeded"); err != nil {
				return err
			}
			return ErrReviewFixLimitExceeded
		}
	}

	return r.doTransition(to, actor, cfg.reason)
}

func (r *Run) doTransition(to RunState, actor Actor, reason string) error {
	if err := ValidateTransition(r.State, to, actor); err != nil {
		return err
	}

	from := r.State

	// Auto-set failure reason when transitioning from verified to failed.
	if from == StateVerified && to == StateFailed {
		reason = FailureReasonPostVerificationCleanup
	}

	// Count review fix cycles.
	if from == StateCodeInReview && to == StateFixingReview {
		r.ReviewFixCount++
	}

	// Record and apply.
	r.History = append(r.History, TransitionRecord{
		From:      from,
		To:        to,
		Actor:     actor,
		Reason:    reason,
		Timestamp: time.Now(),
	})
	r.State = to

	// Set failure reason on the Run when entering failed state.
	if to == StateFailed && reason != "" {
		r.FailureReason = reason
	}

	return nil
}

// String returns a human-readable representation of the run.
func (r *Run) String() string {
	return fmt.Sprintf("Run{id=%s, state=%s, fixes=%d/%d}", r.ID, r.State, r.ReviewFixCount, r.ReviewFixLimit)
}
