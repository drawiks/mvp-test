package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const systemPrompt = `Ты — ассистент по разработке MVP-формулы Dota 2 внутри приложения MVP-Test.
Правила:
1. Не выдумывай статистику и числа. Все данные о матче, игроках и формулах получай только через инструменты.
2. Перед тем как предлагать формулу, вызови validate_formula и preview_formula (или compare_formulas).
3. Формула — это выражение expr-lang по токенам из get_token_catalog. Поддерживаются + - * / **, сравнения, && || !, if/else, min/max/abs/round.
4. Учитывай роль и позицию игрока, интерпретируемость и устойчивость формулы, а не только результат одного матча.
5. Если предлагаешь готовую формулу, вызови propose_formula с полями name, description, expression. Не сохраняй формулу сам.
6. Не используй токен mana_restored, если текущий матч распарсен не через manta.
7. Отвечай по-русски, коротко и по делу. Ссылайся на конкретных игроков и числа из инструментов.
8. Не вызывай explain_player для всех игроков. Для обзора используй preview_formula, а детально разбирай 1–2 ключевых игроков.
9. Не повторяй get_match_context и другие тяжёлые вызовы без необходимости.`

// Run executes an assistant turn, including any tool calls, and appends all
// messages to history.
func Run(ctx context.Context, settings Settings, apiKey string, history *[]Message, tools []Tool, emit func(Event)) error {
	if strings.TrimSpace(apiKey) == "" && !IsLocal(settings.Provider) {
		return fmt.Errorf("не задан API-ключ для %s", ProviderName(settings.Provider))
	}
	rounds := settings.MaxToolRounds
	if rounds <= 0 {
		rounds = 6
	}

	const (
		maxToolCallsPerRound = 6
		maxToolResultChars   = 6000
	)

	byName := make(map[string]Tool, len(tools))
	for _, tool := range tools {
		byName[tool.Name] = tool
	}

	for round := 0; round <= rounds; round++ {
		messages := make([]Message, 0, len(*history)+1)
		messages = append(messages, Message{Role: "system", Content: systemPrompt})
		messages = append(messages, *history...)

		answer, _, err := streamChat(ctx, settings, apiKey, messages, tools, emit)
		if err != nil {
			return err
		}
		*history = append(*history, answer)
		if len(answer.ToolCalls) == 0 {
			return nil
		}

		for callIndex, call := range answer.ToolCalls {
			if callIndex >= maxToolCallsPerRound {
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
			var value any
			var runErr error
			if args == nil {
				args = map[string]any{}
			}
			value, runErr = tool.Run(ctx, args)
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

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// Ping performs a minimal chat request to validate credentials and model access.
func Ping(ctx context.Context, settings Settings, apiKey string) error {
	history := []Message{{Role: "user", Content: "Ответь одним словом: ok"}}
	_, _, err := streamChat(ctx, settings, apiKey, history, nil, nil)
	return err
}
