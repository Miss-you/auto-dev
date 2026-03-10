package runlifecycle_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/lihui/auto-dev/internal/runlifecycle"
)

// ---------------------------------------------------------------------------
// Unit Tests
// ---------------------------------------------------------------------------

func TestNewRunDefaults(t *testing.T) {
	r := runlifecycle.NewRun("run-1")

	if r.ID != "run-1" {
		t.Fatalf("expected ID %q, got %q", "run-1", r.ID)
	}
	if r.State != runlifecycle.StateSpecDrafting {
		t.Fatalf("expected initial state %q, got %q", runlifecycle.StateSpecDrafting, r.State)
	}
	if len(r.History) != 1 {
		t.Fatalf("expected 1 initial history record, got %d records", len(r.History))
	}
	if r.History[0].From != "" {
		t.Fatalf("expected initial record From %q, got %q", "", r.History[0].From)
	}
	if r.History[0].To != runlifecycle.StateSpecDrafting {
		t.Fatalf("expected initial record To %q, got %q", runlifecycle.StateSpecDrafting, r.History[0].To)
	}
	if r.History[0].Actor != runlifecycle.ActorDispatcher {
		t.Fatalf("expected initial record Actor %q, got %q", runlifecycle.ActorDispatcher, r.History[0].Actor)
	}
	if r.History[0].Reason != "run created" {
		t.Fatalf("expected initial record Reason %q, got %q", "run created", r.History[0].Reason)
	}
	if r.ReviewFixLimit != 3 {
		t.Fatalf("expected ReviewFixLimit 3, got %d", r.ReviewFixLimit)
	}
	if r.ReviewFixCount != 0 {
		t.Fatalf("expected ReviewFixCount 0, got %d", r.ReviewFixCount)
	}
	if r.FailureReason != "" {
		t.Fatalf("expected empty FailureReason, got %q", r.FailureReason)
	}
}

func TestNewRunWithOptions(t *testing.T) {
	r := runlifecycle.NewRun("run-2", runlifecycle.WithReviewFixLimit(5))

	if r.ReviewFixLimit != 5 {
		t.Fatalf("expected ReviewFixLimit 5, got %d", r.ReviewFixLimit)
	}
	// Other defaults should still hold.
	if r.State != runlifecycle.StateSpecDrafting {
		t.Fatalf("expected initial state %q, got %q", runlifecycle.StateSpecDrafting, r.State)
	}
}

