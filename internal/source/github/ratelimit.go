package github

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/google/go-github/v75/github"
)

// maxAttempts bounds how many times withRetry will call fn before giving up.
const maxAttempts = 4

// withRetry calls fn, retrying on GitHub rate-limit responses and
// transient (network / 5xx) errors with backoff. It gives up immediately
// on non-transient API errors (404, 401, 422, ...) since retrying those
// can't succeed. Honors ctx cancellation while waiting.
func withRetry(ctx context.Context, fn func() (*github.Response, error)) (*github.Response, error) {
	var lastResp *github.Response
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		resp, err := fn()
		lastResp, lastErr = resp, err
		if err == nil {
			return resp, nil
		}

		wait, retryable := backoffFor(err, attempt)
		if !retryable || attempt == maxAttempts-1 {
			return resp, err
		}
		select {
		case <-ctx.Done():
			return resp, ctx.Err()
		case <-time.After(wait):
		}
	}
	return lastResp, lastErr
}

// backoffFor classifies err and, if it's worth retrying, returns how long
// to wait before the next attempt.
func backoffFor(err error, attempt int) (time.Duration, bool) {
	var rateErr *github.RateLimitError
	if errors.As(err, &rateErr) {
		if d := time.Until(rateErr.Rate.Reset.Time); d > 0 {
			return d, true
		}
		return time.Second, true
	}

	var abuseErr *github.AbuseRateLimitError
	if errors.As(err, &abuseErr) {
		if abuseErr.RetryAfter != nil {
			return *abuseErr.RetryAfter, true
		}
		return expBackoff(attempt), true
	}

	var apiErr *github.ErrorResponse
	if errors.As(err, &apiErr) {
		if apiErr.Response != nil && apiErr.Response.StatusCode >= 500 {
			return expBackoff(attempt), true
		}
		// A well-formed 4xx API error (404, 401, 422, ...) is not
		// transient; retrying it can't succeed.
		return 0, false
	}

	// Not a recognized GitHub API error shape: likely a network-level
	// error (timeout, DNS, connection reset). Worth a bounded retry.
	return expBackoff(attempt), true
}

func expBackoff(attempt int) time.Duration {
	base := time.Second * time.Duration(int64(1)<<uint(attempt)) // 1s, 2s, 4s, 8s
	jitter := time.Duration(rand.Int63n(int64(base)/2 + 1))      //nolint:gosec // G404: retry-backoff jitter, not security-sensitive
	return base + jitter
}
