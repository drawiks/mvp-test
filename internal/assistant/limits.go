package assistant

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
)

// IsRateLimitError reports whether err is a provider rate-limit rejection.
func IsRateLimitError(err error) bool {
	var api *APIError
	return errors.As(err, &api) && api.Status == http.StatusTooManyRequests
}

// IsInvalidKeyError reports Google credential failures.
func IsInvalidKeyError(err error) bool {
	var api *APIError
	if !errors.As(err, &api) {
		return false
	}
	if api.Status == http.StatusUnauthorized {
		return true
	}
	lower := strings.ToLower(api.Body)
	return strings.Contains(lower, "api key not valid") ||
		strings.Contains(lower, "api_key_invalid") ||
		strings.Contains(lower, "please pass a valid api key") ||
		strings.Contains(lower, "missing or invalid authorization header")
}

// FriendlyError turns Google and proxy errors into user-facing messages.
func FriendlyError(err error) string {
	var api *APIError
	if !errors.As(err, &api) {
		return err.Error()
	}
	if IsRateLimitError(err) {
		return "Лимит Google AI Studio исчерпан на текущем ключе. Если добавлены другие ключи, приложение переключится автоматически."
	}
	if IsInvalidKeyError(err) {
		return "Ключ Google недействителен. Проверьте ключи в настройках."
	}
	if api.Status >= http.StatusInternalServerError {
		return "Ошибка прокси Google. Попробуйте позже."
	}
	return "Ошибка Gemini (HTTP " + strconv.Itoa(api.Status) + "): " + truncateText(strings.TrimSpace(api.Body), 240)
}

func truncateText(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
