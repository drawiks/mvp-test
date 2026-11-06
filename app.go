package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"mvp/internal/assistant"
	"mvp/internal/formula"
	"mvp/internal/model"
	"mvp/internal/mvp"
	"mvp/internal/parse"
	"mvp/internal/updater"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const parserURLDefault = "http://localhost:5600"

type App struct {
	ctx          context.Context
	store        *formula.Store
	configPath   string
	parserURL    string
	parserEngine string
	result       *model.Result
	resultMu     sync.Mutex
	parseCancel  context.CancelFunc
	parseMu      sync.Mutex

	updateMu      sync.Mutex
	updateRelease *updater.Release
	updating      bool

	assistantSettings    assistant.Settings
	assistantMu          sync.Mutex
	assistantHistory     []assistant.Message
	assistantRunning     bool
	assistantCancel      context.CancelFunc
	assistantGeneration  int
	assistantProposal    *AssistantProposal
	assistantHistoryPath string

	googleMu        sync.Mutex
	googleKeys      []string
	googleKeyStates []googleKeyState
	googleKeyIndex  int
}

type appConfig struct {
	ParserURL     string             `json:"parser_url"`
	ParserEngine  string             `json:"parser_engine"`
	Assistant     assistant.Settings `json:"assistant"`
	GoogleKeysEnc []string           `json:"google_keys_enc,omitempty"`
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	updater.CleanupOld()
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		cfgDir = "."
	}
	dir := filepath.Join(cfgDir, "mvp")
	_ = os.MkdirAll(dir, 0o755)
	a.configPath = filepath.Join(dir, "config.json")
	a.assistantHistoryPath = filepath.Join(dir, "assistant-history.json")
	a.store = formula.NewStore(filepath.Join(dir, "formulas.json"))
	_ = os.Remove(filepath.Join(dir, "variables.json"))
	_ = os.Remove(filepath.Join(dir, "variables.json.tmp"))
	if err := a.store.ValidatePreset(a.store.Active()); err != nil {
		a.store.SetActive("standard_v2")
		a.store.Save()
	}
	cfg := appConfig{
		ParserURL:    parserURLDefault,
		ParserEngine: parse.EngineOdota,
		Assistant:    assistant.DefaultSettings(),
	}
	if data, err := os.ReadFile(a.configPath); err == nil {
		if err := json.Unmarshal(data, &cfg); err != nil {
			cfg = appConfig{ParserURL: parserURLDefault, ParserEngine: parse.EngineOdota, Assistant: assistant.DefaultSettings()}
		}
	}
	if cfg.ParserURL == "" {
		cfg.ParserURL = parserURLDefault
	}
	if cfg.ParserEngine != parse.EngineManta {
		cfg.ParserEngine = parse.EngineOdota
	}
	validModel := false
	for _, id := range assistant.GoogleFreeModels {
		if id == cfg.Assistant.Model {
			validModel = true
			break
		}
	}
	if !validModel {
		cfg.Assistant.Model = assistant.DefaultModel()
	}
	if cfg.Assistant.MaxToolRounds <= 0 {
		cfg.Assistant.MaxToolRounds = 6
	}
	if cfg.Assistant.Temperature <= 0 || cfg.Assistant.Temperature > 2 {
		cfg.Assistant.Temperature = 0.2
	}
	a.parserURL = cfg.ParserURL
	a.parserEngine = cfg.ParserEngine
	a.assistantSettings = cfg.Assistant
	for _, enc := range cfg.GoogleKeysEnc {
		key, derr := assistant.DecryptSecret(enc)
		if derr != nil || key == "" {
			if derr != nil {
				runtime.LogError(ctx, "assistant google key: "+derr.Error())
			}
			continue
		}
		a.googleKeys = append(a.googleKeys, key)
		a.googleKeyStates = append(a.googleKeyStates, googleKeyState{})
	}
	a.loadAssistantHistory()
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

	parserURL, parserEngine := a.parserURL, a.parserEngine

	go func() {
		defer cancel()
		result, err := parse.ParseReplay(ctx, path, parserURL, parserEngine)
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

func (a *App) GetAppVersion() string { return version }

type UpdateInfo struct {
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	Available      bool   `json:"available"`
	Notes          string `json:"notes"`
}

func (a *App) CheckForUpdates() (UpdateInfo, error) {
	info := UpdateInfo{CurrentVersion: version}
	if version == "dev" {
		return info, nil
	}
	if a.ctx == nil {
		return info, errors.New("приложение ещё не запущено")
	}
	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()
	rel, ok, err := updater.Check(ctx, version)
	if err != nil {
		return info, err
	}

	a.updateMu.Lock()
	defer a.updateMu.Unlock()
	if !ok {
		a.updateRelease = nil
		return info, nil
	}
	info.LatestVersion = rel.Version
	info.Available = true
	info.Notes = rel.Notes
	a.updateRelease = &rel
	return info, nil
}

func (a *App) InstallUpdate() error {
	a.updateMu.Lock()
	rel := a.updateRelease
	if rel == nil {
		a.updateMu.Unlock()
		return errors.New("обновление не найдено")
	}
	if a.updating {
		a.updateMu.Unlock()
		return errors.New("обновление уже загружается")
	}
	a.updating = true
	a.updateMu.Unlock()

	go func() {
		defer func() {
			a.updateMu.Lock()
			a.updating = false
			a.updateMu.Unlock()
		}()

		lastPercent := -1
		progress := func(p updater.Progress) {
			percent := 0
			if p.Total > 0 {
				percent = int(p.Downloaded * 100 / p.Total)
			}
			if p.Stage == "download" && p.Total > 0 && percent == lastPercent {
				return
			}
			lastPercent = percent
			runtime.EventsEmit(a.ctx, "updateProgress", map[string]any{
				"stage": p.Stage, "downloaded": p.Downloaded, "total": p.Total, "percent": percent,
			})
		}

		ctx, cancel := context.WithTimeout(a.ctx, 15*time.Minute)
		defer cancel()
		path, err := updater.Download(ctx, *rel, progress)
		if err != nil {
			runtime.EventsEmit(a.ctx, "updateDone", map[string]any{"ok": false, "error": err.Error()})
			return
		}
		runtime.EventsEmit(a.ctx, "updateProgress", map[string]any{"stage": "apply", "percent": 100})
		if err := updater.ApplyAndRestart(path); err != nil {
			runtime.EventsEmit(a.ctx, "updateDone", map[string]any{"ok": false, "error": err.Error()})
			return
		}
		runtime.EventsEmit(a.ctx, "updateDone", map[string]any{"ok": true})
		runtime.Quit(a.ctx)
	}()
	return nil
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
	return a.saveConfig()
}

func (a *App) GetParserEngine() string { return a.parserEngine }

func (a *App) SetParserEngine(engine string) error {
	if engine != parse.EngineOdota && engine != parse.EngineManta {
		return errors.New("неизвестный движок парсинга: " + engine)
	}
	a.parserEngine = engine
	return a.saveConfig()
}

func (a *App) saveConfig() error {
	googleKeysEnc, err := a.googleKeysEncrypted()
	if err != nil {
		return err
	}
	data, err := json.Marshal(appConfig{
		ParserURL:     a.parserURL,
		ParserEngine:  a.parserEngine,
		Assistant:     a.assistantSettings,
		GoogleKeysEnc: googleKeysEnc,
	})
	if err != nil {
		return err
	}
	return os.WriteFile(a.configPath, data, 0o644)
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
	if err := a.store.ValidatePreset(p); err != nil {
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

func (a *App) CurrentPreset() formula.Preset { return a.store.Active() }

func (a *App) EvalPreview(expr string) ([]PlayerView, error) {
	a.resultMu.Lock()
	defer a.resultMu.Unlock()
	if a.result == nil {
		return nil, errors.New("сначала откройте реплей")
	}
	preset := formula.Preset{ID: "__preview", Name: "__preview", Kind: "expression", Expression: expr}
	return BuildViews(*a.result, preset)
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
	return formula.Breakdown(expression, mvp.PlayerVars(*player, float64(a.result.DurationSec)))
}

// ValidateExpression checks a formula expression without needing a loaded
// replay. It backs the in-editor linter so syntax errors surface immediately.
func (a *App) ValidateExpression(expression string) error {
	return formula.Validate(expression, formula.BaseTokens())
}

func (a *App) Recompute() { a.EmitResult() }

func (a *App) EmitResult() {
	if a.ctx == nil {
		return
	}
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
	duration := float64(result.DurationSec)
	for _, p := range result.Players {
		if p.Team != "radiant" && p.Team != "dire" {
			continue
		}
		score, err := mvp.ComputeScoreAt(p, duration, preset)
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
