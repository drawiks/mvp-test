package assistant

import "strings"

const (
	GroqBaseURL       = "https://api.groq.com/openai/v1"
	OpenRouterBaseURL = "https://openrouter.ai/api/v1"
	OllamaBaseURL     = "http://127.0.0.1:11434/v1"
	LlamaCppBaseURL   = "http://127.0.0.1:9931/v1"
)

// GroqFreeModels is a curated list because Groq's /models response does not
// expose which models are available on the free plan.
var GroqFreeModels = []Model{
	{
		ID: "qwen/qwen3.8-27b", Name: "Qwen3.8 27B", Provider: ProviderGroq,
		ContextLength: 131072, SupportsTools: true, Free: true,
		Privacy: "Groq не сохраняет данные по умолчанию; ZDR включается в Data Controls; данные в US",
	},
	{
		ID: "openai/gpt-oss-120b", Name: "GPT-OSS 120B", Provider: ProviderGroq,
		ContextLength: 131072, SupportsTools: true, Free: true,
		Privacy: "Groq не сохраняет данные по умолчанию; ZDR включается в Data Controls; данные в US",
	},
	{
		ID: "openai/gpt-oss-20b", Name: "GPT-OSS 20B", Provider: ProviderGroq,
		ContextLength: 131072, SupportsTools: true, Free: true,
		Privacy: "Groq не сохраняет данные по умолчанию; ZDR включается в Data Controls; данные в US",
	},
}

var (
	groqBaseURL       = GroqBaseURL
	openRouterBaseURL = OpenRouterBaseURL
)

func BaseURL(provider string) string {
	switch provider {
	case ProviderOpenRouter:
		return openRouterBaseURL
	case ProviderOllama:
		return OllamaBaseURL
	case ProviderLlamaCpp:
		return LlamaCppBaseURL
	default:
		return groqBaseURL
	}
}

// Endpoint returns the OpenAI-compatible base URL for the selected provider.
func Endpoint(settings Settings) string {
	switch settings.Provider {
	case ProviderOpenRouter:
		return openRouterBaseURL
	case ProviderOllama:
		return normalizedURL(settings.OllamaURL, OllamaBaseURL)
	case ProviderLlamaCpp:
		return normalizedURL(settings.LlamaCppURL, LlamaCppBaseURL)
	default:
		return groqBaseURL
	}
}

func IsLocal(provider string) bool {
	return provider == ProviderOllama || provider == ProviderLlamaCpp
}

func normalizedURL(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = fallback
	}
	value = strings.TrimRight(value, "/")
	if !strings.HasSuffix(value, "/v1") {
		value += "/v1"
	}
	return value
}

func localHost(base string) string {
	return strings.TrimSuffix(base, "/v1")
}

func DefaultModel(provider string) string {
	switch provider {
	case ProviderOpenRouter:
		return "nvidia/nemotron-3-ultra-550b-a55b:free"
	case ProviderOllama, ProviderLlamaCpp:
		return ""
	default:
		return "qwen/qwen3.8-27b"
	}
}

func ProviderName(provider string) string {
	switch provider {
	case ProviderOpenRouter:
		return "OpenRouter"
	case ProviderOllama:
		return "Ollama"
	case ProviderLlamaCpp:
		return "llama.cpp"
	default:
		return "GroqCloud"
	}
}
