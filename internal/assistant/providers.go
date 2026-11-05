package assistant

const (
	GroqBaseURL       = "https://api.groq.com/openai/v1"
	OpenRouterBaseURL = "https://openrouter.ai/api/v1"
)

// GroqFreeModels is a curated list because Groq's /models response does not
// expose which models are available on the free plan.
var GroqFreeModels = []Model{
	{
		ID: "qwen/qwen3.8-27b", Name: "Qwen3.8 27B", Provider: ProviderGroq,
		ContextLength: 131072, SupportsTools: true, Free: true,
		Note: "free tier: 30 RPM / 1000 RPD / 8K TPM / 200K TPD",
	},
	{
		ID: "openai/gpt-oss-120b", Name: "GPT-OSS 120B", Provider: ProviderGroq,
		ContextLength: 131072, SupportsTools: true, Free: true,
		Note: "free tier: 30 RPM / 1000 RPD / 8K TPM / 200K TPD",
	},
	{
		ID: "openai/gpt-oss-20b", Name: "GPT-OSS 20B", Provider: ProviderGroq,
		ContextLength: 131072, SupportsTools: true, Free: true,
		Note: "free tier: 30 RPM / 1000 RPD / 8K TPM / 200K TPD",
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
	default:
		return groqBaseURL
	}
}

func DefaultModel(provider string) string {
	switch provider {
	case ProviderOpenRouter:
		return "nvidia/nemotron-3-ultra-550b-a55b:free"
	default:
		return "qwen/qwen3.8-27b"
	}
}

func ProviderName(provider string) string {
	switch provider {
	case ProviderOpenRouter:
		return "OpenRouter"
	default:
		return "GroqCloud"
	}
}
