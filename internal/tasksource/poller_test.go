package tasksource_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/lihui/auto-dev/internal/tasksource"
)

func makeTasks(ids ...string) []tasksource.NormalizedTask {
	now := time.Now()
	tasks := make([]tasksource.NormalizedTask, len(ids))
	for i, id := range ids {
		tasks[i] = tasksource.NormalizedTask{
			ExternalID: id,
			Title:      "Task " + id,
			UpdatedAt:  now,
		}
	}
	return tasks
}

func TestPollerNormalPolling(t *testing.T) {
	provider := &tasksource.MemoryProvider{
		Tasks: makeTasks("1", "2"),
	}

	var mu sync.Mutex
	var collected []tasksource.NormalizedTask

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
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

	// Wait for at least one callback to fire.
	deadline := time.After(1 * time.Second)
	for {
		mu.Lock()
		n := len(collected)
		mu.Unlock()
		if n >= 2 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for tasks to be collected")
		case <-time.After(5 * time.Millisecond):
		}
	}

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(collected) < 2 {
		t.Fatalf("expected at least 2 tasks, got %d", len(collected))
	}

	ids := map[string]bool{}
	for _, task := range collected {
		ids[task.ExternalID] = true
	}
	if !ids["1"] || !ids["2"] {
		t.Fatalf("expected tasks 1 and 2, got %v", ids)
	}
}

func TestPollerDedup(t *testing.T) {
	provider := &tasksource.MemoryProvider{
		Tasks: makeTasks("1", "2"),
	}

	var mu sync.Mutex
	callbackCount := 0
	var firstBatchSize int

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	poller := tasksource.NewPoller(tasksource.PollerConfig{
		Interval: 10 * time.Millisecond,
		Provider: provider,
		OnNewTasks: func(tasks []tasksource.NormalizedTask) {
			mu.Lock()
			defer mu.Unlock()
			callbackCount++
			if callbackCount == 1 {
				firstBatchSize = len(tasks)
			}
		},
	})

	done := make(chan error, 1)
	go func() {
		done <- poller.Run(ctx)
	}()

	// Wait for the first callback, then let a few more poll cycles pass.
	deadline := time.After(1 * time.Second)
	for {
		mu.Lock()
		n := callbackCount
		mu.Unlock()
		if n >= 1 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for first callback")
		case <-time.After(5 * time.Millisecond):
		}
	}

	// Let several more poll cycles complete.
	time.Sleep(100 * time.Millisecond)

	cancel()
	<-done

	mu.Lock()
	defer mu.Unlock()

	if firstBatchSize != 2 {
		t.Fatalf("expected first batch to have 2 tasks, got %d", firstBatchSize)
	}
	// After the first callback, dedup should prevent subsequent callbacks.
	// Only the first call should have fired (subsequent polls yield 0 new tasks).
	if callbackCount != 1 {
		t.Fatalf("expected exactly 1 callback (dedup should suppress repeats), got %d", callbackCount)
	}
}

func TestPollerUpdatedAtChange(t *testing.T) {
	now := time.Now()
	provider := &tasksource.MemoryProvider{
		Tasks: []tasksource.NormalizedTask{
			{ExternalID: "A", Title: "Task A", UpdatedAt: now},
		},
	}

	var mu sync.Mutex
	var batches [][]tasksource.NormalizedTask

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	poller := tasksource.NewPoller(tasksource.PollerConfig{
		Interval: 20 * time.Millisecond,
		Provider: provider,
		OnNewTasks: func(tasks []tasksource.NormalizedTask) {
			mu.Lock()
			defer mu.Unlock()
			cp := make([]tasksource.NormalizedTask, len(tasks))
			copy(cp, tasks)
			batches = append(batches, cp)
		},
	})

	done := make(chan error, 1)
	go func() {
		done <- poller.Run(ctx)
	}()

	// Wait for first callback.
	deadline := time.After(1 * time.Second)
	for {
		mu.Lock()
		n := len(batches)
		mu.Unlock()
		if n >= 1 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for first callback")
		case <-time.After(5 * time.Millisecond):
		}
	}

	// Update the task's UpdatedAt to trigger re-yield.
	provider.Mu.Lock()
	provider.Tasks[0].UpdatedAt = now.Add(1 * time.Minute)
	provider.Mu.Unlock()

	// Wait for second callback.
	deadline = time.After(1 * time.Second)
	for {
		mu.Lock()
		n := len(batches)
		mu.Unlock()
		if n >= 2 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for second callback after UpdatedAt change")
		case <-time.After(5 * time.Millisecond):
		}
	}

	cancel()
	<-done

	mu.Lock()
	defer mu.Unlock()

	if len(batches) < 2 {
		t.Fatalf("expected at least 2 batches, got %d", len(batches))
	}
	if batches[0][0].ExternalID != "A" || batches[1][0].ExternalID != "A" {
		t.Fatal("expected task A in both batches")
	}
}

