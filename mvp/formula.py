from __future__ import annotations

import ast
import json
import logging
import operator
import uuid
from dataclasses import dataclass, field
from pathlib import Path

_log = logging.getLogger(__name__)

STATS: tuple[tuple[str, str], ...] = (
    ("kills", "Kills"),
    ("deaths", "Deaths"),
    ("assists", "Assists"),
    ("last_hits", "Last Hits"),
    ("gpm", "GPM"),
    ("xpm", "XPM"),
    ("stun_duration", "Stun"),
    ("healing", "Healing"),
    ("tower_damage", "Tower Dmg"),
    ("camps_stacked", "Camps Stacked"),
    ("rune_pickups", "Rune Pickups"),
    ("first_blood", "First Blood"),
)

STAT_TOKENS: set[str] = {token for token, _ in STATS}
STAT_LABEL_BY_TOKEN: dict[str, str] = dict(STATS)

EXAMPLES: tuple[tuple[str, str], ...] = (
    (
        "KDA ratio",
        "(kills * 3 + assists * 1.5) / max(deaths, 1)",
    ),
    (
        "Support focus",
        "(kills * 2 + assists * 2.5) / max(deaths, 1)"
        " + healing * 0.005 + camps_stacked * 0.5 + stun_duration * 0.05",
    ),
    (
        "Farm focus",
        "last_hits * 0.003 + gpm * 0.002 + xpm * 0.002 + tower_damage * 0.001",
    ),
    (
        "Carry",
        "kills * 0.3 + (3 - deaths * 0.3) + assists * 0.15"
        " + last_hits * 0.003 + gpm * 0.002 + xpm * 0.002",
    ),
)

DEFAULT_LINEAR_WEIGHTS: dict[str, float] = {
    "kills": 0.3,
    "deaths": 0.3,
    "deaths_base": 3.0,
    "assists": 0.15,
    "last_hits": 0.003,
    "gpm": 0.002,
    "xpm": 0.002,
    "stun": 0.05,
    "healing": 0.004,
    "tower_damage": 0.001,
    "camps": 0.5,
    "runes": 0.2,
    "first_blood": 1.0,
}


class FormulaError(ValueError):
    pass


@dataclass
class Preset:
    id: str
    name: str
    kind: str = "linear"
    weights: dict[str, float] = field(default_factory=dict)
    expression: str = ""

    def to_dict(self) -> dict:
        return {
            "id": self.id,
            "name": self.name,
            "kind": self.kind,
            "weights": dict(self.weights),
            "expression": self.expression,
        }

    @classmethod
    def from_dict(cls, data: dict) -> "Preset":
        weights = {}
        for key, value in (data.get("weights") or {}).items():
            try:
                weights[str(key)] = float(value)
            except (TypeError, ValueError):
                continue
        return cls(
            id=str(data.get("id") or uuid.uuid4().hex),
            name=str(data.get("name") or "Без названия"),
            kind="expression" if data.get("kind") == "expression" else "linear",
            weights=weights,
            expression=str(data.get("expression") or ""),
        )


def standard_preset() -> Preset:
    return Preset(
        id="standard",
        name="Стандартная",
        kind="linear",
        weights=dict(DEFAULT_LINEAR_WEIGHTS),
    )


_EXPR_TERMS: tuple[tuple[str, str], ...] = (
    ("kills", "kills"),
    ("deaths", "deaths"),
    ("assists", "assists"),
    ("last_hits", "last_hits"),
    ("gpm", "gpm"),
    ("xpm", "xpm"),
    ("stun_duration", "stun"),
    ("healing", "healing"),
    ("tower_damage", "tower_damage"),
    ("camps_stacked", "camps"),
    ("rune_pickups", "runes"),
    ("first_blood", "first_blood"),
)

_WEIGHT_BY_TOKEN: dict[str, str] = {token: key for token, key in _EXPR_TERMS}


def _const_value(node: ast.AST) -> float | None:
    if isinstance(node, ast.Constant) and isinstance(node.value, (int, float)):
        return float(node.value)
    if isinstance(node, ast.UnaryOp) and type(node.op) in _UNAOPS:
        val = _const_value(node.operand)
        if val is not None:
            return _UNAOPS[type(node.op)](val)
    return None


