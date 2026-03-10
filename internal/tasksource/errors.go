package tasksource

import (
	"errors"
	"fmt"
	"time"
)

var (
	// ErrAuthFailure indicates the provider rejected authentication (401/403).
	// The poller should stop polling when this error is returned.
	ErrAuthFailure = errors.New("authentication failure")
)

// RateLimitError indicates the provider's rate limit has been exceeded.
// RetryAfter contains the absolute time when requests can resume.
type RateLimitError struct {
	RetryAfter time.Time
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limit exceeded, retry after %s", e.RetryAfter.Format(time.RFC3339))
}
