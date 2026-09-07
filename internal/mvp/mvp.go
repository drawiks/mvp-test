package mvp

import (
	"sort"

	"mvp/internal/formula"
	"mvp/internal/model"
)

type Weights struct {
	Kills          float64
	Deaths         float64
	Assists        float64
	LastHits       float64
	GPM            float64
	XPM            float64
	Stun           float64
	Healing        float64
	TowerDamage    float64
	Camps          float64
	Runes          float64
	FirstBlood     float64
	HeroDamage     float64
	DamageTaken    float64
	GoldSpentWards float64
	GoldSpentSmoke float64
	GoldSpentDust  float64
	BuffsDuration  float64
	Save           float64
	Purge          float64
	ShieldUptime   float64
	BuffStatsDuration float64
	InvisibilityDuration float64
	BuffHasteDuration float64

	FearDuration     float64
	RootsDuration    float64
	LeashDuration    float64
	TrapDuration     float64
	TauntDuration    float64
	SilenceDuration  float64
	BreakDuration    float64
	DisarmDuration   float64
	HealDuration     float64
	HealValue        float64
	CreepsStacked    float64
	TimeDead         float64
	WisdomsCaptured  float64
	WatchersCaptured float64
	LotusesGathered  float64
	CourierKills     float64
}

var DefaultWeights = Weights{
	Kills: 0.3, Deaths: 0.3, Assists: 0.15,
	LastHits: 0.003, GPM: 0.002, XPM: 0.002, Stun: 0.05,
	Healing: 0.004, TowerDamage: 0.001, Camps: 0.5, Runes: 0.2,
	FirstBlood: 1.0,
}

var weightKeys = map[string]weightField{
	"kills":        {"Kills", "kills", func(p *model.Player) float64 { return float64(p.Kills) }},
	"deaths":       {"Deaths", "deaths", func(p *model.Player) float64 { return float64(p.Deaths) }},
	"assists":      {"Assists", "assists", func(p *model.Player) float64 { return float64(p.Assists) }},
	"last_hits":    {"LastHits", "last_hits", func(p *model.Player) float64 { return float64(p.LastHits) }},
	"gpm":          {"GPM", "gpm", func(p *model.Player) float64 { return float64(p.GPM) }},
	"xpm":          {"XPM", "xpm", func(p *model.Player) float64 { return float64(p.XPM) }},
	"stun_duration": {"Stun", "stun_duration", func(p *model.Player) float64 { return p.StunDuration }},
	"healing":      {"Healing", "healing", func(p *model.Player) float64 { return p.Healing }},
	"tower_damage": {"TowerDamage", "tower_damage", func(p *model.Player) float64 { return float64(p.TowerDamage) }},
	"camps_stacked": {"Camps", "camps_stacked", func(p *model.Player) float64 { return float64(p.CampsStacked) }},
	"rune_pickups": {"Runes", "rune_pickups", func(p *model.Player) float64 { return float64(p.RunePickups) }},
	"first_blood": {"FirstBlood", "first_blood", func(p *model.Player) float64 {
		if p.FirstBlood {
			return 1
		}
		return 0
	}},
	"hero_damage":      {"HeroDamage", "hero_damage", func(p *model.Player) float64 { return float64(p.HeroDamage) }},
	"damage_taken":     {"DamageTaken", "damage_taken", func(p *model.Player) float64 { return float64(p.DamageTaken) }},
	"gold_spent_wards": {"GoldSpentWards", "gold_spent_wards", func(p *model.Player) float64 { return float64(p.GoldSpentWards) }},
	"gold_spent_smoke": {"GoldSpentSmoke", "gold_spent_smoke", func(p *model.Player) float64 { return float64(p.GoldSpentSmoke) }},
	"gold_spent_dust":  {"GoldSpentDust", "gold_spent_dust", func(p *model.Player) float64 { return float64(p.GoldSpentDust) }},
	"buff_duration":    {"BuffsDuration", "buff_duration", func(p *model.Player) float64 { return p.BuffsDuration }},
	"save_duration":    {"Save", "save_duration", func(p *model.Player) float64 { return p.Save }},
	"purge_duration":   {"Purge", "purge_duration", func(p *model.Player) float64 { return p.Purge }},
	"shield_duration":  {"ShieldUptime", "shield_duration", func(p *model.Player) float64 { return p.ShieldUptime }},
	"buff_stats_duration": {"BuffStatsDuration", "buff_stats_duration", func(p *model.Player) float64 { return p.BuffStatsDuration }},
	"invisibility_duration": {"InvisibilityDuration", "invisibility_duration", func(p *model.Player) float64 { return p.InvisibilityDuration }},
	"buff_haste_duration": {"BuffHasteDuration", "buff_haste_duration", func(p *model.Player) float64 { return p.BuffHasteDuration }},
	"fear_duration":     {"FearDuration", "fear_duration", func(p *model.Player) float64 { return p.FearDuration }},
	"roots_duration":    {"RootsDuration", "roots_duration", func(p *model.Player) float64 { return p.RootsDuration }},
	"leash_duration":    {"LeashDuration", "leash_duration", func(p *model.Player) float64 { return p.LeashDuration }},
	"trap_duration":     {"TrapDuration", "trap_duration", func(p *model.Player) float64 { return p.TrapDuration }},
	"taunt_duration":    {"TauntDuration", "taunt_duration", func(p *model.Player) float64 { return p.TauntDuration }},
	"silence_duration":  {"SilenceDuration", "silence_duration", func(p *model.Player) float64 { return p.SilenceDuration }},
	"break_duration":    {"BreakDuration", "break_duration", func(p *model.Player) float64 { return p.BreakDuration }},
	"disarm_duration":   {"DisarmDuration", "disarm_duration", func(p *model.Player) float64 { return p.DisarmDuration }},
	"heal_duration":     {"HealDuration", "heal_duration", func(p *model.Player) float64 { return p.HealDuration }},
	"heal_value":        {"HealValue", "heal_value", func(p *model.Player) float64 { return p.HealValue }},
	"time_dead":         {"TimeDead", "time_dead", func(p *model.Player) float64 { return p.TimeDead }},
	"creeps_stacked":    {"CreepsStacked", "creeps_stacked", func(p *model.Player) float64 { return float64(p.CreepsStacked) }},
	"wisdoms_captured":  {"WisdomsCaptured", "wisdoms_captured", func(p *model.Player) float64 { return float64(p.WisdomsCaptured) }},
	"watchers_captured": {"WatchersCaptured", "watchers_captured", func(p *model.Player) float64 { return float64(p.WatchersCaptured) }},
	"lotuses_gathered":  {"LotusesGathered", "lotuses_gathered", func(p *model.Player) float64 { return float64(p.LotusesGathered) }},
	"courier_kills":     {"CourierKills", "courier_kills", func(p *model.Player) float64 { return float64(p.CourierKills) }},
}