def linear_to_expression(weights: dict[str, float]) -> str:
    """Конвертирует линейные коэффициенты в эквивалентное выражение."""
    w = dict(weights)
    terms = []
    for token, key in _EXPR_TERMS:
        if key != "deaths" and w.get(key):
            terms.append(f"{token} * {w[key]:g}")
    if w.get("deaths") or w.get("deaths_base"):
        terms.append(f"({w.get('deaths_base', 0.0):g} - deaths * {w.get('deaths', 0.0):g})")
    return " + ".join(terms)


def _flatten_add(node) -> list[ast.AST]:
    if isinstance(node, ast.BinOp) and type(node.op) is ast.Add:
        return _flatten_add(node.left) + _flatten_add(node.right)
    return [node]


def expression_to_weights(expression: str) -> dict[str, float] | None:
    """Извлекает коэффициенты из выражения вида 'kills * 0.3 + (3 - deaths * 0.3)'.

    Возвращает None, если выражение не сводится к взвешенной сумме переменных.
    """
    try:
        body = ast.parse(expression, mode="eval").body
    except SyntaxError:
        return None
    weights: dict[str, float] = {}
    for term in _flatten_add(body):
        value = _const_value(term)
        if value is not None:
            if "deaths_base" in weights:
                return None
            weights["deaths_base"] = value
            continue
        if isinstance(term, ast.BinOp) and type(term.op) is ast.Mult:
            left, right = term.left, term.right
            left_val = _const_value(left)
            right_val = _const_value(right)
            if isinstance(left, ast.Name) and right_val is not None:
                name, value = left.id, right_val
            elif isinstance(right, ast.Name) and left_val is not None:
                name, value = right.id, left_val
            else:
                return None
            if name not in _WEIGHT_BY_TOKEN:
                return None
            key = _WEIGHT_BY_TOKEN[name]
            if key == "deaths":
                return None
            if key in weights:
                return None
            weights[key] = value
            continue
        if (
            isinstance(term, ast.BinOp)
            and type(term.op) is ast.Sub
            and _const_value(term.left) is not None
            and isinstance(term.right, ast.BinOp)
            and type(term.right.op) is ast.Mult
        ):
            sub_left, sub_right = term.right.left, term.right.right
            if isinstance(sub_left, ast.Name) and _const_value(sub_right) is not None:
                if sub_left.id != "deaths":
                    return None
                if "deaths_base" in weights or "deaths" in weights:
                    return None
                weights["deaths_base"] = _const_value(term.left)
                weights["deaths"] = _const_value(sub_right)
                continue
            if isinstance(sub_right, ast.Name) and _const_value(sub_left) is not None:
                if sub_right.id != "deaths":
                    return None
                if "deaths_base" in weights or "deaths" in weights:
                    return None
                weights["deaths_base"] = _const_value(term.left)
                weights["deaths"] = _const_value(sub_left)
                continue
            return None
        return None
    return weights or None


_FUNCS = {
    "max": max,
    "min": min,
    "abs": abs,
    "round": round,
}
_BINOPS = {
    ast.Add: operator.add,
    ast.Sub: operator.sub,
    ast.Mult: operator.mul,
    ast.Div: operator.truediv,
    ast.Pow: operator.pow,
}
_UNAOPS = {
    ast.UAdd: lambda x: x,
    ast.USub: lambda x: -x,
}


def _check_node(node, allowed: set[str]) -> str | None:
    if isinstance(node, ast.Constant) and isinstance(node.value, (int, float)):
        return None
    if isinstance(node, ast.Name):
        if node.id in allowed:
            return None
        if node.id in _FUNCS:
            return f"'{node.id}' используется неверно — нужно {node.id}(число)"
        return f"Неизвестная переменная: {node.id}. Доступно: {', '.join(sorted(allowed))}"
    if isinstance(node, ast.BinOp) and type(node.op) in _BINOPS:
        return _check_node(node.left, allowed) or _check_node(node.right, allowed)
    if isinstance(node, ast.UnaryOp) and type(node.op) in _UNAOPS:
        return _check_node(node.operand, allowed)
    if isinstance(node, ast.Call) and isinstance(node.func, ast.Name) and node.func.id in _FUNCS:
        if node.keywords:
            return "Ключевые аргументы не поддерживаются"
        for arg in node.args:
            err = _check_node(arg, allowed)
            if err:
                return err
        return None
    return f"Недопустимая часть формулы: {type(node).__name__}"


