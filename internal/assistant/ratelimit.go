package assistant

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	groqDefaultRPM = 30
	maxRateWait    = 2 * time.Minute
	maxRateRetries = 3
)

// RateLimiter paces Groq requests using the limits returned by the API.
type RateLimiter struct {
	mu          sync.Mutex
	state       RateLimit
	lastRequest time.Time
	minInterval time.Duration
	now         func() time.Time
	sleep       func(context.Context, time.Duration) error
}

func newRateLimiter(provider string) *RateLimiter {
	if provider != ProviderGroq {
		return nil
	}
	return &RateLimiter{
		minInterval: time.Minute / groqDefaultRPM,
		now:         time.Now,
		sleep:       sleepContext,
	}
}

func (l *RateLimiter) Update(rate *RateLimit) {
	if l == nil || rate == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.state = *rate
}

// Wait blocks until the next request fits the current limits.
func (l *RateLimiter) Wait(ctx context.Context, estimatedTokens int, emit func(Event)) error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	now := l.now()
	var wait time.Duration
	if !l.lastRequest.IsZero() && now.Sub(l.lastRequest) < l.minInterval {
		wait = l.minInterval - now.Sub(l.lastRequest)
	}
	reason := "requests"
	if l.state.RequestsLimit > 0 && l.state.RequestsRemaining <= 0 {
		if d := resetDuration(l.state.RequestsReset); d > wait {
			wait = d
		}
	}
	if l.state.TokensLimit > 0 && estimatedTokens > l.state.TokensRemaining {
		if d := resetDuration(l.state.TokensReset); d > wait {
			wait = d
			reason = "tokens"
		}
	}
	l.mu.Unlock()

	if wait <= 0 {
		l.markRequest()
		return nil
	}
	if wait > maxRateWait {
		return fmt.Errorf("лимит Groq: следующий запрос возможен через %s", wait.Round(time.Second))
	}
	if emit != nil {
		emit(Event{Type: "rateWait", WaitSeconds: int(math.Ceil(wait.Seconds())), WaitReason: reason})
	}
	if err := l.sleep(ctx, wait); err != nil {
		return err
	}
	l.markRequest()
	return nil
}

// RetryDelay decides whether a failed request can be retried automatically.
func (l *RateLimiter) RetryDelay(err error, rate *RateLimit, emitted bool) (time.Duration, bool) {
	if l == nil || emitted {
		return 0, false
	}
	var api *APIError
	if !errors.As(err, &api) || api.Status != http.StatusTooManyRequests {
		return 0, false
	}
	delay := retryAfterDuration(api.RetryAfter)
	if rate != nil {
		l.Update(rate)
		if d := resetDuration(rate.TokensReset); d > delay {
			delay = d
		}
		if rate.RequestsLimit > 0 && rate.RequestsRemaining <= 0 {
			if d := resetDuration(rate.RequestsReset); d > delay {
				delay = d
			}
		}
	}
	if delay <= 0 {
		delay = l.minInterval
	}
	if delay > maxRateWait {
		return delay, false
	}
	return delay, true
}

func (l *RateLimiter) WaitForRetry(ctx context.Context, delay time.Duration, emit func(Event)) error {
	if emit != nil {
		emit(Event{Type: "rateWait", WaitSeconds: int(math.Ceil(delay.Seconds())), WaitReason: "rate"})
	}
	if err := l.sleep(ctx, delay); err != nil {
		return err
	}
	l.markRequest()
	return nil
}

func (l *RateLimiter) markRequest() {
	l.mu.Lock()
	l.lastRequest = l.now()
	l.mu.Unlock()
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func resetDuration(value string) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if d, err := time.ParseDuration(value); err == nil {
		return d
	}
	if seconds, err := strconv.ParseFloat(value, 64); err == nil {
		return time.Duration(seconds * float64(time.Second))
	}
	return 0
}

func retryAfterDuration(value string) time.Duration {
	return resetDuration(value)
}
