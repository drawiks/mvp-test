from __future__ import annotations

from PyQt6.QtCore import Qt, pyqtSignal
from PyQt6.QtWidgets import (
    QDoubleSpinBox,
    QFormLayout,
    QFrame,
    QLabel,
    QPushButton,
    QScrollArea,
    QToolButton,
    QVBoxLayout,
    QWidget,
)

from ..mvp import DEFAULT_WEIGHTS, Weights

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
)


class WeightsPanel(QWidget):
    changed = pyqtSignal()

    def __init__(self, parent=None):
        super().__init__(parent)
        self._spinboxes: dict[str, QDoubleSpinBox] = {}
        self._updating = False

        outer = QVBoxLayout(self)
        outer.setContentsMargins(0, 0, 0, 0)
        outer.setSpacing(0)

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
                spin.setValue(getattr(DEFAULT_WEIGHTS, key))
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
        self._update_formula(self.weights())

    def _on_toggle(self, checked: bool) -> None:
        self._toggle.setArrowType(Qt.ArrowType.DownArrow if checked else Qt.ArrowType.RightArrow)
        self._body.setVisible(checked)

    def _on_value_changed(self, _value: float) -> None:
        if self._updating:
            return
        self._update_formula(self.weights())
        self.changed.emit()

    def _update_formula(self, weights: Weights) -> None:
        parts = []
        for key, label in _FORMULA_ORDER:
            value = getattr(weights, key)
            text = f"{value:g}·{label}"
            if value > 0 and key not in ("deaths",):
                text = "+" + text
            parts.append(text)
        self._formula.setText("Счёт = " + " ".join(parts))

    def reset(self) -> None:
        self.set_weights(DEFAULT_WEIGHTS)
        self._update_formula(self.weights())
        self.changed.emit()

    def set_weights(self, weights: Weights) -> None:
        self._updating = True
        try:
            for key, spin in self._spinboxes.items():
                spin.setValue(getattr(weights, key))
        finally:
            self._updating = False

    def weights(self) -> Weights:
        return Weights(**{key: spin.value() for key, spin in self._spinboxes.items()})
