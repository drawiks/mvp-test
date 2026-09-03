package formula

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func approx(a, b, eps float64) bool { return math.Abs(a-b) <= eps }

func TestValidateOK(t *testing.T) {
	if err := Validate("(kills * 3 + assists * 1.5) / max(deaths, 1)", statTokens); err != nil {
		t.Fatal(err)
	}
}

func TestValidateUnknownVariable(t *testing.T) {
	err := Validate("kills * kils", statTokens)
	if err == nil || !strings.Contains(err.Error(), "Неизвестная переменная") {
		t.Fatalf("got %v", err)
	}
}

func TestValidateSyntaxError(t *testing.T) {
	if err := Validate("(kills + ", statTokens); err == nil {
		t.Fatal("want syntax error")
	}
}

func TestValidateEmpty(t *testing.T) {
	if err := Validate("   ", statTokens); err == nil {
		t.Fatal("want empty error")
	}
}

func TestAllExamplesValid(t *testing.T) {
	for _, ex := range Examples {
		if err := Validate(ex.Expression, statTokens); err != nil {
			t.Fatalf("%s: %v", ex.Name, err)
		}
	}
}

func TestEvalMath(t *testing.T) {
	vars := map[string]float64{"kills": 10, "deaths": 4, "assists": 5}
	got, err := Eval("kills * 3 + assists * 1.5", vars)
	if err != nil || !approx(got, 37.5, 1e-9) {
		t.Fatalf("got %v err %v", got, err)
	}
	got, err = Eval("(kills * 3 + assists * 1.5) / max(deaths, 1)", vars)
	if err != nil || !approx(got, 37.5/4, 1e-9) {
		t.Fatalf("got %v err %v", got, err)
	}
}

func TestEvalFunctions(t *testing.T) {
	got, err := Eval("min(5, 2) + abs(-3) + round(1.7)", map[string]float64{})
	if err != nil || !approx(got, 7.0, 1e-9) {
		t.Fatalf("got %v err %v", got, err)
	}
}

func TestEvalBlocksBuiltins(t *testing.T) {
	_, err := Eval("__import__('os').system('echo hi')", map[string]float64{"kills": 1})
	if err == nil {
		t.Fatal("want error for builtin access")
	}
}

func TestEvalDivZero(t *testing.T) {
	_, err := Eval("1 / 0", map[string]float64{"kills": 1})
	if err == nil || !strings.Contains(err.Error(), "Деление на ноль") {
		t.Fatalf("got %v", err)
	}
}

func TestStoreRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "formulas.json")
	s := NewStore(path)
	s.Add(Preset{ID: "x", Name: "Моя", Kind: "expression", Expression: "kills*2"})
	if !s.SetActive("x") {
		t.Fatal("set_active failed")
	}
	if !s.Save() {
		t.Fatal("save failed")
	}
	loaded := NewStore(path)
	if loaded.Active().ID != "x" {
		t.Fatal("active not persisted")
	}
	p, _ := loaded.Get("x")
	if p.Expression != "kills*2" {
		t.Fatal("expression not persisted")
	}
}

func TestStoreStandardIsBuiltin(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "f.json"))
	if _, ok := s.Get("standard"); !ok {
		t.Fatal("standard missing")
	}
	if s.Remove("standard") {
		t.Fatal("remove standard should fail")
	}
}

func TestStoreUpsertOverwritesExisting(t *testing.T) {
	// Regression: Upsert/Add must overwrite an existing user preset, so
	// converting a saved linear preset to an expression one actually sticks.
	s := NewStore(filepath.Join(t.TempDir(), "f.json"))
	s.Upsert(Preset{ID: "x", Name: "X", Kind: "linear", Weights: map[string]float64{"deaths": 0.1}})
	s.Upsert(Preset{ID: "x", Name: "X", Kind: "expression", Expression: "kills * 5 + deaths"})

	p, ok := s.Get("x")
	if !ok {
		t.Fatal("preset missing")
	}
	if p.Kind != "expression" || p.Expression != "kills * 5 + deaths" {
		t.Fatalf("expected expression to overwrite linear, got %+v", p)
	}
}

func TestStoreBuiltinsNeverOverwritten(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "f.json"))
	s.Upsert(Preset{ID: "standard", Kind: "expression", Expression: "hacked"})
	s.Add(Preset{ID: "standard_v2", Kind: "linear", Weights: map[string]float64{"kills": 9}})
	if s.presets["standard"].Expression != "" {
		t.Fatal("standard builtin was overwritten")
	}
	if s.presets["standard_v2"].Expression == "" {
		t.Fatal("standard_v2 builtin lost its expression")
	}
}

