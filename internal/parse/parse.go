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

	"github.com/drawiks/manta-cli"
	"github.com/klauspost/compress/zstd"
	"mvp/internal/model"
)

// Errors surfaced to the UI in user-facing messages.
var (
	ErrReplay    = errors.New("не удалось прочитать файл реплея")
	ErrParse     = errors.New("не удалось разобрать реплей")
	ErrNotJSON   = errors.New("файл не является реплеем: некорректный JSON")
	ErrNoPlayers = errors.New("некорректный реплей: нет поля players")
)

const (
	bz2Header  = "BZh"
	zstdHeader = "\x28\xb5\x2f\xfd\x00"
)

var heroAliases = map[string]string{
	"anti_mage":       "antimage",
	"vengeful_spirit": "vengefulspirit",
	"queen_of_pain":   "queenofpain",
	"doom":            "doom_bringer",
}

func canonicalHero(raw string) string {
	slug := strings.TrimPrefix(raw, "npc_dota_hero_")
	if alias, ok := heroAliases[slug]; ok {
		return alias
	}
	return slug
}

// ReadAggregatedJSON maps manta-cli's aggregated match JSON into the app model.
func ReadAggregatedJSON(data []byte) (model.Result, error) {
	var match manta.Match
	if err := json.Unmarshal(bytes.TrimSpace(data), &match); err != nil {
		return model.Result{}, fmt.Errorf("%w: %v", ErrNotJSON, err)
	}
	if len(match.Players) == 0 && !hasPlayersField(data) {
		return model.Result{}, ErrNoPlayers
	}
	return fromManta(&match), nil
}

func hasPlayersField(data []byte) bool {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(bytes.TrimSpace(data), &probe); err != nil {
		return false
	}
	_, ok := probe["players"]
	return ok
}

func fromManta(match *manta.Match) model.Result {
	r := model.Result{
		MatchID:     match.MatchID,
		DurationSec: int64(match.DurationSec),
		RadiantWin:  match.RadiantWin,
	}
	for _, p := range match.Players {
		r.Players = append(r.Players, model.Player{
			SteamID:              p.SteamID,
			PlayerID:             p.PlayerID,
			HeroID:               p.HeroID,
			Hero:                 canonicalHero(p.Hero),
			Team:                 p.Team,
			Name:                 p.Name,
			Position:             p.Position,
			Lane:                 p.Lane,
			Kills:                p.Kills,
			Deaths:               p.Deaths,
			Assists:              p.Assists,
			Level:                p.Level,
			LastHits:             p.LastHits,
			Networth:             p.Networth,
			GPM:                  p.GPM,
			XPM:                  p.XPM,
			Healing:              float64(p.Healing),
			ManaRestored:         float64(p.ManaRestored),
			HeroDamage:           p.HeroDamage,
			DamageTaken:          p.DamageTaken,
			TowerDamage:          p.TowerDamage,
			TimeDead:             p.TimeDead,
			StunDuration:         p.StunDuration,
			GoldSpentWards:       p.GoldSpentWards,
			GoldSpentSmoke:       p.GoldSpentSmoke,
			GoldSpentDust:        p.GoldSpentDust,
			GoldLost:             p.GoldLost,
			CampsStacked:         p.CampsStacked,
			CreepsStacked:        p.CreepsStacked,
			RunePickups:          p.RunePickups,
			FirstBlood:           p.FirstBlood,
			BuffsDuration:        p.BuffDuration,
			Save:                 sumCategory(p.BuffSources, "save"),
			Purge:                sumCategory(p.BuffSources, "purge"),
			ShieldUptime:         sumCategory(p.BuffSources, "shield"),
			BuffStatsDuration:    sumCategory(p.BuffSources, "buff_stats"),
			InvisibilityDuration: sumCategory(p.BuffSources, "invisibility"),
			BuffHasteDuration:    sumCategory(p.BuffSources, "buff_haste"),

			FearDuration:     p.FearDuration,
			RootsDuration:    p.RootsDuration,
			LeashDuration:    p.LeashDuration,
			TrapDuration:     p.TrapDuration,
			TauntDuration:    p.TauntDuration,
			SilenceDuration:  p.SilenceDuration,
			BreakDuration:    p.BreakDuration,
			DisarmDuration:   p.DisarmDuration,
			WisdomsCaptured:  p.WisdomsCaptured,
			WatchersCaptured: p.WatchersCaptured,
			LotusesGathered:  p.LotusesGathered,
			CourierKills:     p.CourierKills,
		})
	}
	return r
}

// sumCategory totals buff stats for one category. Purges are reported as an
// application count; older output carried a duration/value instead.
func sumCategory(sources []manta.SourceEntry, category string) float64 {
	var total float64
	for _, s := range sources {
		if s.Category != category {
			continue
		}
		switch {
		case s.Count != 0:
			total += float64(s.Count)
		case s.Duration != 0:
			total += s.Duration
		default:
			total += s.Value
		}
	}
	return total
}

// ParseReplay parses a manta .dem replay or an already aggregated match JSON.
func ParseReplay(ctx context.Context, path string) (model.Result, error) {
	kind := bySuffix(strings.ToLower(strings.TrimSuffix(strings.TrimSuffix(path, ".bz2"), ".zst")))
	if kind == "ndjson" {
		return model.Result{}, fmt.Errorf("%w: NDJSON больше не поддерживается", ErrParse)
	}
	if kind == "json" {
		input := path
		if compressedHeader(path) {
			tmp, err := decompress(path)
			if err != nil {
				return model.Result{}, err
			}
			input = tmp
			defer os.Remove(input)
		}
		return parseJSONFile(input)
	}
	return parseManta(ctx, path)
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

func parseJSONFile(path string) (model.Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return model.Result{}, fmt.Errorf("%w: %v", ErrReplay, err)
	}
	return ReadAggregatedJSON(data)
}

func parseManta(ctx context.Context, path string) (model.Result, error) {
	match, err := manta.ParseFile(ctx, path)
	if err != nil {
		switch {
		case errors.Is(err, manta.ErrNotReplay):
			return model.Result{}, fmt.Errorf("%w: %v", ErrReplay, err)
		case errors.Is(err, manta.ErrNoPlayerResource):
			return model.Result{}, fmt.Errorf("%w: %v", ErrNoPlayers, err)
		default:
			return model.Result{}, fmt.Errorf("%w: %v", ErrParse, err)
		}
	}
	return fromManta(match), nil
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

// decompress expands a bzip2/zstd JSON file into a temporary uncompressed copy.
func decompress(path string) (string, error) {
	src, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrReplay, err)
	}
	defer src.Close()
	dst, err := os.CreateTemp("", "mvp-*.json")
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
