from __future__ import annotations

from dataclasses import dataclass, field


@dataclass
class Player:
    steam_id: int = 0
    player_id: int = 0
    hero_id: int = 0
    hero: str = ""
    team: str = ""
    name: str = ""
    kills: int = 0
    deaths: int = 0
    assists: int = 0
    level: int = 0
    last_hits: int = 0
    networth: int = 0
    gpm: int = 0
    xpm: int = 0
    healing: float = 0.0
    hero_damage: int = 0
    damage_taken: int = 0
    tower_damage: int = 0
    time_dead: float = 0.0
    stun_duration: float = 0.0
    gold_spent_wards: int = 0
    gold_spent_smoke: int = 0
    gold_spent_dust: int = 0
    camps_stacked: int = 0
    creeps_stacked: int = 0
    rune_pickups: int = 0
    first_blood: bool = False
    buffs_duration: float = 0.0
    save: float = 0.0
    purge: float = 0.0
    shield_uptime: float = 0.0


@dataclass
class Result:
    match_id: int = 0
    duration_sec: int = 0
    radiant_win: bool = True
    players: list[Player] = field(default_factory=list)

    @property
    def winner_team(self) -> str:
        return "dire" if not self.radiant_win else "radiant"

    @property
    def loser_team(self) -> str:
        return "radiant" if not self.radiant_win else "dire"