func TestImportExport(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "f.json"))
	preset := Preset{ID: "y", Name: "Y", Kind: "expression", Expression: "healing*0.005"}
	out := filepath.Join(dir, "y.json")
	if err := store.ExportFile(preset, out); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"healing*0.005"`) {
		t.Fatalf("export content: %s", data)
	}
	other := NewStore(filepath.Join(dir, "other.json"))
	imported, err := other.ImportFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if imported.ID != "y" || imported.Name != "Y" {
		t.Fatalf("imported %+v", imported)
	}
}

func TestStandardPresetDefaults(t *testing.T) {
	p := StandardPreset()
	if p.Weights["kills"] != 0.3 || p.Kind != "linear" {
		t.Fatalf("%+v", p)
	}
}

func TestStandardV2PresetExpressionValid(t *testing.T) {
	p := StandardV2Preset()
	if p.Kind != "expression" || p.ID != "standard_v2" || p.Name != "Стандартная v2" {
		t.Fatalf("%+v", p)
	}
	if err := Validate(p.Expression, statTokens); err != nil {
		t.Fatal(err)
	}
	if _, ok := ExpressionToWeights(p.Expression); ok {
		t.Fatal("v2 must be nonlinear")
	}
}

func TestStoreHasStandardV2Builtin(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "f.json"))
	if _, ok := s.Get("standard_v2"); !ok {
		t.Fatal("standard_v2 missing")
	}
	if s.Active().ID != "standard_v2" {
		t.Fatal("active must default to standard_v2")
	}
}

func TestLinearToExpressionValid(t *testing.T) {
	expr := LinearToExpression(DefaultLinearWeights)
	if err := Validate(expr, statTokens); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"kills * 0.3", "(3 - deaths * 0.3)", "stun_duration * 0.05", "camps_stacked * 0.5"} {
		if !strings.Contains(expr, want) {
			t.Fatalf("%s missing in %s", want, expr)
		}
	}
}

func TestExpressionToWeightsRoundtrip(t *testing.T) {
	expr := LinearToExpression(DefaultLinearWeights)
	var expected = map[string]float64{}
	for k, v := range DefaultLinearWeights {
		if v != 0 || k == "deaths" || k == "deaths_base" {
			expected[k] = v
		}
	}
	got, ok := ExpressionToWeights(expr)
	if !ok {
		t.Fatal("expected linear")
	}
	if len(got) != len(expected) {
		t.Fatalf("got %v want %v", got, expected)
	}
	for k, v := range expected {
		if got[k] != v {
			t.Fatalf("%s: got %v want %v", k, got[k], v)
		}
	}
}

func TestNewStatsInExpression(t *testing.T) {
	expr := "hero_damage * 0.001 + damage_taken * 0.0005 + gold_spent_wards * 0.01 + gold_spent_smoke * 0.02 + gold_spent_dust * 0.03"
	if err := Validate(expr, statTokens); err != nil {
		t.Fatal(err)
	}
	got, err := Eval(expr, map[string]float64{
		"hero_damage": 20000, "damage_taken": 15000, "gold_spent_wards": 500,
		"gold_spent_smoke": 100, "gold_spent_dust": 50,
	})
	if err != nil || !approx(got, 36.0, 1e-9) {
		t.Fatalf("got %v err %v", got, err)
	}
}

func TestExpressionToWeightsSparse(t *testing.T) {
	got, ok := ExpressionToWeights("kills * 3 + (2 - deaths * 0.5)")
	if !ok || got["kills"] != 3.0 || got["deaths_base"] != 2.0 || got["deaths"] != 0.5 {
		t.Fatalf("%v %v", got, ok)
	}
}

func TestExpressionToWeightsUnsupported(t *testing.T) {
	for _, expr := range []string{
		"(kills * 3 + assists * 1.5) / max(deaths, 1)",
		"", "kills * kills", "kills ** 2",
	} {
		if _, ok := ExpressionToWeights(expr); ok {
			t.Fatalf("expected nonlinear: %q", expr)
		}
	}
}

func TestLinearToExpressionSkipsAbsent(t *testing.T) {
	if got := LinearToExpression(map[string]float64{"kills": 3}); got != "kills * 3" {
		t.Fatalf("got %q", got)
	}
}

func TestNegativeCoefficientRoundtrip(t *testing.T) {
	weights := map[string]float64{"kills": 0.3, "deaths": 0.3, "deaths_base": 3.0, "assists": -0.5}
	expr := LinearToExpression(weights)
	got, ok := ExpressionToWeights(expr)
	if !ok {
		t.Fatal("not linear")
	}
	for k, v := range weights {
		if got[k] != v {
			t.Fatalf("%s %v != %v", k, got[k], v)
		}
	}
	if got, ok := ExpressionToWeights("kills * -0.5"); !ok || got["kills"] != -0.5 {
		t.Fatalf("%v %v", got, ok)
	}
}

func TestExpressionToWeightsDeathsAmbiguity(t *testing.T) {
	if _, ok := ExpressionToWeights("deaths * 5"); ok {
		t.Fatal("plain deaths must be refused")
	}
	got, ok := ExpressionToWeights("3 - deaths * 0.3")
	if !ok || got["deaths_base"] != 3.0 || got["deaths"] != 0.3 {
		t.Fatalf("%v %v", got, ok)
	}
	if _, ok := ExpressionToWeights("(2 - deaths * 0.3) + deaths * 0.1"); ok {
		t.Fatal("duplicate deaths must be refused")
	}
}

func TestExpressionToWeightsDuplicateConstant(t *testing.T) {
	if _, ok := ExpressionToWeights("kills * 2 + 3 + 5"); ok {
		t.Fatal("two constants must be refused")
	}
	got, ok := ExpressionToWeights("3 + kills * 2")
	if !ok || got["deaths_base"] != 3.0 || got["kills"] != 2.0 {
		t.Fatalf("%v %v", got, ok)
	}
	if _, ok := ExpressionToWeights("3 + (4 - deaths * 0.3)"); ok {
		t.Fatal("base dup must be refused")
	}
}

func TestSplitExpressionV2(t *testing.T) {
	weights, tail := SplitExpression(StandardV2Formula)
	for k, want := range map[string]float64{"kills": 0.2, "deaths_base": 3.0, "deaths": 0.3, "first_blood": 1.0} {
		if weights[k] != want {
			t.Fatalf("%s %v != %v", k, weights[k], want)
		}
	}
	for _, want := range []string{"min(healing, 8000)", "max(deaths, 1)", "min(buffs_duration, 600)", "min(save + purge + shield_uptime, 500)"} {
		if !strings.Contains(tail, want) {
			t.Fatalf("%s missing in %s", want, tail)
		}
	}
}

func TestSplitExpressionRoundtripV2(t *testing.T) {
	weights, tail := SplitExpression(StandardV2Formula)
	rebuilt := LinearToExpression(weights)
	if tail != "" {
		rebuilt += " + " + tail
	}
	if err := Validate(rebuilt, statTokens); err != nil {
		t.Fatal(err)
	}
	a := EvalOn(t, StandardV2Formula, sampleVars(t))
	b := EvalOn(t, rebuilt, sampleVars(t))
	if !approx(a, b, 1e-9) {
		t.Fatalf("%v != %v", a, b)
	}
}

func sampleVars(t *testing.T) map[string]float64 {
	return map[string]float64{
		"kills": 10, "deaths": 4, "assists": 7, "last_hits": 250, "gpm": 580, "xpm": 640,
		"healing": 9000, "hero_damage": 20000, "damage_taken": 15000, "tower_damage": 3200,
		"stun_duration": 45, "camps_stacked": 9, "rune_pickups": 5, "first_blood": 1,
		"gold_spent_wards": 500, "gold_spent_smoke": 100, "gold_spent_dust": 50,
		"buffs_duration": 700, "save": 300, "purge": 120, "shield_uptime": 40,
	}
}

func EvalOn(t *testing.T, expr string, vars map[string]float64) float64 {
	t.Helper()
	got, err := Eval(expr, vars)
	if err != nil {
		t.Fatalf("%s: %v", expr, err)
	}
	return got
}

func TestSplitExpressionLinearOnly(t *testing.T) {
	weights, tail := SplitExpression("kills * 0.3 + (3 - deaths * 0.3) + assists * 0.15")
	if tail != "" {
		t.Fatalf("tail %q", tail)
	}
	if weights["kills"] != 0.3 || weights["assists"] != 0.15 {
		t.Fatalf("%v", weights)
	}
}

func TestSplitExpressionEmpty(t *testing.T) {
	if w, tail := SplitExpression(""); len(w) != 0 || tail != "" {
		t.Fatalf("%v %q", w, tail)
	}
	if w, tail := SplitExpression("   "); len(w) != 0 || tail != "" {
		t.Fatalf("%v %q", w, tail)
	}
}
