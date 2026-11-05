package main

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"mvp/internal/assistant"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type AssistantSettings struct {
	Provider         string  `json:"provider"`
	Model            string  `json:"model"`
	Temperature      float64 `json:"temperature"`
	MaxToolRounds    int     `json:"maxToolRounds"`
	SendPlayerNames  bool    `json:"sendPlayerNames"`
	HasGroqKey       bool    `json:"hasGroqKey"`
	HasOpenRouterKey bool    `json:"hasOpenRouterKey"`
}

type AssistantModel struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Provider      string `json:"provider"`
	ContextLength int    `json:"contextLength"`
	SupportsTools bool   `json:"supportsTools"`
	Free          bool   `json:"free"`
	Note          string `json:"note,omitempty"`
	Privacy       string `json:"privacy,omitempty"`
}

type AssistantProposal struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Expression  string `json:"expression"`
}

type AssistantLimits struct {
	Provider              string    `json:"provider"`
	RequestsLimit         int       `json:"requestsLimit"`
	RequestsRemaining     int       `json:"requestsRemaining"`
	RequestsReset         string    `json:"requestsReset"`
	TokensLimit           int       `json:"tokensLimit"`
	TokensRemaining       int       `json:"tokensRemaining"`
	TokensReset           string    `json:"tokensReset"`
	FreeRequestsLimit     int       `json:"freeRequestsLimit"`
	FreeRequestsRemaining int       `json:"freeRequestsRemaining"`
	FreeRequestsReset     string    `json:"freeRequestsReset"`
	CreditsRemaining      *float64  `json:"creditsRemaining,omitempty"`
	RetryAfter            string    `json:"retryAfter"`
	ReceivedAt            time.Time `json:"receivedAt"`
}

func toAssistantLimits(r assistant.RateLimit) AssistantLimits {
	return AssistantLimits{
		Provider: r.Provider, RequestsLimit: r.RequestsLimit, RequestsRemaining: r.RequestsRemaining,
		RequestsReset: r.RequestsReset, TokensLimit: r.TokensLimit, TokensRemaining: r.TokensRemaining,
		TokensReset: r.TokensReset, FreeRequestsLimit: r.FreeRequestsLimit,
		FreeRequestsRemaining: r.FreeRequestsRemaining, FreeRequestsReset: r.FreeRequestsReset,
		CreditsRemaining: r.CreditsRemaining, RetryAfter: r.RetryAfter, ReceivedAt: r.ReceivedAt,
	}
}

func (a *App) GetAssistantLimits() AssistantLimits {
	a.assistantMu.Lock()
	limits := a.assistantLimits
	a.assistantMu.Unlock()
	provider := a.assistantSettings.Provider
	if provider == assistant.ProviderOpenRouter && a.assistantKey(provider) != "" && a.ctx != nil {
		ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
		defer cancel()
		if fetched, err := assistant.FetchOpenRouterLimits(ctx, a.assistantKey(provider)); err == nil {
			limits = *fetched
			a.assistantMu.Lock()
			a.assistantLimits = limits
			a.assistantLimitsAt = time.Now()
			a.assistantMu.Unlock()
		}
	}
	return toAssistantLimits(limits)
}

func (a *App) assistantKey(provider string) string {
	if provider == assistant.ProviderOpenRouter {
		return a.openRouterKey
	}
	return a.groqKey
}

func (a *App) GetAssistantSettings() AssistantSettings {
	s := a.assistantSettings
	return AssistantSettings{
		Provider: s.Provider, Model: s.Model, Temperature: s.Temperature,
		MaxToolRounds: s.MaxToolRounds, SendPlayerNames: s.SendPlayerNames,
		HasGroqKey:       a.groqKey != "",
		HasOpenRouterKey: a.openRouterKey != "",
	}
}

