package tasksource_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/lihui/auto-dev/internal/tasksource"
)

// TestE2EFullPipeline exercises the full MemoryProvider -> Poller -> FilterConfig
// pipeline, verifying that only tasks matching the filter reach the callback and
// that deduplication prevents the same tasks from arriving twice.
func TestE2EFullPipeline(t *testing.T) {
	now := time.Now()

	provider := &tasksource.MemoryProvider{
		Tasks: []tasksource.NormalizedTask{
			{
				ExternalID: "t1",
				Title:      "Good task 1",
				Labels:     []string{"auto-dev", "bug"},
				Metadata:   map[string]string{"state": "open"},
				UpdatedAt:  now,
			},
			{
				ExternalID: "t2",
				Title:      "Excluded by label",
				Labels:     []string{"auto-dev", "wontfix"},
				Metadata:   map[string]string{"state": "open"},
				UpdatedAt:  now,
			},
			{
				ExternalID: "t3",
				Title:      "Good task 2",
				Labels:     []string{"auto-dev", "enhancement"},
				Metadata:   map[string]string{"state": "open"},
				UpdatedAt:  now,
			},
			{
				ExternalID: "t4",
				Title:      "Missing required label",
				Labels:     []string{"enhancement"},
				Metadata:   map[string]string{"state": "open"},
				UpdatedAt:  now,
			},
			{
				ExternalID: "t5",
				Title:      "Wrong state",
				Labels:     []string{"auto-dev"},
				Metadata:   map[string]string{"state": "closed"},
				UpdatedAt:  now,
			},
		},
	}

	filter := tasksource.FilterConfig{
		IncludeLabels: []string{"auto-dev"},
		ExcludeLabels: []string{"wontfix"},
		States:        []string{"open"},
	}

	var mu sync.Mutex
	var collected []tasksource.NormalizedTask

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	poller := tasksource.NewPoller(tasksource.PollerConfig{
		Interval: 10 * time.Millisecond,
		Provider: provider,
		Filter:   filter,
		OnNewTasks: func(tasks []tasksource.NormalizedTask) {
			mu.Lock()
			defer mu.Unlock()
			collected = append(collected, tasks...)
		},
	})

	done := make(chan error, 1)
	go func() {
		done <- poller.Run(ctx)
	}()

	// Wait for the first callback to deliver filtered tasks.
	deadline := time.After(2 * time.Second)
	for {
		mu.Lock()
		n := len(collected)
		mu.Unlock()
		if n >= 2 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for filtered tasks to arrive")
		case <-time.After(5 * time.Millisecond):
		}
	}

	// Let several more poll cycles pass to verify dedup prevents re-delivery.
	time.Sleep(100 * time.Millisecond)

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	// Verify only t1 and t3 passed the filter (t2 excluded by label,
	// t4 missing required label, t5 wrong state).
	ids := map[string]bool{}
	for _, task := range collected {
		ids[task.ExternalID] = true
	}

	if !ids["t1"] {
		t.Error("expected task t1 (Good task 1) to pass filter, but it was missing")
	}
	if !ids["t3"] {
		t.Error("expected task t3 (Good task 2) to pass filter, but it was missing")
	}
	if ids["t2"] {
		t.Error("task t2 should have been excluded by ExcludeLabels (wontfix)")
	}
	if ids["t4"] {
		t.Error("task t4 should have been excluded for missing IncludeLabel (auto-dev)")
	}
	if ids["t5"] {
		t.Error("task t5 should have been excluded by state filter (closed vs open)")
	}

	// Verify dedup: exactly 2 tasks should have been collected, not more.
	if len(collected) != 2 {
		t.Errorf("expected exactly 2 tasks (dedup should prevent re-delivery), got %d", len(collected))
	}
}

// TestE2ETransientErrorRecovery verifies that the Poller automatically recovers
// after the provider returns transient (non-fatal) errors. It starts the
// provider with an error, waits for several failed fetch attempts, then clears
// the error and confirms tasks eventually arrive. Run must return nil on
// context cancellation.
func TestE2ETransientErrorRecovery(t *testing.T) {
	now := time.Now()

	provider := &tasksource.MemoryProvider{
		FetchError: fmt.Errorf("simulated transient network error"),
	}

	var mu sync.Mutex
	var collected []tasksource.NormalizedTask

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	poller := tasksource.NewPoller(tasksource.PollerConfig{
		Interval: 10 * time.Millisecond,
		Provider: provider,
		OnNewTasks: func(tasks []tasksource.NormalizedTask) {
			mu.Lock()
			defer mu.Unlock()
			collected = append(collected, tasks...)
		},
	})

	done := make(chan error, 1)
	go func() {
		done <- poller.Run(ctx)
	}()

	// Wait until the provider has been called at least 3 times (all failures).
	deadline := time.After(2 * time.Second)
	for {
		provider.Mu.Lock()
		fc := provider.FetchCount
		provider.Mu.Unlock()
		if fc >= 3 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for multiple failed fetch attempts")
		case <-time.After(5 * time.Millisecond):
		}
	}

	// Clear the error and provide tasks.
	provider.Mu.Lock()
	provider.FetchError = nil
	provider.Tasks = []tasksource.NormalizedTask{
		{ExternalID: "r1", Title: "Recovered task 1", UpdatedAt: now},
		{ExternalID: "r2", Title: "Recovered task 2", UpdatedAt: now},
	}
	provider.Mu.Unlock()

	// Wait for the tasks to arrive via the callback.
	deadline = time.After(2 * time.Second)
	for {
		mu.Lock()
		n := len(collected)
		mu.Unlock()
		if n >= 2 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for tasks after transient error recovery")
		case <-time.After(5 * time.Millisecond):
		}
	}

	cancel()
	err := <-done

	// Run must return nil on context cancellation (transient errors are not fatal).
	if err != nil {
		t.Fatalf("expected Run to return nil on context cancel, got: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(collected) < 2 {
		t.Fatalf("expected at least 2 tasks after recovery, got %d", len(collected))
	}

	ids := map[string]bool{}
	for _, task := range collected {
		ids[task.ExternalID] = true
	}
	if !ids["r1"] || !ids["r2"] {
		t.Fatalf("expected tasks r1 and r2 to be delivered, got IDs: %v", ids)
	}
}

