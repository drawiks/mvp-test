package model

type Player struct {
	SteamID        int64
	PlayerID       int
	HeroID         int
	Hero           string
	Team           string
	Name           string
	Position       int
	Lane           string
	Kills          int
	Deaths         int
	Assists        int
	Level          int
	LastHits       int
	Networth       int
	GPM            int
	XPM            int
	Healing        float64
	HeroDamage     int
	DamageTaken    int
	TowerDamage    int
	TimeDead       float64
	StunDuration   float64
	GoldSpentWards int
	GoldSpentSmoke int
	GoldSpentDust  int
	CampsStacked   int
	CreepsStacked  int
	RunePickups    int
	FirstBlood     bool
	BuffsDuration  float64
	Save           float64
	Purge          float64
	ShieldUptime   float64

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
	GoldLost         float64
	WisdomsCaptured  int
	WatchersCaptured int
	LotusesGathered  int
	CourierKills     int
}

type Result struct {
	MatchID     int64
	DurationSec int64
	RadiantWin  bool
	Players     []Player
}

func (r *Result) WinnerTeam() string {
	if r.RadiantWin {
		return "radiant"
	}
	return "dire"
}

func (r *Result) LoserTeam() string {
	if r.RadiantWin {
		return "dire"
	}
	return "radiant"
}
