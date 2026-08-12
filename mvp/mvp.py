from __future__ import annotations

from dataclasses import asdict, dataclass

from .model import Player, Result

@dataclass(frozen=True)
class Weights:
    kills: float = 0.3
    deaths: float = 0.3
    deaths_base: float = 3.0
    assists: float = 0.15
    last_hits: float = 0.003
    gpm: float = 0.002
    xpm: float = 0.002
    stun: float = 0.05
    healing: float = 0.004
    tower_damage: float = 0.001
    camps: float = 0.5
    runes: float = 0.2
    first_blood: float = 1.0


DEFAULT_WEIGHTS = Weights()

_WEIGHT_FIELDS: tuple[str, ...] = tuple(Weights.__dataclass_fields__.keys())


def weights_from_mapping(mapping: dict) -> Weights:
    return Weights(**{k: float(mapping[k]) for k in _WEIGHT_FIELDS if k in mapping})


def weights_to_mapping(weights: Weights) -> dict:
    return asdict(weights)


def score_breakdown(player: Player, weights: Weights = DEFAULT_WEIGHTS) -> dict[str, float]:
    return {
        "kills": player.kills * weights.kills,
        "deaths": weights.deaths_base - player.deaths * weights.deaths,
        "assists": player.assists * weights.assists,
        "last_hits": player.last_hits * weights.last_hits,
        "gpm": player.gpm * weights.gpm,
        "xpm": player.xpm * weights.xpm,
        "stun": player.stun_duration * weights.stun,
        "healing": player.healing * weights.healing,
        "tower_damage": player.tower_damage * weights.tower_damage,
        "camps": player.camps_stacked * weights.camps,
        "runes": player.rune_pickups * weights.runes,
        "first_blood": weights.first_blood * (1.0 if player.first_blood else 0.0),
    }


def compute_score(player: Player, weights: Weights = DEFAULT_WEIGHTS) -> float:
    return sum(score_breakdown(player, weights).values())


def ranked_players(result: Result, weights: Weights = DEFAULT_WEIGHTS) -> list[Player]:
    return sorted(
        (p for p in result.players if p.team in ("radiant", "dire")),
        key=lambda p: compute_score(p, weights),
        reverse=True,
    )


def select_mvps(result: Result, weights: Weights = DEFAULT_WEIGHTS) -> dict[str, Player | None]:
    winner_ranked = sorted(
        (p for p in result.players if p.team == result.winner_team),
        key=lambda p: compute_score(p, weights),
        reverse=True,
    )
    loser_ranked = sorted(
        (p for p in result.players if p.team == result.loser_team),
        key=lambda p: compute_score(p, weights),
        reverse=True,
    )
    return {
        "winner_top1": winner_ranked[0] if winner_ranked else None,
        "winner_top2": winner_ranked[1] if len(winner_ranked) > 1 else None,
        "loser_top1": loser_ranked[0] if loser_ranked else None,
    }
