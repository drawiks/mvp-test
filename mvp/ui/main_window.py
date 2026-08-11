from __future__ import annotations

import time
from pathlib import Path

from PyQt6.QtCore import QSettings, Qt, QTimer, pyqtSignal
from PyQt6.QtGui import QDragEnterEvent, QDropEvent, QKeySequence, QShortcut
from PyQt6.QtWidgets import (
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

from ..manta_client import MantaNotFoundError, MantaWorker, find_manta_cli
from ..model import Result
from ..mvp import DEFAULT_WEIGHTS, select_mvps, weights_from_mapping, weights_to_mapping
from .mvp_cards import MvpCardsRow
from .stats_table import StatsTable
from .theme import STYLE_SHEET
from .weights_panel import WeightsPanel

REPLAY_FILTER = "Реплеи (*.dem *.dem.bz2 *.dem.zst);;Все файлы (*)"

_ACCEPTED_SUFFIXES = (".dem", ".dem.bz2", ".dem.zst")


class DropZone(QLabel):
    clicked = pyqtSignal()

    def __init__(self, parent=None):
        super().__init__(parent)
        self.setAlignment(Qt.AlignmentFlag.AlignCenter)
        self.setWordWrap(True)
        self.setCursor(Qt.CursorShape.PointingHandCursor)

    def mousePressEvent(self, event) -> None:
        if event.button() == Qt.MouseButton.LeftButton:
            self.clicked.emit()
        super().mousePressEvent(event)


class MainWindow(QMainWindow):
    def __init__(self, parent=None):
        super().__init__(parent)
        self.setWindowTitle("MVP Calculator")
        self.resize(1280, 820)
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

        self._build_ui()
        self._build_menu()
        self.setAcceptDrops(True)

        self._load_settings()
        self._show_idle()

    # ------------------------------------------------------------------ UI
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
        subtitle = QLabel("Расчёт MVP по реплею Dota 2 через manta_cli · коэффициенты редактируются на лету")
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
        self._drop_zone.setObjectName("dropHint")
        self._drop_zone.setMinimumHeight(56)
        self._drop_zone.clicked.connect(self.choose_file)
        root.addWidget(self._drop_zone)

        info_bar = QFrame()
        info_bar.setObjectName("card")
        info_layout = QHBoxLayout(info_bar)
        info_layout.setContentsMargins(14, 8, 14, 8)
        info_layout.setSpacing(24)
        self._info = {}
        for key, caption in [
            ("match", "Match ID"),
            ("duration", "Длительность"),
            ("winner", "Победитель"),
            ("elapsed", "Анализ"),
        ]:
            box = QVBoxLayout()
            cap = QLabel(caption)
            cap.setObjectName("matchInfoCaption")
            val = QLabel("—")
            val.setObjectName("matchInfoValue")
            box.addWidget(cap)
            box.addWidget(val)
            info_layout.addLayout(box)
            self._info[key] = val
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

        self._weights_panel = WeightsPanel()
        self._weights_panel.setMinimumWidth(300)
        self._weights_panel.setMaximumWidth(360)
        self._weights_panel.changed.connect(self._recompute)
        splitter.addWidget(self._weights_panel)
        splitter.setStretchFactor(0, 1)
        splitter.setStretchFactor(1, 0)
        splitter.setSizes([940, 320])

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

    # ------------------------------------------------------------ settings
    def _load_settings(self) -> None:
        custom = self._settings.value("manta/path")
        if custom:
            self._binary = Path(custom)
        mapping = {}
        for key in weights_to_mapping(DEFAULT_WEIGHTS):
            value = self._settings.value(f"weights/{key}")
            if value is not None:
                try:
                    mapping[key] = float(value)
                except (TypeError, ValueError):
                    pass
        if mapping:
            self._weights_panel.set_weights(weights_from_mapping(mapping))

    def _save_weights(self) -> None:
        for key, value in weights_to_mapping(self._weights_panel.weights()).items():
            self._settings.setValue(f"weights/{key}", value)

    # ---------------------------------------------------------------- drag&drop
    def dragEnterEvent(self, event: QDragEnterEvent) -> None:
        if self._urls_from_event(event):
            self._drop_zone.setObjectName("dropHintActive")
            self._drop_zone.style().unpolish(self._drop_zone)
            self._drop_zone.style().polish(self._drop_zone)
            event.acceptProposedAction()

    def dragLeaveEvent(self, event) -> None:
        self._restore_drop_zone_style()
        super().dragLeaveEvent(event)

    def dropEvent(self, event: QDropEvent) -> None:
        self._restore_drop_zone_style()
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

    def _restore_drop_zone_style(self) -> None:
        self._drop_zone.setObjectName("dropHint")
        self._drop_zone.style().unpolish(self._drop_zone)
        self._drop_zone.style().polish(self._drop_zone)

    # ---------------------------------------------------------------- actions
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
        if self._worker is not None and self._worker.is_running():
            return
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
        self._table.clearContents()
        self._table.setRowCount(0)
        for value in self._info.values():
            value.setText("—")

        worker = MantaWorker(binary)
        worker.finished_ok.connect(self._on_parse_ok)
        worker.failed.connect(self._on_parse_failed)
        self._worker = worker

        self._set_busy(True)
        self._drop_zone.setText(f"Анализ реплея… 0 с\n{path}")
        self._started_at = time.monotonic()
        self._elapsed = 0
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

    # ------------------------------------------------------------ processing
    def _set_busy(self, busy: bool) -> None:
        self._open_btn.setEnabled(not busy)

    def _on_tick(self) -> None:
        self._elapsed += 1
        self._info["elapsed"].setText(f"работает… {self._elapsed} с")

    def _on_parse_ok(self, result: Result) -> None:
        elapsed = time.monotonic() - self._started_at
        self._timer.stop()
        self._set_busy(False)
        self._result = result
        self._drop_zone.setText(
            f"{self._last_path.name if self._last_path else 'реплей'}"
            f" · обработано за {elapsed:.1f} с — нажмите для другого файла"
        )
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
        self._restore_drop_zone_style()
        self._drop_zone.setText("Перетащите .dem реплей сюда или нажмите «Выбрать файл»")
        self.statusBar().showMessage("Ошибка анализа реплея", 6000)
        print(f"[mvp] error: {message}", flush=True)
        QMessageBox.critical(self, "Ошибка анализа реплея", message)

    def _recompute(self) -> None:
        if self._result is None:
            return
        weights = self._weights_panel.weights()
        mvps = select_mvps(self._result, weights)
        self._cards.show_mvps(mvps, weights)
        self._table.set_result(self._result, weights)
        self._update_match_info()
        self._save_weights()

    def _update_match_info(self) -> None:
        result = self._result
        if result is None:
            return
        self._info["match"].setText(str(result.match_id) if result.match_id else "—")
        self._info["duration"].setText(self._format_duration(result.duration_sec))
        winner = "Radiant" if result.radiant_win else "Dire"
        loser = "Dire" if result.radiant_win else "Radiant"
        self._info["winner"].setText(f"{winner} (проиграли: {loser})")

    @staticmethod
    def _format_duration(seconds: int) -> str:
        if seconds <= 0:
            return "—"
        minutes, sec = divmod(seconds, 60)
        return f"{minutes} мин {sec} с"

    def _show_idle(self) -> None:
        self._drop_zone.setText(
            "Перетащите файл .dem / .dem.bz2 / .dem.zst сюда или нажмите «Выбрать файл»"
        )

    def _show_about(self) -> None:
        QMessageBox.about(
            self,
            "О программе",
            "MVP Calculator\n\n"
            "Разбор реплея Dota 2 через manta_cli и расчёт MVP по формуле.\n"
            "Коэффициенты формулы редактируются в правой панели.\n"
            "Версия: 1.0.0",
        )
