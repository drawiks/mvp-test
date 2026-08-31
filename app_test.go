package main

import (
	"os"
	"testing"

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
	views, err := BuildViews(*a.result, formula.StandardPreset(), nil)
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

func TestEvalVariableAgainstLoadedPlayer(t *testing.T) {
	a := fixture(t)
	// Use a stored disabled variable store; find first player's base stats.
	p := a.result.Players[0]
	v, err := a.EvalVariable("dead_ratio", "time_dead / max(deaths, 1)", mvp.PlayerVars(p))
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
}
