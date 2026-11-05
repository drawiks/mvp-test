package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
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
	return api.Error()
}
