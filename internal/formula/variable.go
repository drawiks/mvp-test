package formula

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Variable is a user-defined derived stat: a named expression evaluated per
// player and exposed as an extra token inside formula expressions.
type Variable struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Expression string `json:"expression"`
}

// VariableStore persists user variables as JSON at a user path.
type VariableStore struct {
	path      string
	variables map[string]Variable
}

// NewVariableStore opens a variable store, loading any saved file.
func NewVariableStore(path string) *VariableStore {
	s := &VariableStore{
		path:      path,
		variables: map[string]Variable{},
	}
	s.load()
	return s
}

func (s *VariableStore) add(v Variable) { s.variables[v.ID] = v }

// load reads the persisted store file (missing or corrupt files are ignored).
func (s *VariableStore) load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var file struct {
		Variables []jsonVariable `json:"variables"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		return
	}
	for _, v := range file.Variables {
		s.add(v.variable())
	}
}

type jsonVariable struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Expression string `json:"expression"`
}

func (j jsonVariable) variable() Variable {
	v := Variable{ID: j.ID, Name: j.Name, Expression: j.Expression}
	if v.ID == "" {
		v.ID = newID()
	}
	if v.Name == "" {
		v.Name = "Без названия"
	}
	return v
}

// Save writes the store atomically. Returns false on failure.
func (s *VariableStore) Save() bool {
	payload := struct {
		Version   int        `json:"version"`
		Variables []Variable `json:"variables"`
	}{1, s.Variables()}
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return false
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return false
	}
	if err := os.Rename(tmp, s.path); err != nil {
		os.Remove(tmp)
		return false
	}
	return true
}

// Variables returns all stored variables in insertion order.
func (s *VariableStore) Variables() []Variable {
	out := make([]Variable, 0, len(s.variables))
	for _, v := range s.variables {
		out = append(out, v)
	}
	return out
}

// Get returns a variable by id.
func (s *VariableStore) Get(id string) (Variable, bool) {
	v, ok := s.variables[id]
	return v, ok
}

// Add inserts a variable, refreshing its id if it collides with an existing one.
func (s *VariableStore) Add(v Variable) {
	if _, exists := s.variables[v.ID]; exists || v.ID == "" {
		v.ID = newID()
	}
	s.add(v)
}

// Remove deletes a variable. Returns false if the id is unknown.
func (s *VariableStore) Remove(id string) bool {
	if _, ok := s.variables[id]; !ok {
		return false
	}
	delete(s.variables, id)
	return true
}

// Names returns the registered variable names in insertion order.
func (s *VariableStore) Names() []string {
	out := make([]string, 0, len(s.variables))
	for _, v := range s.Variables() {
		out = append(out, v.Name)
	}
	return out
}

// IsVariable reports whether name is a defined user variable.
func (s *VariableStore) IsVariable(name string) bool {
	for _, v := range s.variables {
		if v.Name == name {
			return true
		}
	}
	return false
}

// ValidateVariable checks a variable before storage. A variable's expression
// may reference base stat tokens and other user variables, but not itself.
func (s *VariableStore) ValidateVariable(v Variable) error {
	if strings.TrimSpace(v.Name) == "" {
		return errf("Введите название переменной")
	}
	if strings.TrimSpace(v.Expression) == "" {
		return errf("Введите выражение переменной")
	}
	allowed := s.AllowedNames()
	delete(allowed, v.Name)
	return Validate(v.Expression, allowed)
}

// AllowedNames returns the allowed identifiers for a variable expression:
// base stat tokens plus every stored variable name.
func (s *VariableStore) AllowedNames() map[string]bool {
	allowed := map[string]bool{}
	for t := range statTokens {
		allowed[t] = true
	}
	for _, other := range s.Variables() {
		allowed[other.Name] = true
	}
	return allowed
}

// ResolveVars evaluates the user variables against a base stat map, exposing
// each variable name as an extra key. Variables may depend on base stats and
// on one another; dependencies are resolved over several passes so ordering
// does not matter.
func ResolveVars(base map[string]float64, defs []Variable) (map[string]float64, error) {
	env := make(map[string]float64, len(base)+len(defs))
	for k, v := range base {
		env[k] = v
	}
	names := make([]string, 0, len(defs))
	for _, d := range defs {
		names = append(names, d.Name)
	}
	// Validate every definition up front against base stats + all variable
	// names so ordering of definitions does not matter.
	allowed := map[string]bool{}
	for k := range env {
		allowed[k] = true
	}
	for _, n := range names {
		allowed[n] = true
	}
	for _, d := range defs {
		if err := Validate(d.Expression, allowed); err != nil {
			return nil, errf("Переменная %s: %v", d.Name, err)
		}
	}
	// Seed every variable name so forward references evaluate (then converge)
	// over the bounded passes below.
	for _, n := range names {
		env[n] = 0
	}
	// ponytail: bounded O(n²) passes; topo-sort if dependency chains get long.
	for i := 0; i < len(defs); i++ {
		for _, d := range defs {
			val, err := run(d.Expression, env)
			if err != nil {
				return nil, errf("Переменная %s: %v", d.Name, err)
			}
			env[d.Name] = val
		}
	}
	return env, nil
}
