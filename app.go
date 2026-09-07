package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"mvp/internal/formula"
	"mvp/internal/model"
	"mvp/internal/mvp"
	"mvp/internal/parse"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const parserURLDefault = "http://localhost:5600"

type App struct {
	ctx         context.Context
	store       *formula.Store
	varStore    *formula.VariableStore
	configPath  string
	parserURL   string
	result      *model.Result
	resultMu    sync.Mutex
	parseCancel context.CancelFunc
	parseMu     sync.Mutex
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		cfgDir = "."
	}
	dir := filepath.Join(cfgDir, "mvp")
	_ = os.MkdirAll(dir, 0o755)
	a.configPath = filepath.Join(dir, "config.json")
	a.store = formula.NewStore(filepath.Join(dir, "formulas.json"))
	a.varStore = formula.NewVariableStore(filepath.Join(dir, "variables.json"))
	a.parserURL = parserURLDefault
	if data, err := os.ReadFile(a.configPath); err == nil {
		var cfg struct {
			ParserURL string `json:"parser_url"`
		}
		if json.Unmarshal(data, &cfg) == nil && cfg.ParserURL != "" {
			a.parserURL = cfg.ParserURL
		}
	}
}

func (a *App) ParseReplay(path string) error {
	if a.ctx == nil {
		return errors.New("приложение ещё не запущено")
	}
	if _, err := os.Stat(path); err != nil {
		return errors.New("файл не найден: " + path)
	}
	a.parseMu.Lock()
	if a.parseCancel != nil {
		a.parseCancel()
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.parseCancel = cancel
	a.parseMu.Unlock()

	runtime.EventsEmit(a.ctx, "parseStart", map[string]any{"path": path})
	runtime.EventsEmit(a.ctx, "matchInfo", nil)

	go func() {
		defer cancel()
		result, err := parse.ParseReplay(ctx, path, a.parserURL)
		if err != nil {
			runtime.EventsEmit(a.ctx, "parseDone", map[string]any{"ok": false, "error": err.Error()})
			return
		}
		a.resultMu.Lock()
		a.result = &result
		a.resultMu.Unlock()
		a.EmitResult()
		runtime.EventsEmit(a.ctx, "parseDone", map[string]any{"ok": true})
	}()
	return nil
}

func (a *App) CancelParse() {
	a.parseMu.Lock()
	defer a.parseMu.Unlock()
	if a.parseCancel != nil {
		a.parseCancel()
		a.parseCancel = nil
	}
}

func (a *App) ChooseReplay() {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Выберите файл реплея",
		Filters: []runtime.FileFilter{
			{DisplayName: "Реплеи (*.dem, *.json, *.ndjson, *.bz2, *.zst)", Pattern: "*.dem;*.json;*.ndjson;*.bz2;*.zst;*.zip"},
			{DisplayName: "Все файлы", Pattern: "*"},
		},
	})
	if err != nil {
		return
	}
	if path == "" {
		return
	}
	if err := a.ParseReplay(path); err != nil {
		runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:    runtime.ErrorDialog,
			Title:   "Не удалось открыть",
			Message: err.Error(),
		})
	}
}

func (a *App) GetParserURL() string { return a.parserURL }

func (a *App) SetParserURL(url string) error {
	url = strings.TrimSpace(url)
	if url == "" {
		url = parserURLDefault
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return errors.New("URL должен начинаться с http:// или https://")
	}
	a.parserURL = strings.TrimSuffix(url, "/")
	cfg, _ := json.Marshal(map[string]any{"parser_url": a.parserURL})
	return os.WriteFile(a.configPath, cfg, 0o644)
}

func (a *App) ListPresets() []formula.Preset { return a.store.Presets() }

func (a *App) ActivePresetID() string { return a.store.Active().ID }

func (a *App) SetActivePreset(id string) error {
	if !a.store.SetActive(id) {
		return errors.New("пресет не найден: " + id)
	}
	a.store.Save()
	a.EmitResult()
	return nil
}

func (a *App) UpsertPreset(p formula.Preset) error {
	if err := a.store.ValidatePreset(p, a.varStore.AllowedNames()); err != nil {
		return err
	}
	if p.ID == "standard_v2" || p.ID == "tutorial" {
		return errors.New("встроенные пресеты нельзя изменять")
	}
	if a.store.NameTaken(strings.TrimSpace(p.Name), p.ID) {
		return errors.New("формула с таким названием уже существует")
	}
	a.store.Upsert(p)
	a.store.Save()
	return nil
}

func (a *App) RemovePreset(id string) (bool, error) {
	if id == "standard_v2" || id == "tutorial" {
		return false, errors.New("встроенные пресеты нельзя удалить")
	}
	ok := a.store.Remove(id)
	a.store.Save()
	return ok, nil
}

func (a *App) ExportPreset(id, dest string) error {
	p, ok := a.store.Get(id)
	if !ok {
		return errors.New("пресет не найден: " + id)
	}
	return a.store.ExportFile(p, dest)
}

func (a *App) ImportPreset(path string) (formula.Preset, error) {
	p, err := a.store.ImportFile(path)
	if err != nil {
		return formula.Preset{}, err
	}
	a.store.Save()
	return p, nil
}

