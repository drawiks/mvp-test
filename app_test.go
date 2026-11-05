package main

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
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

func TestAssistantToolsUseGeminiSchemas(t *testing.T) {
	a := fixture(t)
	for _, tool := range a.assistantTools() {
		if tool.Parameters == nil {
			continue
		}
		if _, ok := tool.Parameters["additionalProperties"]; ok {
			t.Errorf("%s: additionalProperties is not supported by Gemini", tool.Name)
		}
		if required, ok := tool.Parameters["required"]; ok && required == nil {
			t.Errorf("%s: required must be a list, not null", tool.Name)
		}
	}
}

func TestAssistantMvpPodium(t *testing.T) {
	a := fixture(t)
	a.assistantSettings = assistant.DefaultSettings()
	ctx, err := a.toolMatchContext()
	if err != nil {
		t.Fatal(err)
	}
	wantRoles := []string{"winner_top1", "loser_top1", "winner_top2"}
	if len(ctx.MVPPodium) != len(wantRoles) {
		t.Fatalf("podium %+v", ctx.MVPPodium)
	}
	for i, role := range wantRoles {
		if ctx.MVPPodium[i].Role != role || ctx.MVPPodium[i].Place != i+1 {
			t.Fatalf("podium[%d] = %+v, want %s", i, ctx.MVPPodium[i], role)
		}
		if ctx.MVPPodium[i].Name != "" {
			t.Fatalf("player names must be masked: %+v", ctx.MVPPodium[i])
		}
	}
	if ctx.MVPPodium[0].Team != ctx.Winner {
		t.Fatalf("MVP team = %s, winner = %s", ctx.MVPPodium[0].Team, ctx.Winner)
	}
	if ctx.MVPPodium[1].Team == ctx.Winner {
		t.Fatalf("loser MVP team = %s", ctx.MVPPodium[1].Team)
	}
}

func TestAssistantPreviewMvp(t *testing.T) {
	a := fixture(t)
	preview, err := a.toolPreview("kills * 2 + assists")
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Ranking) != 10 || len(preview.MVPPodium) != 3 {
		t.Fatalf("preview %+v", preview)
	}
	wantRoles := []string{"winner_top1", "loser_top1", "winner_top2"}
	for i, role := range wantRoles {
		if preview.MVPPodium[i].Role != role {
			t.Fatalf("podium[%d] = %s, want %s", i, preview.MVPPodium[i].Role, role)
		}
	}
	for _, row := range preview.Ranking {
		if row.MvpRole != "" && row.TeamPlace == 0 {
			t.Fatalf("missing team place: %+v", row)
		}
	}
}

func TestAssistantStatLeaders(t *testing.T) {
	a := fixture(t)
	leaders, err := a.toolStatLeaders(map[string]any{"token": "kills", "limit": float64(3)})
	if err != nil {
		t.Fatal(err)
	}
	if len(leaders) != 3 {
		t.Fatalf("leaders %d", len(leaders))
	}
	for i := 1; i < len(leaders); i++ {
		if leaders[i-1].Value < leaders[i].Value {
			t.Fatalf("leaders not sorted: %+v", leaders)
		}
	}
	team, err := a.toolStatLeaders(map[string]any{"token": "kills", "team": "radiant"})
	if err != nil {
		t.Fatal(err)
	}
	for _, leader := range team {
		if leader.Team != "radiant" {
			t.Fatalf("team filter failed: %+v", leader)
		}
	}
}

func TestAssistantDiffFormulas(t *testing.T) {
	a := fixture(t)
	diff, err := a.toolDiff("kills * 2 + assists", "kills * 2 + deaths")
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Added) == 0 || len(diff.Removed) == 0 {
		t.Fatalf("diff %+v", diff)
	}
}

