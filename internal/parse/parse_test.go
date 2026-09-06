package parse

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/klauspost/compress/zstd"
)

const fixturePath = "../testdata/match.json"

func rawFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestReadAggregatedSchema(t *testing.T) {
	r, err := ReadAggregatedJSON(rawFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	if r.MatchID != 8926354517 || r.DurationSec != 2507 || r.RadiantWin || len(r.Players) != 10 {
		t.Fatalf("%+v", r)
	}
}

func TestReadAggregatedBadJSON(t *testing.T) {
	if _, err := ReadAggregatedJSON([]byte("not json {")); err == nil {
		t.Fatal("want error")
	}
}

func TestReadAggregatedNoPlayers(t *testing.T) {
	if _, err := ReadAggregatedJSON([]byte(`{"match_id": 1, "duration_sec": 10}`)); err == nil {
		t.Fatal("want no-players error")
	}
}

func TestAggregatesBuffs(t *testing.T) {
	r, err := ReadAggregatedJSON([]byte(`
	{
	  "match_id": 1,
	  "duration_sec": 60,
	  "radiant_win": true,
	  "players": [
	    {
	      "steam_id": 1, "player_id": 0, "hero_id": 1, "hero": "H",
	      "team": "radiant", "name": "n", "kills": 1, "deaths": 2,
	      "assists": 3, "level": 4, "last_hits": 5, "networth": 6,
	      "gpm": 7, "xpm": 8, "healing": 0, "hero_damage": 0,
	      "damage_taken": 0, "tower_damage": 0, "gold_lost": 120,
	      "buff_duration": 120.5,
	      "buff_sources": [
	        {"inflictor": "a", "category": "save", "duration": 30.0},
	        {"inflictor": "b", "category": "purge", "duration": 10.5},
	        {"inflictor": "c", "category": "shield", "duration": 5.0},
	        {"inflictor": "d", "category": "heal", "duration": 100.0},
	        {"inflictor": "e", "category": "save", "duration": 1.5}
	      ]
	    }
	  ]
	}
	`))
	if err != nil {
		t.Fatal(err)
	}
	p := r.Players[0]
	if p.BuffsDuration != 120.5 || p.Save != 31.5 || p.Purge != 10.5 || p.ShieldUptime != 5.0 {
		t.Fatalf("buffs: %+v", p)
	}
	if p.GoldLost != 120 {
		t.Fatalf("gold_lost %v", p.GoldLost)
	}
}

func TestFromPlayerNoBuffs(t *testing.T) {
	r, err := ReadAggregatedJSON([]byte(`
	{"duration_sec": 1, "radiant_win": true,
	 "players": [
	   {"steam_id": 1, "player_id": 0, "hero_id": 1, "hero": "H",
	    "team": "radiant", "name": "n", "kills": 0, "deaths": 0,
	    "assists": 0, "level": 1, "last_hits": 0, "networth": 0,
	    "gpm": 0, "xpm": 0}
	 ]}
	`))
	if err != nil {
		t.Fatal(err)
	}
	p := r.Players[0]
	if p.Save != 0 || p.Purge != 0 || p.ShieldUptime != 0 || p.BuffsDuration != 0 {
		t.Fatalf("%+v", p)
	}
}

func TestLegacyValueFallback(t *testing.T) {
	r, err := ReadAggregatedJSON([]byte(`
	{"duration_sec": 1, "radiant_win": true,
	 "players": [
	   {"steam_id": 1, "player_id": 0, "hero_id": 1, "hero": "H",
	    "team": "radiant", "name": "n",
	    "buff_sources": [{"inflictor": "x", "category": "save", "value": 7.5}]}
	 ]}
	`))
	if err != nil {
		t.Fatal(err)
	}
	if r.Players[0].Save != 7.5 {
		t.Fatalf("save %v", r.Players[0].Save)
	}
}

func TestParseReplayJSON(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "match.json")
	if err := os.WriteFile(tmp, rawFixture(t), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := ParseReplay(context.Background(), tmp, "http://localhost:5600")
	if err != nil {
		t.Fatal(err)
	}
	if r.MatchID != 8926354517 || len(r.Players) != 10 {
		t.Fatalf("%+v", r)
	}
}

func TestParseReplayBzip2(t *testing.T) {
	compressed := filepath.Join("..", "testdata", "match.json.bz2")
	r, err := ParseReplay(context.Background(), compressed, "http://localhost:5600")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Players) != 10 {
		t.Fatalf("players %d", len(r.Players))
	}
}

func TestParseReplayZstd(t *testing.T) {
	dir := t.TempDir()
	compressed := filepath.Join(dir, "match.json.zst")
	writeZstd(t, compressed, rawFixture(t))

	r, err := ParseReplay(context.Background(), compressed, "http://localhost:5600")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Players) != 10 {
		t.Fatalf("players %d", len(r.Players))
	}
}

func TestParseReplayMissingFile(t *testing.T) {
	if _, err := ParseReplay(context.Background(), filepath.Join(t.TempDir(), "nope.dem"), ""); err == nil {
		t.Fatal("want error")
	}
}

func writeZstd(t *testing.T, path string, data []byte) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	enc, err := zstd.NewWriter(f)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := enc.Write(data); err != nil {
		t.Fatal(err)
	}
	enc.Close()
	f.Close()
}

// canonicalHero must resolve legacy unit-derived snake_case names to the
// valve/opendota api slug (so Steam CDN icon URLs and the display-name map
// line up). id 20 = Vengeful Spirit, 39 = Queen of Pain.
func TestCanonicalHero(t *testing.T) {
	cases := []struct {
		id   int
		raw  string
		want string
	}{
		{20, "vengeful_spirit", "vengefulspirit"},
		{39, "queen_of_pain", "queenofpain"},
		{86, "rubick", "rubick"},
		{0, "vengeless_hero", "vengeless_hero"},
		{0, "npc_dota_hero_doctor_who", "doctor_who"},
	}
	for _, c := range cases {
		got := canonicalHero(c.id, c.raw)
		if got != c.want {
			t.Errorf("canonicalHero(%d, %q) = %q, want %q", c.id, c.raw, got, c.want)
		}
	}
}

func TestFromPlayerNewStats(t *testing.T) {
	r, err := ReadAggregatedJSON([]byte(`
	{"duration_sec": 2507, "radiant_win": true,
	 "players": [
	   {"steam_id": 1, "player_id": 0, "hero_id": 1, "hero": "H",
	    "team": "radiant", "name": "n",
	    "wisdoms_captured": 3, "watchers_captured": 2,
	    "lotuses_gathered": 5, "courier_kills": 1}
	 ]}
	`))
	if err != nil {
		t.Fatal(err)
	}
	p := r.Players[0]
	if p.WisdomsCaptured != 3 || p.WatchersCaptured != 2 || p.LotusesGathered != 5 || p.CourierKills != 1 {
		t.Fatalf("new stats: %+v", p)
	}
	if r.DurationSec != 2507 {
		t.Fatalf("duration_sec %v", r.DurationSec)
	}
}