func TestTransitionRecordsHistory(t *testing.T) {
	r := runlifecycle.NewRun("run-3")

	if err := r.Transition(runlifecycle.StateSpecInReview, runlifecycle.ActorSupervisor); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(r.History) != 2 {
		t.Fatalf("expected 2 history records (1 initial + 1 transition), got %d", len(r.History))
	}

	rec := r.History[1]
	if rec.From != runlifecycle.StateSpecDrafting {
		t.Errorf("expected From %q, got %q", runlifecycle.StateSpecDrafting, rec.From)
	}
	if rec.To != runlifecycle.StateSpecInReview {
		t.Errorf("expected To %q, got %q", runlifecycle.StateSpecInReview, rec.To)
	}
	if rec.Actor != runlifecycle.ActorSupervisor {
		t.Errorf("expected Actor %q, got %q", runlifecycle.ActorSupervisor, rec.Actor)
	}
	if rec.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

// ---------------------------------------------------------------------------
// E2E / Integration Tests (full lifecycle paths)
// ---------------------------------------------------------------------------

// helperTransition is a small helper that calls Transition and fails the test
// on error. It returns the run for chaining convenience.
func helperTransition(t *testing.T, r *runlifecycle.Run, to runlifecycle.RunState, actor runlifecycle.Actor, opts ...runlifecycle.TransitionOption) {
	t.Helper()
	if err := r.Transition(to, actor, opts...); err != nil {
		t.Fatalf("transition to %s by %s failed: %v", to, actor, err)
	}
}

func TestHappyPathFullLifecycle(t *testing.T) {
	r := runlifecycle.NewRun("happy-1")

	ev := runlifecycle.WithCompletedEvidence(runlifecycle.CompletedEvidence{
		VerifyPassed:         true,
		SpecsSynced:          true,
		ChangeArchived:       true,
		AllPRsMergedOrClosed: true,
	})

	steps := []struct {
		to    runlifecycle.RunState
		actor runlifecycle.Actor
		opts  []runlifecycle.TransitionOption
	}{
		{runlifecycle.StateSpecInReview, runlifecycle.ActorSupervisor, nil},
		{runlifecycle.StateSpecApproved, runlifecycle.ActorSupervisor, nil},
		{runlifecycle.StateImplQueued, runlifecycle.ActorDispatcher, nil},
		{runlifecycle.StateImplementing, runlifecycle.ActorSupervisor, nil},
		{runlifecycle.StateCodeInReview, runlifecycle.ActorSupervisor, nil},
		{runlifecycle.StateVerified, runlifecycle.ActorSupervisor, nil},
		{runlifecycle.StateCompleted, runlifecycle.ActorDispatcher, []runlifecycle.TransitionOption{ev}},
	}

	for _, step := range steps {
		helperTransition(t, r, step.to, step.actor, step.opts...)
	}

	if r.State != runlifecycle.StateCompleted {
		t.Fatalf("expected final state %q, got %q", runlifecycle.StateCompleted, r.State)
	}
	// 1 initial record + 7 transition records = 8
	if len(r.History) != len(steps)+1 {
		t.Fatalf("expected %d history records, got %d", len(steps)+1, len(r.History))
	}
}

func TestHappyPathWithReviewFixCycle(t *testing.T) {
	r := runlifecycle.NewRun("fix-cycle-1")

	// Advance to code_in_review.
	helperTransition(t, r, runlifecycle.StateSpecInReview, runlifecycle.ActorSupervisor)
	helperTransition(t, r, runlifecycle.StateSpecApproved, runlifecycle.ActorSupervisor)
	helperTransition(t, r, runlifecycle.StateImplQueued, runlifecycle.ActorDispatcher)
	helperTransition(t, r, runlifecycle.StateImplementing, runlifecycle.ActorSupervisor)
	helperTransition(t, r, runlifecycle.StateCodeInReview, runlifecycle.ActorSupervisor)

	// One review fix cycle.
	helperTransition(t, r, runlifecycle.StateFixingReview, runlifecycle.ActorSupervisor)
	helperTransition(t, r, runlifecycle.StateCodeInReview, runlifecycle.ActorSupervisor)

	// Now verified and completed.
	helperTransition(t, r, runlifecycle.StateVerified, runlifecycle.ActorSupervisor)
	helperTransition(t, r, runlifecycle.StateCompleted, runlifecycle.ActorDispatcher, runlifecycle.WithCompletedEvidence(runlifecycle.CompletedEvidence{
		VerifyPassed:         true,
		SpecsSynced:          true,
		ChangeArchived:       true,
		AllPRsMergedOrClosed: true,
	}))

	if r.State != runlifecycle.StateCompleted {
		t.Fatalf("expected final state %q, got %q", runlifecycle.StateCompleted, r.State)
	}
	if r.ReviewFixCount != 1 {
		t.Fatalf("expected ReviewFixCount 1, got %d", r.ReviewFixCount)
	}
}

func TestFailureReasonAutoAnnotation(t *testing.T) {
	t.Run("verified_to_failed_auto_sets_reason", func(t *testing.T) {
		r := runlifecycle.NewRun("fail-auto-1")

		// Advance to verified.
		helperTransition(t, r, runlifecycle.StateSpecInReview, runlifecycle.ActorSupervisor)
		helperTransition(t, r, runlifecycle.StateSpecApproved, runlifecycle.ActorSupervisor)
		helperTransition(t, r, runlifecycle.StateImplQueued, runlifecycle.ActorDispatcher)
		helperTransition(t, r, runlifecycle.StateImplementing, runlifecycle.ActorSupervisor)
		helperTransition(t, r, runlifecycle.StateCodeInReview, runlifecycle.ActorSupervisor)
		helperTransition(t, r, runlifecycle.StateVerified, runlifecycle.ActorSupervisor)

		// Transition to failed without providing a reason.
		helperTransition(t, r, runlifecycle.StateFailed, runlifecycle.ActorSupervisor)

		expected := runlifecycle.FailureReasonPostVerificationCleanup
		if r.FailureReason != expected {
			t.Fatalf("expected FailureReason %q, got %q", expected, r.FailureReason)
		}

		// Also verify the history record carries the auto-annotated reason.
		lastRec := r.History[len(r.History)-1]
		if lastRec.Reason != expected {
			t.Fatalf("expected history record reason %q, got %q", expected, lastRec.Reason)
		}
	})

	t.Run("implementing_to_failed_does_not_auto_set_reason", func(t *testing.T) {
		r := runlifecycle.NewRun("fail-no-auto-1")

		// Advance to implementing.
		helperTransition(t, r, runlifecycle.StateSpecInReview, runlifecycle.ActorSupervisor)
		helperTransition(t, r, runlifecycle.StateSpecApproved, runlifecycle.ActorSupervisor)
		helperTransition(t, r, runlifecycle.StateImplQueued, runlifecycle.ActorDispatcher)
		helperTransition(t, r, runlifecycle.StateImplementing, runlifecycle.ActorSupervisor)

		// Transition to failed without providing a reason.
		helperTransition(t, r, runlifecycle.StateFailed, runlifecycle.ActorSupervisor)

		if r.FailureReason != "" {
			t.Fatalf("expected empty FailureReason, got %q", r.FailureReason)
		}
	})
}

func TestReviewFixLimitExceeded(t *testing.T) {
	r := runlifecycle.NewRun("limit-1", runlifecycle.WithReviewFixLimit(2))

	// Advance to code_in_review.
	helperTransition(t, r, runlifecycle.StateSpecInReview, runlifecycle.ActorSupervisor)
	helperTransition(t, r, runlifecycle.StateSpecApproved, runlifecycle.ActorSupervisor)
	helperTransition(t, r, runlifecycle.StateImplQueued, runlifecycle.ActorDispatcher)
	helperTransition(t, r, runlifecycle.StateImplementing, runlifecycle.ActorSupervisor)
	helperTransition(t, r, runlifecycle.StateCodeInReview, runlifecycle.ActorSupervisor)

	// Fix cycle 1: code_in_review → fixing_review → code_in_review.
	helperTransition(t, r, runlifecycle.StateFixingReview, runlifecycle.ActorSupervisor)
	helperTransition(t, r, runlifecycle.StateCodeInReview, runlifecycle.ActorSupervisor)

	// Fix cycle 2: code_in_review → fixing_review → code_in_review.
	helperTransition(t, r, runlifecycle.StateFixingReview, runlifecycle.ActorSupervisor)
	helperTransition(t, r, runlifecycle.StateCodeInReview, runlifecycle.ActorSupervisor)

	// 3rd attempt to enter fixing_review should auto-fail and return ErrReviewFixLimitExceeded.
	err := r.Transition(runlifecycle.StateFixingReview, runlifecycle.ActorSupervisor)
	if !errors.Is(err, runlifecycle.ErrReviewFixLimitExceeded) {
		t.Fatalf("expected ErrReviewFixLimitExceeded, got %v", err)
	}

	if r.State != runlifecycle.StateFailed {
		t.Fatalf("expected state %q after limit exceeded, got %q", runlifecycle.StateFailed, r.State)
	}
	if r.FailureReason != "review fix limit exceeded" {
		t.Fatalf("expected FailureReason %q, got %q", "review fix limit exceeded", r.FailureReason)
	}
	if r.ReviewFixCount != 2 {
		t.Fatalf("expected ReviewFixCount 2 (not incremented on auto-fail), got %d", r.ReviewFixCount)
	}
}

func TestTerminalStateRejectsTransition(t *testing.T) {
	terminalCases := []struct {
		name   string
		target runlifecycle.RunState
		actor  runlifecycle.Actor
	}{
		{"completed", runlifecycle.StateCompleted, runlifecycle.ActorDispatcher},
		{"failed", runlifecycle.StateFailed, runlifecycle.ActorSupervisor},
		{"aborted", runlifecycle.StateAborted, runlifecycle.ActorOperator},
	}

	for _, tc := range terminalCases {
		t.Run(tc.name, func(t *testing.T) {
			r := runlifecycle.NewRun("term-" + tc.name)

			// Build a run that reaches the target terminal state.
			switch tc.target {
			case runlifecycle.StateCompleted:
				helperTransition(t, r, runlifecycle.StateSpecInReview, runlifecycle.ActorSupervisor)
				helperTransition(t, r, runlifecycle.StateSpecApproved, runlifecycle.ActorSupervisor)
				helperTransition(t, r, runlifecycle.StateImplQueued, runlifecycle.ActorDispatcher)
				helperTransition(t, r, runlifecycle.StateImplementing, runlifecycle.ActorSupervisor)
				helperTransition(t, r, runlifecycle.StateCodeInReview, runlifecycle.ActorSupervisor)
				helperTransition(t, r, runlifecycle.StateVerified, runlifecycle.ActorSupervisor)
				helperTransition(t, r, runlifecycle.StateCompleted, runlifecycle.ActorDispatcher, runlifecycle.WithCompletedEvidence(runlifecycle.CompletedEvidence{
					VerifyPassed:         true,
					SpecsSynced:          true,
					ChangeArchived:       true,
					AllPRsMergedOrClosed: true,
				}))
			case runlifecycle.StateFailed:
				helperTransition(t, r, runlifecycle.StateFailed, runlifecycle.ActorSupervisor)
			case runlifecycle.StateAborted:
				helperTransition(t, r, runlifecycle.StateAborted, runlifecycle.ActorOperator)
			}

			// Now try to transition out of the terminal state.
			err := r.Transition(runlifecycle.StateSpecDrafting, runlifecycle.ActorSupervisor)
			if err == nil {
				t.Fatal("expected error when transitioning from terminal state, got nil")
			}
			if !errors.Is(err, runlifecycle.ErrTerminalState) {
				t.Fatalf("expected ErrTerminalState, got %v", err)
			}
		})
	}
}

func TestAbortFromAnyState(t *testing.T) {
	r := runlifecycle.NewRun("abort-1")

	// Advance to implementing.
	helperTransition(t, r, runlifecycle.StateSpecInReview, runlifecycle.ActorSupervisor)
	helperTransition(t, r, runlifecycle.StateSpecApproved, runlifecycle.ActorSupervisor)
	helperTransition(t, r, runlifecycle.StateImplQueued, runlifecycle.ActorDispatcher)
	helperTransition(t, r, runlifecycle.StateImplementing, runlifecycle.ActorSupervisor)

	// Abort via operator.
	helperTransition(t, r, runlifecycle.StateAborted, runlifecycle.ActorOperator)

	if r.State != runlifecycle.StateAborted {
		t.Fatalf("expected state %q, got %q", runlifecycle.StateAborted, r.State)
	}
}

func TestJSONRoundTrip(t *testing.T) {
	r := runlifecycle.NewRun("json-1", runlifecycle.WithReviewFixLimit(4))

	// Advance through several states, including a fix cycle and failure.
	helperTransition(t, r, runlifecycle.StateSpecInReview, runlifecycle.ActorSupervisor)
	helperTransition(t, r, runlifecycle.StateSpecApproved, runlifecycle.ActorSupervisor)
	helperTransition(t, r, runlifecycle.StateImplQueued, runlifecycle.ActorDispatcher)
	helperTransition(t, r, runlifecycle.StateImplementing, runlifecycle.ActorSupervisor)
	helperTransition(t, r, runlifecycle.StateCodeInReview, runlifecycle.ActorSupervisor)
	helperTransition(t, r, runlifecycle.StateFixingReview, runlifecycle.ActorSupervisor)
	helperTransition(t, r, runlifecycle.StateCodeInReview, runlifecycle.ActorSupervisor)
	helperTransition(t, r, runlifecycle.StateVerified, runlifecycle.ActorSupervisor)
	helperTransition(t, r, runlifecycle.StateFailed, runlifecycle.ActorSupervisor)

	// Marshal to JSON.
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	// Unmarshal into a new Run.
	var restored runlifecycle.Run
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	// Compare fields (excluding exact timestamp comparison).
	if restored.ID != r.ID {
		t.Errorf("ID mismatch: got %q, want %q", restored.ID, r.ID)
	}
	if restored.State != r.State {
		t.Errorf("State mismatch: got %q, want %q", restored.State, r.State)
	}
	if len(restored.History) != len(r.History) {
		t.Fatalf("History length mismatch: got %d, want %d", len(restored.History), len(r.History))
	}
	if restored.ReviewFixCount != r.ReviewFixCount {
		t.Errorf("ReviewFixCount mismatch: got %d, want %d", restored.ReviewFixCount, r.ReviewFixCount)
	}
	if restored.ReviewFixLimit != r.ReviewFixLimit {
		t.Errorf("ReviewFixLimit mismatch: got %d, want %d", restored.ReviewFixLimit, r.ReviewFixLimit)
	}
	if restored.FailureReason != r.FailureReason {
		t.Errorf("FailureReason mismatch: got %q, want %q", restored.FailureReason, r.FailureReason)
	}

	// Verify history records match (From, To, Actor, Reason); use time tolerance.
	for i, orig := range r.History {
		got := restored.History[i]
		if got.From != orig.From {
			t.Errorf("History[%d].From: got %q, want %q", i, got.From, orig.From)
		}
		if got.To != orig.To {
			t.Errorf("History[%d].To: got %q, want %q", i, got.To, orig.To)
		}
		if got.Actor != orig.Actor {
			t.Errorf("History[%d].Actor: got %q, want %q", i, got.Actor, orig.Actor)
		}
		if got.Reason != orig.Reason {
			t.Errorf("History[%d].Reason: got %q, want %q", i, got.Reason, orig.Reason)
		}
		// Allow up to 1 second of timestamp drift from JSON round-trip (nanosecond truncation).
		diff := got.Timestamp.Sub(orig.Timestamp)
		if diff < 0 {
			diff = -diff
		}
		if diff > 1_000_000_000 { // 1 second
			t.Errorf("History[%d].Timestamp drift too large: %v", i, diff)
		}
	}
}

func TestWithReason(t *testing.T) {
	r := runlifecycle.NewRun("reason-1")
	err := r.Transition(runlifecycle.StateSpecInReview, runlifecycle.ActorSupervisor, runlifecycle.WithReason("spec PR #42 created"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lastRec := r.History[len(r.History)-1]
	if lastRec.Reason != "spec PR #42 created" {
		t.Fatalf("expected reason %q, got %q", "spec PR #42 created", lastRec.Reason)
	}
}

func TestVerifiedToFailedWithExplicitReason(t *testing.T) {
	r := runlifecycle.NewRun("explicit-reason-1")
	helperTransition(t, r, runlifecycle.StateSpecInReview, runlifecycle.ActorSupervisor)
	helperTransition(t, r, runlifecycle.StateSpecApproved, runlifecycle.ActorSupervisor)
	helperTransition(t, r, runlifecycle.StateImplQueued, runlifecycle.ActorDispatcher)
	helperTransition(t, r, runlifecycle.StateImplementing, runlifecycle.ActorSupervisor)
	helperTransition(t, r, runlifecycle.StateCodeInReview, runlifecycle.ActorSupervisor)
	helperTransition(t, r, runlifecycle.StateVerified, runlifecycle.ActorSupervisor)

	// Even with an explicit reason, verified->failed MUST use the mandatory reason.
	err := r.Transition(runlifecycle.StateFailed, runlifecycle.ActorSupervisor, runlifecycle.WithReason("archive service unavailable"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.FailureReason != runlifecycle.FailureReasonPostVerificationCleanup {
		t.Fatalf("expected FailureReason %q (mandatory), got %q", runlifecycle.FailureReasonPostVerificationCleanup, r.FailureReason)
	}
	// History record should also carry the mandatory reason.
	lastRec := r.History[len(r.History)-1]
	if lastRec.Reason != runlifecycle.FailureReasonPostVerificationCleanup {
		t.Fatalf("expected history reason %q, got %q", runlifecycle.FailureReasonPostVerificationCleanup, lastRec.Reason)
	}
}

func TestRunString(t *testing.T) {
	r := runlifecycle.NewRun("str-1", runlifecycle.WithReviewFixLimit(5))
	want := "Run{id=str-1, state=spec_drafting, fixes=0/5}"
	if got := r.String(); got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestCompletedEvidenceValidation(t *testing.T) {
	// Helper: create a run in verified state.
	makeVerified := func(t *testing.T, id string) *runlifecycle.Run {
		t.Helper()
		r := runlifecycle.NewRun(id)
		helperTransition(t, r, runlifecycle.StateSpecInReview, runlifecycle.ActorSupervisor)
		helperTransition(t, r, runlifecycle.StateSpecApproved, runlifecycle.ActorSupervisor)
		helperTransition(t, r, runlifecycle.StateImplQueued, runlifecycle.ActorDispatcher)
		helperTransition(t, r, runlifecycle.StateImplementing, runlifecycle.ActorSupervisor)
		helperTransition(t, r, runlifecycle.StateCodeInReview, runlifecycle.ActorSupervisor)
		helperTransition(t, r, runlifecycle.StateVerified, runlifecycle.ActorSupervisor)
		return r
	}

	t.Run("nil_evidence", func(t *testing.T) {
		r := makeVerified(t, "ev-nil")
		err := r.Transition(runlifecycle.StateCompleted, runlifecycle.ActorDispatcher)
		if !errors.Is(err, runlifecycle.ErrIllegalTransition) {
			t.Fatalf("expected ErrIllegalTransition, got: %v", err)
		}
		if r.State != runlifecycle.StateVerified {
			t.Fatalf("state should remain verified, got %q", r.State)
		}
	})

	t.Run("VerifyPassed_false", func(t *testing.T) {
		r := makeVerified(t, "ev-vp")
		err := r.Transition(runlifecycle.StateCompleted, runlifecycle.ActorDispatcher, runlifecycle.WithCompletedEvidence(runlifecycle.CompletedEvidence{
			VerifyPassed: false, SpecsSynced: true, ChangeArchived: true, AllPRsMergedOrClosed: true,
		}))
		if !errors.Is(err, runlifecycle.ErrIllegalTransition) {
			t.Fatalf("expected ErrIllegalTransition, got: %v", err)
		}
		if r.State != runlifecycle.StateVerified {
			t.Fatalf("state should remain verified, got %q", r.State)
		}
	})

	t.Run("SpecsSynced_false", func(t *testing.T) {
		r := makeVerified(t, "ev-ss")
		err := r.Transition(runlifecycle.StateCompleted, runlifecycle.ActorDispatcher, runlifecycle.WithCompletedEvidence(runlifecycle.CompletedEvidence{
			VerifyPassed: true, SpecsSynced: false, ChangeArchived: true, AllPRsMergedOrClosed: true,
		}))
		if !errors.Is(err, runlifecycle.ErrIllegalTransition) {
			t.Fatalf("expected ErrIllegalTransition, got: %v", err)
		}
		if r.State != runlifecycle.StateVerified {
			t.Fatalf("state should remain verified, got %q", r.State)
		}
	})

	t.Run("ChangeArchived_false", func(t *testing.T) {
		r := makeVerified(t, "ev-ca")
		err := r.Transition(runlifecycle.StateCompleted, runlifecycle.ActorDispatcher, runlifecycle.WithCompletedEvidence(runlifecycle.CompletedEvidence{
			VerifyPassed: true, SpecsSynced: true, ChangeArchived: false, AllPRsMergedOrClosed: true,
		}))
		if !errors.Is(err, runlifecycle.ErrIllegalTransition) {
			t.Fatalf("expected ErrIllegalTransition, got: %v", err)
		}
		if r.State != runlifecycle.StateVerified {
			t.Fatalf("state should remain verified, got %q", r.State)
		}
	})

	t.Run("AllPRsMergedOrClosed_false", func(t *testing.T) {
		r := makeVerified(t, "ev-pr")
		err := r.Transition(runlifecycle.StateCompleted, runlifecycle.ActorDispatcher, runlifecycle.WithCompletedEvidence(runlifecycle.CompletedEvidence{
			VerifyPassed: true, SpecsSynced: true, ChangeArchived: true, AllPRsMergedOrClosed: false,
		}))
		if !errors.Is(err, runlifecycle.ErrIllegalTransition) {
			t.Fatalf("expected ErrIllegalTransition, got: %v", err)
		}
		if r.State != runlifecycle.StateVerified {
			t.Fatalf("state should remain verified, got %q", r.State)
		}
	})
}

func TestReviewFixLimitUnauthorizedActorDoesNotAutoFail(t *testing.T) {
	r := runlifecycle.NewRun("limit-unauth-1", runlifecycle.WithReviewFixLimit(1))

	// Advance to code_in_review.
	helperTransition(t, r, runlifecycle.StateSpecInReview, runlifecycle.ActorSupervisor)
	helperTransition(t, r, runlifecycle.StateSpecApproved, runlifecycle.ActorSupervisor)
	helperTransition(t, r, runlifecycle.StateImplQueued, runlifecycle.ActorDispatcher)
	helperTransition(t, r, runlifecycle.StateImplementing, runlifecycle.ActorSupervisor)
	helperTransition(t, r, runlifecycle.StateCodeInReview, runlifecycle.ActorSupervisor)

	// Use up the limit.
	helperTransition(t, r, runlifecycle.StateFixingReview, runlifecycle.ActorSupervisor)
	helperTransition(t, r, runlifecycle.StateCodeInReview, runlifecycle.ActorSupervisor)

	// Dispatcher is NOT authorized for code_in_review → fixing_review.
	// Even though the limit is reached, the unauthorized actor should be rejected
	// WITHOUT auto-transitioning to failed.
	err := r.Transition(runlifecycle.StateFixingReview, runlifecycle.ActorDispatcher)
	if err == nil {
		t.Fatal("expected error for unauthorized actor, got nil")
	}
	if !errors.Is(err, runlifecycle.ErrIllegalTransition) {
		t.Fatalf("expected ErrIllegalTransition, got: %v", err)
	}
	// State must still be code_in_review, NOT failed.
	if r.State != runlifecycle.StateCodeInReview {
		t.Fatalf("expected state %q (unchanged), got %q", runlifecycle.StateCodeInReview, r.State)
	}
}
