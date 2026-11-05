package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseRetryAfter(t *testing.T) {
	now := time.Now()
	if delay, ok := ParseRetryAfter("3", now); !ok || delay != 3*time.Second {
		t.Fatalf("seconds: %v %v", delay, ok)
	}
	at := now.Add(2 * time.Second).UTC().Format(http.TimeFormat)
	if delay, ok := ParseRetryAfter(at, now); !ok || delay < time.Second || delay > 3*time.Second {
		t.Fatalf("http date: %v %v", delay, ok)
	}
	if _, ok := ParseRetryAfter("not-a-date", now); ok {
		t.Fatal("invalid value accepted")
	}
}

func TestRetryDelay(t *testing.T) {
	if got := RetryDelay(&APIError{Status: 429, RetryAfter: "5"}, 0); got != 5*time.Second {
		t.Fatalf("retry-after delay %v", got)
	}
	got := RetryDelay(&APIError{Status: 503}, 0)
	if got < time.Second || got > 1500*time.Millisecond {
		t.Fatalf("backoff %v", got)
	}
	if got := RetryDelay(&APIError{Status: 503}, 10); got > maxRetryDelay {
		t.Fatalf("max delay %v", got)
	}
}

func TestFriendlyErrorCases(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"invalid key", &APIError{Status: 400, Body: "Please pass a valid API key"}, "недействителен"},
		{"billing", &APIError{Status: 403, Body: "Gemini API free tier is not available"}, "биллинг"},
		{"region", &APIError{Status: 400, Body: "User location is not supported for the API use"}, "биллинг"},
		{"model", &APIError{Status: 404, Body: "model not found"}, "Модель"},
		{"rate", &APIError{Status: 429, Body: "RESOURCE_EXHAUSTED"}, "Лимит"},
		{"internal", &APIError{Status: 500, Body: "internal"}, "Внутренняя"},
		{"proxy", &APIError{Status: 502, Body: "bad gateway"}, "прокси"},
		{"high demand", &APIError{Status: 503, Body: "This model is currently experiencing high demand"}, "перегружена"},
		{"thought signature", &APIError{Status: 400, Body: "Function call is missing a thought_signature"}, "Очистите историю"},
		{"tool schema", &APIError{Status: 400, Body: "value at top-level must be a list"}, "схему запроса"},
		{"timeout", &APIError{Status: 504, Body: "deadline exceeded"}, "вовремя"},
		{"cancelled", context.Canceled, "отменён"},
		{"empty", &EmptyResponseError{}, "пустой"},
		{"no key", &NoAvailableKeyError{}, "Нет доступных"},
		{"partial", &PartialResponseError{Err: errors.New("connection reset")}, "оборвался"},
		{"retry exhausted", &RetryExhaustedError{Err: &APIError{Status: 503, Body: "overloaded"}, Attempts: 4}, "попыток"},
		{"legacy session", errors.New("tls: server did not echo the legacy session ID"), "TLS-соединение"},
		{"certificate", errors.New("tls: failed to verify certificate"), "сертификат"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := FriendlyError(tc.err)
			if !strings.Contains(got, tc.want) {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

type testRotator struct {
	keys []string
	idx  int
}

func (r *testRotator) Key() string { return r.keys[r.idx] }

func (r *testRotator) Rotate(err error) (string, bool) {
	if !IsRateLimitError(err) || r.idx+1 >= len(r.keys) {
		return "", false
	}
	r.idx++
	return r.keys[r.idx], true
}

func TestRunRotatesKeyOnRateLimit(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Header.Get("Authorization") == "Bearer key-1" {
			writeAPIError(w, http.StatusTooManyRequests, "RESOURCE_EXHAUSTED", "0")
			return
		}
		writeSSESuccess(w, "ok")
	}))
	defer srv.Close()
	oldURL := googleBaseURL
	googleBaseURL = srv.URL
	defer func() { googleBaseURL = oldURL }()

	rotator := &testRotator{keys: []string{"key-1", "key-2"}}
	history := []Message{{Role: "user", Content: "hi"}}
	if err := RunWithRotator(context.Background(), Settings{Model: "m"}, rotator, &history, nil, nil); err != nil {
		t.Fatal(err)
	}
	if requests != 2 || rotator.idx != 1 {
		t.Fatalf("requests %d idx %d", requests, rotator.idx)
	}
}

func TestRunRetriesTransientError(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests < 3 {
			writeAPIError(w, http.StatusServiceUnavailable, "high demand", "0")
			return
		}
		writeSSESuccess(w, "ok")
	}))
	defer srv.Close()
	oldURL := googleBaseURL
	googleBaseURL = srv.URL
	defer func() { googleBaseURL = oldURL }()

	history := []Message{{Role: "user", Content: "hi"}}
	var events []Event
	if err := Run(context.Background(), Settings{Model: "m"}, "key", &history, nil, func(e Event) {
		events = append(events, e)
	}); err != nil {
		t.Fatal(err)
	}
	if requests != 3 {
		t.Fatalf("requests %d", requests)
	}
	retries := 0
	for _, event := range events {
		if event.Type == "retry" {
			retries++
		}
	}
	if retries == 0 {
		t.Fatal("retry status event missing")
	}
}

