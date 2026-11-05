package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"mvp/internal/assistant"
)

type assistantHistoryFile struct {
	Version   int                 `json:"version"`
	UpdatedAt string              `json:"updatedAt"`
	History   []assistant.Message `json:"history"`
	Proposal  *AssistantProposal  `json:"proposal,omitempty"`
}

type AssistantChatMessage struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

type AssistantHistoryState struct {
	Messages []AssistantChatMessage `json:"messages"`
	Proposal *AssistantProposal     `json:"proposal,omitempty"`
}

func (a *App) loadAssistantHistory() {
	if a.assistantHistoryPath == "" {
		return
	}
	data, err := os.ReadFile(a.assistantHistoryPath)
	if err != nil {
		return
	}
	var file assistantHistoryFile
	if json.Unmarshal(data, &file) != nil {
		return
	}
	a.assistantMu.Lock()
	defer a.assistantMu.Unlock()
	a.assistantHistory = limitHistoryForStorage(file.History, 60)
	a.assistantProposal = file.Proposal
}

func (a *App) saveAssistantHistoryLocked() {
	if a.assistantHistoryPath == "" {
		return
	}
	file := assistantHistoryFile{
		Version: 1, UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		History: limitHistoryForStorage(a.assistantHistory, 60), Proposal: a.assistantProposal,
	}
	data, err := json.Marshal(file)
	if err != nil {
		return
	}
	dir := filepath.Dir(a.assistantHistoryPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	tmp := a.assistantHistoryPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return
	}
	if err := os.Rename(tmp, a.assistantHistoryPath); err != nil {
		_ = os.Remove(tmp)
	}
}

func (a *App) GetAssistantHistory() AssistantHistoryState {
	a.assistantMu.Lock()
	defer a.assistantMu.Unlock()
	out := AssistantHistoryState{Proposal: a.assistantProposal}
	for _, m := range a.assistantHistory {
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}
		if m.Content == "" {
			continue
		}
		out.Messages = append(out.Messages, AssistantChatMessage{Role: m.Role, Text: m.Content})
	}
	return out
}

func limitHistoryForStorage(history []assistant.Message, max int) []assistant.Message {
	history = trimAssistantHistory(history, max)
	for i := range history {
		if history[i].Role == "tool" && len(history[i].Content) > 8192 {
			history[i].Content = history[i].Content[:8192] + "…"
		}
	}
	return history
}
