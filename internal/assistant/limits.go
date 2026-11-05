package assistant

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

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
		strings.Contains(lower, "missing or invalid authorization header") ||
		strings.Contains(lower, "unauthenticated")
}

// FriendlyError turns provider and transport errors into user-facing messages.
func FriendlyError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.Canceled) {
		return "Запрос отменён."
	}
	var partial *PartialResponseError
	if errors.As(err, &partial) {
		return "Ответ оборвался. Повторите запрос."
	}
	var empty *EmptyResponseError
	if errors.As(err, &empty) {
		return "Gemini вернул пустой ответ. Попробуйте ещё раз."
	}
	var noKey *NoAvailableKeyError
	if errors.As(err, &noKey) {
		return noKey.Error()
	}

	attempts := 0
	var exhausted *RetryExhaustedError
	if errors.As(err, &exhausted) {
		attempts = exhausted.Attempts
	}
	var api *APIError
	if errors.As(err, &api) {
		return friendlyAPIError(api, attempts)
	}
	if IsTimeoutError(err) {
		return "Gemini не ответил вовремя." + retrySuffix(attempts) + " Повторите запрос или выберите более быструю модель."
	}
	if IsNetworkError(err) {
		return "Не удалось связаться с Gemini или прокси." + retrySuffix(attempts) + " Проверьте интернет и повторите запрос."
	}
	return "Не удалось выполнить запрос: " + truncateText(strings.TrimSpace(err.Error()), 240)
}

func friendlyAPIError(api *APIError, attempts int) string {
	lower := strings.ToLower(api.Body)
	suffix := retrySuffix(attempts)

	switch {
	case strings.Contains(lower, "thought_signature"):
		return "История tool-call повреждена: Gemini требует thought_signature. Очистите историю и повторите запрос."
	case strings.Contains(lower, "value at top-level must be a list"):
		return "Gemini отклонил схему запроса. Обновите приложение; если ошибка повторяется, сообщите о ней."
	case IsInvalidKeyError(api):
		return "Ключ Google недействителен или нет доступа. Проверьте ключи в настройках."
	case IsBillingError(api):
		return "Закончились кредиты Google AI Studio или для проекта не включён биллинг. Проверьте баланс и настройки проекта."
	case IsModelNotFoundError(api):
		return "Модель Gemini недоступна для этого ключа. Выберите другую модель в настройках."
	case IsRateLimitError(api):
		if strings.Contains(lower, "daily") || strings.Contains(lower, "per day") || strings.Contains(lower, "quota_exceeded") {
			return "Дневная квота Gemini исчерпана." + suffix + " Подождите сброса квоты или добавьте ключ из другого проекта."
		}
		return "Лимит запросов Gemini исчерпан." + suffix + " Подождите немного или добавьте ключ из другого проекта."
	case api.Status == http.StatusPaymentRequired:
		return "Закончились кредиты Google AI Studio. Пополните баланс."
	case api.Status == http.StatusForbidden || IsPermissionError(api):
		return "Нет доступа к модели или проекту Gemini. Проверьте права ключа и настройки проекта."
	case api.Status == http.StatusRequestTimeout || api.Status == http.StatusGatewayTimeout:
		return "Gemini не ответил вовремя." + suffix + " Повторите запрос или выберите более быструю модель."
	case api.Status == 499:
		return "Запрос отменён."
	case api.Status == http.StatusInternalServerError:
		return "Внутренняя ошибка Gemini." + suffix + " Попробуйте позже."
	case api.Status == http.StatusBadGateway || api.Status == http.StatusServiceUnavailable:
		if strings.Contains(lower, "high demand") || strings.Contains(lower, "overloaded") || strings.Contains(lower, "temporarily unavailable") {
			return "Модель Gemini перегружена из-за высокого спроса." + suffix + " Попробуйте позже или выберите другую модель."
		}
		return "Сервис Gemini или прокси временно недоступен." + suffix + " Попробуйте позже."
	case api.Status == http.StatusNotImplemented:
		return "Эта функция не поддерживается выбранной моделью Gemini."
	case api.Status == http.StatusConflict:
		return "Конфликт запроса Gemini. Повторите запрос."
	case api.Status == http.StatusBadRequest:
		return "Gemini отклонил запрос. Проверьте выбранную модель и формулу. Детали ниже."
	}

	body := truncateText(strings.TrimSpace(api.Body), 240)
	if body == "" {
		body = http.StatusText(api.Status)
	}
	return fmt.Sprintf("Ошибка Gemini (HTTP %d): %s", api.Status, body)
}

func retrySuffix(attempts int) string {
	if attempts <= 1 {
		return ""
	}
	return " Выполнено попыток: " + strconv.Itoa(attempts) + "."
}

func truncateText(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
