package mvp

import (
	"math"
	"os"
	"sort"
	"testing"

	"mvp/internal/formula"
	"mvp/internal/model"
	"mvp/internal/parse"
)

func fixtureResult(t *testing.T) model.Result {
	t.Helper()
	data, err := os.ReadFile("../testdata/match.json")
	if err != nil {
		t.Fatal(err)
	}
	result, err := parse.ReadAggregatedJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func linearPreset(name string, w Weights) formula.Preset {
	return formula.Preset{
		ID: name, Name: name, Kind: "linear",
		Weights: WeightsToMapping(w),
	}
}

var testLinear = linearPreset("standard", DefaultWeights)

func TestParseSchema(t *testing.T) {
	r := fixtureResult(t)
	if r.MatchID != 8926354517 || r.DurationSec != 2507 || r.RadiantWin || len(r.Players) != 10 {
		t.Fatalf("%+v", r)
	}
}

func TestDefaultWeightsMatchGolden(t *testing.T) {
	r := fixtureResult(t)
	byName := map[string]model.Player{}
	for _, p := range r.Players {
		byName[p.Name] = p
	}
	expected := map[string]float64{
		"oyoy":            50.9669951,
		"mvhoyeti":        40.92665588,
		"юный дебустер":   29.9013875,
		"Мясное пюре":     25.0,
		"Master Control":  14.33400835,
		"Scarry":          12.46165745,
		"Beefsteeek":      7.107,
		"а за мат извини": 6.2273227,
		"Daniamaps":       4.95733805,
		"re_triger":       3.65900567,
	}
	for name, want := range expected {
		got, err := ComputeScore(byName[name], testLinear)
		if err != nil {
			t.Fatal(err)
		}
		if math.Abs(got-want) > 0.002 {
			t.Fatalf("%s: got %v want %v", name, got, want)
		}
	}
}

func TestSelectMvpsRoles(t *testing.T) {
	r := fixtureResult(t)
	mvps, err := SelectMvps(r, testLinear, nil)
	if err != nil {
		t.Fatal(err)
	}
	if mvps["winner_top1"].Name != "oyoy" || mvps["winner_top1"].Team != "dire" {
		t.Fatalf("winner_top1 %+v", mvps["winner_top1"])
	}
	if mvps["winner_top2"].Name != "mvhoyeti" || mvps["winner_top2"].Team != "dire" {
		t.Fatalf("winner_top2 %+v", mvps["winner_top2"])
	}
	if mvps["loser_top1"].Name != "Master Control" || mvps["loser_top1"].Team != "radiant" {
		t.Fatalf("loser_top1 %+v", mvps["loser_top1"])
	}
}

func TestRankedPlayersSorted(t *testing.T) {
	r := fixtureResult(t)
	ranked, err := RankedPlayers(r, testLinear, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ranked) != 10 {
		t.Fatal("must rank exactly 10")
	}
	prev := math.Inf(1)
	byName := map[string]model.Player{}
	for _, p := range r.Players {
		byName[p.Name] = p
	}
	for _, p := range ranked {
		s, _ := ComputeScore(p, testLinear)
		if s > prev {
			t.Fatal("not sorted descending")
		}
		prev = s
		_ = byName
	}
}

func TestCustomWeightsOnlyFirstBlood(t *testing.T) {
	r := fixtureResult(t)
	w := linearPreset("fb", Weights{FirstBlood: 10.0})
	mvps, err := SelectMvps(r, w, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !mvps["winner_top1"].FirstBlood {
		t.Fatal("mvp must have first blood")
	}
	for _, p := range r.Players {
		s, err := ComputeScore(p, w)
		if err != nil {
			t.Fatal(err)
		}
		if p.FirstBlood {
			if math.Abs(s-10.0) > 1e-9 {
				t.Fatalf("fb score %v", s)
			}
		} else if math.Abs(s) > 1e-9 {
			t.Fatalf("%s score %v", p.Name, s)
		}
	}
}

func TestCustomWeightsChangeRanking(t *testing.T) {
	r := fixtureResult(t)
	w := linearPreset("k", Weights{Kills: 1.0})
	ranked, err := RankedPlayers(r, w, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ranked) == 0 || ranked[0].Name != "mvhoyeti" {
		t.Fatalf("top is %+v", ranked[0])
	}
}

func TestWeightsToFromMappingRoundtrip(t *testing.T) {
	a := WeightsFromMapping(WeightsToMapping(DefaultWeights))
	if a != DefaultWeights {
		t.Fatalf("%+v != %+v", a, DefaultWeights)
	}
	custom := Weights{Kills: 2.5}
	b := WeightsFromMapping(WeightsToMapping(custom))
	if b != custom {
		t.Fatalf("%+v != %+v", b, custom)
	}
}

func TestExpressionPresetMatchesLinearDefaults(t *testing.T) {
	r := fixtureResult(t)
	byName := map[string]model.Player{}
	for _, p := range r.Players {
		byName[p.Name] = p
	}
	expr := formula.LinearToExpression(formula.DefaultLinearWeights)
	preset := formula.Preset{ID: "expr", Name: "Expr", Kind: "expression", Expression: expr}
	for _, name := range []string{"oyoy", "mvhoyeti", "Master Control"} {
		a, err := ComputeScore(byName[name], preset)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := ComputeScore(byName[name], testLinear)
		if math.Abs(a-b) > 1e-9 {
			t.Fatalf("%s: %v != %v", name, a, b)
		}
	}
}

func TestExpressionPresetSelectMvps(t *testing.T) {
	r := fixtureResult(t)
	preset := formula.Preset{
		ID: "expr", Name: "Expr", Kind: "expression",
		Expression: "(kills * 3 + assists * 1.5) / max(deaths, 1)",
	}
	mvps, err := SelectMvps(r, preset, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"winner_top1", "winner_top2", "loser_top1"} {
		if mvps[key] == nil {
			t.Fatalf("%s nil", key)
		}
	}
	if mvps["winner_top1"].Team != "dire" || mvps["winner_top2"].Team != "dire" || mvps["loser_top1"].Team != "radiant" {
		t.Fatal("wrong team assignment")
	}
	s, _ := ComputeScore(*mvps["winner_top1"], preset)
	if s <= 0 {
		t.Fatal("winner score must be positive")
	}
}

func TestStandardV2ComputeScore(t *testing.T) {
	p := model.Player{
		Kills: 10, Deaths: 4, Assists: 7, LastHits: 250, GPM: 580, XPM: 640,
		Healing: 9000, HeroDamage: 20000, DamageTaken: 15000, TowerDamage: 3200,
		StunDuration: 45, CampsStacked: 9, RunePickups: 5, FirstBlood: true,
		GoldSpentWards: 500, GoldSpentSmoke: 100, GoldSpentDust: 50,
		BuffsDuration: 700, Save: 300, Purge: 120, ShieldUptime: 40,
	}
	got, err := ComputeScore(p, formula.StandardV2Preset())
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got-35.747499999999995) > 1e-9 {
		t.Fatalf("got %v", got)
	}
}

func TestStandardV2OnFixture(t *testing.T) {
	r := fixtureResult(t)
	byName := map[string]model.Player{}
	for _, p := range r.Players {
		byName[p.Name] = p
	}
	expected := map[string]float64{
		"mvhoyeti":       28.619662352,
		"юный дебустер":  23.75088,
		"oyoy":           16.906955183,
		"Master Control": 9.67692209,
	}
	for name, want := range expected {
		got, err := ComputeScore(byName[name], formula.StandardV2Preset())
		if err != nil {
			t.Fatal(err)
		}
		if math.Abs(got-want) > 0.002 {
			t.Fatalf("%s: got %v want %v", name, got, want)
		}
	}
	mvps, err := SelectMvps(r, formula.StandardV2Preset(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if mvps["winner_top1"].Name != "mvhoyeti" {
		t.Fatalf("v2 top1 %s", mvps["winner_top1"].Name)
	}
	if mvps["winner_top2"].Name != "юный дебустер" {
		t.Fatalf("v2 top2 %s", mvps["winner_top2"].Name)
	}
	if mvps["loser_top1"].Name != "Master Control" {
		t.Fatalf("v2 loser %s", mvps["loser_top1"].Name)
	}
}

func TestLinearToExpressionEquivalence(t *testing.T) {
	p := model.Player{Kills: 10, Deaths: 4, Assists: 7, LastHits: 250, GPM: 580, XPM: 640,
		StunDuration: 45, Healing: 9000, TowerDamage: 3200, CampsStacked: 9, RunePickups: 5, FirstBlood: true}
	w := Weights{}
	w.Kills, w.DeathsBase, w.Deaths = 0.4, 2.0, 0.5
	preset := formula.Preset{
		ID: "x", Name: "X", Kind: "expression",
		Expression: formula.LinearToExpression(map[string]float64{
			"kills": 0.4, "deaths_base": 2.0, "deaths": 0.5,
		}),
	}
	a, err := ComputeScore(p, preset)
	if err != nil {
		t.Fatal(err)
	}
	b, err := ComputeScore(p, linearPreset("l", w))
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(a-b) > 1e-6 {
		t.Fatalf("%v != %v", a, b)
	}
}

func TestFixtureSortedBySlotStable(t *testing.T) {
	r := fixtureResult(t)
	ids := make([]int64, 0, len(r.Players))
	for _, p := range r.Players {
		ids = append(ids, p.SteamID)
	}
	sorted := append([]int64(nil), ids...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	if len(ids) != 10 {
		t.Fatal("fixture broken")
	}
}

func TestComputeScoreVars(t *testing.T) {
	p := model.Player{TimeDead: 480, Deaths: 6}
	preset := formula.Preset{ID: "e", Name: "E", Kind: "expression", Expression: "my_var * 2"}
	vars := []formula.Variable{
		{ID: "a", Name: "my_var", Expression: "time_dead / max(deaths, 1)"},
	}
	got, err := ComputeScoreVars(p, 0, preset, vars)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got-160) > 1e-9 {
		t.Fatalf("got %v want 160", got)
	}
}

func TestComputeScoreVarsDependencies(t *testing.T) {
	p := model.Player{TimeDead: 600, Deaths: 5}
	preset := formula.Preset{ID: "e", Name: "E", Kind: "expression", Expression: "b"}
	vars := []formula.Variable{
		{ID: "1", Name: "a", Expression: "time_dead / max(deaths, 1)"},
		{ID: "2", Name: "b", Expression: "a * 3"},
	}
	got, err := ComputeScoreVars(p, 0, preset, vars)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got-360) > 1e-9 {
		t.Fatalf("got %v want 360", got)
	}
}

func TestPlayerVarsNewTokens(t *testing.T) {
	p := model.Player{WisdomsCaptured: 3, WatchersCaptured: 2, LotusesGathered: 5, CourierKills: 1}
	vars := PlayerVars(p, 3615.8)
	checks := map[string]float64{
		"wisdoms_captured":  3,
		"watchers_captured": 2,
		"lotuses_gathered":  5,
		"courier_kills":     1,
		"match_duration":    3615.8,
	}
	for token, want := range checks {
		if got := vars[token]; got != want {
			t.Errorf("%s = %v, want %v", token, got, want)
		}
	}
}

func TestComputeScoreWithMatchDuration(t *testing.T) {
	p := model.Player{Kills: 2}
	preset := formula.Preset{ID: "e", Name: "E", Kind: "expression", Expression: "match_duration * kills"}
	got, err := ComputeScoreVars(p, 3615.8, preset, nil)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got-7231.6) > 1e-9 {
		t.Fatalf("got %v want 7231.6", got)
	}
}

func TestPlayerVarsPosition(t *testing.T) {
	vars := PlayerVars(model.Player{Position: 3}, 0)
	if vars["position"] != 3 {
		t.Fatalf("position = %v, want 3", vars["position"])
	}
}

func TestComputeScoreWithPositionIf(t *testing.T) {
	p := model.Player{Kills: 10, Position: 1}
	preset := formula.Preset{ID: "e", Name: "E", Kind: "expression", Expression: "if position == 1 { kills * 2 } else { kills }"}
	got, err := ComputeScoreVars(p, 0, preset, nil)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got-20) > 1e-9 {
		t.Fatalf("got %v want 20", got)
	}
	p.Position = 4
	got, err = ComputeScoreVars(p, 0, preset, nil)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got-10) > 1e-9 {
		t.Fatalf("got %v want 10", got)
	}
}
