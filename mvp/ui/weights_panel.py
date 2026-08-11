from __future__ import annotations

from PyQt6.QtCore import Qt, pyqtSignal
from PyQt6.QtWidgets import (
    QDoubleSpinBox,
    QFormLayout,
    QGroupBox,
    QLabel,
    QPushButton,
    QScrollArea,
    QVBoxLayout,
    QWidget,
)

from ..mvp import DEFAULT_WEIGHTS, Weights

_FIELDS: tuple[tuple[str, str], ...] = (
    ("kills", "Kills"),
    ("deaths", "Смерти (штраф)"),
    ("deaths_base", "База смерти"),
    ("assists", "Assists"),
    ("last_hits", "LastHits"),
    ("gpm", "GPM"),
    ("xpm", "XPM"),
    ("stun", "StunDuration"),
    ("healing", "Healing"),
    ("tower_damage", "TowerDamage"),
    ("camps", "CampsStacked"),
    ("runes", "RunePickups"),
    ("first_blood", "FirstBlood"),
)


class WeightsPanel(QWidget):
    changed = pyqtSignal()

    def __init__(self, parent=None):
        super().__init__(parent)
        self._spinboxes: dict[str, QDoubleSpinBox] = {}
        self._updating = False

        layout = QVBoxLayout(self)
        layout.setContentsMargins(0, 0, 0, 0)
        layout.setSpacing(8)

        group = QGroupBox("Коэффициенты формулы")
        group_layout = QVBoxLayout(group)
        group_layout.setContentsMargins(10, 10, 10, 10)

        scroll = QScrollArea()
        scroll.setWidgetResizable(True)
        scroll.setFrameShape(QScrollArea.Shape.NoFrame)
        form_widget = QWidget()
        form = QFormLayout(form_widget)
        form.setHorizontalSpacing(12)
        form.setVerticalSpacing(8)
        form.setLabelAlignment(Qt.AlignmentFlag.AlignRight)

        for key, label in _FIELDS:
            spin = QDoubleSpinBox()
            spin.setRange(-100.0, 100.0)
            spin.setDecimals(4)
            spin.setSingleStep(0.001)
            spin.setValue(getattr(DEFAULT_WEIGHTS, key))
            spin.valueChanged.connect(self._on_value_changed)
            form.addRow(f"{label} ×", spin)
            self._spinboxes[key] = spin

        scroll.setWidget(form_widget)
        group_layout.addWidget(scroll)

        reset_btn = QPushButton("Сбросить к значениям по умолчанию")
        reset_btn.clicked.connect(self.reset)
        group_layout.addWidget(reset_btn)

        hint = QLabel("Изменения применяются сразу к уже загруженному реплею.")
        hint.setWordWrap(True)
        hint.setStyleSheet("color: #7d8590; font-size: 11px;")
        group_layout.addWidget(hint)

        layout.addWidget(group)

    def _on_value_changed(self, _value: float) -> None:
        if not self._updating:
            self.changed.emit()

    def reset(self) -> None:
        self.set_weights(DEFAULT_WEIGHTS)
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
