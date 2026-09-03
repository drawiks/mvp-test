package formula

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Preset is a saved formula: either a set of linear weights or a free-form
// expression.
type Preset struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	Kind       string             `json:"kind"` // "linear" | "expression"
	Weights    map[string]float64 `json:"weights"`
	Expression string             `json:"expression"`
}

// StandardPreset returns the built-in linear preset.
func StandardPreset() Preset {
	w := make(map[string]float64, len(DefaultLinearWeights))
	for k, v := range DefaultLinearWeights {
		w[k] = v
	}
	return Preset{ID: "standard", Name: "Стандартная", Kind: "linear", Weights: w}
}

// StandardV2Preset returns the default expression preset.
func StandardV2Preset() Preset {
	return Preset{ID: "standard_v2", Name: "Стандартная v2", Kind: "expression", Expression: StandardV2Formula}
}

func newID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Store persists presets as JSON at a user path.
type Store struct {
	path     string
	presets  map[string]Preset
	activeID string
}

// NewStore opens a store, seeding the two built-ins and loading any saved file.
func NewStore(path string) *Store {
	s := &Store{
		path:    path,
		presets: map[string]Preset{},
	}
	s.add(StandardPreset())
	s.add(StandardV2Preset())
	s.activeID = "standard_v2"
	s.load()
	return s
}

func (s *Store) add(p Preset) { s.presets[p.ID] = p }

// Load reads the persisted store file (missing or corrupt files are ignored).
func (s *Store) load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var file struct {
		Active  string       `json:"active"`
		Presets []jsonPreset `json:"presets"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		return
	}
	for _, p := range file.Presets {
		s.add(p.preset())
	}
	if _, ok := s.presets[file.Active]; ok {
		s.activeID = file.Active
	}
}

type jsonPreset struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	Kind       string             `json:"kind"`
	Weights    map[string]float64 `json:"weights"`
	Expression string             `json:"expression"`
}

func (j jsonPreset) preset() Preset {
	p := Preset{ID: j.ID, Name: j.Name, Kind: j.Kind, Expression: j.Expression}
	p.Weights = map[string]float64{}
	for k, v := range j.Weights {
		p.Weights[k] = v
	}
	if p.ID == "" {
		p.ID = newID()
	}
	if p.Name == "" {
		p.Name = "Без названия"
	}
	if p.Kind != "expression" {
		p.Kind = "linear"
	}
	return p
}

// Save writes the store atomically. Returns false on failure.
func (s *Store) Save() bool {
	payload := struct {
		Version int      `json:"version"`
		Active  string   `json:"active"`
		Presets []Preset `json:"presets"`
	}{1, s.activeID, s.Presets()}
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

// Presets returns all presets in insertion order.
func (s *Store) Presets() []Preset {
	ordered := []string{"standard", "standard_v2"}
	for id := range s.presets {
		if id != "standard" && id != "standard_v2" {
			ordered = append(ordered, id)
		}
	}
	out := make([]Preset, 0, len(ordered))
	for _, id := range ordered {
		if p, ok := s.presets[id]; ok {
			out = append(out, p)
		}
	}
	return out
}

func (s *Store) Get(id string) (Preset, bool) {
	p, ok := s.presets[id]
	return p, ok
}

// Active returns the active preset.
func (s *Store) Active() Preset {
	if p, ok := s.presets[s.activeID]; ok {
		return p
	}
	return StandardPreset()
}

// isBuiltin reports whether id is a read-only built-in preset.
func isBuiltin(id string) bool { return id == "standard" || id == "standard_v2" }

// Add inserts a preset. Built-ins are never overwritten.
func (s *Store) Add(p Preset) {
	if isBuiltin(p.ID) {
		return
	}
	s.add(p)
}

// Upsert adds or replaces a user preset. Built-ins are never overwritten.
func (s *Store) Upsert(p Preset) {
	if isBuiltin(p.ID) {
		return
	}
	s.add(p)
	if s.activeID == p.ID {
		s.activeID = p.ID
	}
}

// ValidatePreset checks a preset before it is stored.
func (s *Store) ValidatePreset(p Preset) error {
	if strings.TrimSpace(p.Name) == "" {
		return errf("Введите название пресета")
	}
	if p.Kind == "expression" {
		return Validate(p.Expression, statTokens)
	}
	if p.Kind != "linear" {
		return errf("Неизвестный вид пресета")
	}
	return nil
}

// Remove deletes a preset; built-ins and unknown ids are refused.
func (s *Store) Remove(id string) bool {
	if id == "standard" {
		return false
	}
	if _, ok := s.presets[id]; !ok {
		return false
	}
	delete(s.presets, id)
	if s.activeID == id {
		s.activeID = "standard"
	}
	return true
}

// SetActive switches the active preset.
func (s *Store) SetActive(id string) bool {
	if _, ok := s.presets[id]; !ok {
		return false
	}
	s.activeID = id
	return true
}

// ImportFile loads a preset from a JSON file. Name/id collisions are
// re-generated so imports never clobber existing presets.
func (s *Store) ImportFile(path string) (Preset, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Preset{}, errf("Не удалось прочитать файл: %v", err)
	}
	var j jsonPreset
	if err := json.Unmarshal(data, &j); err != nil {
		return Preset{}, errf("Не удалось прочитать файл: %v", err)
	}
	p := j.preset()
	if p.Kind == "expression" && strings.TrimSpace(p.Expression) == "" {
		return Preset{}, errf("Файл не содержит выражения формулы")
	}
	if _, existing := s.presets[p.ID]; existing || p.ID == "standard" {
		p.ID = newID()
	}
	s.add(p)
	return p, nil
}

// ExportFile writes a single preset to a JSON file.
func (s *Store) ExportFile(p Preset, path string) error {
	body, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("export: %w", err)
	}
	return os.WriteFile(path, body, 0o644)
}