func (a *App) SaveAssistantSettings(s AssistantSettings, groqKey, openRouterKey string) error {
	if s.Provider != assistant.ProviderGroq && s.Provider != assistant.ProviderOpenRouter {
		return errors.New("неизвестный провайдер ассистента")
	}
	if strings.TrimSpace(s.Model) == "" {
		return errors.New("выберите модель")
	}
	if s.MaxToolRounds <= 0 {
		s.MaxToolRounds = 6
	}
	if s.MaxToolRounds > 10 {
		s.MaxToolRounds = 10
	}
	if s.Temperature < 0 {
		s.Temperature = 0
	}
	if s.Temperature > 2 {
		s.Temperature = 2
	}
	a.assistantSettings = assistant.Settings{
		Provider: s.Provider, Model: strings.TrimSpace(s.Model),
		Temperature: s.Temperature, MaxToolRounds: s.MaxToolRounds,
		SendPlayerNames: s.SendPlayerNames,
	}
	if strings.TrimSpace(groqKey) != "" {
		a.groqKey = strings.TrimSpace(groqKey)
	}
	if strings.TrimSpace(openRouterKey) != "" {
		a.openRouterKey = strings.TrimSpace(openRouterKey)
	}
	return a.saveConfig()
}

func (a *App) ClearAssistantKey(provider string) error {
	switch provider {
	case assistant.ProviderGroq:
		a.groqKey = ""
	case assistant.ProviderOpenRouter:
		a.openRouterKey = ""
	default:
		return errors.New("неизвестный провайдер")
	}
	return a.saveConfig()
}

func (a *App) ListAssistantModels(provider string, refresh bool) ([]AssistantModel, error) {
	if provider == "" {
		provider = a.assistantSettings.Provider
	}
	if provider != assistant.ProviderGroq && provider != assistant.ProviderOpenRouter {
		return nil, errors.New("неизвестный провайдер ассистента")
	}
	if a.ctx == nil {
		return nil, errors.New("приложение ещё не запущено")
	}
	ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
	defer cancel()
	models, err := assistant.ListModels(ctx, provider, a.assistantKey(provider), refresh)
	if err != nil {
		return nil, err
	}
	out := make([]AssistantModel, 0, len(models))
	for _, m := range models {
		out = append(out, AssistantModel{
			ID: m.ID, Name: m.Name, Provider: m.Provider, ContextLength: m.ContextLength,
			SupportsTools: m.SupportsTools, Free: m.Free, Note: m.Note, Privacy: m.Privacy,
		})
	}
	return out, nil
}

func (a *App) TestAssistantConnection() error {
	key := a.assistantKey(a.assistantSettings.Provider)
	if key == "" {
		return errors.New("сначала укажите API-ключ")
	}
	if a.ctx == nil {
		return errors.New("приложение ещё не запущено")
	}
	ctx, cancel := context.WithTimeout(a.ctx, 60*time.Second)
	defer cancel()
	return assistant.Ping(ctx, a.assistantSettings, key)
}

