from __future__ import annotations

from PyQt6.QtCore import Qt, pyqtSignal
from PyQt6.QtWidgets import (
    QComboBox,
    QDoubleSpinBox,
    QFormLayout,
    QFrame,
    QHBoxLayout,
    QLabel,
    QPushButton,
    QScrollArea,
    QToolButton,
    QVBoxLayout,
    QWidget,
)

from ..formula import (
    DEFAULT_LINEAR_WEIGHTS,
    Preset,
    PresetStore,
    linear_to_expression,
    split_expression,
)
from ..mvp import Weights

_GROUPS: tuple[tuple[str, tuple[tuple[str, str], ...]], ...] = (
    (
        "Файт",
        (
            ("kills", "Kills"),
            ("deaths", "Смерти (штраф)"),
            ("deaths_base", "База смерти"),
            ("assists", "Assists"),
            ("first_blood", "First Blood"),
            ("healing", "Healing"),
            ("hero_damage", "Hero Damage"),
            ("damage_taken", "Damage Taken"),
        ),
    ),
    (
        "Фарм",
        (
            ("last_hits", "Last Hits"),
            ("gpm", "GPM"),
            ("xpm", "XPM"),
        ),
    ),
    (
        "Утилити",
        (
            ("stun", "Stun Duration"),
            ("tower_damage", "Tower Damage"),
            ("camps", "Camps Stacked"),
            ("runes", "Rune Pickups"),
            ("gold_spent_wards", "Wards Gold"),
            ("gold_spent_smoke", "Smoke Gold"),
            ("gold_spent_dust", "Dust Gold"),
            ("buffs_duration", "Buffs Duration"),
            ("save", "Save Uptime"),
            ("purge", "Purge Uptime"),
            ("shield_uptime", "Shield Uptime"),
        ),
    ),
)

_FORMULA_ORDER = (
    ("kills", "Kills"),
    ("deaths", "Deaths"),
    ("deaths_base", "База"),
    ("assists", "Assists"),
    ("last_hits", "LH"),
    ("gpm", "GPM"),
    ("xpm", "XPM"),
    ("stun", "Stun"),
    ("healing", "Heal"),
    ("tower_damage", "TDmg"),
    ("camps", "Camps"),
    ("runes", "Runes"),
    ("first_blood", "FB"),
    ("hero_damage", "HeroDmg"),
    ("damage_taken", "DmgTaken"),
    ("gold_spent_wards", "WardsG"),
    ("gold_spent_smoke", "SmokeG"),
    ("gold_spent_dust", "DustG"),
    ("buffs_duration", "BuffsDur"),
    ("save", "SaveUp"),
    ("purge", "PurgeUp"),
    ("shield_uptime", "ShieldUp"),
)

_WEIGHT_KEYS = {key for group, fields in _GROUPS for key, _ in fields}


