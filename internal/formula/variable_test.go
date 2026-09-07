package formula

import (
	"math"
	"path/filepath"
	"strings"
	"testing"
)

func TestVariableStoreRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "variables.json")
	s := NewVariableStore(path)
	s.Add(Variable{Name: "dead_ratio", Expression: "time_dead / max(deaths, 1)"})
	if !s.Save() {
		t.Fatal("save failed")
	}
	s2 := NewVariableStore(path)
	list := s2.Variables()
	if len(list) != 1 || list[0].Name != "dead_ratio" || list[0].Expression != "time_dead / max(deaths, 1)" {
		t.Fatalf("got %+v", list)
	}
	if !s2.Remove(list[0].ID) {
		t.Fatal("remove failed")
	}
	if !s2.Save() {
		t.Fatal("save after remove failed")
	}
}

func TestValidateVariableRejectsUnknownToken(t *testing.T) {
	s := NewVariableStore("")
	err := s.ValidateVariable(Variable{Name: "x", Expression: "time_dead / nonsense"})
	if err == nil || !strings.Contains(err.Error(), "Неизвестная переменная") {
		t.Fatalf("got %v", err)
	}
}

func TestValidateVariableAllowsOtherVariables(t *testing.T) {
	s := NewVariableStore("")
	s.Add(Variable{ID: "a", Name: "rat", Expression: "time_dead / max(deaths, 1)"})
	err := s.ValidateVariable(Variable{Name: "twice", Expression: "rat * 2"})
	if err != nil {
		t.Fatalf("got %v", err)
	}
}

func TestValidateVariableRejectsMissingName(t *testing.T) {
	s := NewVariableStore("")
	if err := s.ValidateVariable(Variable{Name: "  ", Expression: "kills"}); err == nil {
		t.Fatal("want name error")
	}
}

func TestResolveVarsOrderIndependent(t *testing.T) {
	vars := map[string]float64{"time_dead": 600, "deaths": 5}
	defs := []Variable{
		{Name: "b", Expression: "a * 3"},
		{Name: "a", Expression: "time_dead / max(deaths, 1)"},
	}
	env, err := ResolveVars(vars, defs)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := env["a"]; !ok {
		t.Fatal("a missing")
	}
	if math.Abs(env["b"]-360) > 1e-9 {
		t.Fatalf("b=%v want 360", env["b"])
	}
}

func TestTutorialPreset(t *testing.T) {
	s := NewStore("")
	found := false
	for _, p := range s.Presets() {
		if p.ID == "tutorial" {
			found = true
			if !isBuiltin(p.ID) || p.Kind != "expression" {
				t.Fatalf("tutorial preset wrong flags: %+v", p)
			}
		}
	}
	if !found {
		t.Fatal("tutorial preset missing from store")
	}
	if s.Remove("tutorial") {
		t.Fatal("built-in tutorial preset was removed")
	}
}

func TestVariableStoreNameTaken(t *testing.T) {
	s := NewVariableStore(filepath.Join(t.TempDir(), "v.json"))
	s.Add(Variable{ID: "a", Name: "dead_ratio", Expression: "time_dead / max(deaths,1)"})
	if !s.NameTaken("DEAD_RATIO", "") || !s.NameTaken("dead_ratio", "") {
		t.Fatal("case-insensitive duplicate not detected")
	}
	if s.NameTaken("dead_ratio", "a") {
		t.Fatal("self-id should not count")
	}
}
