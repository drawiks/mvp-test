package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	maxToolCallsPerRound = 6
	maxToolResultChars   = 4000
)

type KeyRotator interface {
	Key() string
	Rotate(err error) (string, bool)
}

type singleKeyRotator struct{ key string }

func (r singleKeyRotator) Key() string { return r.key }

func (r singleKeyRotator) Rotate(error) (string, bool) { return "", false }

// Run executes an assistant turn with a single API key.
func Run(ctx context.Context, settings Settings, apiKey string, history *[]Message, tools []Tool, emit func(Event)) error {
	return RunWithRotator(ctx, settings, singleKeyRotator{key: apiKey}, history, tools, emit)
}

// RunWithRotator executes an assistant turn and can switch keys between tool
// rounds when Gemini reports quota or credential errors.
func RunWithRotator(ctx context.Context, settings Settings, rotator KeyRotator, history *[]Message, tools []Tool, emit func(Event)) error {
	if rotator == nil || strings.TrimSpace(rotator.Key()) == "" {
		return &NoAvailableKeyError{}
	}
	rounds := settings.MaxToolRounds
	if rounds <= 0 {
		rounds = 6
	}

	byName := make(map[string]Tool, len(tools))
	for _, tool := range tools {
		byName[tool.Name] = tool
	}

	for round := 0; round <= rounds; round++ {
		messages := make([]Message, 0, len(*history)+1)
		messages = append(messages, Message{Role: "system", Content: systemPrompt})
		messages = append(messages, *history...)

		var (
			answer Message
			err    error
		)
		answer, err = chatWithRetry(ctx, &settings, rotator, messages, tools, emit, round == 0)
		if err != nil {
			return err
		}
		*history = append(*history, answer)
		if len(answer.ToolCalls) == 0 {
			return nil
		}

		for i, call := range answer.ToolCalls {
			if i >= maxToolCallsPerRound {
				result := `{"error":"слишком много вызовов инструментов за один шаг; используй preview_formula и точечные get_player/explain_player"}`
				if emit != nil {
					emit(Event{Type: "toolResult", ToolName: call.Function.Name, ToolResult: result})
				}
				*history = append(*history, Message{Role: "tool", ToolCallID: call.ID, Name: call.Function.Name, Content: result})
				continue
			}
			tool, ok := byName[call.Function.Name]
			if !ok {
				result := fmt.Sprintf(`{"error":"unknown tool %s"}`, call.Function.Name)
				*history = append(*history, Message{Role: "tool", ToolCallID: call.ID, Name: call.Function.Name, Content: result})
				continue
			}
			if emit != nil {
				emit(Event{Type: "tool", ToolName: call.Function.Name, ToolArgs: call.Function.Arguments})
			}
			var args map[string]any
			if strings.TrimSpace(call.Function.Arguments) != "" {
				if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
					args = map[string]any{}
				}
			}
			if args == nil {
				args = map[string]any{}
			}
			value, runErr := tool.Run(ctx, args)
			var result string
			if runErr != nil {
				raw, _ := json.Marshal(map[string]string{"error": runErr.Error()})
				result = string(raw)
			} else {
				raw, err := json.Marshal(value)
				if err != nil {
					raw, _ = json.Marshal(map[string]string{"error": err.Error()})
				}
				result = string(raw)
			}
			if len(result) > maxToolResultChars {
				result = `{"error":"результат инструмента слишком большой; уточните запрос или используйте более точечный инструмент"}`
			}
			if emit != nil {
				emit(Event{Type: "toolResult", ToolName: call.Function.Name, ToolResult: truncate(result, 2000)})
			}
			*history = append(*history, Message{Role: "tool", ToolCallID: call.ID, Name: call.Function.Name, Content: result})
		}
	}
	return fmt.Errorf("ассистент превысил лимит шагов инструментов (%d)", rounds)
}

func chatWithRetry(ctx context.Context, settings *Settings, rotator KeyRotator, messages []Message, tools []Tool, emit func(Event), allowFallback bool) (Message, error) {
	totalAttempts := 0
	modelAttempts := 0
	fallbackUsed := false

	for {
		key := ""
		if rotator != nil {
			key = rotator.Key()
		}
		if strings.TrimSpace(key) == "" {
			return Message{}, &NoAvailableKeyError{}
		}

		emitted := false
		wrapped := func(event Event) {
			if event.Type == "delta" {
				emitted = true
			}
			if emit != nil {
				emit(event)
			}
		}

		answer, err := streamChat(ctx, *settings, key, messages, tools, wrapped)
		totalAttempts++
		modelAttempts++
		if err == nil {
			return answer, nil
		}
		if ctx.Err() != nil {
			return Message{}, ctx.Err()
		}
		if emitted {
			return Message{}, &PartialResponseError{Err: err}
		}

		if modelAttempts >= maxAttempts {
			// Only switch models on the first round: later history contains
			// thought signatures tied to the original model.
			if allowFallback && !fallbackUsed && IsHighDemandError(err) {
				if next, ok := FallbackModel(settings.Model); ok {
					emitRetry(emit, fmt.Sprintf("Модель %s перегружена. Переключаюсь на %s.", settings.Model, next), 0, 0, 0)
					settings.Model = next
					fallbackUsed = true
					modelAttempts = 0
					continue
				}
			}
			return Message{}, &RetryExhaustedError{Err: err, Attempts: totalAttempts}
		}

		if rotator != nil {
			if _, ok := rotator.Rotate(err); ok {
				emitRetry(emit, "Переключаю Google-ключ.", totalAttempts, maxAttempts, 0)
				continue
			}
		}
		if !IsTransientError(err) {
			return Message{}, err
		}

		delay := RetryDelay(err, modelAttempts-1)
		emitRetry(emit, fmt.Sprintf("%s. Повторяю через %s (попытка %d/%d).", RetryReason(err), delay.Round(time.Millisecond), modelAttempts+1, maxAttempts), modelAttempts+1, maxAttempts, delay)
		if err := SleepContext(ctx, delay); err != nil {
			return Message{}, err
		}
	}
}

func emitRetry(emit func(Event), text string, attempt, max int, delay time.Duration) {
	if emit == nil {
		return
	}
	emit(Event{
		Type: "retry", Text: text,
		RetryAttempt: attempt, RetryMax: max,
		RetryDelayMS: int(delay / time.Millisecond),
	})
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// Ping performs a minimal chat request to validate credentials and model access.
func Ping(ctx context.Context, settings Settings, apiKey string) error {
	history := []Message{{Role: "user", Content: "Ответь одним словом: ok"}}
	_, err := chatWithRetry(ctx, &settings, singleKeyRotator{key: apiKey}, history, nil, nil, true)
	return err
}
