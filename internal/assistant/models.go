package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const modelCacheTTL = time.Hour

var modelCache = struct {
	sync.Mutex
	data []Model
	at   time.Time
}{}

// ListModels returns the curated free-tier Gemini models available to the key.
func ListModels(ctx context.Context, apiKey string, force bool) ([]Model, error) {
	modelCache.Lock()
	if !force && modelCache.data != nil && time.Since(modelCache.at) < modelCacheTTL {
		out := append([]Model(nil), modelCache.data...)
		modelCache.Unlock()
		return out, nil
	}
	modelCache.Unlock()

	models, err := fetchGoogleModels(ctx, apiKey)
	if err != nil {
		return nil, err
	}
	modelCache.Lock()
	modelCache.data = append([]Model(nil), models...)
	modelCache.at = time.Now()
	modelCache.Unlock()
	return models, nil
}

func fetchGoogleModels(ctx context.Context, apiKey string) ([]Model, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("google: не задан API-ключ")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleModelsURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-goog-api-key", apiKey)
	req.Header.Set("User-Agent", "mvp-test-assistant")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, &APIError{Status: resp.StatusCode, Body: extractErrorBody(raw)}
	}
	var payload struct {
		Models []struct {
			Name                       string   `json:"name"`
			DisplayName                string   `json:"displayName"`
			InputTokenLimit            int      `json:"inputTokenLimit"`
			SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	byID := make(map[string]struct {
		name    string
		context int
	}, len(payload.Models))
	for _, m := range payload.Models {
		if !contains(m.SupportedGenerationMethods, "generateContent") {
			continue
		}
		id := strings.TrimPrefix(m.Name, "models/")
		name := m.DisplayName
		if name == "" {
			name = id
		}
		byID[id] = struct {
			name    string
			context int
		}{name, m.InputTokenLimit}
	}
	out := make([]Model, 0, len(GoogleFreeModels))
	for _, id := range GoogleFreeModels {
		if info, ok := byID[id]; ok {
			out = append(out, Model{ID: id, Name: info.name, ContextLength: info.context})
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("для этого ключа/проекта не найдено бесплатных моделей Gemini")
	}
	return out, nil
}

func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
