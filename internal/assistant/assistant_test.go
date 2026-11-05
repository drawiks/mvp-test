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

func TestLookupModelFilterOpenRouter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":[
			{"id":"vendor/free:free","name":"Free","context_length":1000,"supported_parameters":["tools"],"pricing":{"prompt":"0","completion":"0"},"architecture":{"output_modalities":["text"]}},
			{"id":"vendor/paid","name":"Paid","context_length":1000,"supported_parameters":["tools"],"pricing":{"prompt":"0.1","completion":"0.1"},"architecture":{"output_modalities":["text"]}},
			{"id":"vendor/notools:free","name":"No tools","context_length":1000,"supported_parameters":[],"pricing":{"prompt":"0","completion":"0"},"architecture":{"output_modalities":["text"]}},
			{"id":"vendor/zeropriced","name":"Zero","context_length":1000,"supported_parameters":["tools"],"pricing":{"prompt":"0","completion":"0"},"architecture":{"output_modalities":["text"]}}
		]}`)
	}))
	defer srv.Close()
	old := openRouterBaseURL
	openRouterBaseURL = srv.URL
	defer func() { openRouterBaseURL = old }()

	models, err := ListModels(context.Background(), ProviderOpenRouter, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 || models[0].ID != "vendor/free:free" {
		t.Fatalf("models %+v", models)
	}
}

func TestGroqCuratedWithoutKey(t *testing.T) {
	models, err := ListModels(context.Background(), ProviderGroq, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != len(GroqFreeModels) {
		t.Fatalf("models %+v", models)
	}
}

func TestRunToolLoop(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		flusher, _ := w.(http.Flusher)
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
			if flusher != nil {
				flusher.Flush()
			}
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"1}\"}}]}}]}\n\n")
			fmt.Fprint(w, "data: [DONE]\n\n")
			return
		}
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"готово\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()
	old := groqBaseURL
	groqBaseURL = srv.URL
	defer func() { groqBaseURL = old }()

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
	err := Run(context.Background(), Settings{Provider: ProviderGroq, Model: "m", MaxToolRounds: 3}, "key", &history, tools, func(e Event) {
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

func TestParseRate(t *testing.T) {
	h := http.Header{}
	h.Set("x-ratelimit-limit-requests", "1000")
	h.Set("x-ratelimit-remaining-requests", "987")
	h.Set("x-ratelimit-reset-requests", "2m")
	h.Set("x-ratelimit-limit-tokens", "8000")
	h.Set("x-ratelimit-remaining-tokens", "7000")
	r := parseRate(h, ProviderGroq)
	if r == nil || r.RequestsLimit != 1000 || r.RequestsRemaining != 987 || r.TokensLimit != 8000 {
		t.Fatalf("rate %+v", r)
	}
}

func TestFriendlyError429(t *testing.T) {
	err := &APIError{Status: 429, Rate: &RateLimit{
		Provider: ProviderGroq, RequestsLimit: 1000, RequestsRemaining: 0, RequestsReset: "1h",
	}}
	got := FriendlyError(err)
	if !strings.Contains(got, "Groq") || !strings.Contains(got, "1h") {
		t.Fatalf("message %q", got)
	}
}

func TestFetchOpenRouterLimits(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":{"limit_remaining":9.5,"free_model_daily_requests":{"used":3,"limit":50,"remaining":47}}}`)
	}))
	defer srv.Close()
	old := openRouterBaseURL
	openRouterBaseURL = srv.URL
	defer func() { openRouterBaseURL = old }()
	r, err := FetchOpenRouterLimits(context.Background(), "key")
	if err != nil {
		t.Fatal(err)
	}
	if r.FreeRequestsRemaining != 47 || r.FreeRequestsLimit != 50 || r.CreditsRemaining == nil || *r.CreditsRemaining != 9.5 {
		t.Fatalf("rate %+v", r)
	}
}
