from __future__ import annotations

import json

import pytest

from mvp.formula import (
    DEFAULT_LINEAR_WEIGHTS,
    EXAMPLES,
    FormulaError,
    Preset,
    PresetStore,
    expression_to_weights,
    linear_to_expression,
    safe_eval,
    standard_preset,
    validate_expression,
)
from mvp.model import Player


def test_validate_ok() -> None:
    assert validate_expression("(kills * 3 + assists * 1.5) / max(deaths, 1)") is None


def test_validate_unknown_variable() -> None:
    err = validate_expression("kills * kils")
    assert err is not None
    assert "Неизвестная переменная" in err


def test_validate_syntax_error() -> None:
    assert validate_expression("(kills + ") is not None


def test_validate_empty() -> None:
    assert validate_expression("   ") is not None


def test_all_examples_valid() -> None:
    for name, expression in EXAMPLES:
        assert validate_expression(expression) is None, name


def test_safe_eval_math() -> None:
    vars_ = {"kills": 10, "deaths": 4, "assists": 5}
    assert safe_eval("kills * 3 + assists * 1.5", vars_) == pytest.approx(37.5)
    assert safe_eval(
        "(kills * 3 + assists * 1.5) / max(deaths, 1)", vars_
    ) == pytest.approx(37.5 / 4)


def test_safe_eval_functions() -> None:
    assert safe_eval("min(5, 2) + abs(-3) + round(1.7)", {}) == pytest.approx(7.0)


def test_safe_eval_blocks_builtins() -> None:
    with pytest.raises(FormulaError):
        safe_eval("__import__('os').system('echo hi')", {"kills": 1})


def test_safe_eval_div_zero() -> None:
    with pytest.raises(FormulaError):
        safe_eval("1 / 0", {"kills": 1})


def test_store_roundtrip(tmp_path) -> None:
    store = PresetStore(tmp_path / "formulas.json")
    store.add(Preset(id="x", name="Моя", kind="expression", expression="kills*2"))
    store.set_active("x")
    store.save()

    loaded = PresetStore(tmp_path / "formulas.json")
    assert loaded.active().id == "x"
    assert loaded.get("x").expression == "kills*2"


def test_store_standard_is_builtin(tmp_path) -> None:
    store = PresetStore(tmp_path / "formulas.json")
    assert store.get("standard") is not None
    assert store.remove("standard") is False


def test_import_export(tmp_path) -> None:
    store = PresetStore(tmp_path / "formulas.json")
    preset = Preset(id="y", name="Y", kind="expression", expression="healing*0.005")
    out = tmp_path / "y.json"
    store.export_file(preset, out)

    data = json.loads(out.read_text("utf-8"))
    assert data["expression"] == "healing*0.005"

    other = PresetStore(tmp_path / "other.json")
    imported = other.import_file(out)
    assert imported.id == "y"
    assert other.get("y").name == "Y"


def test_standard_preset_defaults() -> None:
    assert standard_preset().weights["kills"] == 0.3
    assert standard_preset().kind == "linear"


def test_linear_to_expression_valid() -> None:
    expr = linear_to_expression(DEFAULT_LINEAR_WEIGHTS)
    assert validate_expression(expr) is None
    assert "kills * 0.3" in expr
    assert "(3 - deaths * 0.3)" in expr
    assert "stun_duration * 0.05" in expr
    assert "camps_stacked * 0.5" in expr


def test_linear_to_expression_equivalence() -> None:
    from mvp.mvp import compute_score, Weights

    player = Player(
        name="P",
        kills=10,
        deaths=4,
        assists=7,
        last_hits=250,
        gpm=580,
        xpm=640,
        stun_duration=45.0,
        healing=9000.0,
        tower_damage=3200.0,
        camps_stacked=9,
        rune_pickups=5,
        first_blood=True,
    )
    weights = {"kills": 0.4, "deaths_base": 2.0, "deaths": 0.5}
    linear = Weights(**{key: weights.get(key, 0.0) for key in DEFAULT_LINEAR_WEIGHTS})
    preset = Preset(id="x", name="X", kind="expression", expression=linear_to_expression(weights))
    assert compute_score(player, preset) == pytest.approx(
        compute_score(player, linear), abs=1e-6
    )


def test_expression_to_weights_roundtrip() -> None:
    expr = linear_to_expression(DEFAULT_LINEAR_WEIGHTS)
    assert expression_to_weights(expr) == DEFAULT_LINEAR_WEIGHTS


def test_expression_to_weights_sparse() -> None:
    assert expression_to_weights("kills * 3 + (2 - deaths * 0.5)") == {
        "kills": 3.0,
        "deaths_base": 2.0,
        "deaths": 0.5,
    }


def test_expression_to_weights_unsupported() -> None:
    assert expression_to_weights("(kills * 3 + assists * 1.5) / max(deaths, 1)") is None
    assert expression_to_weights("") is None
    assert expression_to_weights("kills * kills") is None
    assert expression_to_weights("kills ** 2") is None


def test_linear_to_expression_skips_absent() -> None:
    assert linear_to_expression({"kills": 3}) == "kills * 3"


def test_negative_coefficient_roundtrip() -> None:
    weights = {"kills": 0.3, "deaths": 0.3, "deaths_base": 3.0, "assists": -0.5}
    expr = linear_to_expression(weights)
    assert expression_to_weights(expr) == weights
    assert expression_to_weights("kills * -0.5") == {"kills": -0.5}


def test_expression_to_weights_deaths_ambiguity() -> None:
    assert expression_to_weights("deaths * 5") is None
    assert expression_to_weights("3 - deaths * 0.3") == {
        "deaths_base": 3.0,
        "deaths": 0.3,
    }
    assert expression_to_weights("(2 - deaths * 0.3) + deaths * 0.1") is None


def test_expression_to_weights_duplicate_constant() -> None:
    assert expression_to_weights("kills * 2 + 3 + 5") is None
    assert expression_to_weights("3 + kills * 2") == {"deaths_base": 3.0, "kills": 2.0}
    assert expression_to_weights("3 + (4 - deaths * 0.3)") is None
