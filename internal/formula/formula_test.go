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

func TestStoreOverwritesExisting(t *testing.T) {
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
	s.Upsert(Preset{ID: "standard_v2", Kind: "linear", Weights: map[string]float64{"kills": 9}})
	s.Add(Preset{ID: "tutorial", Kind: "linear", Weights: map[string]float64{"kills": 9}})
	if s.presets["standard_v2"].Expression == "" {
		t.Fatal("standard_v2 builtin lost its expression")
	}
	if s.presets["tutorial"].Expression == "" {
		t.Fatal("tutorial builtin lost its expression")
	}
}

func TestValidatePresetAllowsUserVariables(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "f.json"))
	expr := "kills * 2 + Initiation_Density"

	// Without extra, an unknown identifier is rejected.
	if err := s.ValidatePreset(Preset{Name: "X", Kind: "expression", Expression: expr}, nil); err == nil {
		t.Fatal("want error without user-variable allowed set")
	}

	// With the variable name allowed, the same expression validates.
	if err := s.ValidatePreset(Preset{Name: "X", Kind: "expression", Expression: expr}, map[string]bool{"Initiation_Density": true}); err != nil {
		t.Fatalf("unexpected error: %v", err)
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

func TestDefaultLinearWeights(t *testing.T) {
	if DefaultLinearWeights["kills"] != 0.3 {
		t.Fatalf("%+v", DefaultLinearWeights)
	}
	if _, ok := DefaultLinearWeights["deaths_base"]; ok {
		t.Fatalf("deaths_base must be gone")
	}
	if _, ok := DefaultLinearWeights["gold_lost"]; ok {
		t.Fatalf("gold_lost must be gone")
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
	for _, want := range []string{"kills * 0.3", "(0 - deaths * 0.3)", "stun_duration * 0.05", "camps_stacked * 0.5"} {
		if !strings.Contains(expr, want) {
			t.Fatalf("%s missing in %s", want, expr)
		}
	}
}

func TestExpressionToWeightsRoundtrip(t *testing.T) {
	expr := LinearToExpression(DefaultLinearWeights)
	var expected = map[string]float64{}
	for k, v := range DefaultLinearWeights {
		if v != 0 || k == "deaths" {
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
	if !ok || got["kills"] != 3.0 || got["deaths"] != 0.5 {
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
	weights := map[string]float64{"kills": 0.3, "deaths": 0.3, "assists": -0.5}
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
	got, ok := ExpressionToWeights("(0 - deaths * 0.3)")
	if !ok || got["deaths"] != 0.3 {
		t.Fatalf("%v %v", got, ok)
	}
	if _, ok := ExpressionToWeights("(0 - deaths * 0.3) + deaths * 0.1"); ok {
		t.Fatal("duplicate deaths must be refused")
	}
}

func TestExpressionToWeightsDuplicateConstant(t *testing.T) {
	got, ok := ExpressionToWeights("kills * 2 + 3 + 5")
	if !ok || got["kills"] != 2.0 {
		t.Fatalf("%v %v", got, ok)
	}
}

func TestSplitExpressionV2(t *testing.T) {
	weights, tail := SplitExpression(StandardV2Formula)
	for k, want := range map[string]float64{"kills": 0.2, "deaths": 0.2, "first_blood": 1.0} {
		if weights[k] != want {
			t.Fatalf("%s %v != %v", k, weights[k], want)
		}
	}
	for _, want := range []string{"min(healing, 8000)", "max(deaths, 1)", "min(buff_duration, 600)", "min(save_duration + purge_duration + shield_duration, 500)"} {
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
		"buff_duration": 700, "save_duration": 300, "purge_duration": 120, "shield_duration": 40,
		"position": 1,
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

func TestStatsV2Registered(t *testing.T) {
	tokens := []string{
		"wisdoms_captured", "watchers_captured", "lotuses_gathered", "courier_kills",
		"match_duration",
	}
	for _, tok := range tokens {
		if !StatTokens(tok) {
			t.Errorf("%s missing from statTokens", tok)
		}
	}
	if err := Validate("match_duration * 3 + wisdoms_captured + courier_kills", statTokens); err != nil {
		t.Fatal(err)
	}
}

func TestExpressionToWeightsNewStats(t *testing.T) {
	got, ok := ExpressionToWeights("wisdoms_captured * 2 + courier_kills * 3.5 + lotuses_gathered * 1")
	if !ok || got["wisdoms_captured"] != 2 || got["courier_kills"] != 3.5 || got["lotuses_gathered"] != 1 {
		t.Fatalf("got %v ok %v", got, ok)
	}
	if _, ok := ExpressionToWeights("match_duration * 3"); ok {
		t.Fatal("match_duration is expression-only, must not convert to linear weights")
	}
}

func TestPositionTokenRegistered(t *testing.T) {
	if !StatTokens("position") {
		t.Fatal("position missing from statTokens")
	}
	if err := Validate("if position == 1 { kills * 2 } else { kills }", statTokens); err != nil {
		t.Fatal(err)
	}
	if _, ok := ExpressionToWeights("position * 2"); ok {
		t.Fatal("position is expression-only, must not convert to linear weights")
	}
}

func TestEvalIfElse(t *testing.T) {
	vars := map[string]float64{"position": 1, "kills": 10, "deaths": 4}
	expr := "if position == 1 { (kills * 21) * 0.11 } else { kills }"
	if got := EvalOn(t, expr, vars); !approx(got, 23.1, 1e-9) {
		t.Fatalf("got %v", got)
	}
	vars["position"] = 3
	if got := EvalOn(t, expr, vars); !approx(got, 10, 1e-9) {
		t.Fatalf("got %v", got)
	}
}

func TestEvalIfElifChain(t *testing.T) {
	vars := map[string]float64{"position": 0, "kills": 10, "assists": 5}
	expr := "if position == 1 { kills } else if position == 2 { assists } else if position == 3 { kills + assists } else { 0 }"
	if got := EvalOn(t, expr, vars); got != 0 {
		t.Fatalf("got %v", got)
	}
	vars["position"] = 2
	if got := EvalOn(t, expr, vars); got != 5 {
		t.Fatalf("got %v", got)
	}
	vars["position"] = 3
	if got := EvalOn(t, expr, vars); got != 15 {
		t.Fatalf("got %v", got)
	}
}

func TestEvalConditionOperators(t *testing.T) {
	vars := map[string]float64{"position": 1, "kills": 12, "assists": 3, "deaths": 5}
	if got := EvalOn(t, "if kills >= 10 && assists < 5 { kills } else { assists }", vars); got != 12 {
		t.Fatalf("&& got %v", got)
	}
	if got := EvalOn(t, "if kills >= 10 || assists > 9 { kills + assists } else { 0 }", vars); got != 15 {
		t.Fatalf("|| got %v", got)
	}
	if got := EvalOn(t, "if !(deaths > 3) { kills } else { deaths }", vars); got != 5 {
		t.Fatalf("! got %v", got)
	}
	if got := EvalOn(t, "if position <= 2 && position != 0 { kills } else { 0 }", vars); got != 12 {
		t.Fatalf("<= != got %v", got)
	}
	if got := EvalOn(t, "kills > deaths ? kills : deaths", vars); got != 12 {
		t.Fatalf("ternary got %v", got)
	}
}

func TestValidateRejectsDisallowedOperators(t *testing.T) {
	cases := []string{
		"kills % 2",
		"kills in [1, 2]",
		"kills and assists",
		"kills or assists",
		"not kills",
		`if kills >= 1 { "abc" } else { "def" }`,
	}
	for _, expr := range cases {
		if err := Validate(expr, statTokens); err == nil {
			t.Errorf("%q validated unexpectedly", expr)
		}
	}
}

func TestStripComments(t *testing.T) {
	got := stripComments("kills * 2 # бонус\n+ assists # вторая строка\n# весь файл комментарий\n")
	want := "kills * 2 \n+ assists \n\n"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestValidateWithComments(t *testing.T) {
	if err := Validate("# комментарий\nkills * 2 # в конце строки\n+ assists", statTokens); err != nil {
		t.Fatalf("commented formula rejected: %v", err)
	}
	if err := Validate("# только комментарий", statTokens); err == nil {
		t.Fatal("comment-only expression validated, want error")
	}
}

func TestEvalWithComments(t *testing.T) {
	vars := map[string]float64{"kills": 10, "assists": 5}
	if got := EvalOn(t, "kills * 2 # x\n+ assists", vars); !approx(got, 25, 1e-9) {
		t.Fatalf("got %v", got)
	}
}

func TestExpressionToWeightsWithComments(t *testing.T) {
	weights, ok := ExpressionToWeights("# линейная\nkills * 0.3 + assists * 0.15 # коммент")
	if !ok {
		t.Fatal("comments broke linear extraction")
	}
	if weights["kills"] != 0.3 || weights["assists"] != 0.15 {
		t.Fatalf("weights %+v", weights)
	}
}

func TestTutorialFormulaValid(t *testing.T) {
	if err := Validate(TutorialFormula, statTokens); err != nil {
		t.Fatalf("tutorial formula invalid: %v", err)
	}
	if _, ok := ExpressionToWeights(TutorialFormula); ok {
		t.Fatal("tutorial is nonlinear, must not convert to weights")
	}
}

func TestMultilineSemicolonSums(t *testing.T) {
	expr := "kills * 2;\nmax(deaths, 1) + assists * 3"
	a := EvalOn(t, expr, map[string]float64{"kills": 5, "deaths": 4, "assists": 2})
	want := 5*2 + 4 + 2*3
	if !approx(a, float64(want), 1e-9) {
		t.Fatalf("got %v want %v", a, want)
	}
}

func TestMigrateExpressionRenamesTokens(t *testing.T) {
	got := MigrateExpression("buffs_duration * 1 + save * 2 + purge * 3 + shield_uptime * 4")
	for _, tok := range []string{"buff_duration", "save_duration", "purge_duration", "shield_duration"} {
		if !strings.Contains(got, tok) {
			t.Fatalf("%s missing in %s", tok, got)
		}
	}
	if strings.Contains(got, "shield_uptime") || strings.Contains(got, "buffs_duration") {
		t.Fatalf("old tokens remain: %s", got)
	}
}

func TestMigrateExpressionNeutralizesRemoved(t *testing.T) {
	for _, old := range []string{"deaths_base", "gold_lost"} {
		expr := "kills * 2 + " + old + " * 5"
		got := MigrateExpression(expr)
		if err := Validate(got, statTokens); err != nil {
			t.Fatalf("%s not neutralized: %s (%v)", old, got, err)
		}
		if strings.Contains(got, old) {
			t.Fatalf("%s remains: %s", old, got)
		}
	}
}

func TestBreakdownSumsToTotal(t *testing.T) {
	expr := "kills * 2;\n assists * 3"
	vars := map[string]float64{"kills": 5, "assists": 2}
	rows, err := Breakdown(expr, vars)
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
	total, _ := Eval(expr, vars)
	if !approx(sum, total, 1e-9) {
		t.Fatalf("sum %v != total %v", sum, total)
	}
}

func TestBreakdownFlatSumTerms(t *testing.T) {
	expr := "kills * 2\n+ assists * 3\n+ last_hits * 0.5"
	vars := map[string]float64{"kills": 5, "assists": 2, "last_hits": 100}
	rows, err := Breakdown(expr, vars)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("rows %d: %+v", len(rows), rows)
	}
	var sum float64
	for _, r := range rows {
		sum += r.Value
	}
	if !approx(sum, EvalOn(t, expr, vars), 1e-9) {
		t.Fatalf("sum %v != total", sum)
	}
	if rows[0].Label != "kills * 2" || rows[1].Label != "assists * 3" {
		t.Fatalf("term labels %q", []string{rows[0].Label, rows[1].Label, rows[2].Label})
	}
}

func TestBreakdownSignedTerms(t *testing.T) {
	expr := "kills * 2 - deaths * 3 + assists"
	vars := map[string]float64{"kills": 5, "deaths": 2, "assists": 4}
	rows, err := Breakdown(expr, vars)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("rows %d: %+v", len(rows), rows)
	}
	if rows[0].Label != "kills * 2" || rows[1].Label != "- deaths * 3" || !approx(rows[1].Value, -6, 1e-9) {
		t.Fatalf("signed term %q = %v", rows[1].Label, rows[1].Value)
	}
	var sum float64
	for _, r := range rows {
		sum += r.Value
	}
	if !approx(sum, EvalOn(t, expr, vars), 1e-9) {
		t.Fatalf("sum %v != total", sum)
	}
}

func TestBreakdownKeepsWholeNestedExpression(t *testing.T) {
	expr := "(kills * 3 + assists * 1.5) / max(deaths, 1)"
	vars := map[string]float64{"kills": 10, "assists": 5, "deaths": 4}
	rows, err := Breakdown(expr, vars)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows %d: %+v", len(rows), rows)
	}
	if !approx(rows[0].Value, EvalOn(t, expr, vars), 1e-9) {
		t.Fatalf("row %v != total", rows[0].Value)
	}
}

func TestSplitTermsUnaryMinusGuard(t *testing.T) {
	if got := splitTerms("a + b ** -2"); len(got) != 2 || got[0] != "a" || got[1] != "b ** -2" {
		t.Fatalf("terms %q", got)
	}
	if got := splitTerms("-a + b"); len(got) != 2 || got[0] != "-a" || got[1] != "b" {
		t.Fatalf("terms %q", got)
	}
	if got := splitTerms("if position == 1 { kills - assists } else { assists }"); len(got) != 1 {
		t.Fatalf("if-block must stay one term, got %q", got)
	}
}

func TestBreakdownTutorialFormula(t *testing.T) {
	vars := sampleVars(t)
	total := EvalOn(t, TutorialFormula, vars)
	rows, err := Breakdown(TutorialFormula, vars)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) < 10 {
		t.Fatalf("tutorial must decompose into summands, got %d rows", len(rows))
	}
	var sum float64
	for _, r := range rows {
		sum += r.Value
	}
	if !approx(sum, total, 1e-6) {
		t.Fatalf("sum %v != total %v", sum, total)
	}
}

func TestStoreNameTaken(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "f.json"))
	s.Add(Preset{ID: "a", Name: "My Preset", Kind: "linear"})
	if !s.NameTaken("my preset", "") {
		t.Fatal("case-insensitive duplicate not detected")
	}
	if s.NameTaken("my preset", "a") {
		t.Fatal("self-id should not count as duplicate")
	}
	if s.NameTaken("Other", "") {
		t.Fatal("distinct name flagged")
	}
}

func TestStoreLoadMigratesLegacyKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f.json")
	legacy := `{"active":"p","presets":[
		{"id":"p","name":"P","kind":"linear",
		 "weights":{"kills":0.3,"stun":0.05,"camps":0.5,"runes":0.2,"deaths_base":3.0,"gold_lost":0.1}},
		{"id":"c","name":"C","kind":"expression","expression":"buffs_duration * 1 + deaths_base * 2"},
		{"id":"standard","name":"Standard","kind":"linear","weights":{"kills":9}}
	]}`
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewStore(path)
	p, ok := s.Get("p")
	if !ok {
		t.Fatal("preset p missing")
	}
	for k, want := range map[string]float64{"kills": 0.3, "stun_duration": 0.05, "camps_stacked": 0.5, "rune_pickups": 0.2} {
		if p.Weights[k] != want {
			t.Fatalf("%s %v != %v", k, p.Weights[k], want)
		}
	}
	for _, bad := range []string{"deaths_base", "gold_lost", "stun", "camps", "runes"} {
		if _, ok := p.Weights[bad]; ok {
			t.Fatalf("legacy key %s survived", bad)
		}
	}
	c, ok := s.Get("c")
	if !ok || c.Expression != "buff_duration * 1 + 0 * 2" {
		t.Fatalf("expression not migrated: %+v", c)
	}
	if _, ok := s.Get("standard"); ok {
		t.Fatal("legacy standard preset should be dropped")
	}
}
