package parse

import (
	"bytes"
	"compress/bzip2"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/drawiks/odota-cli/parser"
	"github.com/klauspost/compress/zstd"
	"mvp/internal/model"
)

// Errors surfaced to the UI in user-facing messages.
var (
	ErrReplay     = errors.New("не удалось прочитать файл реплея")
	ErrParse      = errors.New("не удалось разобрать реплей")
	ErrHTTP       = errors.New("не удалось связаться с парсером")
	ErrNotJSON    = errors.New("файл не является реплеем: некорректный JSON")
	ErrNoPlayers  = errors.New("некорректный реплей: нет поля players")
	ErrParserDown = errors.New("парсер не отвечает. Запустите odota/parser на localhost:5600")
)

const (
	bz2Header  = "BZh"
	zstdHeader = "\x28\xb5\x2f\xfd\x00"
)

// slugByID maps hero ID -> valve/opendota api slug. The odota parser derives
// the "hero" string from the in-game unit name (snake_case of the
// CDOTA_Unit_Hero_* camelCase), which differs from the canonical api slug for
// a few legacy heroes (e.g. "vengeful_spirit" vs "vengefulspirit",
// "queen_of_pain" vs "queenofpain"). Canonicalizing by HeroID keeps icon URLs
// and display names aligned with the Steam CDN / opendota names.
var slugByID = func() map[int]string {
	m := make(map[int]string, len(parser.HeroMap))
	for key, id := range parser.HeroMap {
		m[id] = strings.TrimPrefix(key, "npc_dota_hero_")
	}
	return m
}()

func canonicalHero(id int, raw string) string {
	if slug, ok := slugByID[id]; ok {
		return slug
	}
	return strings.TrimPrefix(raw, "npc_dota_hero_")
}

// ReadAggregatedJSON maps an already-aggregated parser Match object into the
// app model (i.e. the JSON that odota-cli prints after parsing).
func ReadAggregatedJSON(data []byte) (model.Result, error) {
	var match parser.Match
	if err := json.Unmarshal(bytes.TrimSpace(data), &match); err != nil {
		return model.Result{}, fmt.Errorf("%w: %v", ErrNotJSON, err)
	}
	if len(match.Players) == 0 && !hasPlayersField(data) {
		return model.Result{}, ErrNoPlayers
	}
	return FromMatch(&match), nil
}

func hasPlayersField(data []byte) bool {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(bytes.TrimSpace(data), &probe); err != nil {
		return false
	}
	_, ok := probe["players"]
	return ok
}

// FromMatch maps an odota-cli aggregate into the app model, summing buff
// sources into the save/purge/shield uptime fields.
func FromMatch(m *parser.Match) model.Result {
	r := model.Result{
		MatchID:     m.MatchID,
		DurationSec: int64(m.DurationSec),
		RadiantWin:  m.RadiantWin,
	}
	for _, p := range m.Players {
		r.Players = append(r.Players, FromPlayer(p))
	}
	return r
}

// FromPlayer maps a single parsed player, deriving save/purge/shield_uptime
// from buff_sources categories.
func FromPlayer(p parser.Player) model.Player {
	return model.Player{
		SteamID:        p.SteamID,
		PlayerID:       p.PlayerID,
		HeroID:         p.HeroID,
		Hero:           canonicalHero(p.HeroID, p.Hero),
		Team:           p.Team,
		Name:           p.Name,
		Kills:          p.Kills,
		Deaths:         p.Deaths,
		Assists:        p.Assists,
		Level:          p.Level,
		LastHits:       p.LastHits,
		Networth:       p.Networth,
		GPM:            p.GPM,
		XPM:            p.XPM,
		Healing:        float64(p.Healing),
		HeroDamage:     p.HeroDamage,
		DamageTaken:    p.DamageTaken,
		TowerDamage:    p.TowerDamage,
		TimeDead:       p.TimeDead,
		StunDuration:   p.StunDuration,
		GoldSpentWards: p.GoldSpentWards,
		GoldSpentSmoke: p.GoldSpentSmoke,
		GoldSpentDust:  p.GoldSpentDust,
		CampsStacked:   p.CampsStacked,
		CreepsStacked:  p.CreepsStacked,
		RunePickups:    p.RunePickups,
		FirstBlood:     p.FirstBlood,
		BuffsDuration:  p.BuffDuration,
		Save:           sumCategory(p.BuffSources, "save"),
		Purge:          sumCategory(p.BuffSources, "purge"),
		ShieldUptime:   sumCategory(p.BuffSources, "shield"),

		FearDuration:    p.FearDuration,
		RootsDuration:   p.RootsDuration,
		LeashDuration:   p.LeashDuration,
		TrapDuration:    p.TrapDuration,
		TauntDuration:   p.TauntDuration,
		SilenceDuration: p.SilenceDuration,
		BreakDuration:   p.BreakDuration,
		DisarmDuration:  p.DisarmDuration,
		HealDuration:    p.HealDuration,
		HealValue:       p.HealValue,
		GoldLost:        float64(p.GoldLost),
	}
}

