package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	tpmLimitRe     = regexp.MustCompile(`(?i)limit\s+(\d+)`)
	tpmRequestedRe = regexp.MustCompile(`(?i)requested\s+(\d+)`)
	retryAfterRe   = regexp.MustCompile(`(?i)retry after\s+(\d+)`)
)

func parseRate(h http.Header, provider string) *RateLimit {
	r := &RateLimit{Provider: provider, ReceivedAt: time.Now(), RetryAfter: h.Get("Retry-After")}
	setInt := func(dst *int, key string) {
		if v := h.Get(key); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				*dst = n
			}
		}
	}
	if provider == ProviderOpenRouter {
		setInt(&r.RequestsLimit, "X-RateLimit-Limit")
		setInt(&r.RequestsRemaining, "X-RateLimit-Remaining")
		r.RequestsReset = h.Get("X-RateLimit-Reset")
	} else {
		setInt(&r.RequestsLimit, "x-ratelimit-limit-requests")
		setInt(&r.RequestsRemaining, "x-ratelimit-remaining-requests")
		r.RequestsReset = h.Get("x-ratelimit-reset-requests")
		setInt(&r.TokensLimit, "x-ratelimit-limit-tokens")
		setInt(&r.TokensRemaining, "x-ratelimit-remaining-tokens")
		r.TokensReset = h.Get("x-ratelimit-reset-tokens")
	}
	if r.RequestsLimit == 0 && r.RequestsRemaining == 0 && r.TokensLimit == 0 && r.TokensRemaining == 0 && r.RetryAfter == "" {
		return nil
	}
	return r
}