func TestRunFallsBackOnHighDemand(t *testing.T) {
	oldModels := GoogleFreeModels
	GoogleFreeModels = []string{"m", "fallback"}
	defer func() { GoogleFreeModels = oldModels }()

	requests := 0
	models := []string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		var body struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		models = append(models, body.Model)
		if body.Model == "m" {
			writeAPIError(w, http.StatusServiceUnavailable, "high demand", "0")
			return
		}
		writeSSESuccess(w, "ok")
	}))
	defer srv.Close()
	oldURL := googleBaseURL
	googleBaseURL = srv.URL
	defer func() { googleBaseURL = oldURL }()

	history := []Message{{Role: "user", Content: "hi"}}
	var events []Event
	if err := Run(context.Background(), Settings{Model: "m"}, "key", &history, nil, func(e Event) {
		events = append(events, e)
	}); err != nil {
		t.Fatal(err)
	}
	if requests != maxAttempts+1 {
		t.Fatalf("requests %d", requests)
	}
	if len(models) != maxAttempts+1 || models[maxAttempts] != "fallback" {
		t.Fatalf("models %+v", models)
	}
	foundFallback := false
	for _, event := range events {
		if event.Type == "retry" && strings.Contains(event.Text, "Переключаюсь") {
			foundFallback = true
		}
	}
	if !foundFallback {
		t.Fatal("fallback status event missing")
	}
}

func TestRunStopsAfterRetryBudget(t *testing.T) {
	oldModels := GoogleFreeModels
	GoogleFreeModels = []string{"m"}
	defer func() { GoogleFreeModels = oldModels }()

	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		writeAPIError(w, http.StatusServiceUnavailable, "high demand", "0")
	}))
	defer srv.Close()
	oldURL := googleBaseURL
	googleBaseURL = srv.URL
	defer func() { googleBaseURL = oldURL }()

	history := []Message{{Role: "user", Content: "hi"}}
	err := Run(context.Background(), Settings{Model: "m"}, "key", &history, nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if requests != maxAttempts {
		t.Fatalf("requests %d", requests)
	}
	var exhausted *RetryExhaustedError
	if !errors.As(err, &exhausted) || exhausted.Attempts != maxAttempts {
		t.Fatalf("error %#v", err)
	}
	if !strings.Contains(FriendlyError(err), "перегружена") {
		t.Fatalf("friendly error %q", FriendlyError(err))
	}
}

func TestRunDoesNotRetryBadRequest(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		writeAPIError(w, http.StatusBadRequest, "invalid argument", "")
	}))
	defer srv.Close()
	oldURL := googleBaseURL
	googleBaseURL = srv.URL
	defer func() { googleBaseURL = oldURL }()

	history := []Message{{Role: "user", Content: "hi"}}
	err := Run(context.Background(), Settings{Model: "m"}, "key", &history, nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if requests != 1 {
		t.Fatalf("requests %d", requests)
	}
	if !strings.Contains(FriendlyError(err), "отклонил") {
		t.Fatalf("friendly error %q", FriendlyError(err))
	}
}

func TestRunDoesNotRetryPartialResponse(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"partial\"}}]}\n\n")
		fmt.Fprint(w, "data: {\"error\":{\"code\":503,\"message\":\"overloaded\"}}\n\n")
	}))
	defer srv.Close()
	oldURL := googleBaseURL
	googleBaseURL = srv.URL
	defer func() { googleBaseURL = oldURL }()

	history := []Message{{Role: "user", Content: "hi"}}
	err := Run(context.Background(), Settings{Model: "m"}, "key", &history, nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if requests != 1 {
		t.Fatalf("requests %d", requests)
	}
	var partial *PartialResponseError
	if !errors.As(err, &partial) {
		t.Fatalf("error %#v", err)
	}
}

func TestStreamErrorStringCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"error\":{\"code\":\"unavailable\",\"message\":\"high demand\"}}\n\n")
	}))
	defer srv.Close()
	oldURL := googleBaseURL
	googleBaseURL = srv.URL
	defer func() { googleBaseURL = oldURL }()

	_, err := streamChat(context.Background(), Settings{Model: "m"}, "key", nil, nil, nil)
	var api *APIError
	if !errors.As(err, &api) || api.Status != http.StatusServiceUnavailable {
		t.Fatalf("error %#v", err)
	}
}

func writeAPIError(w http.ResponseWriter, status int, message, retryAfter string) {
	if retryAfter != "" {
		w.Header().Set("Retry-After", retryAfter)
	}
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"error":{"code":%d,"message":%q,"status":"UNAVAILABLE"}}`, status, message)
}

func writeSSESuccess(w http.ResponseWriter, text string) {
	w.Header().Set("Content-Type", "text/event-stream")
	fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":%q}}]}\n\n", text)
	fmt.Fprint(w, "data: [DONE]\n\n")
}