// sumCategory totals buff durations for one category. Entries that only carry
// a legacy "value" (seconds) are counted through that field instead.
func sumCategory(sources []parser.SourceEntry, category string) float64 {
	var total float64
	for _, s := range sources {
		if s.Category != category {
			continue
		}
		if s.Duration != 0 {
			total += s.Duration
		} else {
			total += s.Value
		}
	}
	return total
}

// ParseReplay is the full pipeline: decompress (bz2/zstd), then either POST
// the replay to odota/parser or aggregate local event JSON.
func ParseReplay(ctx context.Context, path, parserURL string) (model.Result, error) {
	kind := bySuffix(strings.ToLower(strings.TrimSuffix(strings.TrimSuffix(path, ".bz2"), ".zst")))

	input := path
	if compressedHeader(path) {
		tmp, err := decompress(path)
		if err != nil {
			return model.Result{}, err
		}
		input = tmp
		defer os.Remove(input)
	}

	switch kind {
	case "ndjson":
		return parseNDJSONFile(input)
	case "json":
		return parseJSONFile(input)
	default:
		return parseFromHTTP(ctx, input, parserURL)
	}
}

func bySuffix(lower string) string {
	switch {
	case strings.HasSuffix(lower, ".ndjson"):
		return "ndjson"
	case strings.HasSuffix(lower, ".json"):
		return "json"
	default:
		return "dem"
	}
}

func parseNDJSONFile(path string) (model.Result, error) {
	events, err := parser.ReadNDJSONFile(path)
	if err != nil {
		return model.Result{}, fmt.Errorf("%w: %v", ErrParse, err)
	}
	return aggregate(events)
}

func parseJSONFile(path string) (model.Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return model.Result{}, fmt.Errorf("%w: %v", ErrReplay, err)
	}
	// Aggregated Match objects carry a players array; raw event files do not.
	if hasPlayersField(data) {
		return ReadAggregatedJSON(data)
	}
	events, err := readNDJSON(data)
	if err != nil {
		return model.Result{}, fmt.Errorf("%w: %v", ErrParse, err)
	}
	return aggregate(events)
}

func readNDJSON(data []byte) ([]parser.RawEvent, error) {
	tmp, err := os.CreateTemp("", "mvp-events-*.ndjson")
	if err != nil {
		return nil, err
	}
	path := tmp.Name()
	defer os.Remove(path)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return nil, err
	}
	tmp.Close()
	return parser.ReadNDJSONFile(path)
}

func aggregate(events []parser.RawEvent) (model.Result, error) {
	match, err := parser.Aggregate(events)
	if err != nil {
		return model.Result{}, fmt.Errorf("%w: %v", ErrParse, err)
	}
	return FromMatch(match), nil
}

func parseFromHTTP(ctx context.Context, path, parserURL string) (model.Result, error) {
	demData, err := os.ReadFile(path)
	if err != nil {
		return model.Result{}, fmt.Errorf("%w: %v", ErrReplay, err)
	}
	events, err := parser.FetchFromParser(demData, parserURL)
	if err != nil {
		return model.Result{}, fmt.Errorf("%w: %v", ErrHTTP, err)
	}
	return aggregate(events)
}

// compressedHeader reports whether the file begins with a bzip2 or zstd magic.
func compressedHeader(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	header := make([]byte, 4)
	n, _ := io.ReadFull(f, header)
	header = header[:n]
	return string(header[:min(n, 3)]) == bz2Header || bytes.Equal(header, []byte(zstdHeader[:len(header)]))
}

// decompress expands a bzip2/zstd file into a temporary uncompressed copy.
func decompress(path string) (string, error) {
	src, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrReplay, err)
	}
	defer src.Close()
	dst, err := os.CreateTemp("", "mvp-*.dem")
	if err != nil {
		return "", err
	}
	defer dst.Close()

	header := make([]byte, 3)
	n, _ := io.ReadFull(src, header)
	bz := string(header[:n]) == bz2Header

	if _, err := src.Seek(0, io.SeekStart); err != nil {
		os.Remove(dst.Name())
		return "", err
	}
	var reader io.Reader = src
	if bz {
		reader = bzip2.NewReader(src)
	} else {
		dec, err := zstd.NewReader(src)
		if err != nil {
			os.Remove(dst.Name())
			return "", fmt.Errorf("%w: %v", ErrReplay, err)
		}
		defer dec.Close()
		reader = dec
	}
	if _, err := io.Copy(dst, reader); err != nil {
		os.Remove(dst.Name())
		return "", fmt.Errorf("%w: %v", ErrReplay, err)
	}
	return dst.Name(), nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
