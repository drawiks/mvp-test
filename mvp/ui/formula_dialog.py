from __future__ import annotations

from PyQt6.QtWidgets import (
    QComboBox,
    QDialog,
    QDialogButtonBox,
    QFormLayout,
    QHBoxLayout,
    QLabel,
    QLineEdit,
    QPlainTextEdit,
    QPushButton,
    QScrollArea,
    QVBoxLayout,
    QWidget,
)

from ..formula import EXAMPLES, STATS, FormulaError, Preset, validate_expression
from ..model import Player
from ..mvp import player_vars, safe_eval


def _demo_player() -> Player:
    return Player(
        name="Демо-игрок",
        kills=12,
        deaths=4,
        assists=16,
        last_hits=300,
        gpm=620,
        xpm=680,
        stun_duration=60.0,
        healing=12000.0,
        tower_damage=4500.0,
        camps_stacked=12,
        rune_pickups=8,
        first_blood=True,
    )


class FormulaEditorDialog(QDialog):
    def __init__(
        self,
        preset: Preset | None = None,
        players: list[Player] | None = None,
        parent=None,
    ):
        super().__init__(parent)
        self.setWindowTitle("Редактор формулы")
        self.setMinimumSize(560, 560)
        self._players = players or []
        self._preset = preset or Preset(id="", name="Новая формула", kind="expression")

        layout = QVBoxLayout(self)
        layout.setSpacing(10)

        form = QFormLayout()
        self._name = QLineEdit(self._preset.name)
        form.addRow("Название:", self._name)
        layout.addLayout(form)

        layout.addWidget(QLabel("Статистики — клик вставляет имя в формулу:"))
        chips = QWidget()
        chips_layout = QVBoxLayout(chips)
        chips_layout.setContentsMargins(0, 0, 0, 0)
        chips_layout.setSpacing(6)
        row = QHBoxLayout()
        row.setSpacing(4)
        for index, (token, label) in enumerate(STATS):
            button = QPushButton(label)
            button.setObjectName("chip")
            button.clicked.connect(lambda _=False, tok=token: self._insert(tok))
            row.addWidget(button)
            if index % 6 == 5 and index != len(STATS) - 1:
                chips_layout.addLayout(row)
                row = QHBoxLayout()
                row.setSpacing(4)
        chips_layout.addLayout(row)
        chips_layout.addStretch(1)
        chips_scroll = QScrollArea()
        chips_scroll.setWidgetResizable(True)
        chips_scroll.setFrameShape(QScrollArea.Shape.NoFrame)
        chips_scroll.setFixedHeight(120)
        chips_scroll.setWidget(chips)
        layout.addWidget(chips_scroll)

        ops_row = QHBoxLayout()
        ops_row.setSpacing(4)
        for token in ("+", "-", "*", "/", "(", ")", "max(", "min(", "abs(", "round("):
            button = QPushButton(token)
            button.setObjectName("chip")
            button.clicked.connect(lambda _=False, t=token: self._insert(t))
            ops_row.addWidget(button)
        ops_row.addStretch(1)
        layout.addLayout(ops_row)

        self._expr = QPlainTextEdit()
        self._expr.setPlaceholderText(
            "Например: (kills * 3 + assists * 1.5) / max(deaths, 1)"
        )
        self._expr.setPlainText(self._preset.expression)
        self._expr.setMinimumHeight(120)
        self._expr.textChanged.connect(self._update_preview)
        layout.addWidget(self._expr)

        self._validation = QLabel("")
        self._validation.setWordWrap(True)
        layout.addWidget(self._validation)

        preview_row = QHBoxLayout()
        preview_row.addWidget(QLabel("Предпросмотр:"))
        self._players_combo = QComboBox()
        self._players_combo.addItem("Демо-игрок")
        for player in self._players:
            label = f"{player.name} ({player.hero})" if player.hero else player.name
            self._players_combo.addItem(label, player)
        self._players_combo.currentIndexChanged.connect(self._update_preview)
        preview_row.addWidget(self._players_combo, 1)
        self._score = QLabel("—")
        self._score.setObjectName("previewScore")
        preview_row.addWidget(self._score)
        layout.addLayout(preview_row)

        example_row = QHBoxLayout()
        example_row.addWidget(QLabel("Примеры:"))
        self._examples = QComboBox()
        for name, _ in EXAMPLES:
            self._examples.addItem(name)
        example_row.addWidget(self._examples, 1)
        use_button = QPushButton("Вставить")
        use_button.clicked.connect(self._insert_example)
        example_row.addWidget(use_button)
        layout.addLayout(example_row)

        buttons = QDialogButtonBox(
            QDialogButtonBox.StandardButton.Save | QDialogButtonBox.StandardButton.Cancel
        )
        buttons.button(QDialogButtonBox.StandardButton.Save).setText("Сохранить")
        buttons.button(QDialogButtonBox.StandardButton.Cancel).setText("Отмена")
        buttons.accepted.connect(self._on_save)
        buttons.rejected.connect(self.reject)
        layout.addWidget(buttons)

        self._update_preview()

    def _insert(self, token: str) -> None:
        self._expr.textCursor().insertText(token)
        self._expr.setFocus()

    def _insert_example(self) -> None:
        index = self._examples.currentIndex()
        self._expr.setPlainText(EXAMPLES[index][1])
        self._update_preview()

    def _update_preview(self) -> None:
        text = self._expr.toPlainText()
        err = validate_expression(text)
        if err:
            self._validation.setText(f"⚠ {err}")
            self._validation.setStyleSheet("color: #f85149;")
            self._score.setText("—")
            return
        self._validation.setText("✓ Формула корректна")
        self._validation.setStyleSheet("color: #3fb950;")
        player = self._players_combo.currentData()
        if player is None:
            player = _demo_player()
        try:
            value = safe_eval(text, player_vars(player))
        except FormulaError:
            self._score.setText("—")
            return
        self._score.setText(f"Счёт: {value:.2f}")

    def _on_save(self) -> None:
        err = validate_expression(self._expr.toPlainText())
        if err:
            self._validation.setText(f"⚠ {err}")
            self._validation.setStyleSheet("color: #f85149;")
            return
        self._preset.name = self._name.text().strip() or "Без названия"
        self._preset.kind = "expression"
        self._preset.expression = self._expr.toPlainText().strip()
        self.accept()

    def preset(self) -> Preset:
        return self._preset