def validate_expression(expression: str, allowed: set[str] | None = None) -> str | None:
    allowed = STAT_TOKENS if allowed is None else allowed
    if not expression.strip():
        return "Формула пустая"
    try:
        tree = ast.parse(expression, mode="eval")
    except SyntaxError as exc:
        return f"Ошибка в формуле: {exc.msg}"
    return _check_node(tree.body, allowed)


def safe_eval(expression: str, variables: dict[str, float]) -> float:
    err = validate_expression(expression, allowed=set(variables) | set(_FUNCS))
    if err:
        raise FormulaError(err)
    code = compile(ast.parse(expression, mode="eval"), "<formula>", "eval")
    env = dict(_FUNCS)
    env.update(variables)
    try:
        return float(eval(code, {"__builtins__": {}}, env))
    except ZeroDivisionError:
        raise FormulaError("Деление на ноль") from None
    except FormulaError:
        raise
    except Exception as exc:  # noqa: BLE001
        raise FormulaError(f"Ошибка вычисления: {exc}") from None


class PresetStore:
    def __init__(self, path: Path):
        self._path = Path(path)
        self._presets: dict[str, Preset] = {"standard": standard_preset()}
        self.active_id = "standard"
        self._load()

    def _load(self) -> None:
        if not self._path.is_file():
            return
        try:
            data = json.loads(self._path.read_text("utf-8"))
        except (OSError, json.JSONDecodeError):
            return
        for item in data.get("presets", []):
            preset = Preset.from_dict(item)
            self._presets[preset.id] = preset
        if data.get("active") in self._presets:
            self.active_id = data["active"]

    def save(self) -> bool:
        try:
            self._path.parent.mkdir(parents=True, exist_ok=True)
            payload = {
                "version": 1,
                "active": self.active_id,
                "presets": [preset.to_dict() for preset in self._presets.values()],
            }
            tmp = self._path.with_name(self._path.name + ".tmp")
            tmp.write_text(json.dumps(payload, ensure_ascii=False, indent=2), "utf-8")
            tmp.replace(self._path)
        except OSError as exc:
            _log.warning("Не удалось сохранить формулы: %s", exc)
            return False
        return True

    @property
    def presets(self) -> list[Preset]:
        return list(self._presets.values())

    def get(self, preset_id: str) -> Preset | None:
        return self._presets.get(preset_id)

    def active(self) -> Preset:
        return self._presets.get(self.active_id, standard_preset())

    def add(self, preset: Preset) -> None:
        self._presets[preset.id] = preset

    def remove(self, preset_id: str) -> bool:
        if preset_id == "standard" or preset_id not in self._presets:
            return False
        del self._presets[preset_id]
        if self.active_id == preset_id:
            self.active_id = "standard"
        return True

    def set_active(self, preset_id: str) -> bool:
        if preset_id not in self._presets:
            return False
        self.active_id = preset_id
        return True

    def import_file(self, path: Path) -> Preset:
        try:
            data = json.loads(Path(path).read_text("utf-8"))
        except (OSError, json.JSONDecodeError) as exc:
            raise FormulaError(f"Не удалось прочитать файл: {exc}") from exc
        preset = Preset.from_dict(data)
        if not preset.expression and preset.kind == "expression":
            raise FormulaError("Файл не содержит выражения формулы")
        if preset.id == "standard" or preset.id in self._presets:
            preset.id = uuid.uuid4().hex
        self._presets[preset.id] = preset
        return preset

    def export_file(self, preset: Preset, path: Path) -> None:
        Path(path).write_text(
            json.dumps(preset.to_dict(), ensure_ascii=False, indent=2),
            "utf-8",
        )
