from __future__ import annotations

from pathlib import Path

import pytest

from mvp.manta_client import parse_result
from mvp.model import Result
from mvp.mvp import (
    DEFAULT_WEIGHTS,
    Weights,
    compute_score,
    ranked_players,
    select_mvps,
)

FIXTURE = Path(__file__).parent / "fixtures" / "match.json"


@pytest.fixture(scope="module")
def result() -> Result:
    return parse_result(FIXTURE.read_bytes())


def test_parse_result_schema(result: Result) -> None:
    assert result.match_id == 8926354517
    assert result.duration_sec == 1890
    assert result.radiant_win is False
    assert len(result.players) == 10


def test_default_weights_match_formula(result: Result) -> None:
    by_name = {p.name: p for p in result.players}
    expected = {
        "oyoy": 52.243995,
        "mvhoyeti": 46.113656,
        "юный дебустер": 32.418388,
        "Мясное пюре": 25.182000,
        "Master Control": 14.334008,
        "Scarry": 12.463657,
        "Beefsteeek": 7.107000,
        "а за мат извини": 6.227323,
        "Daniamaps": 4.957338,
        "re_triger": 3.655006,
    }
    for name, score in expected.items():
        assert compute_score(by_name[name], DEFAULT_WEIGHTS) == pytest.approx(score, abs=0.002)


def test_select_mvps_roles(result: Result) -> None:
    mvps = select_mvps(result, DEFAULT_WEIGHTS)
    assert mvps["winner_top1"].name == "oyoy"
    assert mvps["winner_top2"].name == "mvhoyeti"
    assert mvps["loser_top1"].name == "Master Control"
    assert mvps["winner_top1"].team == "dire"
    assert mvps["winner_top2"].team == "dire"
    assert mvps["loser_top1"].team == "radiant"


def test_ranked_players_sorted(result: Result) -> None:
    ranked = ranked_players(result, DEFAULT_WEIGHTS)
    scores = [compute_score(p, DEFAULT_WEIGHTS) for p in ranked]
    assert scores == sorted(scores, reverse=True)
    assert len(ranked) == 10


def test_custom_weights_only_first_blood(result: Result) -> None:
    weights = Weights(
        first_blood=10.0,
        deaths_base=0.0,
        deaths=0.0,
        kills=0.0,
        assists=0.0,
        last_hits=0.0,
        gpm=0.0,
        xpm=0.0,
        stun=0.0,
        healing=0.0,
        tower_damage=0.0,
        camps=0.0,
        runes=0.0,
    )
    mvps = select_mvps(result, weights)
    assert mvps["winner_top1"].first_blood is True
    assert compute_score(mvps["winner_top1"], weights) == pytest.approx(10.0)
    for p in result.players:
        if not p.first_blood:
            assert compute_score(p, weights) == pytest.approx(0.0)


def test_custom_weights_change_ranking(result: Result) -> None:
    weights = Weights(kills=1.0, deaths=0.0, deaths_base=0.0)
    top = ranked_players(result, weights)[0]
    assert top.name == "mvhoyeti"


def test_weights_to_from_mapping_roundtrip() -> None:
    from mvp.mvp import weights_from_mapping, weights_to_mapping

    assert weights_from_mapping(weights_to_mapping(DEFAULT_WEIGHTS)) == DEFAULT_WEIGHTS
    custom = Weights(kills=2.5, first_blood=0.0)
    assert weights_from_mapping(weights_to_mapping(custom)) == custom


def test_expression_preset_matches_linear_defaults(result: Result) -> None:
    from mvp.formula import Preset

    expression = (
        "kills * 0.3 + (3 - deaths * 0.3) + assists * 0.15 + last_hits * 0.003"
        " + gpm * 0.002 + xpm * 0.002 + stun_duration * 0.05 + healing * 0.004"
        " + tower_damage * 0.001 + camps_stacked * 0.5 + rune_pickups * 0.2 + first_blood"
    )
    preset = Preset(id="expr", name="Expr", kind="expression", expression=expression)
    by_name = {p.name: p for p in result.players}
    for name, expected in {
        "oyoy": 52.243995,
        "mvhoyeti": 46.113656,
        "Master Control": 14.334008,
    }.items():
        assert compute_score(by_name[name], preset) == pytest.approx(expected, abs=0.002)


def test_expression_preset_select_mvps(result: Result) -> None:
    from mvp.formula import Preset

    preset = Preset(
        id="expr",
        name="Expr",
        kind="expression",
        expression="(kills * 3 + assists * 1.5) / max(deaths, 1)",
    )
    mvps = select_mvps(result, preset)
    assert mvps["winner_top1"].team == "dire"
    assert mvps["winner_top2"].team == "dire"
    assert mvps["loser_top1"].team == "radiant"
    assert compute_score(mvps["winner_top1"], preset) > 0
