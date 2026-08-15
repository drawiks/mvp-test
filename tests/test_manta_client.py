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


def test_parse_result_aggregates_buffs() -> None:
    raw = """
    {
      "match_id": 1,
      "duration_sec": 60,
      "radiant_win": true,
      "players": [
        {
          "steam_id": 1, "player_id": 0, "hero_id": 1, "hero": "H",
          "team": "radiant", "name": "n", "kills": 1, "deaths": 2,
          "assists": 3, "level": 4, "last_hits": 5, "networth": 6,
          "gpm": 7, "xpm": 8, "healing": 0.0, "hero_damage": 0,
          "damage_taken": 0, "tower_damage": 0, "time_dead": 0.0,
          "stun_duration": 0.0, "gold_spent_wards": 0, "gold_spent_smoke": 0,
          "gold_spent_dust": 0, "camps_stacked": 0, "creeps_stacked": 0,
          "rune_pickups": 0, "first_blood": false,
          "buffs_duration": 120.5,
          "buff_sources": [
            {"inflictor": "a", "category": "save", "value": 30.0, "count": 1},
            {"inflictor": "b", "category": "purge", "value": 10.5, "count": 1},
            {"inflictor": "c", "category": "shield", "value": 5.0, "count": 1},
            {"inflictor": "d", "category": "heal", "value": 100.0, "count": 2},
            {"inflictor": "e", "category": "save", "value": 1.5, "count": 1}
          ],
          "stun_sources": [{"inflictor": "x", "value": 3.0, "count": 1}]
        }
      ]
    }
    """
    result = parse_result(raw)
    p = result.players[0]
    assert p.buffs_duration == pytest.approx(120.5)
    assert p.save == pytest.approx(31.5)
    assert p.purge == pytest.approx(10.5)
    assert p.shield_uptime == pytest.approx(5.0)


def test_parse_result_without_buffs() -> None:
    raw = """
    {
      "duration_sec": 1,
      "radiant_win": true,
      "players": [
        {"steam_id": 1, "player_id": 0, "hero_id": 1, "hero": "H",
         "team": "radiant", "name": "n", "kills": 0, "deaths": 0,
         "assists": 0, "level": 1, "last_hits": 0, "networth": 0,
         "gpm": 0, "xpm": 0, "healing": 0.0, "hero_damage": 0,
         "damage_taken": 0, "tower_damage": 0, "time_dead": 0.0,
         "stun_duration": 0.0, "gold_spent_wards": 0, "gold_spent_smoke": 0,
         "gold_spent_dust": 0, "camps_stacked": 0, "creeps_stacked": 0,
         "rune_pickups": 0, "first_blood": false}
      ]
    }
    """
    p = parse_result(raw).players[0]
    assert p.save == 0.0
    assert p.purge == 0.0
    assert p.shield_uptime == 0.0
    assert p.buffs_duration == 0.0
