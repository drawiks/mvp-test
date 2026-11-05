package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"mvp/internal/assistant"
	"mvp/internal/formula"
	"mvp/internal/model"
	"mvp/internal/mvp"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type assistantPlayerInfo struct {
	PlayerID    int                `json:"player_id"`
	Hero        string             `json:"hero"`
	Team        string             `json:"team"`
	Position    int                `json:"position,omitempty"`
	Lane        string             `json:"lane,omitempty"`
	Score       float64            `json:"score"`
	GlobalPlace int                `json:"global_place"`
	TeamPlace   int                `json:"team_place"`
	MvpRole     string             `json:"mvp_role,omitempty"`
	Name        string             `json:"name,omitempty"`
	SteamID     int64              `json:"steam_id,omitempty"`
	Stats       map[string]float64 `json:"stats"`
}

type assistantMatchInfo struct {
	MatchID     int64                 `json:"match_id"`
	DurationSec int64                 `json:"duration_sec"`
	Winner      string                `json:"winner"`
	Formula     formula.Preset        `json:"formula"`
	Players     []assistantPlayerInfo `json:"players"`
}

type assistantRanking struct {
	PlayerID    int     `json:"player_id"`
	Hero        string  `json:"hero"`
	Team        string  `json:"team"`
	Score       float64 `json:"score"`
	GlobalPlace int     `json:"global_place"`
	Name        string  `json:"name,omitempty"`
}

func (a *App) assistantTools() []assistant.Tool {
	return []assistant.Tool{
		{
			Name:        "get_match_context",
			Description: "Вернуть текущий матч, активную формулу и статистику всех игроков.",
			Parameters:  objectSchema(nil),
			Run: func(ctx context.Context, args map[string]any) (any, error) {
				return a.toolMatchContext()
			},
		},
		{
			Name:        "get_player",
			Description: "Вернуть полную статистику одного игрока по player_id.",
			Parameters: objectSchema(map[string]any{
				"player_id": map[string]any{"type": "integer", "description": "player_id от 0 до 9"},
			}, "player_id"),
			Run: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := intArg(args, "player_id")
				if err != nil {
					return nil, err
				}
				return a.toolPlayer(id)
			},
		},
		{
			Name:        "get_active_formula",
			Description: "Вернуть активную формулу (пресет).",
			Parameters:  objectSchema(nil),
			Run: func(ctx context.Context, args map[string]any) (any, error) {
				return a.store.Active(), nil
			},
		},
		{
			Name:        "list_formulas",
			Description: "Вернуть все сохранённые формулы.",
			Parameters:  objectSchema(nil),
			Run: func(ctx context.Context, args map[string]any) (any, error) {
				return a.store.Presets(), nil
			},
		},
		{
			Name:        "get_token_catalog",
			Description: "Вернуть список доступных токенов формулы с подписями.",
			Parameters:  objectSchema(nil),
			Run: func(ctx context.Context, args map[string]any) (any, error) {
				return formula.Stats, nil
			},
		},
		{
			Name:        "list_variables",
			Description: "Вернуть пользовательские переменные.",
			Parameters:  objectSchema(nil),
			Run: func(ctx context.Context, args map[string]any) (any, error) {
				return a.varStore.Variables(), nil
			},
		},
		{
			Name:        "validate_formula",
			Description: "Проверить синтаксис и токены формулы.",
			Parameters: objectSchema(map[string]any{
				"expression": map[string]any{"type": "string"},
			}, "expression"),
			Run: func(ctx context.Context, args map[string]any) (any, error) {
				expr := stringArg(args, "expression")
				if err := formula.Validate(expr, a.varStore.AllowedNames()); err != nil {
					return map[string]any{"valid": false, "error": err.Error()}, nil
				}
				return map[string]any{"valid": true}, nil
			},
		},
		{
			Name:        "preview_formula",
			Description: "Оценить формулу на текущем матче и вернуть ранжирование.",
			Parameters: objectSchema(map[string]any{
				"expression": map[string]any{"type": "string"},
			}, "expression"),
			Run: func(ctx context.Context, args map[string]any) (any, error) {
				return a.toolPreview(stringArg(args, "expression"))
			},
		},
		{
			Name:        "explain_player",
			Description: "Показать вклад слагаемых формулы для одного игрока.",
			Parameters: objectSchema(map[string]any{
				"expression": map[string]any{"type": "string"},
				"player_id":  map[string]any{"type": "integer", "description": "player_id от 0 до 9"},
			}, "expression", "player_id"),
			Run: func(ctx context.Context, args map[string]any) (any, error) {
				id, err := intArg(args, "player_id")
				if err != nil {
					return nil, err
				}
				return a.toolExplain(stringArg(args, "expression"), id)
			},
		},
		{
			Name:        "compare_formulas",
			Description: "Сравнить ранжирование двух формул на текущем матче.",
			Parameters: objectSchema(map[string]any{
				"expression_a": map[string]any{"type": "string"},
				"expression_b": map[string]any{"type": "string"},
			}, "expression_a", "expression_b"),
			Run: func(ctx context.Context, args map[string]any) (any, error) {
				return a.toolCompare(stringArg(args, "expression_a"), stringArg(args, "expression_b"))
			},
		},
		{
			Name:        "propose_formula",
			Description: "Предложить формулу. Сохраняет только предложение, не меняет пресеты.",
			Parameters: objectSchema(map[string]any{
				"name":        map[string]any{"type": "string"},
				"description": map[string]any{"type": "string"},
				"expression":  map[string]any{"type": "string"},
			}, "name", "expression"),
			Run: func(ctx context.Context, args map[string]any) (any, error) {
				expr := stringArg(args, "expression")
				if err := formula.Validate(expr, a.varStore.AllowedNames()); err != nil {
					return nil, fmt.Errorf("формула не прошла валидацию: %v", err)
				}
				name := stringArg(args, "name")
				if strings.TrimSpace(name) == "" {
					name = "Предложение ассистента"
				}
				proposal := &AssistantProposal{
					Name: name, Description: stringArg(args, "description"), Expression: expr,
				}
				a.assistantMu.Lock()
				a.assistantProposal = proposal
				a.assistantMu.Unlock()
				runtime.EventsEmit(a.ctx, "assistantProposal", proposal)
				return map[string]any{"accepted": true}, nil
			},
		},
	}
}