// FetchOpenRouterLimits reads free-model and credit limits for the current key.
func FetchOpenRouterLimits(ctx context.Context, apiKey string) (*RateLimit, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, openRouterBaseURL+"/key", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("User-Agent", "mvp-test-assistant")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openrouter key: %s", resp.Status)
	}
	var payload struct {
		Data struct {
			LimitRemaining *float64 `json:"limit_remaining"`
			FreeRequests   struct {
				Used      int `json:"used"`
				Limit     int `json:"limit"`
				Remaining int `json:"remaining"`
			} `json:"free_model_daily_requests"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return &RateLimit{
		Provider:              ProviderOpenRouter,
		FreeRequestsLimit:     payload.Data.FreeRequests.Limit,
		FreeRequestsRemaining: payload.Data.FreeRequests.Remaining,
		FreeRequestsReset:     "00:00 UTC",
		CreditsRemaining:      payload.Data.LimitRemaining,
		ReceivedAt:            time.Now(),
	}, nil
}

// FriendlyError turns provider errors into messages suitable for the UI.
func FriendlyError(err error) string {
	var api *APIError
	if !errors.As(err, &api) {
		return err.Error()
	}
	lower := strings.ToLower(api.Body)
	if strings.Contains(lower, "request too large") ||
		strings.Contains(lower, "input tokens per minute") ||
		strings.Contains(lower, "tokens per minute") ||
		strings.Contains(lower, "itpm") || strings.Contains(lower, "tpm") {
		limit, requested, retry := tpmNumbers(api.Body)
		msg := "Запрос слишком большой для Groq"
		if limit > 0 && requested > 0 {
			msg += fmt.Sprintf(": %d токенов при лимите %d в минуту", requested, limit)
		} else if limit > 0 {
			msg += fmt.Sprintf(": лимит %d токенов в минуту", limit)
		}
		msg += ". Сократите историю"
		if retry > 0 {
			msg += fmt.Sprintf(" или повторите через %d сек", retry)
		} else if api.RetryAfter != "" {
			msg += " или повторите через " + api.RetryAfter
		}
		return msg + "."
	}
	if strings.Contains(lower, "context length") || strings.Contains(lower, "maximum context") {
		return "Запрос превышает контекст модели. Очистите историю или начните новый диалог."
	}
	switch api.Status {
	case http.StatusUnauthorized:
		return "Неверный API-ключ. Проверьте ключ провайдера."
	case http.StatusForbidden:
		return "Доступ запрещён: " + api.Body
	case http.StatusPaymentRequired:
		return "Недостаточно кредитов на аккаунте провайдера."
	case http.StatusTooManyRequests:
		if api.Rate != nil {
			if api.Rate.Provider == ProviderOpenRouter {
				if api.Rate.FreeRequestsLimit > 0 {
					return fmt.Sprintf("Дневной лимит free-моделей OpenRouter исчерпан (%d/%d). Сброс: %s.",
						api.Rate.FreeRequestsLimit-api.Rate.FreeRequestsRemaining, api.Rate.FreeRequestsLimit, api.Rate.FreeRequestsReset)
				}
				if api.Rate.RequestsLimit > 0 {
					return fmt.Sprintf("Лимит OpenRouter исчерпан (%d/%d). Сброс: %s.",
						api.Rate.RequestsLimit-api.Rate.RequestsRemaining, api.Rate.RequestsLimit, api.Rate.RequestsReset)
				}
			} else {
				if api.Rate.RequestsLimit > 0 && api.Rate.RequestsRemaining == 0 {
					return fmt.Sprintf("Дневной лимит Groq исчерпан (%d/%d). Сброс: %s.",
						api.Rate.RequestsLimit-api.Rate.RequestsRemaining, api.Rate.RequestsLimit, api.Rate.RequestsReset)
				}
				if api.Rate.TokensLimit > 0 && api.Rate.TokensRemaining == 0 {
					return fmt.Sprintf("Лимит токенов Groq исчерпан (%d/%d). Сброс: %s.",
						api.Rate.TokensLimit-api.Rate.TokensRemaining, api.Rate.TokensLimit, api.Rate.TokensReset)
				}
			}
		}
		if api.RetryAfter != "" {
			return "Лимит запросов исчерпан. Повторите через " + api.RetryAfter + "."
		}
		return "Лимит запросов исчерпан. " + strings.TrimSpace(api.Body)
	}
	return "Ошибка провайдера (HTTP " + strconv.Itoa(api.Status) + "): " + truncateText(strings.TrimSpace(api.Body), 240)
}

func tpmNumbers(body string) (limit, requested, retry int) {
	if m := tpmLimitRe.FindStringSubmatch(body); len(m) == 2 {
		limit, _ = strconv.Atoi(m[1])
	}
	if m := tpmRequestedRe.FindStringSubmatch(body); len(m) == 2 {
		requested, _ = strconv.Atoi(m[1])
	}
	if m := retryAfterRe.FindStringSubmatch(body); len(m) == 2 {
		retry, _ = strconv.Atoi(m[1])
	}
	return
}

func truncateText(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// FriendlyErrorFor adds provider-specific connection and model guidance.
func FriendlyErrorFor(err error, provider string) string {
	var api *APIError
	if errors.As(err, &api) {
		if provider == ProviderGoogle {
			if api.Status == http.StatusTooManyRequests {
				return "Лимит Google AI Studio исчерпан на текущем ключе. Если добавлены другие ключи, приложение переключится автоматически."
			}
			if IsInvalidKeyError(err) {
				return "Ключ Google недействителен. Проверьте ключи в настройках."
			}
			if api.Status >= 500 {
				return "Ошибка прокси Google. Попробуйте позже."
			}
		}
		if api.Status == http.StatusNotFound && IsLocal(provider) {
			return "Модель не найдена на локальном сервере. Обновите список моделей или загрузите её."
		}
		return FriendlyError(err)
	}
	lower := strings.ToLower(err.Error())
	connection := strings.Contains(lower, "connection refused") ||
		strings.Contains(lower, "connect: connection refused") ||
		strings.Contains(lower, "no such host") ||
		strings.Contains(lower, "dial tcp") ||
		strings.Contains(lower, "i/o timeout") ||
		strings.Contains(lower, "connection reset")
	if connection {
		switch provider {
		case ProviderOllama:
			return "Ollama не отвечает. Запустите `ollama serve` или проверьте URL в настройках."
		case ProviderLlamaCpp:
			return "llama-server не отвечает. Запустите его с `--jinja` и проверьте URL/порт (обычно 9931)."
		}
	}
	return err.Error()
}

// IsRateLimitError reports whether err is a provider rate-limit rejection.
func IsRateLimitError(err error) bool {
	var api *APIError
	return errors.As(err, &api) && api.Status == http.StatusTooManyRequests
}

// IsInvalidKeyError reports credential failures, including Gemini's 400 body.
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
