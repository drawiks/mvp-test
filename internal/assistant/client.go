package assistant

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

type toolDefinition struct {
	Type     string          `json:"type"`
	Function toolFunctionDef `json:"function"`
}

type toolFunctionDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type chatRequest struct {
	Model       string           `json:"model"`
	Messages    []Message        `json:"messages"`
	Tools       []toolDefinition `json:"tools,omitempty"`
	ToolChoice  string           `json:"tool_choice,omitempty"`
	Temperature float64          `json:"temperature,omitempty"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	Stream      bool             `json:"stream"`
}

type streamChunk struct {
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
	Choices []struct {
		Delta struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				Index    int    `json:"index"`
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

func streamChat(ctx context.Context, settings Settings, apiKey string, messages []Message, tools []Tool, emit func(Event)) (Message, error) {
	defs := make([]toolDefinition, 0, len(tools))
	for _, t := range tools {
		defs = append(defs, toolDefinition{
			Type: "function",
			Function: toolFunctionDef{
				Name: t.Name, Description: t.Description, Parameters: t.Parameters,
			},
		})
	}

	body, err := json.Marshal(chatRequest{
		Model:       settings.Model,
		Messages:    messages,
		Tools:       defs,
		ToolChoice:  "auto",
		Temperature: settings.Temperature,
		MaxTokens:   2048,
		Stream:      true,
	})
	if err != nil {
		return Message{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, BaseURL(settings.Provider)+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Message{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("User-Agent", "mvp-test-assistant")
	if settings.Provider == ProviderOpenRouter {
		req.Header.Set("HTTP-Referer", "https://github.com/drawiks/mvp-test")
		req.Header.Set("X-Title", "MVP-Test")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Message{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		body := strings.TrimSpace(string(raw))
		var parsed struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(raw, &parsed) == nil && parsed.Error.Message != "" {
			body = parsed.Error.Message
		}
		return Message{}, &APIError{Status: resp.StatusCode, Body: body, RetryAfter: resp.Header.Get("Retry-After")}
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	var content strings.Builder
	calls := map[int]*ToolCall{}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if chunk.Error != nil && chunk.Error.Message != "" {
			return Message{}, &APIError{Status: http.StatusBadGateway, Body: chunk.Error.Message}
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				content.WriteString(choice.Delta.Content)
				if emit != nil {
					emit(Event{Type: "delta", Text: choice.Delta.Content})
				}
			}
			for _, tc := range choice.Delta.ToolCalls {
				call := calls[tc.Index]
				if call == nil {
					call = &ToolCall{Type: "function"}
					calls[tc.Index] = call
				}
				if tc.ID != "" {
					call.ID = tc.ID
				}
				if tc.Type != "" {
					call.Type = tc.Type
				}
				if tc.Function.Name != "" {
					call.Function.Name = tc.Function.Name
				}
				call.Function.Arguments += tc.Function.Arguments
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return Message{}, err
	}

	indexes := make([]int, 0, len(calls))
	for idx := range calls {
		indexes = append(indexes, idx)
	}
	sort.Ints(indexes)
	toolCalls := make([]ToolCall, 0, len(indexes))
	for _, idx := range indexes {
		call := *calls[idx]
		if call.ID == "" {
			call.ID = fmt.Sprintf("call_%d", idx)
		}
		if call.Function.Name == "" {
			continue
		}
		toolCalls = append(toolCalls, call)
	}
	if content.Len() == 0 && len(toolCalls) == 0 {
		return Message{}, fmt.Errorf("ассистент вернул пустой ответ")
	}
	return Message{Role: "assistant", Content: content.String(), ToolCalls: toolCalls}, nil
}