type weightField struct {
	field  string
	token  string
	getter func(*model.Player) float64
}

func (w *Weights) value(key string) float64 {
	f := weightKeys[key]
	switch f.field {
	case "Kills":
		return w.Kills
	case "Deaths":
		return w.Deaths
	case "Assists":
		return w.Assists
	case "LastHits":
		return w.LastHits
	case "GPM":
		return w.GPM
	case "XPM":
		return w.XPM
	case "Stun":
		return w.Stun
	case "Healing":
		return w.Healing
	case "TowerDamage":
		return w.TowerDamage
	case "Camps":
		return w.Camps
	case "Runes":
		return w.Runes
	case "FirstBlood":
		return w.FirstBlood
	case "HeroDamage":
		return w.HeroDamage
	case "DamageTaken":
		return w.DamageTaken
	case "GoldSpentWards":
		return w.GoldSpentWards
	case "GoldSpentSmoke":
		return w.GoldSpentSmoke
	case "GoldSpentDust":
		return w.GoldSpentDust
	case "BuffsDuration":
		return w.BuffsDuration
	case "Save":
		return w.Save
	case "Purge":
		return w.Purge
	case "ShieldUptime":
		return w.ShieldUptime
	case "BuffStatsDuration":
		return w.BuffStatsDuration
	case "InvisibilityDuration":
		return w.InvisibilityDuration
	case "BuffHasteDuration":
		return w.BuffHasteDuration
	case "FearDuration":
		return w.FearDuration
	case "RootsDuration":
		return w.RootsDuration
	case "LeashDuration":
		return w.LeashDuration
	case "TrapDuration":
		return w.TrapDuration
	case "TauntDuration":
		return w.TauntDuration
	case "SilenceDuration":
		return w.SilenceDuration
	case "BreakDuration":
		return w.BreakDuration
	case "DisarmDuration":
		return w.DisarmDuration
	case "HealDuration":
		return w.HealDuration
	case "HealValue":
		return w.HealValue
	case "CreepsStacked":
		return w.CreepsStacked
	case "TimeDead":
		return w.TimeDead
	case "WisdomsCaptured":
		return w.WisdomsCaptured
	case "WatchersCaptured":
		return w.WatchersCaptured
	case "LotusesGathered":
		return w.LotusesGathered
	case "CourierKills":
		return w.CourierKills
	}
	return 0
}

