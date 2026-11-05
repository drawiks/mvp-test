package assistant

import (
	"context"
	"time"
)

const (
	ProviderGroq       = "groq"
	ProviderOpenRouter = "openrouter"
	ProviderOllama     = "ollama"
	ProviderLlamaCpp   = "llamacpp"
	ProviderGoogle     = "google"
)

// Settings are the persisted assistant options. API keys are not part of this
// struct; the app stores them encrypted.
type Settings struct {
	Provider        string  `json:"provider"`
	Model           string  `json:"model"`
	Temperature     float64 `json:"temperature"`
	MaxToolRounds   int     `json:"maxToolRounds"`
	SendPlayerNames bool    `json:"sendPlayerNames"`
	OllamaURL       string  `json:"ollamaUrl"`
	LlamaCppURL     string  `json:"llamaCppUrl"`
}

func DefaultSettings() Settings {
	return Settings{
		Provider:      ProviderGroq,
		Model:         "qwen/qwen3.8-27b",
		Temperature:   0.2,
		MaxToolRounds: 6,
		OllamaURL:     OllamaBaseURL,
		LlamaCppURL:   LlamaCppBaseURL,
	}
}

// Model is one selectable chat model.
type Model struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Provider      string `json:"provider"`
	ContextLength int    `json:"contextLength"`
	SupportsTools bool   `json:"supportsTools"`
	Free          bool   `json:"free"`
	Local         bool   `json:"local"`
	Privacy       string `json:"privacy,omitempty"`
}

// RateLimit is the latest provider quota information.
type RateLimit struct {
	Provider              string    `json:"provider"`
	RequestsLimit         int       `json:"requestsLimit"`
	RequestsRemaining     int       `json:"requestsRemaining"`
	RequestsReset         string    `json:"requestsReset,omitempty"`
	TokensLimit           int       `json:"tokensLimit"`
	TokensRemaining       int       `json:"tokensRemaining"`
	TokensReset           string    `json:"tokensReset,omitempty"`
	FreeRequestsLimit     int       `json:"freeRequestsLimit"`
	FreeRequestsRemaining int       `json:"freeRequestsRemaining"`
	FreeRequestsReset     string    `json:"freeRequestsReset,omitempty"`
	CreditsRemaining      *float64  `json:"creditsRemaining,omitempty"`
	RetryAfter            string    `json:"retryAfter,omitempty"`
	ReceivedAt            time.Time `json:"receivedAt"`
}

// Message is one OpenAI-compatible chat message.
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Tool is a local function exposed to the model.
type Tool struct {
	Name        string
	Description string
	Parameters  map[string]any
	Run         func(ctx context.Context, args map[string]any) (any, error)
}

// Proposal is a formula suggested by the assistant. It is never saved
// automatically.
type Proposal struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Expression  string `json:"expression"`
}

// Event is emitted while an assistant turn is running.
type Event struct {
	Type        string
	Text        string
	ToolName    string
	ToolArgs    string
	ToolResult  string
	Proposal    *Proposal
	Rate        *RateLimit
	WaitSeconds int
	WaitUntil   time.Time
	WaitReason  string
	Err         error
}

// APIError is a non-2xx response from a provider.
type APIError struct {
	Status     int
	Body       string
	RetryAfter string
	Rate       *RateLimit
}

func (e *APIError) Error() string {
	if e.RetryAfter != "" {
		return e.Body + " (retry after " + e.RetryAfter + ")"
	}
	return e.Body
}
