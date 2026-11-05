package assistant

import (
	"context"
	"errors"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	maxAttempts    = 4
	baseRetryDelay = time.Second
	maxRetryDelay  = 30 * time.Second
)

// EmptyResponseError marks a completed stream that produced no text or tools.
type EmptyResponseError struct{}

func (e *EmptyResponseError) Error() string {
	return "ассистент вернул пустой ответ"
}

// NoAvailableKeyError marks a key pool with no usable credentials.
type NoAvailableKeyError struct{}

func (e *NoAvailableKeyError) Error() string {
	return "Нет доступных Google-ключей. Проверьте ключи в настройках."
}

// PartialResponseError marks a stream that failed after emitting text.
type PartialResponseError struct{ Err error }

func (e *PartialResponseError) Error() string {
	return "ответ оборвался: " + e.Err.Error()
}
func (e *PartialResponseError) Unwrap() error { return e.Err }

// RetryExhaustedError wraps the last transient error and the number of calls.
type RetryExhaustedError struct {
	Err      error
	Attempts int
}

func (e *RetryExhaustedError) Error() string { return e.Err.Error() }
func (e *RetryExhaustedError) Unwrap() error { return e.Err }

// IsRateLimitError reports whether err is a provider rate-limit or quota rejection.
func IsRateLimitError(err error) bool {
	var api *APIError
	if !errors.As(err, &api) {
		return false
	}
	if api.Status == http.StatusTooManyRequests {
		return true
	}
	lower := strings.ToLower(api.Body)
	return strings.Contains(lower, "resource_exhausted") ||
		strings.Contains(lower, "rate limit") ||
		strings.Contains(lower, "quota") ||
		strings.Contains(lower, "too many requests")
}

// IsHighDemandError reports Gemini model capacity errors.
func IsHighDemandError(err error) bool {
	var api *APIError
	if !errors.As(err, &api) {
		return false
	}
	if api.Status == http.StatusServiceUnavailable {
		return true
	}
	lower := strings.ToLower(api.Body)
	return strings.Contains(lower, "high demand") ||
		strings.Contains(lower, "overloaded") ||
		strings.Contains(lower, "temporarily unavailable")
}

// IsModelNotFoundError reports an unavailable model or resource.
func IsModelNotFoundError(err error) bool {
	var api *APIError
	if !errors.As(err, &api) {
		return false
	}
	if api.Status == http.StatusNotFound {
		return true
	}
	lower := strings.ToLower(api.Body)
	return strings.Contains(lower, "model not found") ||
		strings.Contains(lower, "not_found") ||
		strings.Contains(lower, "is not found")
}

// IsBillingError reports depleted credits or unavailable free tier.
func IsBillingError(err error) bool {
	var api *APIError
	if !errors.As(err, &api) {
		return false
	}
	if api.Status == http.StatusPaymentRequired {
		return true
	}
	lower := strings.ToLower(api.Body)
	return strings.Contains(lower, "billing") ||
		strings.Contains(lower, "prepay") ||
		strings.Contains(lower, "credit") ||
		strings.Contains(lower, "free tier") ||
		strings.Contains(lower, "location is not supported") ||
		strings.Contains(lower, "not available in your country") ||
		strings.Contains(lower, "enable billing")
}

// IsPermissionError reports missing access to the model or project.
func IsPermissionError(err error) bool {
	var api *APIError
	if !errors.As(err, &api) {
		return false
	}
	if api.Status == http.StatusForbidden {
		return true
	}
	lower := strings.ToLower(api.Body)
	return strings.Contains(lower, "permission_denied") ||
		strings.Contains(lower, "permission denied") ||
		strings.Contains(lower, "forbidden")
}

// IsTimeoutError reports request or network timeouts.
func IsTimeoutError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

// IsNetworkError reports transport failures before a response is received.
func IsNetworkError(err error) bool {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr)
}

// IsTransientError reports errors worth retrying without changing the request.
func IsTransientError(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}
	lower := strings.ToLower(err.Error())
	if IsLegacySessionIDError(err) || strings.Contains(lower, "tls: failed to verify certificate") || strings.Contains(lower, "x509:") {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var empty *EmptyResponseError
	if errors.As(err, &empty) {
		return true
	}
	var api *APIError
	if errors.As(err, &api) {
		switch api.Status {
		case http.StatusRequestTimeout, http.StatusTooManyRequests,
			http.StatusInternalServerError, http.StatusBadGateway,
			http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			return true
		}
		return IsRateLimitError(err)
	}
	return IsNetworkError(err) || IsTimeoutError(err)
}

// ParseRetryAfter parses seconds and HTTP-date forms of the Retry-After header.
func ParseRetryAfter(value string, now time.Time) (time.Duration, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds < 0 {
			return 0, false
		}
		return time.Duration(seconds) * time.Second, true
	}
	if at, err := http.ParseTime(value); err == nil {
		delay := at.Sub(now)
		if delay < 0 {
			delay = 0
		}
		return delay, true
	}
	return 0, false
}

// RetryDelay returns exponential backoff with jitter, honoring Retry-After.
func RetryDelay(err error, attempt int) time.Duration {
	var api *APIError
	if errors.As(err, &api) && api.RetryAfter != "" {
		if delay, ok := ParseRetryAfter(api.RetryAfter, time.Now()); ok {
			if delay > maxRetryDelay {
				delay = maxRetryDelay
			}
			return delay
		}
	}
	if attempt < 0 {
		attempt = 0
	}
	if attempt > 5 {
		attempt = 5
	}
	delay := baseRetryDelay << attempt
	if delay > maxRetryDelay {
		delay = maxRetryDelay
	}
	delay += time.Duration(rand.IntN(500)) * time.Millisecond
	if delay > maxRetryDelay {
		delay = maxRetryDelay
	}
	return delay
}

// SleepContext waits for delay or until ctx is cancelled.
func SleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// RetryReason returns a short human-readable reason for UI retry status.
func RetryReason(err error) string {
	var empty *EmptyResponseError
	switch {
	case errors.As(err, &empty):
		return "Пустой ответ Gemini"
	case IsRateLimitError(err):
		return "Лимит Gemini"
	case IsHighDemandError(err):
		return "Модель Gemini перегружена"
	case IsTimeoutError(err):
		return "Таймаут Gemini"
	case IsNetworkError(err):
		return "Нет связи с Gemini"
	default:
		return "Временная ошибка Gemini"
	}
}
