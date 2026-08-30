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

// App is the Wails application root. Every exported method is bindable.
type App struct {
	ctx         context.Context
	store       *formula.Store
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

// startup initialises the app: config dir, preset store, parser URL.
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

// ---- Replays ----

// ParseReplay starts async parsing of a replay. Rows of progress are emitted
// as "parseProgress", completion as "parseDone".
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

// CancelParse aborts the in-flight parse, if any.
func (a *App) CancelParse() {
	a.parseMu.Lock()
	defer a.parseMu.Unlock()
	if a.parseCancel != nil {
		a.parseCancel()
		a.parseCancel = nil
	}
}

// ChooseReplay opens the native file dialog and starts parsing the selection.
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

// GetParserURL returns the configured parser endpoint.
func (a *App) GetParserURL() string { return a.parserURL }

// SetParserURL updates and persists the parser endpoint.
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

// ---- Presets ----

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

// UpsertPreset adds or replaces a preset after validation.
func (a *App) UpsertPreset(p formula.Preset) error {
	if err := a.store.ValidatePreset(p); err != nil {
		return err
	}
	if p.ID == "standard" || p.ID == "standard_v2" {
		return errors.New("встроенные пресеты нельзя изменять")
	}
	a.store.Upsert(p)
	a.store.Save()
	return nil
}

func (a *App) RemovePreset(id string) (bool, error) {
	if id == "standard" || id == "standard_v2" {
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

// ImportPresetDialog imports a preset through the native file dialog.
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

// ExportPresetDialog exports the preset through the native save dialog.
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

// CurrentPreset returns the active preset (for binding).
func (a *App) CurrentPreset() formula.Preset { return a.store.Active() }

// ---- Scores ----

// EvalPreview re-scores every played hero with the given expression without
// changing the active preset.
func (a *App) EvalPreview(expr string) ([]PlayerView, error) {
	a.resultMu.Lock()
	defer a.resultMu.Unlock()
	if a.result == nil {
		return nil, errors.New("сначала откройте реплей")
	}
	preset := formula.Preset{ID: "__preview", Name: "__preview", Kind: "expression", Expression: expr}
	return BuildViews(*a.result, preset)
}

// Recompute re-scores the stored result with the active preset and emits
// "resultUpdated".
func (a *App) Recompute() { a.EmitResult() }

// EmitResult broadcasts the current result + active preset scores to the UI.
func (a *App) EmitResult() {
	a.resultMu.Lock()
	defer a.resultMu.Unlock()
	if a.result == nil {
		return
	}
	views, err := BuildViews(*a.result, a.store.Active())
	if err != nil {
		runtime.LogError(a.ctx, err.Error())
		return
	}
	runtime.EventsEmit(a.ctx, "resultUpdated", views)
	runtime.EventsEmit(a.ctx, "matchInfo", matchInfoOf(*a.result, a.store.Active().Name))
}

// MatchInfo is the header bar summary of the loaded replay.
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

// HeroImageURL returns a CDN url of the hero portrait.
func (a *App) HeroImageURL(hero string) string {
	if hero == "" {
		return ""
	}
	return "https://cdn.cloudflare.steamstatic.com/apps/dota2/images/dota_react/heroes/" + hero + ".png"
}

func BuildViews(result model.Result, preset formula.Preset) ([]PlayerView, error) {
	ranked, err := mvp.RankedPlayers(result, preset)
	if err != nil {
		return nil, err
	}
	globalPlace := map[int64]int{}
	for i, p := range ranked {
		globalPlace[p.SteamID] = i + 1
	}
	teamPlace := map[int64]int{}
	for _, team := range []string{result.WinnerTeam(), result.LoserTeam()} {
		list, err := mvp.RankTeam(result, team, preset)
		if err != nil {
			return nil, err
		}
		for i, p := range list {
			teamPlace[p.SteamID] = i + 1
		}
	}
	mvps, err := mvp.SelectMvps(result, preset)
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
	for _, p := range result.Players {
		if p.Team != "radiant" && p.Team != "dire" {
			continue
		}
		score, err := mvp.ComputeScore(p, preset)
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
			Stats:       mvp.PlayerVars(p),
		})
	}
	return views, nil
}

// PlayerView is a player flattened with scoring metadata.
type PlayerView struct {
	model.Player
	Score       float64            `json:"Score"`
	TeamPlace   int                `json:"TeamPlace"`
	GlobalPlace int                `json:"GlobalPlace"`
	IsWinner    bool               `json:"IsWinner"`
	MvpRole     string             `json:"MvpRole"`
	Stats       map[string]float64 `json:"Stats"`
}