func (a *App) ImportPresetDialog() (formula.Preset, bool) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Импорт пресета",
		Filters: []runtime.FileFilter{{DisplayName: "Пресеты (*.json)", Pattern: "*.json"}, {DisplayName: "Все файлы", Pattern: "*"}},
	})
	if err != nil || path == "" {
		return formula.Preset{}, false
	}
	p, err := a.store.ImportFile(path)
	if err != nil {
		runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{Type: runtime.ErrorDialog, Title: "Импорт", Message: err.Error()})
		return formula.Preset{}, false
	}
	a.store.Save()
	return p, true
}

func (a *App) ExportPresetDialog(id string) (bool, error) {
	p, ok := a.store.Get(id)
	if !ok {
		return false, errors.New("пресет не найден: " + id)
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Экспорт пресета",
		DefaultFilename: "preset.json",
		Filters:         []runtime.FileFilter{{DisplayName: "Пресеты (*.json)", Pattern: "*.json"}},
	})
	if err != nil || path == "" {
		return false, nil
	}
	if err := a.store.ExportFile(p, path); err != nil {
		runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{Type: runtime.ErrorDialog, Title: "Экспорт", Message: err.Error()})
		return false, err
	}
	return true, nil
}

func (a *App) ListVariables() []formula.Variable { return a.varStore.Variables() }

func (a *App) UpsertVariable(v formula.Variable) error {
	if err := a.varStore.ValidateVariable(v); err != nil {
		return err
	}
	if a.varStore.NameTaken(strings.TrimSpace(v.Name), v.ID) {
		return errors.New("переменная с таким названием уже существует")
	}
	a.varStore.Add(v)
	a.varStore.Save()
	a.EmitResult()
	return nil
}

func (a *App) RemoveVariable(id string) error {
	if !a.varStore.Remove(id) {
		return errors.New("переменная не найдена")
	}
	a.varStore.Save()
	a.EmitResult()
	return nil
}

func (a *App) EvalVariable(name, expr string, base map[string]float64) (float64, error) {
	if strings.TrimSpace(expr) == "" {
		return 0, errors.New("Введите выражение переменной")
	}
	allowed := a.varStore.AllowedNames()
	delete(allowed, name)
	if err := formula.Validate(expr, allowed); err != nil {
		return 0, err
	}
	env, err := formula.ResolveVars(base, a.varStore.Variables())
	if err != nil {
		return 0, err
	}
	return formula.Eval(expr, env)
}

func (a *App) TestPlayerStats() map[string]float64 {
	p := model.Player{
		Kills: 10, Deaths: 4, Assists: 7, LastHits: 250, GPM: 580, XPM: 640,
		Healing: 9000, HeroDamage: 20000, DamageTaken: 15000, TowerDamage: 3200,
		StunDuration: 45, CampsStacked: 9, RunePickups: 5, FirstBlood: true,
		GoldSpentWards: 500, GoldSpentSmoke: 100, GoldSpentDust: 50,
		BuffsDuration: 700, Save: 300, Purge: 120, ShieldUptime: 40,
		BuffStatsDuration: 200, InvisibilityDuration: 30, BuffHasteDuration: 60,
		CreepsStacked: 4, TimeDead: 480, Position: 2, Lane: "mid",
	}
	return mvp.PlayerVars(p, 3600)
}

func (a *App) ImportVariableDialog() (formula.Variable, bool) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Импорт переменной",
		Filters: []runtime.FileFilter{{DisplayName: "Переменные (*.json)", Pattern: "*.json"}, {DisplayName: "Все файлы", Pattern: "*"}},
	})
	if err != nil || path == "" {
		return formula.Variable{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{Type: runtime.ErrorDialog, Title: "Импорт", Message: "Не удалось прочитать файл: " + err.Error()})
		return formula.Variable{}, false
	}
	var v formula.Variable
	if err := json.Unmarshal(data, &v); err != nil {
		runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{Type: runtime.ErrorDialog, Title: "Импорт", Message: "Не удалось прочитать файл"})
		return formula.Variable{}, false
	}
	if err := a.varStore.ValidateVariable(v); err != nil {
		runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{Type: runtime.ErrorDialog, Title: "Импорт", Message: err.Error()})
		return formula.Variable{}, false
	}
	a.varStore.Add(v)
	a.varStore.Save()
	a.EmitResult()
	return v, true
}

func (a *App) ExportVariableDialog(id string) (bool, error) {
	target, ok := a.varStore.Get(id)
	if !ok {
		return false, errors.New("переменная не найдена")
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Экспорт переменной",
		DefaultFilename: "variable.json",
		Filters:         []runtime.FileFilter{{DisplayName: "Переменные (*.json)", Pattern: "*.json"}},
	})
	if err != nil || path == "" {
		return false, nil
	}
	body, err := json.MarshalIndent(target, "", "  ")
	if err != nil {
		return false, err
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return false, err
	}
	return true, nil
}

func (a *App) CurrentPreset() formula.Preset { return a.store.Active() }