// TestE2ERestartReYield simulates a process restart by creating two sequential
// Poller instances against the same MemoryProvider. Because each Poller has a
// fresh seen map, the second instance must re-yield all tasks even though the
// first instance already delivered them.
func TestE2ERestartReYield(t *testing.T) {
	now := time.Now()

	provider := &tasksource.MemoryProvider{
		Tasks: []tasksource.NormalizedTask{
			{ExternalID: "s1", Title: "Stable task 1", UpdatedAt: now},
			{ExternalID: "s2", Title: "Stable task 2", UpdatedAt: now},
			{ExternalID: "s3", Title: "Stable task 3", UpdatedAt: now},
		},
	}

	// --- Poller #1 ---
	var mu1 sync.Mutex
	var collected1 []tasksource.NormalizedTask

	ctx1, cancel1 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel1()

	poller1 := tasksource.NewPoller(tasksource.PollerConfig{
		Interval: 10 * time.Millisecond,
		Provider: provider,
		OnNewTasks: func(tasks []tasksource.NormalizedTask) {
			mu1.Lock()
			defer mu1.Unlock()
			collected1 = append(collected1, tasks...)
		},
	})

	done1 := make(chan error, 1)
	go func() {
		done1 <- poller1.Run(ctx1)
	}()

	// Wait for Poller #1 to deliver all 3 tasks.
	deadline := time.After(2 * time.Second)
	for {
		mu1.Lock()
		n := len(collected1)
		mu1.Unlock()
		if n >= 3 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for Poller #1 to deliver tasks")
		case <-time.After(5 * time.Millisecond):
		}
	}

	cancel1()
	if err := <-done1; err != nil {
		t.Fatalf("Poller #1 Run returned unexpected error: %v", err)
	}

	// Verify Poller #1 collected the expected tasks.
	mu1.Lock()
	ids1 := map[string]bool{}
	for _, task := range collected1 {
		ids1[task.ExternalID] = true
	}
	mu1.Unlock()

	if !ids1["s1"] || !ids1["s2"] || !ids1["s3"] {
		t.Fatalf("Poller #1 did not collect all tasks: got %v", ids1)
	}

	// --- Poller #2 (simulates restart with fresh seen map) ---
	var mu2 sync.Mutex
	var collected2 []tasksource.NormalizedTask

	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()

	poller2 := tasksource.NewPoller(tasksource.PollerConfig{
		Interval: 10 * time.Millisecond,
		Provider: provider,
		OnNewTasks: func(tasks []tasksource.NormalizedTask) {
			mu2.Lock()
			defer mu2.Unlock()
			collected2 = append(collected2, tasks...)
		},
	})

	done2 := make(chan error, 1)
	go func() {
		done2 <- poller2.Run(ctx2)
	}()

	// Wait for Poller #2 to deliver all 3 tasks.
	deadline = time.After(2 * time.Second)
	for {
		mu2.Lock()
		n := len(collected2)
		mu2.Unlock()
		if n >= 3 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for Poller #2 to re-yield tasks")
		case <-time.After(5 * time.Millisecond):
		}
	}

	// Let a few more cycles pass to confirm dedup works within this instance.
	time.Sleep(100 * time.Millisecond)

	cancel2()
	if err := <-done2; err != nil {
		t.Fatalf("Poller #2 Run returned unexpected error: %v", err)
	}

	mu2.Lock()
	defer mu2.Unlock()

	// Verify Poller #2 re-yielded all tasks.
	ids2 := map[string]bool{}
	for _, task := range collected2 {
		ids2[task.ExternalID] = true
	}

	if !ids2["s1"] || !ids2["s2"] || !ids2["s3"] {
		t.Fatalf("Poller #2 did not re-yield all tasks: got %v", ids2)
	}

	// Verify dedup within Poller #2: exactly 3 tasks, not more.
	if len(collected2) != 3 {
		t.Errorf("expected exactly 3 tasks from Poller #2 (dedup within instance), got %d", len(collected2))
	}
}
