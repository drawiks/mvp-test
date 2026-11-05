package assistant

const (
	GoogleBaseURL   = "https://gemini-proxy.drawksr.workers.dev/v1beta/openai"
	GoogleModelsURL = "https://gemini-proxy.drawksr.workers.dev/v1beta/models"
)

// GoogleFreeModels is the curated free-tier text model pool, in priority order.
var GoogleFreeModels = []string{
	"gemini-3.5-flash-lite",
	"gemini-3.1-flash-lite",
	"gemini-3.8-flash",
	"gemini-3.6-flash",
	"gemini-3-flash-preview",
}

var (
	googleBaseURL   = GoogleBaseURL
	googleModelsURL = GoogleModelsURL
)

func DefaultModel() string {
	return GoogleFreeModels[0]
}

// FallbackModel returns the first curated model different from current.
func FallbackModel(current string) (string, bool) {
	for _, id := range GoogleFreeModels {
		if id != current {
			return id, true
		}
	}
	return "", false
}
