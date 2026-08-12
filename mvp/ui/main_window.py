from __future__ import annotations

import time
from pathlib import Path

from PyQt6.QtCore import QSettings, QStandardPaths, Qt, QTimer, pyqtSignal
from PyQt6.QtGui import QColor, QDragEnterEvent, QDropEvent, QKeySequence, QPainter, QPen, QShortcut
from PyQt6.QtWidgets import (
    QDialog,
    QFileDialog,
    QFrame,
    QHBoxLayout,
    QLabel,
    QMainWindow,
    QMenu,
    QMessageBox,
    QPushButton,
    QSplitter,
    QVBoxLayout,
    QWidget,
)

from ..formula import DEFAULT_LINEAR_WEIGHTS, FormulaError, PresetStore
from ..manta_client import MantaNotFoundError, MantaWorker, find_manta_cli
from ..model import Result
from ..mvp import DEFAULT_WEIGHTS, select_mvps, weights_to_mapping
from . import theme
from .formula_dialog import FormulaEditorDialog
from .formula_manager import FormulaManagerDialog
from .icons import folder_pixmap
from .mvp_cards import MvpCardsRow
from .stats_table import StatsTable
from .theme import STYLE_SHEET
from .weights_panel import WeightsPanel

REPLAY_FILTER = "Реплеи (*.dem *.dem.bz2 *.dem.zst);;Все файлы (*)"

_ACCEPTED_SUFFIXES = (".dem", ".dem.bz2", ".dem.zst")


def _set_object_name(widget: QWidget, name: str) -> None:
    widget.setObjectName(name)
    widget.style().unpolish(widget)
    widget.style().polish(widget)


class Spinner(QWidget):
    def __init__(self, parent=None):
        super().__init__(parent)
        self._angle = 0
        self.setFixedSize(20, 20)
        self._timer = QTimer(self)
        self._timer.setInterval(16)
        self._timer.timeout.connect(self._tick)

    def start(self) -> None:
        self._timer.start()

    def stop(self) -> None:
        self._timer.stop()

    def _tick(self) -> None:
        self._angle = (self._angle + 8) % 360
        self.update()

    def paintEvent(self, event) -> None:
        painter = QPainter(self)
        painter.setRenderHint(QPainter.RenderHint.Antialiasing)
        rect = self.rect().adjusted(2, 2, -2, -2)
        pen = QPen(QColor(theme.ACCENT), 3)
        pen.setCapStyle(Qt.PenCapStyle.RoundCap)
        painter.setPen(pen)
        painter.drawArc(rect, -self._angle * 16, 300 * 16)