func (a *App) EvalPreview(expr string) ([]PlayerView, error) {
	a.resultMu.Lock()
	defer a.resultMu.Unlock()
	if a.result == nil {
		return nil, errors.New("сначала откройте реплей")
	}
	preset := formula.Preset{ID: "__preview", Name: "__preview", Kind: "expression", Expression: expr}
	return BuildViews(*a.result, preset, a.varStore.Variables())
}

// EvalBreakdown evaluates each top-level term of an expression against a single
// player, returning one row per term (used by the score breakdown table).
func (a *App) EvalBreakdown(expression string, playerID int) ([]formula.BreakdownRow, error) {
	a.resultMu.Lock()
	defer a.resultMu.Unlock()
	if a.result == nil {
		return nil, errors.New("сначала откройте реплей")
	}
	var player *model.Player
	for i := range a.result.Players {
		if a.result.Players[i].PlayerID == playerID {
			player = &a.result.Players[i]
			break
		}
	}
	if player == nil {
		return nil, errors.New("игрок не найден")
	}
	env, err := formula.ResolveVars(mvp.PlayerVars(*player, float64(a.result.DurationSec)), a.varStore.Variables())
	if err != nil {
		return nil, err
	}
	return formula.Breakdown(expression, env)
}

// ValidateExpression checks a formula expression without needing a loaded
// replay. It backs the in-editor linter so syntax errors surface immediately.
func (a *App) ValidateExpression(expression string) error {
	return formula.Validate(expression, a.varStore.AllowedNames())
}

func (a *App) Recompute() { a.EmitResult() }

func (a *App) EmitResult() {
	a.resultMu.Lock()
	defer a.resultMu.Unlock()
	if a.result == nil {
		return
	}
	views, err := BuildViews(*a.result, a.store.Active(), a.varStore.Variables())
	if err != nil {
		runtime.LogError(a.ctx, err.Error())
		return
	}
	runtime.EventsEmit(a.ctx, "resultUpdated", views)
	runtime.EventsEmit(a.ctx, "matchInfo", matchInfoOf(*a.result, a.store.Active().Name))
}

type MatchInfo struct {
	MatchID      int64  `json:"matchId"`
	DurationSec  int64  `json:"durationSec"`
	RadiantScore int    `json:"radiantScore"`
	DireScore    int    `json:"direScore"`
	Winner       string `json:"winner"`
	PresetName   string `json:"presetName"`
}

func matchInfoOf(result model.Result, presetName string) MatchInfo {
	var radiant, dire int
	for _, p := range result.Players {
		if p.Team == "radiant" {
			radiant += p.Kills
		} else if p.Team == "dire" {
			dire += p.Kills
		}
	}
	return MatchInfo{
		MatchID:      result.MatchID,
		DurationSec:  result.DurationSec,
		RadiantScore: radiant,
		DireScore:    dire,
		Winner:       result.WinnerTeam(),
		PresetName:   presetName,
	}
}

func (a *App) HeroImageURL(hero string) string {
	if hero == "" {
		return ""
	}
	return "https://cdn.cloudflare.steamstatic.com/apps/dota2/images/dota_react/heroes/" + hero + ".png"
}

func BuildViews(result model.Result, preset formula.Preset, vars []formula.Variable) ([]PlayerView, error) {
	ranked, err := mvp.RankedPlayers(result, preset, vars)
	if err != nil {
		return nil, err
	}
	globalPlace := map[int64]int{}
	for i, p := range ranked {
		globalPlace[p.SteamID] = i + 1
	}
	teamPlace := map[int64]int{}
	for _, team := range []string{result.WinnerTeam(), result.LoserTeam()} {
		list, err := mvp.RankTeam(result, team, preset, vars)
		if err != nil {
			return nil, err
		}
		for i, p := range list {
			teamPlace[p.SteamID] = i + 1
		}
	}
	mvps, err := mvp.SelectMvps(result, preset, vars)
	if err != nil {
		return nil, err
	}
	mvpRole := map[int64]string{}
	for role, p := range mvps {
		if p != nil {
			mvpRole[p.SteamID] = role
		}
	}
	views := make([]PlayerView, 0, len(result.Players))
	duration := float64(result.DurationSec)
	for _, p := range result.Players {
		if p.Team != "radiant" && p.Team != "dire" {
			continue
		}
		score, err := mvp.ComputeScoreVars(p, duration, preset, vars)
		if err != nil {
			return nil, err
		}
		views = append(views, PlayerView{
			Player:      p,
			Score:       score,
			TeamPlace:   teamPlace[p.SteamID],
			GlobalPlace: globalPlace[p.SteamID],
			IsWinner:    p.Team == result.WinnerTeam(),
			MvpRole:     mvpRole[p.SteamID],
			Stats:       mvp.PlayerVars(p, duration),
		})
	}
	return views, nil
}

type PlayerView struct {
	model.Player
	Score       float64            `json:"Score"`
	TeamPlace   int                `json:"TeamPlace"`
	GlobalPlace int                `json:"GlobalPlace"`
	IsWinner    bool               `json:"IsWinner"`
	MvpRole     string             `json:"MvpRole"`
	Stats       map[string]float64 `json:"Stats"`
}