func TestPollerSeenMapReset(t *testing.T) {
	now := time.Now()
	provider := &tasksource.MemoryProvider{
		Tasks: []tasksource.NormalizedTask{
			{ExternalID: "A", Title: "Task A", UpdatedAt: now},
		},
	}

	var mu sync.Mutex
	var callbackCount int

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	poller := tasksource.NewPoller(tasksource.PollerConfig{
		Interval:           10 * time.Millisecond,
		Provider:           provider,
		SeenMapResetCycles: 3,
		OnNewTasks: func(tasks []tasksource.NormalizedTask) {
			mu.Lock()
			defer mu.Unlock()
			callbackCount++
		},
	})

	done := make(chan error, 1)
	go func() {
		done <- poller.Run(ctx)
	}()

	// Wait for at least 2 callbacks: first poll + after reset at cycle 3.
	deadline := time.After(1 * time.Second)
	for {
		mu.Lock()
		n := callbackCount
		mu.Unlock()
		if n >= 2 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for seen map reset to trigger re-yield")
		case <-time.After(5 * time.Millisecond):
		}
	}

	cancel()
	<-done

	mu.Lock()
	defer mu.Unlock()

	if callbackCount < 2 {
		t.Fatalf("expected at least 2 callbacks (initial + after reset), got %d", callbackCount)
	}
}

func TestPollerRateLimitBackoff(t *testing.T) {
	retryAfter := time.Now().Add(150 * time.Millisecond)
	now := time.Now()

	provider := &tasksource.MemoryProvider{
		FetchError: &tasksource.RateLimitError{RetryAfter: retryAfter},
	}

	var mu sync.Mutex
	var collected []tasksource.NormalizedTask

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
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

	// Wait for at least one fetch attempt (rate limited), then clear the error.
	deadline := time.After(1 * time.Second)
	for {
		provider.Mu.Lock()
		fc := provider.FetchCount
		provider.Mu.Unlock()
		if fc >= 1 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for first fetch")
		case <-time.After(5 * time.Millisecond):
		}
	}

	time.Sleep(50 * time.Millisecond)

	provider.Mu.Lock()
	fetchCountDuringBackoff := provider.FetchCount
	provider.Mu.Unlock()
	if fetchCountDuringBackoff != 1 {
		t.Fatalf("expected exactly 1 fetch during backoff window, got %d", fetchCountDuringBackoff)
	}

	// Clear the error and set tasks.
	provider.Mu.Lock()
	provider.FetchError = nil
	provider.Tasks = []tasksource.NormalizedTask{
		{ExternalID: "1", Title: "Task 1", UpdatedAt: now},
	}
	provider.Mu.Unlock()

	// Wait for tasks to be delivered.
	deadline = time.After(1 * time.Second)
	for {
		mu.Lock()
		n := len(collected)
		mu.Unlock()
		if n >= 1 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for tasks after rate limit cleared")
		case <-time.After(5 * time.Millisecond):
		}
	}

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(collected) < 1 {
		t.Fatal("expected at least 1 task after rate limit recovery")
	}
}