func (a *App) toolMatchContext() (assistantMatchInfo, error) {
	result, views, err := a.assistantViews()
	if err != nil {
		return assistantMatchInfo{}, err
	}
	showNames := a.assistantShowNames()
	players := make([]assistantPlayerInfo, 0, len(views))
	for _, v := range views {
		players = append(players, assistantPlayerInfo{
			PlayerID: v.PlayerID, Hero: v.Hero, Team: v.Team, Position: v.Position, Lane: v.Lane,
			Score: v.Score, GlobalPlace: v.GlobalPlace, TeamPlace: v.TeamPlace, MvpRole: v.MvpRole,
			Name: optionalName(v.Name, showNames), SteamID: optionalSteamID(v.SteamID, showNames),
			Stats: compactMatchStats(v.Stats),
		})
	}
	return assistantMatchInfo{
		MatchID: result.MatchID, DurationSec: result.DurationSec, Winner: result.WinnerTeam(),
		Formula: a.store.Active(), Players: players,
	}, nil
}

func (a *App) toolPlayer(playerID int) (assistantPlayerInfo, error) {
	_, views, err := a.assistantViews()
	if err != nil {
		return assistantPlayerInfo{}, err
	}
	showNames := a.assistantShowNames()
	for _, v := range views {
		if v.PlayerID != playerID {
			continue
		}
		return assistantPlayerInfo{
			PlayerID: v.PlayerID, Hero: v.Hero, Team: v.Team, Position: v.Position, Lane: v.Lane,
			Score: v.Score, GlobalPlace: v.GlobalPlace, TeamPlace: v.TeamPlace, MvpRole: v.MvpRole,
			Name: optionalName(v.Name, showNames), SteamID: optionalSteamID(v.SteamID, showNames),
			Stats: v.Stats,
		}, nil
	}
	return assistantPlayerInfo{}, fmt.Errorf("игрок %d не найден", playerID)
}

func (a *App) toolPreview(expression string) ([]assistantRanking, error) {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return nil, errors.New("пустое выражение")
	}
	if err := formula.Validate(expression, a.varStore.AllowedNames()); err != nil {
		return nil, err
	}
	result, err := a.assistantResult()
	if err != nil {
		return nil, err
	}
	preset := formula.Preset{ID: "__assistant", Name: "__assistant", Kind: "expression", Expression: expression}
	views, err := BuildViews(*result, preset, a.varStore.Variables())
	if err != nil {
		return nil, err
	}
	showNames := a.assistantShowNames()
	out := make([]assistantRanking, 0, len(views))
	for _, v := range views {
		out = append(out, assistantRanking{
			PlayerID: v.PlayerID, Hero: v.Hero, Team: v.Team, Score: v.Score,
			GlobalPlace: v.GlobalPlace, Name: optionalName(v.Name, showNames),
		})
	}
	return out, nil
}

