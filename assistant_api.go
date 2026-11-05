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

const assistantHistoryBudgetChars = 12000

type AssistantSettings struct {
	Model           string  `json:"model"`
	Temperature     float64 `json:"temperature"`
	MaxToolRounds   int     `json:"maxToolRounds"`
	SendPlayerNames bool    `json:"sendPlayerNames"`
	GoogleKeyCount  int     `json:"googleKeyCount"`
	GoogleActiveKey int     `json:"googleActiveKey"`
}

type AssistantModel struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ContextLength int    `json:"contextLength"`
}

type AssistantProposal struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	Expression        string `json:"expression"`
	ActionID          string `json:"actionId,omitempty"`
	Pending           bool   `json:"pending,omitempty"`
	Mode              string `json:"mode,omitempty"`
	PresetID          string `json:"presetId,omitempty"`
	TargetName        string `json:"targetName,omitempty"`
	CurrentExpression string `json:"currentExpression,omitempty"`
	Activate          bool   `json:"activate,omitempty"`
	BaseActiveID      string `json:"baseActiveId,omitempty"`
}

func (a *App) GetAssistantSettings() AssistantSettings {
	s := a.assistantSettings
	return AssistantSettings{
		Model: s.Model, Temperature: s.Temperature, MaxToolRounds: s.MaxToolRounds,
		SendPlayerNames: s.SendPlayerNames,
		GoogleKeyCount:  len(a.googleKeysSnapshot()), GoogleActiveKey: a.googleKeyIndex + 1,
	}
}

func (a *App) SaveAssistantSettings(s AssistantSettings) error {
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
		Model: strings.TrimSpace(s.Model), Temperature: s.Temperature,
		MaxToolRounds: s.MaxToolRounds, SendPlayerNames: s.SendPlayerNames,
	}
	return a.saveConfig()
}

func (a *App) ListAssistantModels(refresh bool) ([]AssistantModel, error) {
	if a.ctx == nil {
		return nil, errors.New("приложение ещё не запущено")
	}
	keys := a.googleKeysSnapshot()
	if len(keys) == 0 {
		return nil, errors.New("добавьте Google-ключ в настройках")
	}
	ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
	defer cancel()
	var (
		models []assistant.Model
		err    error
	)
	for _, key := range keys {
		models, err = assistant.ListModels(ctx, key, refresh)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, errors.New(assistant.FriendlyError(err))
	}
	out := make([]AssistantModel, 0, len(models))
	for _, m := range models {
		out = append(out, AssistantModel{ID: m.ID, Name: m.Name, ContextLength: m.ContextLength})
	}
	return out, nil
}

func (a *App) TestAssistantConnection() error {
	if strings.TrimSpace(a.assistantSettings.Model) == "" {
		return errors.New("выберите модель")
	}
	key := a.currentGoogleKey()
	if key == "" {
		return errors.New("нет доступных Google-ключей")
	}
	if a.ctx == nil {
		return errors.New("приложение ещё не запущено")
	}
	ctx, cancel := context.WithTimeout(a.ctx, 60*time.Second)
	defer cancel()
	if err := assistant.Ping(ctx, a.assistantSettings, key); err != nil {
		return errors.New(assistant.FriendlyError(err))
	}
	return nil
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
	if strings.TrimSpace(settings.Model) == "" {
		a.assistantMu.Unlock()
		return errors.New("выберите модель")
	}
	if a.currentGoogleKey() == "" {
		a.assistantMu.Unlock()
		return errors.New("нет доступных Google-ключей. Проверьте ключи в настройках")
	}
	history := trimHistoryByBudget(a.assistantHistory, assistantHistoryBudgetChars)
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
			case "retry":
				runtime.EventsEmit(a.ctx, "assistantRetry", map[string]any{
					"message": e.Text, "attempt": e.RetryAttempt,
					"maxAttempts": e.RetryMax, "delayMs": e.RetryDelayMS,
				})
			}
		}
		err := assistant.RunWithRotator(ctx, settings, &appGoogleRotator{a: a}, &history, a.assistantTools(), emit)
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
				} else {
					detail = err.Error()
				}
			}
			runtime.EventsEmit(a.ctx, "assistantDone", map[string]any{"ok": false, "error": message, "detail": detail})
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
	if a.assistantHistoryPath != "" {
		_ = os.Remove(a.assistantHistoryPath)
	}
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
