package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDefaultModel(t *testing.T) {
	if DefaultModel() != "gemini-3.5-flash-lite" {
		t.Fatalf("default %q", DefaultModel())
	}
}

func TestGoogleModelsCurated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-goog-api-key") != "key" {
			t.Errorf("missing google key header")
		}
		fmt.Fprint(w, `{"models":[
			{"name":"models/gemini-3.8-flash","displayName":"Gemini 3.8 Flash","inputTokenLimit":1000000,"supportedGenerationMethods":["generateContent"]},
			{"name":"models/gemini-4-argon","displayName":"Gemini 4 Argon","inputTokenLimit":1000000,"supportedGenerationMethods":["generateContent"]},
			{"name":"models/gemini-3.5-flash-lite","displayName":"Gemini 3.5 Flash-Lite","inputTokenLimit":1000000,"supportedGenerationMethods":["generateContent"]},
			{"name":"models/embedding-001","displayName":"Embedding","inputTokenLimit":2048,"supportedGenerationMethods":["embedContent"]}
		]}`)
	}))
	defer srv.Close()
	old := googleModelsURL
	googleModelsURL = srv.URL
	defer func() { googleModelsURL = old }()

	models, err := ListModels(context.Background(), "key", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 2 || models[0].ID != "gemini-3.5-flash-lite" || models[1].ID != "gemini-3.8-flash" {
		t.Fatalf("models %+v", models)
	}
}

func TestExtractErrorBodyArray(t *testing.T) {
	got := extractErrorBody([]byte(`[{"error":{"code":400,"message":"Please pass a valid API key"}}]`))
	if got != "Please pass a valid API key" {
		t.Fatalf("got %q", got)
	}
}

func TestGeminiToolCallsWithoutIndex(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"id\":\"a\",\"type\":\"function\",\"function\":{\"name\":\"echo\",\"arguments\":\"{\\\"x\\\":1}\"}}]}}]}\n\n")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"id\":\"b\",\"type\":\"function\",\"function\":{\"name\":\"echo\",\"arguments\":\"{\\\"x\\\":2}\"}}]}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()
	old := googleBaseURL
	googleBaseURL = srv.URL
	defer func() { googleBaseURL = old }()
	msg, err := streamChat(context.Background(), Settings{Model: "m"}, "key", nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(msg.ToolCalls) != 2 || msg.ToolCalls[0].ID != "a" || msg.ToolCalls[1].ID != "b" {
		t.Fatalf("tool calls %+v", msg.ToolCalls)
	}
}

func TestRunToolLoop(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "text/event-stream")
		if requests == 1 {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if _, ok := body["tools"]; !ok {
				t.Error("tools missing from request")
			}
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"echo\",\"arguments\":\"{\\\"x\\\":\"}}]}}]}\n\n")
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"1}\"}}]}}]}\n\n")
			fmt.Fprint(w, "data: [DONE]\n\n")
			return
		}
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"готово\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()
	old := googleBaseURL
	googleBaseURL = srv.URL
	defer func() { googleBaseURL = old }()

	var events []Event
	history := []Message{{Role: "user", Content: "test"}}
	tools := []Tool{{
		Name: "echo", Description: "echo",
		Parameters: map[string]any{"type": "object"},
		Run: func(ctx context.Context, args map[string]any) (any, error) {
			if args["x"] != float64(1) {
				t.Fatalf("args %+v", args)
			}
			return map[string]any{"ok": true}, nil
		},
	}}
	err := Run(context.Background(), Settings{Model: "m", MaxToolRounds: 3}, "key", &history, tools, func(e Event) {
		events = append(events, e)
	})
	if err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("requests %d", requests)
	}
	if history[len(history)-1].Content != "готово" {
		t.Fatalf("history %+v", history)
	}
	foundTool := false
	for _, e := range events {
		if e.Type == "tool" && e.ToolName == "echo" {
			foundTool = true
		}
	}
	if !foundTool {
		t.Fatalf("tool event missing: %+v", events)
	}
	if !strings.Contains(events[len(events)-1].Text, "готово") {
		t.Fatalf("delta missing: %+v", events)
	}
}

func TestFriendlyError(t *testing.T) {
	if got := FriendlyError(&APIError{Status: 400, Body: "API key not valid"}); !strings.Contains(got, "недействителен") {
		t.Fatalf("got %q", got)
	}
	if got := FriendlyError(&APIError{Status: 429, Body: "RESOURCE_EXHAUSTED"}); !strings.Contains(got, "Лимит") {
		t.Fatalf("got %q", got)
	}
}
