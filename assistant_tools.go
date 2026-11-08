package main

import (
	"context"
	"errors"
	"fmt"
	"sort"
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
	Stats       map[string]float64 `json:"stats,omitempty"`
}

type assistantMvpInfo struct {
	Place       int     `json:"place"`
	Role        string  `json:"role"`
	Label       string  `json:"label"`
	PlayerID    int     `json:"player_id"`
	Hero        string  `json:"hero"`
	Team        string  `json:"team"`
	Score       float64 `json:"score"`
	GlobalPlace int     `json:"global_place"`
	TeamPlace   int     `json:"team_place"`
	Name        string  `json:"name,omitempty"`
}

type assistantMatchInfo struct {
	MatchID     int64                 `json:"match_id"`
	DurationSec int64                 `json:"duration_sec"`
	Winner      string                `json:"winner"`
	Formula     string                `json:"formula"`
	Players     []assistantPlayerInfo `json:"players"`
	MVPPodium   []assistantMvpInfo    `json:"mvp_podium"`
}

type assistantRanking struct {
	PlayerID    int     `json:"player_id"`
	Hero        string  `json:"hero"`
	Team        string  `json:"team"`
	Score       float64 `json:"score"`
	GlobalPlace int     `json:"global_place"`
	TeamPlace   int     `json:"team_place"`
	MvpRole     string  `json:"mvp_role,omitempty"`
	Name        string  `json:"name,omitempty"`
}

type assistantPreview struct {
	Ranking   []assistantRanking `json:"ranking"`
	MVPPodium []assistantMvpInfo `json:"mvp_podium"`
}

type assistantStatLeader struct {
	PlayerID    int     `json:"player_id"`
	Hero        string  `json:"hero"`
	Team        string  `json:"team"`
	Value       float64 `json:"value"`
	GlobalPlace int     `json:"global_place"`
	TeamPlace   int     `json:"team_place"`
	MvpRole     string  `json:"mvp_role,omitempty"`
	Name        string  `json:"name,omitempty"`
}

type assistantTokenInfo struct {
	Token       string `json:"token"`
	Label       string `json:"label"`
	Unit        string `json:"unit,omitempty"`
	Description string `json:"description"`
}

