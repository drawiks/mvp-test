from __future__ import annotations

import pytest

from mvp.manta_client import MantaParseError, parse_result


def test_parse_result_full() -> None:
    raw = """
    {
      "match_id": 123,
      "duration_sec": 300,
      "radiant_win": true,
      "players": [
        {
          "steam_id": 1, "player_id": 0, "hero_id": 62, "hero": "Slark",
          "team": "radiant", "name": "n", "kills": 2, "deaths": 3,
          "assists": 4, "level": 5, "last_hits": 6, "networth": 7,
          "gpm": 8, "xpm": 9, "healing": 1.5, "hero_damage": 10,
          "damage_taken": 11, "tower_damage": 12, "time_dead": 13.5,
          "stun_duration": 14.25, "gold_spent_wards": 1, "gold_spent_smoke": 2,
          "gold_spent_dust": 3, "camps_stacked": 4, "creeps_stacked": 5,
          "rune_pickups": 6, "first_blood": true
        }
      ]
    }
    """
    result = parse_result(raw)
    assert result.match_id == 123
    assert result.duration_sec == 300
    assert result.radiant_win is True
    assert len(result.players) == 1
    p = result.players[0]
    assert p.hero == "Slark"
    assert p.team == "radiant"
    assert p.kills == 2
    assert p.stun_duration == pytest.approx(14.25)
    assert p.first_blood is True


def test_parse_result_match_id_omitted() -> None:
    raw = '{"duration_sec": 100, "radiant_win": false, "players": []}'
    result = parse_result(raw)
    assert result.match_id == 0
    assert result.radiant_win is False
    assert result.players == []


def test_parse_result_bad_json() -> None:
    with pytest.raises(MantaParseError):
        parse_result("not json {{{")


def test_parse_result_no_players() -> None:
    with pytest.raises(MantaParseError):
        parse_result('{"match_id": 1, "duration_sec": 10, "radiant_win": true}')


def test_parse_result_bytes() -> None:
    raw = b'{"duration_sec": 1, "radiant_win": true, "players": []}'
    assert parse_result(raw).duration_sec == 1