func (a *App) toolExplain(expression string, playerID int) (any, error) {
	result, err := a.assistantResult()
	if err != nil {
		return nil, err
	}
	var player *model.Player
	for i := range result.Players {
		if result.Players[i].PlayerID == playerID {
			player = &result.Players[i]
			break
		}
	}
	if player == nil {
		return nil, fmt.Errorf("игрок %d не найден", playerID)
	}
	env := mvp.PlayerVars(*player, float64(result.DurationSec))
	rows, err := formula.Breakdown(expression, env)
	if err != nil {
		return nil, err
	}
	return map[string]any{"player_id": playerID, "breakdown": rows}, nil
}

func (a *App) toolCompare(aExpr, bExpr string) (any, error) {
	aRanks, err := a.toolPreview(aExpr)
	if err != nil {
		return nil, fmt.Errorf("формула A: %v", err)
	}
	bRanks, err := a.toolPreview(bExpr)
	if err != nil {
		return nil, fmt.Errorf("формула B: %v", err)
	}
	aPlace := map[int]int{}
	for _, r := range aRanks {
		aPlace[r.PlayerID] = r.GlobalPlace
	}
	bPlace := map[int]int{}
	for _, r := range bRanks {
		bPlace[r.PlayerID] = r.GlobalPlace
	}
	changes := make([]map[string]any, 0, len(aRanks))
	for _, r := range aRanks {
		before := aPlace[r.PlayerID]
		after := bPlace[r.PlayerID]
		if before == after {
			continue
		}
		changes = append(changes, map[string]any{
			"player_id": r.PlayerID, "hero": r.Hero, "before": before, "after": after,
		})
	}
	return map[string]any{
		"formula_a": aRanks, "formula_b": bRanks, "rank_changes": changes,
	}, nil
}

var compactStats = []string{
	"kills", "deaths", "assists", "last_hits", "gpm", "xpm", "networth",
	"healing", "mana_restored", "hero_damage", "damage_taken", "tower_damage",
	"stun_duration", "save_duration", "purge_count", "shield_duration",
	"buff_duration", "heal_duration", "heal_value", "time_dead", "gold_lost",
	"camps_stacked", "rune_pickups", "wisdoms_captured", "watchers_captured",
	"lotuses_gathered", "courier_kills", "first_blood", "match_duration", "position",
}

func compactMatchStats(stats map[string]float64) map[string]float64 {
	out := make(map[string]float64, len(compactStats))
	for _, key := range compactStats {
		if value, ok := stats[key]; ok {
			out[key] = value
		}
	}
	return out
}

func (a *App) assistantResult() (*model.Result, error) {
	a.resultMu.Lock()
	defer a.resultMu.Unlock()
	if a.result == nil {
		return nil, errors.New("сначала откройте реплей")
	}
	copyResult := *a.result
	copyResult.Players = append([]model.Player(nil), a.result.Players...)
	return &copyResult, nil
}

func (a *App) assistantViews() (*model.Result, []PlayerView, error) {
	result, err := a.assistantResult()
	if err != nil {
		return nil, nil, err
	}
	views, err := BuildViews(*result, a.store.Active(), a.varStore.Variables())
	if err != nil {
		return nil, nil, err
	}
	return result, views, nil
}

func (a *App) assistantShowNames() bool {
	a.assistantMu.Lock()
	defer a.assistantMu.Unlock()
	return a.assistantSettings.SendPlayerNames
}

func optionalName(name string, show bool) string {
	if show {
		return name
	}
	return ""
}

func optionalSteamID(id int64, show bool) int64 {
	if show {
		return id
	}
	return 0
}

func stringArg(args map[string]any, key string) string {
	if v, ok := args[key].(string); ok {
		return v
	}
	return ""
}

func intArg(args map[string]any, key string) (int, error) {
	switch v := args[key].(type) {
	case float64:
		return int(v), nil
	case int:
		return v, nil
	default:
		return 0, fmt.Errorf("параметр %s должен быть числом", key)
	}
}

func objectSchema(properties map[string]any, required ...string) map[string]any {
	if properties == nil {
		properties = map[string]any{}
	}
	return map[string]any{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
		"required":             required,
	}
}