func (a *App) assistantTools() []assistant.Tool {
	return []assistant.Tool{
		{
			Name:        "get_match_context",
			Description: "Текущий матч: победитель, длительность, активная формула, все игроки с рейтингом и MVP-подиум. Вызывай один раз в начале анализа.",
			Parameters:  objectSchema(nil),
			Run: func(ctx context.Context, args map[string]any) (any, error) {
				return a.toolMatchContext()
			},
		},
		{
			Name:        "get_player",
			Description: "Полная статистика одного игрока. Используй для 1–2 ключевых игроков, не для всех.",
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
			Description: "Активная формула с выражением или весами. Вызывай, когда нужно объяснить текущую формулу.",
			Parameters:  objectSchema(nil),
			Run: func(ctx context.Context, args map[string]any) (any, error) {
				return a.store.Active(), nil
			},
		},
		{
			Name:        "list_formulas",
			Description: "Сохранённые формулы с выражениями и весами. Используй для сравнения или выбора существующей формулы.",
			Parameters:  objectSchema(nil),
			Run: func(ctx context.Context, args map[string]any) (any, error) {
				return a.store.Presets(), nil
			},
		},
		{
			Name:        "get_token_catalog",
			Description: "Подробное описание каждого токена: единицы и точный смысл. Обязателен перед созданием, критикой или существенным изменением формулы; не угадывай смысл по названию.",
			Parameters:  objectSchema(nil),
			Run: func(ctx context.Context, args map[string]any) (any, error) {
				return assistantTokenCatalog(), nil
			},
		},
		{
			Name:        "validate_formula",
			Description: "Проверить синтаксис, операторы и допустимые токены формулы. Вызывай до preview_formula и propose_formula.",
			Parameters: objectSchema(map[string]any{
				"expression": map[string]any{"type": "string"},
			}, "expression"),
			Run: func(ctx context.Context, args map[string]any) (any, error) {
				expr := stringArg(args, "expression")
				if err := formula.Validate(expr, formula.BaseTokens()); err != nil {
					result := map[string]any{"valid": false, "error": err.Error()}
					if hint := formulaValidationHint(err); hint != "" {
						result["hint"] = hint
					}
					return result, nil
				}
				return map[string]any{"valid": true}, nil
			},
		},
		{
			Name:        "preview_formula",
			Description: "Оценить формулу на текущем матче: рейтинг игроков и MVP-подиум. Вызывай до propose_formula.",
			Parameters: objectSchema(map[string]any{
				"expression": map[string]any{"type": "string"},
			}, "expression"),
			Run: func(ctx context.Context, args map[string]any) (any, error) {
				return a.toolPreview(stringArg(args, "expression"))
			},
		},
		{
			Name:        "explain_player",
			Description: "Показать вклад слагаемых формулы для одного игрока. Используй только для 1–2 ключевых игроков.",
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
			Description: "Сравнить две формулы: рейтинги, MVP-подиумы и изменения мест на текущем матче.",
			Parameters: objectSchema(map[string]any{
				"expression_a": map[string]any{"type": "string"},
				"expression_b": map[string]any{"type": "string"},
			}, "expression_a", "expression_b"),
			Run: func(ctx context.Context, args map[string]any) (any, error) {
				return a.toolCompare(stringArg(args, "expression_a"), stringArg(args, "expression_b"))
			},
		},
		{
			Name:        "diff_formulas",
			Description: "Показать добавленные и удалённые верхнеуровневые слагаемые между двумя формулами.",
			Parameters: objectSchema(map[string]any{
				"expression_a": map[string]any{"type": "string"},
				"expression_b": map[string]any{"type": "string"},
			}, "expression_a", "expression_b"),
			Run: func(ctx context.Context, args map[string]any) (any, error) {
				return a.toolDiff(stringArg(args, "expression_a"), stringArg(args, "expression_b"))
			},
		},
		{
			Name:        "get_stat_leaders",
			Description: "Отсортировать игроков по одному базовому токену. Используй для вопросов «кто лучший по X».",
			Parameters: objectSchema(map[string]any{
				"token": map[string]any{"type": "string", "description": "имя токена"},
				"team":  map[string]any{"type": "string", "enum": []string{"radiant", "dire"}, "description": "необязательно: только одна команда"},
				"limit": map[string]any{"type": "integer", "description": "сколько игроков вернуть, по умолчанию 5"},
			}, "token"),
			Run: func(ctx context.Context, args map[string]any) (any, error) {
				return a.toolStatLeaders(args)
			},
		},
		{
			Name:        "request_formula_change",
			Description: "Запросить создание или обновление формулы с подтверждением пользователя. Ничего не сохраняет до подтверждения. Приоритет — обновление активной пользовательской формулы; встроенную формулу можно только скопировать. Удаление токенов при обновлении запрещено без allow_token_removal.",
			Parameters: objectSchema(map[string]any{
				"expression":          map[string]any{"type": "string"},
				"name":                map[string]any{"type": "string"},
				"description":         map[string]any{"type": "string"},
				"mode":                map[string]any{"type": "string", "enum": []string{"auto", "update", "create"}, "description": "auto: обновить активную пользовательскую формулу или создать копию встроенной"},
				"preset_id":           map[string]any{"type": "string", "description": "необязательно: какую пользовательскую формулу обновить"},
				"allow_token_removal": map[string]any{"type": "boolean", "description": "true только если пользователь прямо попросил удалить токены"},
			}, "expression"),
			Run: func(ctx context.Context, args map[string]any) (any, error) {
				return a.stageAssistantFormulaChange(args)
			},
		},
		{
			Name:        "propose_formula",
			Description: "Предложить формулу после validate_formula и preview_formula. Сохраняет только предложение, не меняет пресеты.",
			Parameters: objectSchema(map[string]any{
				"name":        map[string]any{"type": "string"},
				"description": map[string]any{"type": "string"},
				"expression":  map[string]any{"type": "string"},
			}, "name", "expression"),
			Run: func(ctx context.Context, args map[string]any) (any, error) {
				expr := stringArg(args, "expression")
				if err := formula.Validate(expr, formula.BaseTokens()); err != nil {
					if hint := formulaValidationHint(err); hint != "" {
						return nil, fmt.Errorf("формула не прошла валидацию: %v. %s", err, hint)
					}
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
		})
	}
	return assistantMatchInfo{
		MatchID: result.MatchID, DurationSec: result.DurationSec, Winner: result.WinnerTeam(),
		Formula: a.store.Active().Name, Players: players, MVPPodium: buildMVPPodium(views, showNames),
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

func (a *App) toolPreview(expression string) (assistantPreview, error) {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return assistantPreview{}, errors.New("пустое выражение")
	}
	if err := formula.Validate(expression, formula.BaseTokens()); err != nil {
		return assistantPreview{}, err
	}
	result, err := a.assistantResult()
	if err != nil {
		return assistantPreview{}, err
	}
	preset := formula.Preset{ID: "__assistant", Name: "__assistant", Kind: "expression", Expression: expression}
	views, err := BuildViews(*result, preset)
	if err != nil {
		return assistantPreview{}, err
	}
	showNames := a.assistantShowNames()
	out := make([]assistantRanking, 0, len(views))
	for _, v := range views {
		out = append(out, assistantRanking{
			PlayerID: v.PlayerID, Hero: v.Hero, Team: v.Team, Score: v.Score,
			GlobalPlace: v.GlobalPlace, TeamPlace: v.TeamPlace, MvpRole: v.MvpRole,
			Name: optionalName(v.Name, showNames),
		})
	}
	return assistantPreview{Ranking: out, MVPPodium: buildMVPPodium(views, showNames)}, nil
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
	rows, err := formula.Breakdown(expression, mvp.PlayerVars(*player, float64(result.DurationSec)))
	if err != nil {
		return nil, err
	}
	return map[string]any{"player_id": playerID, "breakdown": rows}, nil
}

func (a *App) toolCompare(aExpr, bExpr string) (any, error) {
	aPreview, err := a.toolPreview(aExpr)
	if err != nil {
		return nil, fmt.Errorf("формула A: %v", err)
	}
	bPreview, err := a.toolPreview(bExpr)
	if err != nil {
		return nil, fmt.Errorf("формула B: %v", err)
	}
	aPlace := map[int]int{}
	for _, r := range aPreview.Ranking {
		aPlace[r.PlayerID] = r.GlobalPlace
	}
	bPlace := map[int]int{}
	for _, r := range bPreview.Ranking {
		bPlace[r.PlayerID] = r.GlobalPlace
	}
	changes := make([]map[string]any, 0, len(aPreview.Ranking))
	for _, r := range aPreview.Ranking {
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
		"formula_a": aPreview, "formula_b": bPreview, "rank_changes": changes,
	}, nil
}

var mvpRoleLabels = map[string]string{
	"winner_top1": "MVP матча",
	"loser_top1":  "Лучший в проигравшей команде",
	"winner_top2": "2-й лучший в победившей команде",
}

func buildMVPPodium(views []PlayerView, showNames bool) []assistantMvpInfo {
	byRole := make(map[string]PlayerView, 3)
	for _, view := range views {
		if view.MvpRole != "" {
			byRole[view.MvpRole] = view
		}
	}
	out := make([]assistantMvpInfo, 0, 3)
	for _, role := range []string{"winner_top1", "loser_top1", "winner_top2"} {
		view, ok := byRole[role]
		if !ok {
			continue
		}
		out = append(out, assistantMvpInfo{
			Place: len(out) + 1, Role: role, Label: mvpRoleLabels[role],
			PlayerID: view.PlayerID, Hero: view.Hero, Team: view.Team, Score: view.Score,
			GlobalPlace: view.GlobalPlace, TeamPlace: view.TeamPlace,
			Name: optionalName(view.Name, showNames),
		})
	}
	return out
}

func (a *App) toolDiff(aExpr, bExpr string) (FormulaDiff, error) {
	aExpr = strings.TrimSpace(aExpr)
	bExpr = strings.TrimSpace(bExpr)
	if aExpr == "" || bExpr == "" {
		return FormulaDiff{}, errors.New("нужны обе формулы")
	}
	allowed := formula.BaseTokens()
	if err := formula.Validate(aExpr, allowed); err != nil {
		return FormulaDiff{}, fmt.Errorf("формула A: %v", err)
	}
	if err := formula.Validate(bExpr, allowed); err != nil {
		return FormulaDiff{}, fmt.Errorf("формула B: %v", err)
	}
	return a.DiffFormula(aExpr, bExpr), nil
}

func (a *App) toolStatLeaders(args map[string]any) ([]assistantStatLeader, error) {
	token := strings.TrimSpace(stringArg(args, "token"))
	if token == "" {
		return nil, errors.New("укажите token")
	}
	if !formula.StatTokens(token) {
		return nil, fmt.Errorf("неизвестный токен: %s", token)
	}
	team := strings.ToLower(strings.TrimSpace(stringArg(args, "team")))
	if team != "" && team != "radiant" && team != "dire" {
		return nil, errors.New("team должен быть radiant или dire")
	}
	limit := 5
	if _, ok := args["limit"]; ok {
		n, err := intArg(args, "limit")
		if err != nil {
			return nil, err
		}
		limit = n
	}
	if limit <= 0 {
		limit = 5
	}
	if limit > 10 {
		limit = 10
	}

	_, views, err := a.assistantViews()
	if err != nil {
		return nil, err
	}
	showNames := a.assistantShowNames()
	out := make([]assistantStatLeader, 0, len(views))
	for _, view := range views {
		if team != "" && view.Team != team {
			continue
		}
		value, ok := view.Stats[token]
		if !ok {
			return nil, fmt.Errorf("токен %s недоступен", token)
		}
		out = append(out, assistantStatLeader{
			PlayerID: view.PlayerID, Hero: view.Hero, Team: view.Team, Value: value,
			GlobalPlace: view.GlobalPlace, TeamPlace: view.TeamPlace, MvpRole: view.MvpRole,
			Name: optionalName(view.Name, showNames),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Value == out[j].Value {
			return out[i].PlayerID < out[j].PlayerID
		}
		return out[i].Value > out[j].Value
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func assistantTokenCatalog() []assistantTokenInfo {
	out := make([]assistantTokenInfo, 0, len(formula.Stats))
	for _, stat := range formula.Stats {
		out = append(out, assistantTokenInfo{
			Token: stat.Token, Label: stat.Label,
			Unit: tokenUnits[stat.Token], Description: tokenDescriptions[stat.Token],
		})
	}
	return out
}

var tokenUnits = map[string]string{
	"kills": "count", "deaths": "count", "assists": "count", "last_hits": "count",
	"gpm": "gold/min", "xpm": "xp/min", "stun_duration": "seconds",
	"healing": "HP", "mana_restored": "mana", "tower_damage": "HP",
	"camps_stacked": "count", "rune_pickups": "count", "first_blood": "0/1",
	"hero_damage": "HP", "damage_taken": "HP",
	"gold_spent_wards": "gold", "gold_spent_smoke": "gold", "gold_spent_dust": "gold", "gold_lost": "gold",
	"buff_duration": "seconds", "save_duration": "seconds", "purge_count": "count",
	"shield_duration": "seconds", "buff_stats_duration": "seconds",
	"invisibility_duration": "seconds", "buff_haste_duration": "seconds",
	"fear_duration": "seconds", "roots_duration": "seconds", "leash_duration": "seconds",
	"trap_duration": "seconds", "taunt_duration": "seconds", "silence_duration": "seconds",
	"break_duration": "seconds", "disarm_duration": "seconds",
	"heal_duration": "seconds", "heal_value": "HP", "time_dead": "seconds",
	"creeps_stacked": "count", "wisdoms_captured": "count", "watchers_captured": "count",
	"lotuses_gathered": "count", "courier_kills": "count", "match_duration": "seconds",
	"position": "1-5",
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
	views, err := BuildViews(*result, a.store.Active())
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

func boolArg(args map[string]any, key string) bool {
	v, _ := args[key].(bool)
	return v
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

func formulaValidationHint(err error) string {
	lower := strings.ToLower(err.Error())
	switch {
	case strings.Contains(lower, "bool"):
		return "Логическое условие нельзя умножать или складывать с числом. Используй if condition { A } else { B } или condition ? A : B."
	case strings.Contains(lower, "callnode"):
		return "Условие записывается как if condition { A } else { B } или condition ? A : B, без if(condition, A, B)."
	case strings.Contains(lower, "неизвестная переменная"):
		return "Используй только токены из get_token_catalog или объяви локальную переменную через let."
	default:
		return ""
	}
}

func objectSchema(properties map[string]any, required ...string) map[string]any {
	if properties == nil {
		return nil
	}
	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}