func TestAssistantTokenCatalogUnits(t *testing.T) {
	catalog := assistantTokenCatalog()
	byToken := map[string]assistantTokenInfo{}
	for _, token := range catalog {
		byToken[token.Token] = token
	}
	if byToken["stun_duration"].Unit != "seconds" {
		t.Fatalf("stun unit = %q", byToken["stun_duration"].Unit)
	}
	if byToken["position"].Note == "" {
		t.Fatal("position note missing")
	}
	data, err := json.Marshal(catalog)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) > 4000 {
		t.Fatalf("catalog too large: %d bytes", len(data))
	}
}

func TestAssistantExplainWithVariable(t *testing.T) {
	a := fixture(t)
	a.varStore.Add(formula.Variable{Name: "impact", Expression: "kills * 2 + assists"})
	_, err := a.toolExplain("impact + deaths", 0)
	if err != nil {
		t.Fatal(err)
	}
	player, err := a.toolPlayer(0)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := player.Variables["impact"]; !ok {
		t.Fatalf("resolved variables missing: %+v", player.Variables)
	}
}

func TestAssistantCompareFitsToolLimit(t *testing.T) {
	a := fixture(t)
	out, err := a.toolCompare("kills * 2 + assists", "kills + deaths")
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) > 4000 {
		t.Fatalf("compare result too large: %d bytes", len(data))
	}
}

func TestAssistantToolResultsFitLimit(t *testing.T) {
	a := fixture(t)
	a.assistantSettings = assistant.DefaultSettings()
	match, err := a.toolMatchContext()
	if err != nil {
		t.Fatal(err)
	}
	preview, err := a.toolPreview("kills * 2 + assists")
	if err != nil {
		t.Fatal(err)
	}
	leaders, err := a.toolStatLeaders(map[string]any{"token": "kills"})
	if err != nil {
		t.Fatal(err)
	}
	player, err := a.toolPlayer(0)
	if err != nil {
		t.Fatal(err)
	}
	compare, err := a.toolCompare("kills * 2 + assists", "kills + deaths")
	if err != nil {
		t.Fatal(err)
	}
	outputs := map[string]any{
		"match": match, "preview": preview, "leaders": leaders, "player": player,
		"compare": compare, "tokens": assistantTokenCatalog(),
		"formulas": a.store.Presets(), "variables": a.varStore.Variables(),
	}
	for name, output := range outputs {
		data, err := json.Marshal(output)
		if err != nil {
			t.Fatal(err)
		}
		if len(data) > 4000 {
			t.Errorf("%s result too large: %d bytes", name, len(data))
		}
	}
}

func TestDiffFormula(t *testing.T) {
	a := &App{}
	d := a.DiffFormula("kills * 2 + assists * 3", "kills * 2 + deaths * 1")
	if len(d.Added) != 1 || d.Added[0] != "deaths * 1" || len(d.Removed) != 1 || d.Removed[0] != "assists * 3" {
		t.Fatalf("diff %+v", d)
	}
}

func TestAssistantHistoryRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "assistant-history.json")
	a := &App{assistantHistoryPath: path}
	a.assistantMu.Lock()
	a.assistantHistory = []assistant.Message{{Role: "user", Content: "привет"}, {Role: "assistant", Content: "ответ"}}
	a.assistantProposal = &AssistantProposal{Name: "P", Expression: "kills"}
	a.saveAssistantHistoryLocked()
	a.assistantMu.Unlock()

	b := &App{assistantHistoryPath: path}
	b.loadAssistantHistory()
	h := b.GetAssistantHistory()
	if len(h.Messages) != 2 || h.Messages[1].Text != "ответ" || h.Proposal == nil || h.Proposal.Expression != "kills" {
		t.Fatalf("history %+v", h)
	}
}

func TestTrimHistoryByBudget(t *testing.T) {
	history := []assistant.Message{
		{Role: "user", Content: strings.Repeat("a", 1000)},
		{Role: "assistant", Content: "old"},
		{Role: "user", Content: "current question"},
		{Role: "tool", Content: strings.Repeat("b", 5000)},
	}
	got := trimHistoryByBudget(history, 3000)
	if len(got) == 0 || got[0].Role != "user" || got[0].Content != "current question" {
		t.Fatalf("history %+v", got)
	}
	if len(got[len(got)-1].Content) > 2100 {
		t.Fatalf("tool result not truncated: %d", len(got[len(got)-1].Content))
	}
}