func WeightsToMapping(w Weights) map[string]float64 {
	out := map[string]float64{}
	for key := range weightKeys {
		out[key] = w.value(key)
	}
	return out
}

func WeightsFromMapping(mapping map[string]float64) Weights {
	var w Weights
	for key, v := range mapping {
		f, ok := weightKeys[key]
		if !ok {
			continue
		}
		switch f.field {
		case "Kills":
			w.Kills = v
		case "Deaths":
			w.Deaths = v
		case "Assists":
			w.Assists = v
		case "LastHits":
			w.LastHits = v
		case "GPM":
			w.GPM = v
		case "XPM":
			w.XPM = v
		case "Stun":
			w.Stun = v
		case "Healing":
			w.Healing = v
		case "TowerDamage":
			w.TowerDamage = v
		case "Camps":
			w.Camps = v
		case "Runes":
			w.Runes = v
		case "FirstBlood":
			w.FirstBlood = v
		case "HeroDamage":
			w.HeroDamage = v
		case "DamageTaken":
			w.DamageTaken = v
		case "GoldSpentWards":
			w.GoldSpentWards = v
		case "GoldSpentSmoke":
			w.GoldSpentSmoke = v
		case "GoldSpentDust":
			w.GoldSpentDust = v
		case "BuffsDuration":
			w.BuffsDuration = v
		case "Save":
			w.Save = v
		case "Purge":
			w.Purge = v
		case "ShieldUptime":
			w.ShieldUptime = v
		case "BuffStatsDuration":
			w.BuffStatsDuration = v
		case "InvisibilityDuration":
			w.InvisibilityDuration = v
		case "BuffHasteDuration":
			w.BuffHasteDuration = v
		case "FearDuration":
			w.FearDuration = v
		case "RootsDuration":
			w.RootsDuration = v
		case "LeashDuration":
			w.LeashDuration = v
		case "TrapDuration":
			w.TrapDuration = v
		case "TauntDuration":
			w.TauntDuration = v
		case "SilenceDuration":
			w.SilenceDuration = v
		case "BreakDuration":
			w.BreakDuration = v
		case "DisarmDuration":
			w.DisarmDuration = v
		case "HealDuration":
			w.HealDuration = v
		case "HealValue":
			w.HealValue = v
		case "CreepsStacked":
			w.CreepsStacked = v
		case "TimeDead":
			w.TimeDead = v
		case "WisdomsCaptured":
			w.WisdomsCaptured = v
		case "WatchersCaptured":
			w.WatchersCaptured = v
		case "LotusesGathered":
			w.LotusesGathered = v
		case "CourierKills":
			w.CourierKills = v
		}
	}
	return w
}

func PlayerVars(p model.Player, duration float64) map[string]float64 {
	fb := 0.0
	if p.FirstBlood {
		fb = 1.0
	}
	return map[string]float64{
		"kills": float64(p.Kills), "deaths": float64(p.Deaths), "assists": float64(p.Assists),
		"last_hits": float64(p.LastHits), "gpm": float64(p.GPM), "xpm": float64(p.XPM),
		"stun_duration": p.StunDuration, "healing": p.Healing, "tower_damage": float64(p.TowerDamage),
		"camps_stacked": float64(p.CampsStacked), "creeps_stacked": float64(p.CreepsStacked),
		"rune_pickups": float64(p.RunePickups),
		"first_blood":  fb, "hero_damage": float64(p.HeroDamage), "damage_taken": float64(p.DamageTaken),
		"gold_spent_wards": float64(p.GoldSpentWards), "gold_spent_smoke": float64(p.GoldSpentSmoke),
		"gold_spent_dust": float64(p.GoldSpentDust), "buff_duration": p.BuffsDuration,
		"save_duration": p.Save, "purge_duration": p.Purge, "shield_duration": p.ShieldUptime,
		"buff_stats_duration": p.BuffStatsDuration, "invisibility_duration": p.InvisibilityDuration,
		"buff_haste_duration": p.BuffHasteDuration,
		"fear_duration": p.FearDuration, "roots_duration": p.RootsDuration,
		"leash_duration": p.LeashDuration, "trap_duration": p.TrapDuration,
		"taunt_duration": p.TauntDuration, "silence_duration": p.SilenceDuration,
		"break_duration": p.BreakDuration, "disarm_duration": p.DisarmDuration,
		"heal_duration": p.HealDuration, "heal_value": p.HealValue,
		"time_dead":        p.TimeDead,
		"wisdoms_captured": float64(p.WisdomsCaptured), "watchers_captured": float64(p.WatchersCaptured),
		"lotuses_gathered": float64(p.LotusesGathered), "courier_kills": float64(p.CourierKills),
		"match_duration": duration,
		"position":       float64(p.Position),
	}
}

