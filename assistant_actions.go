package main

import (
	"errors"
	"fmt"
	"strings"

	"mvp/internal/formula"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) stageAssistantFormulaChange(args map[string]any) (any, error) {
	expression := strings.TrimSpace(stringArg(args, "expression"))
	if expression == "" {
		return nil, errors.New("пустое выражение")
	}
	allowed := a.varStore.AllowedNames()
	if err := formula.Validate(expression, allowed); err != nil {
		result := map[string]any{"status": "invalid", "error": err.Error()}
		if hint := formulaValidationHint(err); hint != "" {
			result["hint"] = hint
		}
		return result, nil
	}

	mode := strings.ToLower(strings.TrimSpace(stringArg(args, "mode")))
	if mode == "" {
		mode = "auto"
	}
	if mode != "auto" && mode != "update" && mode != "create" {
		return nil, errors.New("mode должен быть auto, update или create")
	}
	presetID := strings.TrimSpace(stringArg(args, "preset_id"))
	if mode == "create" && presetID != "" {
		return nil, errors.New("preset_id используется только для обновления")
	}

	active := a.store.Active()
	target := active
	if presetID != "" {
		preset, ok := a.store.Get(presetID)
		if !ok {
			return nil, fmt.Errorf("формула не найдена: %s", presetID)
		}
		target = preset
	}

	finalMode := "update"
	if mode == "create" {
		finalMode = "create"
	}
	if finalMode == "update" && isBuiltinPreset(target.ID) {
		if mode == "update" {
			return nil, errors.New("встроенную формулу нельзя изменить; используй mode=create")
		}
		finalMode = "create"
	}

	name := strings.TrimSpace(stringArg(args, "name"))
	if finalMode == "update" {
		if name == "" {
			name = target.Name
		}
		if a.store.NameTaken(name, target.ID) {
			return nil, errors.New("формула с таким названием уже существует")
		}
	} else {
		if name == "" {
			if target.ID == active.ID && isBuiltinPreset(active.ID) {
				name = target.Name + " (ассистент)"
			} else {
				name = "Формула ассистента"
			}
			name = a.uniquePresetName(name)
		} else if a.store.NameTaken(name, "") {
			return nil, errors.New("формула с таким названием уже существует")
		}
	}

	presetIDFinal := target.ID
	if finalMode == "create" {
		presetIDFinal = formula.NewID()
	}
	preset := formula.Preset{
		ID: presetIDFinal, Name: name, Kind: "expression",
		Expression: expression, Weights: map[string]float64{},
	}
	if err := a.store.ValidatePreset(preset, allowed); err != nil {
		return nil, err
	}

	current := a.GetActiveFormulaExpression()
	if finalMode == "update" {
		current = presetExpression(target)
	}
	diff := a.DiffFormula(current, expression)
	proposal := &AssistantProposal{
		Name: name, Description: strings.TrimSpace(stringArg(args, "description")),
		Expression: expression, ActionID: formula.NewID(), Pending: true,
		Mode: finalMode, PresetID: presetIDFinal, TargetName: target.Name,
		CurrentExpression: current, Activate: finalMode == "create" || target.ID == active.ID,
		BaseActiveID: active.ID,
	}

	a.assistantMu.Lock()
	a.assistantProposal = proposal
	a.saveAssistantHistoryLocked()
	a.assistantMu.Unlock()
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "assistantProposal", proposal)
	}

	return map[string]any{
		"status": "approval_required", "action_id": proposal.ActionID,
		"mode": finalMode, "preset_id": presetIDFinal, "name": name,
		"diff":    diff,
		"message": "Формула подготовлена. Пользователь должен подтвердить действие в интерфейсе.",
	}, nil
}

func (a *App) ApproveAssistantAction(id string) error {
	a.assistantMu.Lock()
	if a.assistantProposal == nil || !a.assistantProposal.Pending || a.assistantProposal.ActionID != id {
		a.assistantMu.Unlock()
		return errors.New("действие устарело. Попросите ассистента подготовить его заново")
	}
	action := *a.assistantProposal
	a.assistantMu.Unlock()

	if action.BaseActiveID != "" && a.store.Active().ID != action.BaseActiveID {
		return errors.New("активная формула изменилась. Попросите ассистента пересчитать действие")
	}
	if action.Mode == "update" && isBuiltinPreset(action.PresetID) {
		return errors.New("встроенную формулу нельзя изменить")
	}
	if action.Mode != "create" && action.Mode != "update" {
		return errors.New("неизвестное действие")
	}

	preset := formula.Preset{
		ID: action.PresetID, Name: action.Name, Kind: "expression",
		Expression: action.Expression, Weights: map[string]float64{},
	}
	if err := a.store.ValidatePreset(preset, a.varStore.AllowedNames()); err != nil {
		return err
	}
	if a.store.NameTaken(preset.Name, preset.ID) {
		return errors.New("формула с таким названием уже существует")
	}
	if action.Mode == "update" {
		if _, ok := a.store.Get(preset.ID); !ok {
			return errors.New("формула для обновления не найдена")
		}
	} else if _, ok := a.store.Get(preset.ID); ok {
		return errors.New("формула с таким идентификатором уже существует")
	}

	a.store.Upsert(preset)
	if action.Activate {
		a.store.SetActive(preset.ID)
	}
	a.store.Save()

	a.assistantMu.Lock()
	if a.assistantProposal != nil && a.assistantProposal.ActionID == id {
		a.assistantProposal = nil
		a.saveAssistantHistoryLocked()
	}
	a.assistantMu.Unlock()

	a.EmitResult()
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "assistantProposal", nil)
	}
	return nil
}

func (a *App) RejectAssistantAction(id string) error {
	a.assistantMu.Lock()
	if a.assistantProposal == nil || !a.assistantProposal.Pending || a.assistantProposal.ActionID != id {
		a.assistantMu.Unlock()
		return errors.New("действие устарело")
	}
	a.assistantProposal = nil
	a.saveAssistantHistoryLocked()
	a.assistantMu.Unlock()
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "assistantProposal", nil)
	}
	return nil
}

func (a *App) uniquePresetName(base string) string {
	name := base
	for i := 2; a.store.NameTaken(name, ""); i++ {
		name = fmt.Sprintf("%s %d", base, i)
	}
	return name
}

func presetExpression(preset formula.Preset) string {
	if preset.Kind == "expression" {
		return strings.TrimSpace(preset.Expression)
	}
	return formula.LinearToExpression(preset.Weights)
}

func isBuiltinPreset(id string) bool {
	return id == "standard_v2" || id == "tutorial"
}
