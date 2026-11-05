package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

const modelCacheTTL = time.Hour

var modelCache = struct {
	sync.Mutex
	data map[string][]Model
	at   map[string]time.Time
}{
	data: map[string][]Model{},
	at:   map[string]time.Time{},
}

// ListModels returns the free tool-capable chat models for a provider.
func ListModels(ctx context.Context, provider, apiKey string, force bool) ([]Model, error) {
	modelCache.Lock()
	if !force {
		if models, ok := modelCache.data[provider]; ok && time.Since(modelCache.at[provider]) < modelCacheTTL {
			out := append([]Model(nil), models...)
			modelCache.Unlock()
			return out, nil
		}
	}
	modelCache.Unlock()

	var (
		models []Model
		err    error
	)
	switch provider {
	case ProviderOpenRouter:
		models, err = fetchOpenRouterModels(ctx)
	default:
		models, err = fetchGroqModels(ctx, apiKey)
	}
	if err != nil {
		return nil, err
	}

	modelCache.Lock()
	modelCache.data[provider] = append([]Model(nil), models...)
	modelCache.at[provider] = time.Now()
	modelCache.Unlock()
	return models, nil
}

func fetchGroqModels(ctx context.Context, apiKey string) ([]Model, error) {
	if strings.TrimSpace(apiKey) == "" {
		return append([]Model(nil), GroqFreeModels...), nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, groqBaseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("User-Agent", "mvp-test-assistant")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return append([]Model(nil), GroqFreeModels...), nil
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("groq: неверный API-ключ")
	}
	if resp.StatusCode != http.StatusOK {
		return append([]Model(nil), GroqFreeModels...), nil
	}

	var payload struct {
		Data []struct {
			ID                string   `json:"id"`
			Name              string   `json:"name"`
			ContextLength     int      `json:"context_length"`
			SupportedFeatures []string `json:"supported_features"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return append([]Model(nil), GroqFreeModels...), nil
	}
	available := make(map[string]struct {
		name    string
		context int
		tools   bool
	}, len(payload.Data))
	for _, m := range payload.Data {
		tools := false
		for _, f := range m.SupportedFeatures {
			if f == "tools" {
				tools = true
			}
		}
		available[m.ID] = struct {
			name    string
			context int
			tools   bool
		}{m.Name, m.ContextLength, tools}
	}

	out := make([]Model, 0, len(GroqFreeModels))
	for _, model := range GroqFreeModels {
		if info, ok := available[model.ID]; ok {
			if info.context > 0 {
				model.ContextLength = info.context
			}
			if !info.tools {
				continue
			}
			out = append(out, model)
			continue
		}
	}
	return out, nil
}

func fetchOpenRouterModels(ctx context.Context) ([]Model, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, openRouterBaseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "mvp-test-assistant")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openrouter: %s", resp.Status)
	}

	var payload struct {
		Data []struct {
			ID                  string   `json:"id"`
			Name                string   `json:"name"`
			ContextLength       int      `json:"context_length"`
			SupportedParameters []string `json:"supported_parameters"`
			Pricing             struct {
				Prompt     string `json:"prompt"`
				Completion string `json:"completion"`
			} `json:"pricing"`
			Architecture struct {
				OutputModalities []string `json:"output_modalities"`
			} `json:"architecture"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	priority := map[string]int{
		"nvidia/nemotron-3-ultra-550b-a55b:free": 0,
		"xingchen-agi/xing4.0-29b-a4b:free":      1,
		"nvidia/nemotron-3.5-lightning:free":     2,
		"google/gemma-4-26b-a4b-it:free":         3,
	}
	out := make([]Model, 0)
	for _, m := range payload.Data {
		if !isZeroPrice(m.Pricing.Prompt) || !isZeroPrice(m.Pricing.Completion) {
			continue
		}
		if m.ID != "openrouter/free" && !strings.HasSuffix(m.ID, ":free") {
			continue
		}
		if len(m.Architecture.OutputModalities) > 0 && !contains(m.Architecture.OutputModalities, "text") {
			continue
		}
		if !contains(m.SupportedParameters, "tools") {
			continue
		}
		model := Model{
			ID: m.ID, Name: m.Name, Provider: ProviderOpenRouter,
			ContextLength: m.ContextLength, SupportsTools: true, Free: true,
			Privacy: "free-маршруты OpenRouter могут использовать данные для обучения; имена маскируются",
		}
		if m.ID == "openrouter/free" {
			model.Note = "авто-роутер: выбирает случайную бесплатную модель"
		}
		out = append(out, model)
	}
	sort.SliceStable(out, func(i, j int) bool {
		pi, iok := priority[out[i].ID]
		pj, jok := priority[out[j].ID]
		switch {
		case iok && !jok:
			return true
		case !iok && jok:
			return false
		case iok && jok && pi != pj:
			return pi < pj
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func isZeroPrice(v string) bool {
	v = strings.TrimSpace(v)
	return v == "" || v == "0" || v == "0.0" || v == "0.00" || v == "0e0"
}

func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