func TestAssistantSettings(t *testing.T) {
	a := &App{
		configPath:        filepath.Join(t.TempDir(), "config.json"),
		assistantSettings: assistant.DefaultSettings(),
	}
	err := a.SaveAssistantSettings(AssistantSettings{Model: "gemini-3.5-flash-lite"})
	if err != nil {
		t.Fatal(err)
	}
	s := a.GetAssistantSettings()
	if s.Model != "gemini-3.5-flash-lite" {
		t.Fatalf("settings %+v", s)
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

func TestGoogleKeyRotation(t *testing.T) {
	a := &App{
		googleKeys:      []string{"key-1", "key-2"},
		googleKeyStates: []googleKeyState{{}, {}},
	}
	r := &appGoogleRotator{a: a}
	if r.Key() != "key-1" {
		t.Fatalf("key %q", r.Key())
	}
	next, ok := r.Rotate(&assistant.APIError{Status: 429, RetryAfter: "1"})
	if !ok || next != "key-2" {
		t.Fatalf("next %q ok %v", next, ok)
	}
	if r.Key() != "key-2" {
		t.Fatalf("key %q", r.Key())
	}
	infos := a.GetGoogleKeys()
	if len(infos) != 2 || infos[0].Status != "limited" || !infos[1].Active {
		t.Fatalf("infos %+v", infos)
	}
	if infos[0].Masked == "key-1" {
		t.Fatal("key must be masked")
	}
}

func TestGoogleKeyRotationInvalidKey(t *testing.T) {
	a := &App{
		googleKeys:      []string{"key-1", "key-2"},
		googleKeyStates: []googleKeyState{{}, {}},
	}
	r := &appGoogleRotator{a: a}
	next, ok := r.Rotate(&assistant.APIError{Status: 401, Body: "API key not valid"})
	if !ok || next != "key-2" {
		t.Fatalf("next %q ok %v", next, ok)
	}
	infos := a.GetGoogleKeys()
	if infos[0].Status != "invalid" || !infos[1].Active {
		t.Fatalf("infos %+v", infos)
	}
}

func TestGoogleKeyRotationSingleKey(t *testing.T) {
	a := &App{
		googleKeys:      []string{"key-1"},
		googleKeyStates: []googleKeyState{{}},
	}
	r := &appGoogleRotator{a: a}
	next, ok := r.Rotate(&assistant.APIError{Status: 429, RetryAfter: "2"})
	if ok || next != "" {
		t.Fatalf("next %q ok %v", next, ok)
	}
	if r.Key() != "key-1" {
		t.Fatalf("single key must stay available, got %q", r.Key())
	}
	if !a.googleKeyStates[0].CooldownUntil.IsZero() {
		t.Fatal("single key must not be put on cooldown")
	}
}

func TestGoogleKeysAddRemove(t *testing.T) {
	a := &App{configPath: filepath.Join(t.TempDir(), "config.json")}
	if err := a.AddGoogleKey("key-1"); err != nil {
		t.Fatal(err)
	}
	if err := a.AddGoogleKey("key-2"); err != nil {
		t.Fatal(err)
	}
	if err := a.AddGoogleKey("key-1"); err == nil {
		t.Fatal("duplicate key must be rejected")
	}
	if len(a.GetGoogleKeys()) != 2 {
		t.Fatal("keys not added")
	}
	if err := a.RemoveGoogleKey(0); err != nil {
		t.Fatal(err)
	}
	keys := a.GetGoogleKeys()
	if len(keys) != 1 || keys[0].Status != "ok" {
		t.Fatalf("keys %+v", keys)
	}
}