func (a *App) SendAssistantMessage(text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return errors.New("введите сообщение")
	}
	if a.ctx == nil {
		return errors.New("приложение ещё не запущено")
	}

	a.assistantMu.Lock()
	if a.assistantRunning {
		a.assistantMu.Unlock()
		return errors.New("ассистент уже отвечает")
	}
	settings := a.assistantSettings
	key := a.assistantKey(settings.Provider)
	if key == "" {
		a.assistantMu.Unlock()
		return errors.New("укажите API-ключ в настройках ассистента")
	}
	history := trimHistoryByBudget(a.assistantHistory, assistantHistoryBudget(settings.Provider))
	history = append(history, assistant.Message{Role: "user", Content: text})
	ctx, cancel := context.WithCancel(a.ctx)
	generation := a.assistantGeneration
	a.assistantRunning = true
	a.assistantCancel = cancel
	a.assistantMu.Unlock()

	go func() {
		defer func() {
			cancel()
			a.assistantMu.Lock()
			if a.assistantGeneration == generation {
				a.assistantHistory = limitHistoryForStorage(history, 60)
				a.saveAssistantHistoryLocked()
			}
			a.assistantRunning = false
			a.assistantCancel = nil
			a.assistantMu.Unlock()
		}()

		emit := func(e assistant.Event) {
			switch e.Type {
			case "delta":
				runtime.EventsEmit(a.ctx, "assistantDelta", map[string]any{"text": e.Text})
			case "tool":
				runtime.EventsEmit(a.ctx, "assistantTool", map[string]any{"name": e.ToolName, "args": e.ToolArgs})
			case "toolResult":
				runtime.EventsEmit(a.ctx, "assistantToolResult", map[string]any{"name": e.ToolName, "result": e.ToolResult})
			case "rate":
				if e.Rate != nil {
					a.assistantMu.Lock()
					a.assistantLimits = *e.Rate
					a.assistantLimitsAt = time.Now()
					a.assistantMu.Unlock()
					runtime.EventsEmit(a.ctx, "assistantLimits", toAssistantLimits(*e.Rate))
				}
			}
		}
		err := assistant.Run(ctx, settings, key, &history, a.assistantTools(), emit)
		if err != nil {
			message := err.Error()
			detail := ""
			if errors.Is(err, context.Canceled) {
				message = "запрос отменён"
			} else {
				message = assistant.FriendlyError(err)
				var apiErr *assistant.APIError
				if errors.As(err, &apiErr) {
					detail = apiErr.Body
				}
			}
			runtime.EventsEmit(a.ctx, "assistantDone", map[string]any{"ok": false, "error": message, "detail": detail})
			return
		}
		if settings.Provider == assistant.ProviderOpenRouter && key != "" {
			limitCtx, limitCancel := context.WithTimeout(a.ctx, 15*time.Second)
			if limits, err := assistant.FetchOpenRouterLimits(limitCtx, key); err == nil {
				a.assistantMu.Lock()
				a.assistantLimits = *limits
				a.assistantLimitsAt = time.Now()
				a.assistantMu.Unlock()
				runtime.EventsEmit(a.ctx, "assistantLimits", toAssistantLimits(*limits))
			}
			limitCancel()
		}
		runtime.EventsEmit(a.ctx, "assistantDone", map[string]any{"ok": true})
	}()
	return nil
}

func (a *App) CancelAssistant() {
	a.assistantMu.Lock()
	defer a.assistantMu.Unlock()
	if a.assistantCancel != nil {
		a.assistantCancel()
		a.assistantCancel = nil
	}
}

func (a *App) ClearAssistantHistory() {
	a.assistantMu.Lock()
	defer a.assistantMu.Unlock()
	if a.assistantCancel != nil {
		a.assistantCancel()
		a.assistantCancel = nil
	}
	a.assistantHistory = nil
	a.assistantProposal = nil
	a.assistantGeneration++
	if a.assistantHistoryPath != "" {
		_ = os.Remove(a.assistantHistoryPath)
	}
}

func assistantHistoryBudget(provider string) int {
	if provider == assistant.ProviderOpenRouter {
		return 40000
	}
	return 18000
}

func messageChars(m assistant.Message) int {
	n := len(m.Content) + len(m.Name)
	for _, call := range m.ToolCalls {
		n += len(call.Function.Name) + len(call.Function.Arguments)
	}
	return n
}

func historyChars(history []assistant.Message) int {
	total := 0
	for _, m := range history {
		total += messageChars(m)
	}
	return total
}

func trimHistoryByBudget(history []assistant.Message, maxChars int) []assistant.Message {
	out := trimAssistantHistory(history, 60)
	if historyChars(out) <= maxChars {
		return out
	}
	for {
		first := -1
		for i, m := range out {
			if m.Role == "user" {
				first = i
				break
			}
		}
		if first < 0 {
			break
		}
		next := -1
		for i := first + 1; i < len(out); i++ {
			if out[i].Role == "user" {
				next = i
				break
			}
		}
		if next < 0 {
			break
		}
		out = out[next:]
		if historyChars(out) <= maxChars {
			return out
		}
	}
	for i := range out {
		if out[i].Role == "tool" && len(out[i].Content) > 2000 {
			out[i].Content = out[i].Content[:2000] + "…"
		}
		if out[i].Role == "assistant" && len(out[i].Content) > 4000 {
			out[i].Content = out[i].Content[:4000] + "…"
		}
	}
	return out
}

func trimAssistantHistory(history []assistant.Message, max int) []assistant.Message {
	if len(history) <= max {
		return append([]assistant.Message(nil), history...)
	}
	start := len(history) - max
	for start < len(history) && history[start].Role != "user" {
		start++
	}
	if start >= len(history) {
		start = len(history) - max
	}
	return append([]assistant.Message(nil), history[start:]...)
}