func ScoreBreakdown(p model.Player, w Weights) map[string]float64 {
	out := map[string]float64{}
	for key, f := range weightKeys {
		if f.token == "" {
			continue
		}
		if f.token == "deaths" {
			continue
		}
		out[f.token] = f.getter(&p) * w.value(key)
	}
	out["deaths"] = -float64(p.Deaths) * w.Deaths
	return out
}

func PresetWeights(p formula.Preset) Weights {
	return WeightsFromMapping(p.Weights)
}

func ComputeScore(p model.Player, preset formula.Preset) (float64, error) {
	return ComputeScoreVars(p, 0, preset, nil)
}

func ComputeScoreVars(p model.Player, duration float64, preset formula.Preset, vars []formula.Variable) (float64, error) {
	if preset.Kind == "expression" {
		if preset.Expression == "" {
			return 0, nil
		}
		env := PlayerVars(p, duration)
		if len(vars) > 0 {
			var err error
			env, err = formula.ResolveVars(env, vars)
			if err != nil {
				return 0, err
			}
		}
		return formula.Eval(preset.Expression, env)
	}
	w := PresetWeights(preset)
	total := 0.0
	for _, v := range ScoreBreakdown(p, w) {
		total += v
	}
	return total, nil
}

func ScoreLinear(p model.Player, w Weights) float64 {
	total := 0.0
	wm := map[string]float64{}
	for key := range weightKeys {
		wm[key] = w.value(key)
	}
	preset := formula.Preset{ID: "linear", Name: "linear", Kind: "linear", Weights: wm}
	total, _ = ComputeScore(p, preset)
	return total
}

func RankedPlayers(result model.Result, preset formula.Preset, vars []formula.Variable) ([]model.Player, error) {
	type scored struct {
		player model.Player
		score  float64
	}
	list := make([]scored, 0, len(result.Players))
	duration := float64(result.DurationSec)
	for _, p := range result.Players {
		if p.Team != "radiant" && p.Team != "dire" {
			continue
		}
		s, err := ComputeScoreVars(p, duration, preset, vars)
		if err != nil {
			return nil, err
		}
		list = append(list, scored{p, s})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].score > list[j].score })
	out := make([]model.Player, len(list))
	for i := range list {
		out[i] = list[i].player
	}
	return out, nil
}

func RankTeam(result model.Result, team string, preset formula.Preset, vars []formula.Variable) ([]model.Player, error) {
	filtered := make([]model.Player, 0, 5)
	for _, p := range result.Players {
		if p.Team == team {
			filtered = append(filtered, p)
		}
	}
	full := result
	full.Players = filtered
	return RankedPlayers(full, preset, vars)
}

func SelectMvps(result model.Result, preset formula.Preset, vars []formula.Variable) (map[string]*model.Player, error) {
	type scored struct {
		player model.Player
		score  float64
	}
	pick := func(team string) ([]scored, error) {
		var list []scored
		duration := float64(result.DurationSec)
		for i := range result.Players {
			if result.Players[i].Team != team {
				continue
			}
			s, err := ComputeScoreVars(result.Players[i], duration, preset, vars)
			if err != nil {
				return nil, err
			}
			list = append(list, scored{result.Players[i], s})
		}
		sort.Slice(list, func(a, b int) bool { return list[a].score > list[b].score })
		return list, nil
	}
	winner, err := pick(result.WinnerTeam())
	if err != nil {
		return nil, err
	}
	loser, err := pick(result.LoserTeam())
	if err != nil {
		return nil, err
	}
	mvps := map[string]*model.Player{
		"winner_top1": nil,
		"winner_top2": nil,
		"loser_top1":  nil,
	}
	if len(winner) > 0 {
		mvps["winner_top1"] = &winner[0].player
	}
	if len(winner) > 1 {
		mvps["winner_top2"] = &winner[1].player
	}
	if len(loser) > 0 {
		mvps["loser_top1"] = &loser[0].player
	}
	return mvps, nil
}
