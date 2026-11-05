package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
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

// ListModels returns the tool-capable chat models for a provider.
func ListModels(ctx context.Context, provider, apiKey, baseURL string, force bool) ([]Model, error) {
	cacheKey := provider
	if IsLocal(provider) {
		cacheKey += "|" + baseURL
	}
	modelCache.Lock()
	if !force {
		if models, ok := modelCache.data[cacheKey]; ok && time.Since(modelCache.at[cacheKey]) < modelCacheTTL {
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
	case ProviderOllama:
		models, err = fetchOllamaModels(ctx, baseURL)
	case ProviderLlamaCpp:
		models, err = fetchLlamaCppModels(ctx, baseURL)
	case ProviderGoogle:
		models, err = fetchGoogleModels(ctx, apiKey)
	default:
		models, err = fetchGroqModels(ctx, apiKey)
	}
	if err != nil {
		return nil, err
	}

	modelCache.Lock()
	modelCache.data[cacheKey] = append([]Model(nil), models...)
	modelCache.at[cacheKey] = time.Now()
	modelCache.Unlock()
	return models, nil
}

func fetchOllamaModels(ctx context.Context, baseURL string) ([]Model, error) {
	base := normalizedURL(baseURL, OllamaBaseURL)
	ids, err := fetchModelIDs(ctx, base+"/models")
	if err != nil || len(ids) == 0 {
		ids, err = fetchOllamaTags(ctx, localHost(base))
	}
	if err != nil {
		return nil, err
	}
	host := localHost(base)
	out := make([]Model, 0, len(ids))
	for _, id := range ids {
		model := Model{
			ID: id, Name: id, Provider: ProviderOllama, SupportsTools: true, Free: true, Local: true,
			Privacy: "Локальная модель, данные не покидают компьютер",
		}
		tools, contextLength, known := fetchOllamaShow(ctx, host, id)
		if contextLength > 0 {
			model.ContextLength = contextLength
		}
		if known {
			model.SupportsTools = tools
		}
		out = append(out, model)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func fetchModelIDs(ctx context.Context, url string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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
		return nil, fmt.Errorf("%s", resp.Status)
	}
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(payload.Data))
	for _, m := range payload.Data {
		if strings.TrimSpace(m.ID) != "" {
			ids = append(ids, m.ID)
		}
	}
	return ids, nil
}

func fetchOllamaTags(ctx context.Context, host string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, host+"/api/tags", nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s", resp.Status)
	}
	var payload struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(payload.Models))
	for _, m := range payload.Models {
		if strings.TrimSpace(m.Name) != "" {
			out = append(out, m.Name)
		}
	}
	return out, nil
}

func fetchOllamaShow(ctx context.Context, host, id string) (tools bool, contextLength int, known bool) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	body, _ := json.Marshal(map[string]string{"model": id})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, host+"/api/show", bytes.NewReader(body))
	if err != nil {
		return true, 0, false
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return true, 0, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return true, 0, false
	}
	var payload struct {
		Capabilities []string       `json:"capabilities"`
		ModelInfo    map[string]any `json:"model_info"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return true, 0, false
	}
	tools = contains(payload.Capabilities, "tools")
	for key, value := range payload.ModelInfo {
		if strings.HasSuffix(key, ".context_length") {
			switch v := value.(type) {
			case float64:
				contextLength = int(v)
			case string:
				contextLength, _ = strconv.Atoi(v)
			}
			break
		}
	}
	return tools, contextLength, len(payload.Capabilities) > 0
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
	priority := map[string]int{
		"gemini-3.5-flash-lite": 0,
		"gemini-3.1-flash-lite": 1,
		"gemini-3.8-flash":      2,
		"gemini-3.6-flash":      3,
	}
	out := make([]Model, 0)
	for _, m := range payload.Models {
		if !contains(m.SupportedGenerationMethods, "generateContent") {
			continue
		}
		id := strings.TrimPrefix(m.Name, "models/")
		name := m.DisplayName
		if name == "" {
			name = id
		}
		out = append(out, Model{
			ID: id, Name: name, Provider: ProviderGoogle,
			ContextLength: m.InputTokenLimit, SupportsTools: true, Free: true,
			Privacy: "Google AI Studio: free-tier запросы могут использоваться для улучшения продуктов",
		})
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

func fetchLlamaCppModels(ctx context.Context, baseURL string) ([]Model, error) {
	base := normalizedURL(baseURL, LlamaCppBaseURL)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/models", nil)
	if err == nil {
		resp, rerr := http.DefaultClient.Do(req)
		if rerr == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				var payload struct {
					Data []struct {
						ID   string `json:"id"`
						Meta struct {
							Context int `json:"n_ctx_train"`
						} `json:"meta"`
					} `json:"data"`
				}
				if json.NewDecoder(resp.Body).Decode(&payload) == nil && len(payload.Data) > 0 {
					out := make([]Model, 0, len(payload.Data))
					for _, m := range payload.Data {
						if m.ID == "" {
							continue
						}
						out = append(out, llamaCppModel(m.ID, m.Meta.Context))
					}
					if len(out) > 0 {
						return out, nil
					}
				}
			}
		}
	}

	req, err = http.NewRequestWithContext(ctx, http.MethodGet, localHost(base)+"/props", nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("llama.cpp: %s", resp.Status)
	}
	var props struct {
		ModelAlias string `json:"model_alias"`
		ModelPath  string `json:"model_path"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&props); err != nil {
		return nil, err
	}
	id := props.ModelAlias
	if id == "" {
		id = props.ModelPath
	}
	if id == "" {
		return nil, fmt.Errorf("llama.cpp: модель не загружена")
	}
	return []Model{llamaCppModel(id, 0)}, nil
}

func llamaCppModel(id string, contextLength int) Model {
	name := id
	if strings.ContainsAny(id, `/\`) {
		name = filepath.Base(id)
	}
	return Model{
		ID: id, Name: name, Provider: ProviderLlamaCpp, ContextLength: contextLength,
		SupportsTools: true, Free: true, Local: true,
		Privacy: "Локальная модель, данные не покидают компьютер",
	}
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
