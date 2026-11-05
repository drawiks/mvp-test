package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"mvp/internal/assistant"
)

type googleKeyState struct {
	CooldownUntil time.Time
	Invalid       bool
	LastError     string
}

type GoogleKeyInfo struct {
	Index         int    `json:"index"`
	Masked        string `json:"masked"`
	Active        bool   `json:"active"`
	Status        string `json:"status"`
	CooldownUntil string `json:"cooldownUntil"`
	LastError     string `json:"lastError"`
}

func maskKey(key string) string {
	if len(key) <= 8 {
		return "••••••"
	}
	return key[:4] + "…" + key[len(key)-4:]
}

func (a *App) googleKeysSnapshot() []string {
	a.googleMu.Lock()
	defer a.googleMu.Unlock()
	return append([]string(nil), a.googleKeys...)
}

func (a *App) currentGoogleKey() string {
	a.googleMu.Lock()
	defer a.googleMu.Unlock()
	return a.currentGoogleKeyLocked()
}

func (a *App) currentGoogleKeyLocked() string {
	if len(a.googleKeys) == 0 {
		return ""
	}
	if a.googleKeyIndex < 0 || a.googleKeyIndex >= len(a.googleKeys) {
		a.googleKeyIndex = 0
	}
	now := time.Now()
	for offset := 0; offset < len(a.googleKeys); offset++ {
		i := (a.googleKeyIndex + offset) % len(a.googleKeys)
		st := a.googleKeyStates[i]
		if st.Invalid || st.CooldownUntil.After(now) {
			continue
		}
		a.googleKeyIndex = i
		return a.googleKeys[i]
	}
	return ""
}

func (a *App) GetGoogleKeys() []GoogleKeyInfo {
	a.googleMu.Lock()
	defer a.googleMu.Unlock()
	now := time.Now()
	out := make([]GoogleKeyInfo, 0, len(a.googleKeys))
	for i, key := range a.googleKeys {
		st := a.googleKeyState(i)
		status := "ok"
		cooldown := ""
		if st.Invalid {
			status = "invalid"
		} else if st.CooldownUntil.After(now) {
			status = "limited"
			cooldown = st.CooldownUntil.UTC().Format(time.RFC3339)
		}
		out = append(out, GoogleKeyInfo{
			Index: i, Masked: maskKey(key), Active: i == a.googleKeyIndex,
			Status: status, CooldownUntil: cooldown, LastError: st.LastError,
		})
	}
	return out
}

func (a *App) googleKeyState(i int) googleKeyState {
	if i < 0 || i >= len(a.googleKeyStates) {
		return googleKeyState{}
	}
	return a.googleKeyStates[i]
}

func (a *App) AddGoogleKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("ключ пустой")
	}
	a.googleMu.Lock()
	for _, existing := range a.googleKeys {
		if existing == key {
			a.googleMu.Unlock()
			return errors.New("такой ключ уже добавлен")
		}
	}
	a.googleKeys = append(a.googleKeys, key)
	a.googleKeyStates = append(a.googleKeyStates, googleKeyState{})
	a.googleMu.Unlock()
	return a.saveConfig()
}

func (a *App) RemoveGoogleKey(index int) error {
	a.googleMu.Lock()
	if index < 0 || index >= len(a.googleKeys) {
		a.googleMu.Unlock()
		return errors.New("ключ не найден")
	}
	a.googleKeys = append(a.googleKeys[:index], a.googleKeys[index+1:]...)
	a.googleKeyStates = append(a.googleKeyStates[:index], a.googleKeyStates[index+1:]...)
	if a.googleKeyIndex >= len(a.googleKeys) {
		a.googleKeyIndex = 0
	} else if index < a.googleKeyIndex {
		a.googleKeyIndex--
	}
	a.googleMu.Unlock()
	return a.saveConfig()
}

func (a *App) TestGoogleKeys() error {
	keys := a.googleKeysSnapshot()
	if len(keys) == 0 {
		return errors.New("добавьте хотя бы один Google-ключ")
	}
	okCount := 0
	for i, key := range keys {
		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		_, err := assistant.ListModels(ctx, key, true)
		cancel()
		a.googleMu.Lock()
		st := a.googleKeyState(i)
		if err == nil {
			st.Invalid = false
			st.CooldownUntil = time.Time{}
			st.LastError = ""
			okCount++
		} else {
			st.Invalid = assistant.IsInvalidKeyError(err)
			st.LastError = err.Error()
		}
		a.googleKeyStates[i] = st
		a.googleMu.Unlock()
	}
	if okCount == 0 {
		return errors.New("ни один Google-ключ не прошёл проверку")
	}
	return nil
}

type appGoogleRotator struct{ a *App }

func (r *appGoogleRotator) Key() string { return r.a.currentGoogleKey() }

func (r *appGoogleRotator) Rotate(err error) (string, bool) {
	r.a.googleMu.Lock()
	defer r.a.googleMu.Unlock()
	if len(r.a.googleKeys) == 0 {
		return "", false
	}
	i := r.a.googleKeyIndex
	if i < 0 || i >= len(r.a.googleKeys) {
		i = 0
		r.a.googleKeyIndex = 0
	}
	st := r.a.googleKeyState(i)
	now := time.Now()
	switch {
	case assistant.IsInvalidKeyError(err):
		st.Invalid = true
	case assistant.IsRateLimitError(err):
		delay := time.Minute
		var api *assistant.APIError
		if errors.As(err, &api) && api.RetryAfter != "" {
			if d, perr := time.ParseDuration(api.RetryAfter); perr == nil {
				delay = d
			}
		}
		st.CooldownUntil = now.Add(delay)
	default:
		return "", false
	}
	st.LastError = err.Error()
	r.a.googleKeyStates[i] = st
	for offset := 1; offset <= len(r.a.googleKeys); offset++ {
		next := (i + offset) % len(r.a.googleKeys)
		nextState := r.a.googleKeyStates[next]
		if nextState.Invalid || nextState.CooldownUntil.After(now) {
			continue
		}
		r.a.googleKeyIndex = next
		return r.a.googleKeys[next], true
	}
	return "", false
}

func (a *App) googleKeysEncrypted() ([]string, error) {
	keys := a.googleKeysSnapshot()
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		enc, err := assistant.EncryptSecret(key)
		if err != nil {
			return nil, fmt.Errorf("encrypt google key: %w", err)
		}
		out = append(out, enc)
	}
	return out, nil
}
