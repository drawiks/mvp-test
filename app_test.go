package main

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	"mvp/internal/assistant"
	"mvp/internal/formula"
	"mvp/internal/mvp"
	"mvp/internal/parse"
)

func fixture(t *testing.T) *App {
	t.Helper()
	data, err := os.ReadFile("internal/testdata/match.json")
	if err != nil {
		t.Fatal(err)
	}
	result, err := parse.ReadAggregatedJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	return &App{result: &result, store: formula.NewStore(""), varStore: formula.NewVariableStore("")}
}

func TestBuildViewsStandard(t *testing.T) {
	a := fixture(t)
	views, err := BuildViews(*a.result, a.store.Active(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 10 {
		t.Fatalf("len %d", len(views))
	}
	winners := 0
	for _, v := range views {
		if v.IsWinner {
			winners++
		}
		if v.GlobalPlace < 1 || v.GlobalPlace > 10 {
			t.Fatal("bad global place")
		}
		if v.Score < 0 {
			t.Fatal("negative score")
		}
	}
	if winners != 5 {
		t.Fatalf("winners %d", winners)
	}
	roles := map[string]bool{}
	for _, v := range views {
		if v.MvpRole != "" {
			roles[v.MvpRole] = true
		}
	}
	for _, want := range []string{"winner_top1", "winner_top2", "loser_top1"} {
		if !roles[want] {
			t.Fatalf("role %s missing", want)
		}
	}
}

func TestBuildViewsTopScores(t *testing.T) {
	a := fixture(t)
	views, err := BuildViews(*a.result, formula.Preset{
		ID: "test", Name: "test", Kind: "linear",
		Weights: mvp.WeightsToMapping(mvp.DefaultWeights),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	byPlace := map[int]string{}
	for _, v := range views {
		byPlace[v.GlobalPlace] = v.Name
	}
	want := map[int]string{
		1: "oyoy", 2: "mvhoyeti", 3: "юный дебустер", 4: "Мясное пюре",
		8: "а за мат извини", 9: "Daniamaps", 10: "re_triger",
	}
	for place, name := range want {
		if byPlace[place] != name {
			t.Fatalf("place %d = %s, want %s", place, byPlace[place], name)
		}
	}
}

func TestAppParserURLRoundtrip(t *testing.T) {
	a := &App{}
	if a.GetParserURL() != "" {
		t.Fatal("expected empty before startup")
	}
	_ = a
}

func TestAssistantTools(t *testing.T) {
	a := fixture(t)
	a.assistantSettings = assistant.DefaultSettings()

	ctx, err := a.toolMatchContext()
	if err != nil {
		t.Fatal(err)
	}
	if len(ctx.Players) != 10 {
		t.Fatalf("players %d", len(ctx.Players))
	}
	if ctx.Players[0].Name != "" || ctx.Players[0].SteamID != 0 {
		t.Fatalf("names must be masked: %+v", ctx.Players[0])
	}
	if _, err := a.toolPreview("kills * 2 + assists"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.toolPreview("kills * +"); err == nil {
		t.Fatal("invalid formula must fail")
	}
	a.assistantSettings.SendPlayerNames = true
	player, err := a.toolPlayer(ctx.Players[0].PlayerID)
	if err != nil {
		t.Fatal(err)
	}
	if player.Name == "" || player.SteamID == 0 {
		t.Fatalf("names must be included: %+v", player)
	}
	if _, err := a.toolExplain("kills * 2 + assists", ctx.Players[0].PlayerID); err != nil {
		t.Fatal(err)
	}
	if _, err := a.toolCompare("kills", "assists"); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateMethodsWhenDev(t *testing.T) {
	a := &App{}
	if a.GetAppVersion() != version {
		t.Fatalf("version %q", a.GetAppVersion())
	}
	info, err := a.CheckForUpdates()
	if err != nil {
		t.Fatal(err)
	}
	if info.CurrentVersion != version || info.Available {
		t.Fatalf("info %+v", info)
	}
	if err := a.InstallUpdate(); err == nil {
		t.Fatal("want error without a checked release")
	}
}

func TestSetParserEngine(t *testing.T) {
	a := &App{configPath: filepath.Join(t.TempDir(), "config.json"), parserURL: parserURLDefault}
	if err := a.SetParserEngine("bogus"); err == nil {
		t.Fatal("want error for unknown engine")
	}
	if err := a.SetParserEngine("manta"); err != nil {
		t.Fatal(err)
	}
	if a.GetParserEngine() != "manta" {
		t.Fatalf("engine %q", a.GetParserEngine())
	}
	data, err := os.ReadFile(a.configPath)
	if err != nil {
		t.Fatal(err)
	}
	var cfg appConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.ParserEngine != "manta" || cfg.ParserURL != parserURLDefault {
		t.Fatalf("config %+v", cfg)
	}
}

func TestEvalVariableAgainstLoadedPlayer(t *testing.T) {
	a := fixture(t)
	// Use a stored disabled variable store; find first player's base stats.
	p := a.result.Players[0]
	v, err := a.EvalVariable("dead_ratio", "time_dead / max(deaths, 1)", mvp.PlayerVars(p, 0))
	if err != nil {
		t.Fatal(err)
	}
	if v < 0 {
		t.Fatalf("negative ratio %v", v)
	}
}

func TestEvalVariableRejectsUnknownToken(t *testing.T) {
	a := fixture(t)
	_, err := a.EvalVariable("x", "time_dead / nonsense", nil)
	if err == nil {
		t.Fatal("want error for unknown token")
	}
}

func TestTestPlayerStatsNonEmpty(t *testing.T) {
	a := NewApp()
	s := a.TestPlayerStats()
	if s["kills"] != 10 || s["deaths"] != 4 || s["time_dead"] != 480 {
		t.Fatalf("unexpected test stats: %+v", s)
	}
	if s["position"] != 2 {
		t.Fatalf("position %v, want 2", s["position"])
	}
}

func TestUpsertPresetAllowsUserVariables(t *testing.T) {
	a := fixture(t)
	// Register a user variable that the preset's expression will reference.
	a.varStore.Add(formula.Variable{Name: "Initiation_Density", Expression: "kills * 2"})
	preset := formula.Preset{ID: "p", Name: "P", Kind: "expression", Expression: "kills * 2 + Initiation_Density"}
	if err := a.UpsertPreset(preset); err != nil {
		t.Fatalf("upsert with user variable failed: %v", err)
	}
	got, ok := a.store.Get("p")
	if !ok || got.Expression != "kills * 2 + Initiation_Density" {
		t.Fatalf("preset not persisted: %+v ok=%v", got, ok)
	}
}

func TestUpsertPresetRejectsUnknownToken(t *testing.T) {
	a := fixture(t)
	preset := formula.Preset{ID: "p", Name: "P", Kind: "expression", Expression: "kills * nonexistent_x"}
	if err := a.UpsertPreset(preset); err == nil {
		t.Fatal("want error for unknown token")
	}
}

func TestEvalBreakdown(t *testing.T) {
	a := fixture(t)
	p := a.result.Players[0]
	rows, err := a.EvalBreakdown("kills * 2;\n assists * 3", p.PlayerID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows %d", len(rows))
	}
	var sum float64
	for _, r := range rows {
		sum += r.Value
	}
	total, err := a.EvalVariable("total", "kills * 2 + assists * 3", mvp.PlayerVars(p, 0))
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(sum-total) > 1e-9 {
		t.Fatalf("sum %v != total %v", sum, total)
	}
	if got, err := a.EvalBreakdown("kills * 2", 999999); err == nil || got != nil {
		t.Fatal("unknown steamID should error")
	}
}
