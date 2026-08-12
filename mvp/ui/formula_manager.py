from __future__ import annotations

import uuid

from PyQt6.QtCore import pyqtSignal
from PyQt6.QtWidgets import (
    QDialog,
    QFileDialog,
    QHBoxLayout,
    QLabel,
    QListWidget,
    QListWidgetItem,
    QMessageBox,
    QPushButton,
    QVBoxLayout,
)

from ..formula import FormulaError, Preset, PresetStore, linear_to_expression
from ..model import Player
from .formula_dialog import FormulaEditorDialog


class FormulaManagerDialog(QDialog):
    presets_changed = pyqtSignal()

    def __init__(self, store: PresetStore, players: list[Player] | None = None, parent=None):
        super().__init__(parent)
        self.setWindowTitle("Формулы")
        self.setMinimumSize(480, 420)
        self._store = store
        self._players = players or []

        layout = QVBoxLayout(self)
        layout.setSpacing(8)

        hint = QLabel(
            "Выбор формулы применяется сразу. Клик по строке — активировать."
        )
        hint.setWordWrap(True)
        layout.addWidget(hint)

        self._list = QListWidget()
        self._list.itemSelectionChanged.connect(self._on_selection)
        layout.addWidget(self._list, 1)

        buttons = QHBoxLayout()
        buttons.setSpacing(6)
        create = QPushButton("Создать")
        edit = QPushButton("Редактировать")
        duplicate = QPushButton("Дублировать")
        delete = QPushButton("Удалить")
        buttons.addWidget(create)
        buttons.addWidget(edit)
        buttons.addWidget(duplicate)
        buttons.addWidget(delete)
        buttons.addStretch(1)
        layout.addLayout(buttons)

        file_buttons = QHBoxLayout()
        file_buttons.setSpacing(6)
        export = QPushButton("Экспорт…")
        import_btn = QPushButton("Импорт…")
        close = QPushButton("Закрыть")
        file_buttons.addWidget(export)
        file_buttons.addWidget(import_btn)
        file_buttons.addStretch(1)
        file_buttons.addWidget(close)
        layout.addLayout(file_buttons)

        create.clicked.connect(self._create)
        edit.clicked.connect(self._edit)
        duplicate.clicked.connect(self._duplicate)
        delete.clicked.connect(self._delete)
        export.clicked.connect(self._export)
        import_btn.clicked.connect(self._import)
        close.clicked.connect(self.accept)

        self._refresh()

    def _refresh(self) -> None:
        selected = self._selected()
        selected_id = selected.id if selected is not None else self._store.active_id

        self._list.blockSignals(True)
        try:
            self._list.clear()
            selected_item = None
            for preset in self._store.presets:
                kind = "выражение" if preset.kind == "expression" else "коэффициенты"
                marker = "✓ " if preset.id == self._store.active_id else ""
                item = QListWidgetItem(f"{marker}{preset.name}  ·  {kind}")
                item.setData(1, preset.id)
                self._list.addItem(item)
                if selected_id is not None and preset.id == selected_id:
                    selected_item = item
            if selected_item is not None:
                self._list.setCurrentItem(selected_item)
        finally:
            self._list.blockSignals(False)

    def _selected(self) -> Preset | None:
        item = self._list.currentItem()
        if item is None:
            return None
        return self._store.get(item.data(1))

    def _on_selection(self) -> None:
        preset = self._selected()
        if preset is None:
            return
        if self._store.set_active(preset.id):
            self._store.save()
            self._refresh()
            self.presets_changed.emit()

    def _create(self) -> None:
        dialog = FormulaEditorDialog(players=self._players, parent=self)
        if dialog.exec() != QDialog.DialogCode.Accepted:
            return
        preset = dialog.preset()
        preset.id = uuid.uuid4().hex
        self._store.add(preset)
        self._store.set_active(preset.id)
        self._store.save()
        self._refresh()
        self.presets_changed.emit()

    def _edit(self) -> None:
        preset = self._selected()
        if preset is None:
            return
        if preset.id == "standard":
            QMessageBox.information(
                self,
                "Стандартная формула",
                "«Стандартная» редактируется ползунками коэффициентов. "
                "Для текстовой формулы создайте новую или продублируйте её.",
            )
            return

        if preset.kind == "linear":
            edit_preset = Preset(
                id=preset.id,
                name=preset.name,
                kind="expression",
                expression=linear_to_expression(preset.weights),
            )
        else:
            edit_preset = Preset(
                id=preset.id,
                name=preset.name,
                kind=preset.kind,
                weights=dict(preset.weights),
                expression=preset.expression,
            )

        dialog = FormulaEditorDialog(edit_preset, self._players, parent=self)
        if dialog.exec() != QDialog.DialogCode.Accepted:
            return
        self._store.add(edit_preset)
        self._store.save()
        self._refresh()
        self.presets_changed.emit()

    def _duplicate(self) -> None:
        preset = self._selected()
        if preset is None:
            return
        copy = Preset(
            id=uuid.uuid4().hex,
            name=f"{preset.name} (копия)",
            kind=preset.kind,
            weights=dict(preset.weights),
            expression=preset.expression,
        )
        self._store.add(copy)
        self._store.save()
        self._refresh()
        self.presets_changed.emit()

    def _delete(self) -> None:
        preset = self._selected()
        if preset is None:
            return
        if preset.id == "standard":
            return
        answer = QMessageBox.question(
            self,
            "Удалить формулу",
            f"Удалить формулу «{preset.name}»?",
        )
        if answer != QMessageBox.StandardButton.Yes:
            return
        self._store.remove(preset.id)
        self._store.save()
        self._refresh()
        self.presets_changed.emit()

    def _export(self) -> None:
        preset = self._selected()
        if preset is None:
            return
        path, _ = QFileDialog.getSaveFileName(
            self, "Экспорт формулы", f"{preset.name}.json", "JSON (*.json)"
        )
        if not path:
            return
        try:
            self._store.export_file(preset, path)
        except OSError as exc:
            QMessageBox.critical(self, "Ошибка экспорта", str(exc))

    def _import(self) -> None:
        path, _ = QFileDialog.getOpenFileName(
            self, "Импорт формулы", "", "JSON (*.json)"
        )
        if not path:
            return
        try:
            preset = self._store.import_file(path)
        except (FormulaError, OSError) as exc:
            QMessageBox.critical(self, "Ошибка импорта", str(exc))
            return
        self._store.set_active(preset.id)
        self._store.save()
        self._refresh()
        self.presets_changed.emit()