class DropZone(QFrame):
    clicked = pyqtSignal()

    def __init__(self, parent=None):
        super().__init__(parent)
        self.setObjectName("dropZone")
        self.setCursor(Qt.CursorShape.PointingHandCursor)
        self.setMinimumHeight(86)

        layout = QVBoxLayout(self)
        layout.setContentsMargins(16, 10, 16, 10)
        layout.setSpacing(6)

        self._busy_row = QWidget()
        busy_layout = QHBoxLayout(self._busy_row)
        busy_layout.setContentsMargins(0, 0, 0, 0)
        busy_layout.setSpacing(10)
        self._spinner = Spinner()
        self._busy_text = QLabel("")
        self._busy_text.setAlignment(Qt.AlignmentFlag.AlignVCenter)
        busy_layout.addStretch(1)
        busy_layout.addWidget(self._spinner)
        busy_layout.addWidget(self._busy_text, 1)
        busy_layout.addStretch(1)
        layout.addWidget(self._busy_row, 1)

        self._icon = QLabel()
        self._icon.setPixmap(folder_pixmap((30, 30)))
        self._icon.setAlignment(Qt.AlignmentFlag.AlignCenter)
        layout.addWidget(self._icon, 0, Qt.AlignmentFlag.AlignHCenter)

        self._text = QLabel("")
        self._text.setAlignment(Qt.AlignmentFlag.AlignCenter)
        self._text.setWordWrap(True)
        layout.addWidget(self._text)

        self._button = QPushButton("Выбрать файл")
        self._button.setObjectName("ghost")
        self._button.setCursor(Qt.CursorShape.PointingHandCursor)
        self._button.clicked.connect(self.clicked)
        layout.addWidget(self._button, 0, Qt.AlignmentFlag.AlignHCenter)

        self.set_idle()

    def set_idle(self) -> None:
        self._busy_row.setVisible(False)
        self._icon.setVisible(True)
        self._button.setVisible(True)
        self._text.setText(
            "Перетащите файл .dem / .dem.bz2 / .dem.zst сюда или нажмите «Выбрать файл»"
        )

    def set_busy(self, path: str, elapsed: int) -> None:
        self._spinner.start()
        self._icon.setVisible(False)
        self._button.setVisible(False)
        self._busy_row.setVisible(True)
        self._busy_text.setText(f"Анализ реплея… {elapsed} с\n{path}")

    def set_done(self, path: str, elapsed: float) -> None:
        self._spinner.stop()
        self._busy_row.setVisible(False)
        self._icon.setVisible(True)
        self._button.setVisible(True)
        self._text.setText(
            f"{path} · обработано за {elapsed:.1f} с - нажмите для другого файла"
        )

    def set_error(self) -> None:
        self._spinner.stop()
        self._busy_row.setVisible(False)
        self._icon.setVisible(True)
        self._button.setVisible(True)
        self.set_idle()

    def mousePressEvent(self, event) -> None:
        if event.button() == Qt.MouseButton.LeftButton:
            self.clicked.emit()
        super().mousePressEvent(event)


