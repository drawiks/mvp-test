from __future__ import annotations

from dataclasses import asdict, dataclass

from .formula import Preset, safe_eval
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
    hero_damage: float = 0.0
    damage_taken: float = 0.0
    gold_spent_wards: float = 0.0
    gold_spent_smoke: float = 0.0
    gold_spent_dust: float = 0.0


DEFAULT_WEIGHTS = Weights()

_WEIGHT_FIELDS: tuple[str, ...] = tuple(Weights.__dataclass_fields__.keys())


def weights_from_mapping(mapping: dict) -> Weights:
    return Weights(**{k: float(mapping[k]) for k in _WEIGHT_FIELDS if k in mapping})


def weights_to_mapping(weights: Weights) -> dict:
    return asdict(weights)


def player_vars(player: Player) -> dict[str, float]:
    return {
        "kills": float(player.kills),
        "deaths": float(player.deaths),
        "assists": float(player.assists),
        "last_hits": float(player.last_hits),
        "gpm": float(player.gpm),
        "xpm": float(player.xpm),
        "stun_duration": float(player.stun_duration),
        "healing": float(player.healing),
        "tower_damage": float(player.tower_damage),
        "camps_stacked": float(player.camps_stacked),
        "rune_pickups": float(player.rune_pickups),
        "first_blood": 1.0 if player.first_blood else 0.0,
        "hero_damage": float(player.hero_damage),
        "damage_taken": float(player.damage_taken),
        "gold_spent_wards": float(player.gold_spent_wards),
        "gold_spent_smoke": float(player.gold_spent_smoke),
        "gold_spent_dust": float(player.gold_spent_dust),
    }


def _preset_weights(preset: Preset) -> Weights:
    return weights_from_mapping(
        {k: v for k, v in preset.weights.items() if k in _WEIGHT_FIELDS}
    )


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
        "hero_damage": player.hero_damage * weights.hero_damage,
        "damage_taken": player.damage_taken * weights.damage_taken,
        "gold_spent_wards": player.gold_spent_wards * weights.gold_spent_wards,
        "gold_spent_smoke": player.gold_spent_smoke * weights.gold_spent_smoke,
        "gold_spent_dust": player.gold_spent_dust * weights.gold_spent_dust,
    }


def as_weights(weights: Weights | Preset) -> Weights | None:
    if isinstance(weights, Weights):
        return weights
    if weights.kind == "linear":
        return _preset_weights(weights)
    return None


def compute_score(player: Player, weights: Weights | Preset = DEFAULT_WEIGHTS) -> float:
    if isinstance(weights, Preset):
        if weights.kind == "expression":
            if not weights.expression.strip():
                return 0.0
            return safe_eval(weights.expression, player_vars(player))
        weights = _preset_weights(weights)
    return sum(score_breakdown(player, weights).values())


def ranked_players(result: Result, weights: Weights | Preset = DEFAULT_WEIGHTS) -> list[Player]:
    return sorted(
        (p for p in result.players if p.team in ("radiant", "dire")),
        key=lambda p: compute_score(p, weights),
        reverse=True,
    )


def select_mvps(result: Result, weights: Weights | Preset = DEFAULT_WEIGHTS) -> dict[str, Player | None]:
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
