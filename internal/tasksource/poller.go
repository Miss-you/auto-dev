package tasksource

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

// DefaultPollInterval is the default interval between poll cycles.
const DefaultPollInterval = 30 * time.Second

// DefaultSeenMapResetCycles is the number of poll cycles after which the seen map is cleared.
const DefaultSeenMapResetCycles = 100

// PollerConfig configures the Poller.
type PollerConfig struct {
	// Interval between poll cycles. Default: 30s.
	Interval time.Duration

	// Provider is the task source to poll.
	Provider Provider

	// Filter is applied to tasks after fetching. Zero value means no filtering.
	Filter FilterConfig

	// OnNewTasks is called synchronously with new/updated tasks on each poll cycle.
	// It is never called after Run returns.
	OnNewTasks func([]NormalizedTask)

	// SeenMapResetCycles is the number of cycles after which the seen map is cleared.
	// Default: 100. Values <= 0 are treated as default.
	SeenMapResetCycles int
}

// Poller periodically fetches tasks from a Provider, filters and deduplicates them,
// and delivers new tasks via the OnNewTasks callback.
//
// Concurrency invariant: the seen map is only accessed from the Run goroutine.
// The OnNewTasks callback is invoked synchronously, so no mutex is needed.
type Poller struct {
	interval           time.Duration
	provider           Provider
	filter             FilterConfig
	onNewTasks         func([]NormalizedTask)
	seenMapResetCycles int
	seen               map[string]time.Time // ExternalID → UpdatedAt
	cycleCount         int
}

// NewPoller creates a new Poller with the given configuration.
// It panics if cfg.Provider is nil.
func NewPoller(cfg PollerConfig) *Poller {
	if cfg.Provider == nil {
		panic("tasksource: NewPoller requires a non-nil Provider")
	}
	interval := cfg.Interval
	if interval <= 0 {
		interval = DefaultPollInterval
	}
	resetCycles := cfg.SeenMapResetCycles
	if resetCycles <= 0 {
		resetCycles = DefaultSeenMapResetCycles
	}

	return &Poller{
		interval:           interval,
		provider:           cfg.Provider,
		filter:             cfg.Filter,
		onNewTasks:         cfg.OnNewTasks,
		seenMapResetCycles: resetCycles,
		seen:               make(map[string]time.Time),
	}
}

// Run starts the polling loop. It blocks until ctx is cancelled or a fatal error occurs.
// On normal context cancellation, Run returns nil.
// On ErrAuthFailure from the provider, Run returns the error.
func (p *Poller) Run(ctx context.Context) error {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	// Do an immediate first poll before waiting for the first tick.
	if err := p.handlePollError(ctx, p.poll(ctx)); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := p.handlePollError(ctx, p.poll(ctx)); err != nil {
				return err
			}
		}
	}
}

func (p *Poller) poll(ctx context.Context) error {
	// Check context before polling.
	if ctx.Err() != nil {
		return nil
	}

	tasks, err := p.provider.FetchCandidateTasks(ctx)
	if err != nil {
		return err
	}

	// Apply filter.
	tasks = p.filter.Apply(tasks)

	// Dedup: only yield new or updated tasks.
	var newTasks []NormalizedTask
	for _, t := range tasks {
		lastSeen, exists := p.seen[t.ExternalID]
		if !exists || !t.UpdatedAt.Equal(lastSeen) {
			newTasks = append(newTasks, t)
			p.seen[t.ExternalID] = t.UpdatedAt
		}
	}

	// Deliver new tasks via callback (only if there are any and ctx is still active).
	if len(newTasks) > 0 && ctx.Err() == nil && p.onNewTasks != nil {
		p.onNewTasks(newTasks)
	}

	// Periodic seen map reset.
	p.cycleCount++
	if p.seenMapResetCycles > 0 && p.cycleCount >= p.seenMapResetCycles {
		p.seen = make(map[string]time.Time)
		p.cycleCount = 0
		slog.Debug("seen map reset", "after_cycles", p.seenMapResetCycles)
	}

	return nil
}

// handlePollError processes a poll error. Returns non-nil only for fatal errors
// (ErrAuthFailure). Rate limit errors cause backoff; context cancellation and
// transient errors are handled gracefully.
func (p *Poller) handlePollError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	// Context cancellation is a normal shutdown path.
	if ctx.Err() != nil {
		return nil
	}

	if errors.Is(err, ErrAuthFailure) {
		return err
	}

	var rateLimitErr *RateLimitError
	if errors.As(err, &rateLimitErr) {
		waitDuration := time.Until(rateLimitErr.RetryAfter)
		if waitDuration > 0 {
			slog.Warn("rate limited, backing off", "wait", waitDuration)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(waitDuration):
			}
		}
		return nil
	}

	// Transient error: log and retry on next tick.
	slog.Warn("poll error, will retry", "error", err)
	return nil
}
