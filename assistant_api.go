package main

import (
	"context"
	"errors"
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
}

type AssistantProposal struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Expression  string `json:"expression"`
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
			SupportsTools: m.SupportsTools, Free: m.Free, Note: m.Note,
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
	history := trimAssistantHistory(a.assistantHistory, 40)
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
				a.assistantHistory = history
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
			}
		}
		err := assistant.Run(ctx, settings, key, &history, a.assistantTools(), emit)
		if err != nil {
			message := err.Error()
			if errors.Is(err, context.Canceled) {
				message = "запрос отменён"
			}
			runtime.EventsEmit(a.ctx, "assistantDone", map[string]any{"ok": false, "error": message})
			return
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