class WeightsPanel(QWidget):
    changed = pyqtSignal()
    edit_requested = pyqtSignal()

    def __init__(self, store: PresetStore, parent=None):
        super().__init__(parent)
        self._store = store
        self._spinboxes: dict[str, QDoubleSpinBox] = {}
        self._updating = False

        outer = QVBoxLayout(self)
        outer.setContentsMargins(0, 0, 0, 0)
        outer.setSpacing(0)

        preset_row = QHBoxLayout()
        preset_row.setContentsMargins(0, 0, 0, 0)
        preset_row.setSpacing(6)
        preset_caption = QLabel("Формула:")
        preset_caption.setObjectName("weightSectionTitle")
        self._preset_combo = QComboBox()
        self._preset_combo.currentIndexChanged.connect(self._on_preset_changed)
        preset_row.addWidget(preset_caption)
        preset_row.addWidget(self._preset_combo, 1)
        outer.addLayout(preset_row)

        self._toggle = QToolButton()
        self._toggle.setText("Коэффициенты формулы")
        self._toggle.setCheckable(True)
        self._toggle.setChecked(True)
        self._toggle.setArrowType(Qt.ArrowType.DownArrow)
        self._toggle.setToolButtonStyle(Qt.ToolButtonStyle.ToolButtonTextBesideIcon)
        self._toggle.setStyleSheet(
            "QToolButton { border: none; background: transparent; color: #e6edf3;"
            " font-weight: 700; font-size: 13px; padding: 4px 2px; text-align: left; }"
            "QToolButton:hover { color: #f5c542; }"
        )
        self._toggle.toggled.connect(self._on_toggle)
        outer.addWidget(self._toggle)

        self._body = QWidget()
        body = QVBoxLayout(self._body)
        body.setContentsMargins(0, 6, 0, 0)
        body.setSpacing(8)

        self._formula = QLabel("")
        self._formula.setObjectName("formulaText")
        self._formula.setWordWrap(True)
        body.addWidget(self._formula)

        scroll = QScrollArea()
        scroll.setWidgetResizable(True)
        scroll.setFrameShape(QScrollArea.Shape.NoFrame)
        content = QWidget()
        sections = QVBoxLayout(content)
        sections.setContentsMargins(0, 0, 0, 0)
        sections.setSpacing(8)

        for group_title, fields in _GROUPS:
            section = QFrame()
            section.setObjectName("weightSection")
            section_layout = QVBoxLayout(section)
            section_layout.setContentsMargins(12, 10, 12, 10)
            section_layout.setSpacing(6)
            title = QLabel(group_title)
            title.setObjectName("weightSectionTitle")
            section_layout.addWidget(title)
            form = QFormLayout()
            form.setHorizontalSpacing(10)
            form.setVerticalSpacing(6)
            for key, label in fields:
                spin = QDoubleSpinBox()
                spin.setRange(-100.0, 100.0)
                spin.setDecimals(4)
                spin.setSingleStep(0.001)
                spin.setFixedWidth(96)
                spin.valueChanged.connect(self._on_value_changed)
                form.addRow(f"{label} ×", spin)
                self._spinboxes[key] = spin
            section_layout.addLayout(form)
            sections.addWidget(section)

        scroll.setWidget(content)
        body.addWidget(scroll, 1)

        self._reset_btn = QPushButton("Сбросить к значениям по умолчанию")
        self._reset_btn.setObjectName("ghost")
        self._reset_btn.clicked.connect(self.reset)
        body.addWidget(self._reset_btn)

        hint = QLabel("Изменения применяются сразу к уже загруженному реплею.")
        hint.setObjectName("weightHint")
        hint.setWordWrap(True)
        body.addWidget(hint)

        outer.addWidget(self._body, 1)

        self._expr_widget = QWidget()
        expr_layout = QVBoxLayout(self._expr_widget)
        expr_layout.setContentsMargins(0, 6, 0, 0)
        expr_layout.setSpacing(8)
        expr_caption = QLabel("Текстовая формула")
        expr_caption.setObjectName("weightSectionTitle")
        expr_layout.addWidget(expr_caption)
        self._expr_text = QLabel("")
        self._expr_text.setObjectName("formulaText")
        self._expr_text.setWordWrap(True)
        expr_layout.addWidget(self._expr_text)
        self._expr_btn = QPushButton("Открыть редактор…")
        self._expr_btn.clicked.connect(self.edit_requested)
        expr_layout.addWidget(self._expr_btn)
        outer.addWidget(self._expr_widget)

        self.refresh_presets()

    def refresh_presets(self) -> None:
        self._updating = True
        try:
            self._preset_combo.clear()
            for preset in self._store.presets:
                self._preset_combo.addItem(preset.name, preset.id)
            index = self._preset_combo.findData(self._store.active_id)
            self._preset_combo.setCurrentIndex(index if index >= 0 else 0)
        finally:
            self._updating = False
        self._apply_preset()

    def _on_preset_changed(self, _index: int) -> None:
        if self._updating:
            return
        preset_id = self._preset_combo.currentData()
        if preset_id and self._store.set_active(preset_id):
            self._store.save()
        self._apply_preset()
        self.changed.emit()

    def _apply_preset(self) -> None:
        preset = self._store.active()
        if preset.kind == "expression":
            weights, tail = split_expression(preset.expression)
            if not weights:
                self._toggle.setChecked(False)
                self._toggle.setEnabled(False)
                self._body.setVisible(False)
                self._expr_widget.setVisible(True)
                self._expr_text.setText(preset.expression or "—")
                self._formula.setText("")
                return
            self._toggle.setEnabled(True)
            self._toggle.setChecked(True)
            self._body.setVisible(True)
            self._expr_widget.setVisible(True)
            self._expr_text.setText(preset.expression)
            self._set_spinboxes(weights)
            self._update_formula(self._weights_from_mapping(weights), tail)
            return
        self._toggle.setEnabled(True)
        self._toggle.setChecked(True)
        self._expr_widget.setVisible(False)
        self._body.setVisible(True)
        self._set_spinboxes(preset.weights)
        self._update_formula(self._weights_from_preset(preset))

    def _set_spinboxes(self, weights: dict[str, float]) -> None:
        self._updating = True
        try:
            for key, spin in self._spinboxes.items():
                spin.setValue(weights.get(key, 0.0))
        finally:
            self._updating = False

    @staticmethod
    def _weights_from_preset(preset: Preset) -> Weights:
        return WeightsPanel._weights_from_mapping(preset.weights)

    @staticmethod
    def _weights_from_mapping(mapping: dict) -> Weights:
        return Weights(
            **{key: float(mapping.get(key, 0.0)) for key in _WEIGHT_KEYS}
        )

    def _on_toggle(self, checked: bool) -> None:
        self._toggle.setArrowType(Qt.ArrowType.DownArrow if checked else Qt.ArrowType.RightArrow)
        preset = self._store.active()
        if preset.kind == "expression":
            weights, _ = split_expression(preset.expression)
            if not weights:
                self._body.setVisible(False)
                return
        self._body.setVisible(checked)

    def _on_value_changed(self, _value: float) -> None:
        if self._updating:
            return
        preset = self._store.active()
        if preset.kind == "linear":
            preset.weights.update({key: spin.value() for key, spin in self._spinboxes.items()})
            self._store.save()
        elif preset.kind == "expression":
            weights, tail = split_expression(preset.expression)
            if not weights:
                self.changed.emit()
                return
            weights.update({key: spin.value() for key, spin in self._spinboxes.items()})
            preset.expression = linear_to_expression(weights)
            if tail:
                preset.expression += " + " + tail
            self._store.save()
            self._expr_text.setText(preset.expression)
        weights = (
            preset.weights
            if preset.kind == "linear"
            else split_expression(preset.expression)[0]
        )
        tail = "" if preset.kind == "linear" else split_expression(preset.expression)[1]
        self._update_formula(self._weights_from_mapping(weights), tail)
        self.changed.emit()

    def _update_formula(self, weights: Weights, tail: str = "") -> None:
        parts = []
        for key, label in _FORMULA_ORDER:
            value = getattr(weights, key)
            if value == 0.0:
                continue
            text = f"{value:g}·{label}"
            if value > 0 and key not in ("deaths", "deaths_base"):
                text = "+" + text
            parts.append(text)
        text = "Счёт = " + " ".join(parts)
        if tail:
            text += "  +  " + tail
        self._formula.setText(text)

    def reset(self) -> None:
        preset = self._store.active()
        if preset.kind == "expression":
            _, tail = split_expression(preset.expression)
            preset.expression = linear_to_expression(DEFAULT_LINEAR_WEIGHTS)
            if tail:
                preset.expression += " + " + tail
        else:
            preset.weights.update(DEFAULT_LINEAR_WEIGHTS)
        self._store.save()
        self._apply_preset()
        self.changed.emit()

    def preset(self) -> Preset:
        return self._store.active()