func TestPollerAuthFailureStops(t *testing.T) {
	provider := &tasksource.MemoryProvider{
		FetchError: tasksource.ErrAuthFailure,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	poller := tasksource.NewPoller(tasksource.PollerConfig{
		Interval: 10 * time.Millisecond,
		Provider: provider,
		OnNewTasks: func(tasks []tasksource.NormalizedTask) {
			t.Error("OnNewTasks should not be called on auth failure")
		},
	})

	err := poller.Run(ctx)
	if !errors.Is(err, tasksource.ErrAuthFailure) {
		t.Fatalf("expected ErrAuthFailure, got: %v", err)
	}
}

func TestPollerContextCancel(t *testing.T) {
	provider := &tasksource.MemoryProvider{
		Tasks: makeTasks("1"),
	}

	ctx, cancel := context.WithCancel(context.Background())

	poller := tasksource.NewPoller(tasksource.PollerConfig{
		Interval: 50 * time.Millisecond,
		Provider: provider,
		OnNewTasks: func(tasks []tasksource.NormalizedTask) {
			// First callback fires, then we cancel.
			cancel()
		},
	})

	done := make(chan error, 1)
	go func() {
		done <- poller.Run(ctx)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected nil on context cancel, got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Run to return after context cancel")
	}
}

func TestPollerTransientErrorRecovery(t *testing.T) {
	now := time.Now()
	provider := &tasksource.MemoryProvider{
		FetchError: fmt.Errorf("transient network error"),
	}

	var mu sync.Mutex
	var collected []tasksource.NormalizedTask

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
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

	// Wait for at least one failed fetch, then clear the error.
	deadline := time.After(1 * time.Second)
	for {
		provider.Mu.Lock()
		fc := provider.FetchCount
		provider.Mu.Unlock()
		if fc >= 1 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for first fetch")
		case <-time.After(5 * time.Millisecond):
		}
	}

	// Clear error and set tasks.
	provider.Mu.Lock()
	provider.FetchError = nil
	provider.Tasks = []tasksource.NormalizedTask{
		{ExternalID: "1", Title: "Task 1", UpdatedAt: now},
	}
	provider.Mu.Unlock()

	// Wait for tasks.
	deadline = time.After(1 * time.Second)
	for {
		mu.Lock()
		n := len(collected)
		mu.Unlock()
		if n >= 1 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for tasks after transient error recovery")
		case <-time.After(5 * time.Millisecond):
		}
	}

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(collected) < 1 {
		t.Fatal("expected at least 1 task after recovery")
	}
}

func TestPollerNoCallbackAfterCancel(t *testing.T) {
	now := time.Now()
	provider := &tasksource.MemoryProvider{
		Tasks: []tasksource.NormalizedTask{
			{ExternalID: "1", Title: "Task 1", UpdatedAt: now},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())

	var mu sync.Mutex
	callbackCount := 0

	poller := tasksource.NewPoller(tasksource.PollerConfig{
		Interval: 10 * time.Millisecond,
		Provider: provider,
		OnNewTasks: func(tasks []tasksource.NormalizedTask) {
			mu.Lock()
			callbackCount++
			mu.Unlock()
		},
	})

	done := make(chan error, 1)
	go func() {
		done <- poller.Run(ctx)
	}()

	// Wait for the first callback.
	deadline := time.After(1 * time.Second)
	for {
		mu.Lock()
		n := callbackCount
		mu.Unlock()
		if n >= 1 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for first callback")
		case <-time.After(5 * time.Millisecond):
		}
	}

	// Cancel the context.
	cancel()

	// Wait for Run to return.
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected nil on cancel, got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Run to return")
	}

	// Record callback count after Run exits.
	mu.Lock()
	countAfterStop := callbackCount
	mu.Unlock()

	// Wait a bit and verify no more callbacks fire.
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	finalCount := callbackCount
	mu.Unlock()

	if finalCount != countAfterStop {
		t.Fatalf("callback invoked after Run returned: count went from %d to %d", countAfterStop, finalCount)
	}
}