class MainWindow(QMainWindow):
    def __init__(self, parent=None):
        super().__init__(parent)
        self.setWindowTitle("MVP Calculator")
        self.resize(1320, 860)
        self.setStyleSheet(STYLE_SHEET)

        self._settings = QSettings()
        self._result: Result | None = None
        self._worker: MantaWorker | None = None
        self._binary: Path | None = None
        self._last_path: Path | None = None
        self._started_at = 0.0
        self._elapsed = 0
        self._timer = QTimer(self)
        self._timer.setInterval(1000)
        self._timer.timeout.connect(self._on_tick)

        data_dir = QStandardPaths.writableLocation(
            QStandardPaths.StandardLocation.AppDataLocation
        )
        store_path = Path(data_dir) / "formulas.json" if data_dir else None
        self._store = PresetStore(store_path) if store_path else PresetStore(Path.home() / ".mvp-calculator" / "formulas.json")

        self._build_ui()
        self._build_menu()
        self.setAcceptDrops(True)

        self._load_settings()

    def _build_ui(self) -> None:
        central = QWidget()
        self.setCentralWidget(central)
        root = QVBoxLayout(central)
        root.setContentsMargins(16, 14, 16, 10)
        root.setSpacing(10)

        header = QHBoxLayout()
        title_box = QVBoxLayout()
        title = QLabel("MVP Calculator")
        title.setObjectName("pageTitle")
        subtitle = QLabel("by drawiks")
        subtitle.setObjectName("pageSubtitle")
        title_box.addWidget(title)
        title_box.addWidget(subtitle)
        header.addLayout(title_box, 1)
        self._open_btn = QPushButton("Выбрать файл")
        self._open_btn.setObjectName("accent")
        self._open_btn.clicked.connect(self.choose_file)
        header.addWidget(self._open_btn)
        root.addLayout(header)

        self._drop_zone = DropZone()
        self._drop_zone.clicked.connect(self.choose_file)
        root.addWidget(self._drop_zone)

        info_bar = QFrame()
        info_bar.setObjectName("card")
        info_layout = QHBoxLayout(info_bar)
        info_layout.setContentsMargins(14, 10, 14, 10)
        info_layout.setSpacing(26)
        self._info = {}

        for key, caption in [("match", "Match ID"), ("duration", "Длительность"), ("elapsed", "Анализ")]:
            box = QVBoxLayout()
            cap = QLabel(caption)
            cap.setObjectName("matchInfoCaption")
            val = QLabel("-")
            val.setObjectName("matchInfoValue")
            box.addWidget(cap)
            box.addWidget(val)
            info_layout.addLayout(box)
            self._info[key] = val

        score_box = QVBoxLayout()
        score_cap = QLabel("Счёт команд")
        score_cap.setObjectName("matchInfoCaption")
        score_row = QHBoxLayout()
        score_row.setSpacing(2)
        self._score_r = QLabel("-")
        self._score_d = QLabel("-")
        self._score_r.setMinimumWidth(22)
        self._score_d.setMinimumWidth(22)
        self._score_r.setAlignment(Qt.AlignmentFlag.AlignCenter)
        self._score_d.setAlignment(Qt.AlignmentFlag.AlignCenter)
        self._score_r.setStyleSheet(f"color: {theme.RADIANT}; font-weight: 700; font-size: 14px;")
        self._score_d.setStyleSheet(f"color: {theme.DIRE}; font-weight: 700; font-size: 14px;")
        sep = QLabel(":")
        sep.setStyleSheet("color: #6e7681;")
        score_row.addWidget(self._score_r)
        score_row.addWidget(sep)
        score_row.addWidget(self._score_d)
        score_box.addWidget(score_cap)
        score_box.addLayout(score_row)
        info_layout.addLayout(score_box)

        winner_box = QVBoxLayout()
        winner_cap = QLabel("Победитель")
        winner_cap.setObjectName("matchInfoCaption")
        self._winner_badge = QLabel("-")
        self._winner_badge.hide()
        winner_box.addWidget(winner_cap)
        winner_box.addWidget(self._winner_badge)
        info_layout.addLayout(winner_box)

        info_layout.addStretch(1)
        root.addWidget(info_bar)

        splitter = QSplitter(Qt.Orientation.Horizontal)

        left = QWidget()
        left_layout = QVBoxLayout(left)
        left_layout.setContentsMargins(0, 0, 0, 0)
        left_layout.setSpacing(10)
        self._cards = MvpCardsRow()
        self._table = StatsTable()
        left_layout.addWidget(self._cards)
        left_layout.addWidget(self._table, 1)
        splitter.addWidget(left)

        self._weights_panel = WeightsPanel(self._store)
        self._weights_panel.setMinimumWidth(300)
        self._weights_panel.setMaximumWidth(380)
        self._weights_panel.changed.connect(self._recompute)
        self._weights_panel.edit_requested.connect(self._edit_active_formula)
        splitter.addWidget(self._weights_panel)
        splitter.setStretchFactor(0, 1)
        splitter.setStretchFactor(1, 0)
        splitter.setSizes([950, 340])

        root.addWidget(splitter, 1)

        self.statusBar().showMessage("Загрузите .dem реплей")

    def _build_menu(self) -> None:
        file_menu = QMenu("Файл", self)
        open_action = file_menu.addAction("Открыть реплей…")
        open_action.setShortcut(QKeySequence.StandardKey.Open)
        open_action.triggered.connect(self.choose_file)
        quit_action = file_menu.addAction("Выход")
        quit_action.triggered.connect(self.close)
        self.menuBar().addMenu(file_menu)

        settings_menu = QMenu("Настройки", self)
        formulas_action = settings_menu.addAction("Формулы…")
        formulas_action.triggered.connect(self._open_formulas)
        manta_action = settings_menu.addAction("Путь к manta_cli…")
        manta_action.triggered.connect(self.choose_manta_binary)
        reset_action = settings_menu.addAction("Сбросить коэффициенты")
        reset_action.triggered.connect(self._weights_panel.reset)
        self.menuBar().addMenu(settings_menu)

        help_menu = QMenu("Справка", self)
        about_action = help_menu.addAction("О программе")
        about_action.triggered.connect(self._show_about)
        self.menuBar().addMenu(help_menu)

        QShortcut(QKeySequence("Ctrl+O"), self, self.choose_file)

    def _load_settings(self) -> None:
        custom = self._settings.value("manta/path")
        if custom:
            self._binary = Path(custom)
        legacy = {}
        for key in weights_to_mapping(DEFAULT_WEIGHTS):
            value = self._settings.value(f"weights/{key}")
            if value is not None:
                try:
                    legacy[key] = float(value)
                except (TypeError, ValueError):
                    pass
        if legacy:
            standard = self._store.get("standard")
            if standard is not None and standard.weights == DEFAULT_LINEAR_WEIGHTS:
                standard.weights.update(legacy)
                self._store.save()
            for key in legacy:
                self._settings.remove(f"weights/{key}")
        self._drop_zone.set_idle()

    def _open_formulas(self) -> None:
        players = self._result.players if self._result else None
        dialog = FormulaManagerDialog(self._store, players, parent=self)
        dialog.presets_changed.connect(self._on_presets_changed)
        dialog.exec()

    def _edit_active_formula(self) -> None:
        preset = self._weights_panel.preset()
        if preset.kind != "expression":
            QMessageBox.information(
                self,
                "Стандартная формула",
                "«Стандартная» редактируется ползунками коэффициентов. "
                "Для текстовой формулы создайте новую в меню «Формулы…».",
            )
            return
        players = self._result.players if self._result else None
        dialog = FormulaEditorDialog(preset, players, parent=self)
        if dialog.exec() == QDialog.DialogCode.Accepted:
            self._store.save()
            self._weights_panel.refresh_presets()
            self._recompute()

    def _on_presets_changed(self) -> None:
        self._weights_panel.refresh_presets()
        self._recompute()

    def dragEnterEvent(self, event: QDragEnterEvent) -> None:
        if self._urls_from_event(event):
            _set_object_name(self._drop_zone, "dropZoneActive")
            event.acceptProposedAction()

    def dragLeaveEvent(self, event) -> None:
        _set_object_name(self._drop_zone, "dropZone")
        super().dragLeaveEvent(event)

    def dropEvent(self, event: QDropEvent) -> None:
        _set_object_name(self._drop_zone, "dropZone")
        paths = self._urls_from_event(event)
        if paths:
            self.load_replay(Path(paths[0]))

    @staticmethod
    def _urls_from_event(event) -> list[str]:
        urls = []
        for url in event.mimeData().urls():
            path = url.toLocalFile()
            if path:
                urls.append(path)
        return urls

    def choose_file(self) -> None:
        path, _ = QFileDialog.getOpenFileName(self, "Выберите реплей Dota 2", "", REPLAY_FILTER)
        if path:
            self.load_replay(Path(path))

    def choose_manta_binary(self) -> None:
        path, _ = QFileDialog.getOpenFileName(
            self,
            "Укажите бинарник manta_cli",
            "",
            "Исполняемые файлы (*.exe);;Все файлы (*)",
        )
        if not path:
            return
        self._binary = Path(path)
        self._settings.setValue("manta/path", str(path))
        self.statusBar().showMessage(f"manta_cli: {path}", 5000)

    def load_replay(self, path: Path) -> None:
        if self._worker is not None:
            if self._worker.is_running():
                self._worker.cancel()
            self._worker.deleteLater()
            self._worker = None

        if not self._is_replay_path(path):
            QMessageBox.warning(
                self, "Неверный файл",
                "Поддерживаются только файлы .dem, .dem.bz2 или .dem.zst",
            )
            return
        if not path.is_file():
            QMessageBox.warning(self, "Файл не найден", f"Не удалось найти файл:\n{path}")
            return

        try:
            binary = self._binary or find_manta_cli(self._settings.value("manta/path"))
        except MantaNotFoundError as exc:
            self._prompt_manta_missing(str(exc))
            return
        self._binary = binary
        self._last_path = path

        self._result = None
        self._cards.clear()
        self._table.setRowCount(0)
        for value in self._info.values():
            value.setText("-")
        self._score_r.setText("-")
        self._score_d.setText("-")
        self._winner_badge.hide()

        worker = MantaWorker(binary)
        worker.finished_ok.connect(self._on_parse_ok)
        worker.failed.connect(self._on_parse_failed)
        self._worker = worker

        self._set_busy(True)
        self._elapsed = 0
        self._started_at = time.monotonic()
        self._drop_zone.set_busy(path.name, 0)
        self._info["elapsed"].setText("работает…")
        self._timer.start()
        worker.start(path)

    def _is_replay_path(self, path: Path) -> bool:
        name = path.name.lower()
        return name.endswith(_ACCEPTED_SUFFIXES)

    def _prompt_manta_missing(self, message: str) -> None:
        print(f"[mvp] error: {message}", flush=True)
        box = QMessageBox(self)
        box.setIcon(QMessageBox.Icon.Warning)
        box.setWindowTitle("manta_cli не найден")
        box.setText("Не найден бинарник manta_cli")
        box.setInformativeText(
            message
            + "\n\nВы можете указать путь к manta_cli вручную или скопировать его "
            "рядом с приложением."
        )
        choose = box.addButton("Указать путь…", QMessageBox.ButtonRole.ActionRole)
        box.addButton("Отмена", QMessageBox.ButtonRole.RejectRole)
        box.exec()
        if box.clickedButton() is choose:
            self.choose_manta_binary()

    def _set_busy(self, busy: bool) -> None:
        self._open_btn.setEnabled(not busy)

    def _on_tick(self) -> None:
        self._elapsed += 1
        self._info["elapsed"].setText(f"работает… {self._elapsed} с")
        if self._last_path is not None:
            self._drop_zone.set_busy(self._last_path.name, self._elapsed)

    def _on_parse_ok(self, result: Result) -> None:
        elapsed = time.monotonic() - self._started_at
        self._timer.stop()
        self._set_busy(False)
        self._finish_worker()
        self._result = result
        if self._last_path is not None:
            self._drop_zone.set_done(self._last_path.name, elapsed)
        self._info["elapsed"].setText(f"{elapsed:.1f} с")
        self._recompute()
        print(f"[mvp] ok match_id={result.match_id} duration={result.duration_sec}s", flush=True)
        self.statusBar().showMessage(
            f"Реплей обработан за {elapsed:.1f} с · игроков: {len(result.players)}",
            6000,
        )

    def _on_parse_failed(self, message: str) -> None:
        self._timer.stop()
        self._set_busy(False)
        self._finish_worker()
        _set_object_name(self._drop_zone, "dropZone")
        self._drop_zone.set_error()
        self.statusBar().showMessage("Ошибка анализа реплея", 6000)
        print(f"[mvp] error: {message}", flush=True)
        QMessageBox.critical(self, "Ошибка анализа реплея", message)

    def _finish_worker(self) -> None:
        if self._worker is not None:
            worker, self._worker = self._worker, None
            worker.deleteLater()

    def _recompute(self) -> None:
        if self._result is None:
            return
        preset = self._weights_panel.preset()
        try:
            mvps = select_mvps(self._result, preset)
        except FormulaError as exc:
            self.statusBar().showMessage(f"Ошибка формулы: {exc}", 6000)
            return
        self._cards.show_mvps(mvps, preset)
        self._table.set_result(self._result, preset)
        self._update_match_info()

    def _update_match_info(self) -> None:
        result = self._result
        if result is None:
            return
        self._info["match"].setText(str(result.match_id) if result.match_id else "-")
        self._info["duration"].setText(self._format_duration(result.duration_sec))

        radiant_kills = sum(p.kills for p in result.players if p.team == "radiant")
        dire_kills = sum(p.kills for p in result.players if p.team == "dire")
        self._score_r.setText(str(radiant_kills))
        self._score_d.setText(str(dire_kills))

        winner = "Radiant" if result.radiant_win else "Dire"
        self._winner_badge.setText(f"{winner} побеждают")
        _set_object_name(
            self._winner_badge,
            "matchBadgeRadiant" if result.radiant_win else "matchBadgeDire",
        )
        self._winner_badge.show()

    @staticmethod
    def _format_duration(seconds: int) -> str:
        if seconds <= 0:
            return "-"
        minutes, sec = divmod(seconds, 60)
        return f"{minutes} мин {sec} с"

    def _show_about(self) -> None:
        QMessageBox.about(self, "О программе", "by drawiks")
