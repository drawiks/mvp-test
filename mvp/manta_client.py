from __future__ import annotations

import json
import os
import shutil
import sys
from pathlib import Path

from PyQt6.QtCore import QObject, QProcess, pyqtSignal

from .model import Player, Result


class MantaNotFoundError(FileNotFoundError):
    pass


class MantaParseError(RuntimeError):
    pass


def _binary_name() -> str:
    return "manta_cli.exe" if os.name == "nt" else "manta_cli"


def find_manta_cli(custom_path: str | None = None) -> Path:
    name = _binary_name()
    candidates: list[Path] = []
    if custom_path:
        candidates.append(Path(custom_path))
    if getattr(sys, "frozen", False):
        candidates.append(Path(sys.executable).resolve().parent / name)
    else:
        package_parent = Path(__file__).resolve().parent.parent
        candidates.append(package_parent / name)
        candidates.append(Path(__file__).resolve().parent / name)
    for candidate in candidates:
        if candidate.is_file():
            return candidate
    which = shutil.which(name)
    if which:
        return Path(which)
    raise MantaNotFoundError(
        f"Не найден бинарник {name}. Положите его рядом с приложением, "
        "добавьте в PATH или укажите путь вручную в настройках."
    )


def parse_result(raw: str | bytes) -> Result:
    if isinstance(raw, bytes):
        raw = raw.decode("utf-8", errors="replace")
    try:
        data = json.loads(raw)
    except json.JSONDecodeError as exc:
        raise MantaParseError(f"Некорректный JSON от manta_cli: {exc}") from exc
    if not isinstance(data, dict) or not isinstance(data.get("players"), list):
        raise MantaParseError("Некорректный ответ manta_cli: нет поля players")
    players: list[Player] = []
    for item in data["players"]:
        if not isinstance(item, dict):
            continue
        item.setdefault("team", "unknown")
        players.append(Player(**item))
    return Result(
        match_id=int(data.get("match_id") or 0),
        duration_sec=int(data.get("duration_sec") or 0),
        radiant_win=bool(data.get("radiant_win", True)),
        players=players,
    )


class MantaWorker(QObject):
    finished_ok = pyqtSignal(object)
    failed = pyqtSignal(str)

    def __init__(self, binary: Path, parent: QObject | None = None):
        super().__init__(parent)
        self._binary = Path(binary)
        self._proc = QProcess(self)
        self._proc.readyReadStandardOutput.connect(self._on_stdout)
        self._proc.readyReadStandardError.connect(self._on_stderr)
        self._proc.finished.connect(self._on_finished)
        self._stdout = bytearray()
        self._stderr = bytearray()

    @property
    def binary(self) -> Path:
        return self._binary

    def start(self, replay_path: Path) -> None:
        self._stdout.clear()
        self._stderr.clear()
        self._proc.start(str(self._binary), [str(Path(replay_path))])

    def is_running(self) -> bool:
        return self._proc.state() != QProcess.ProcessState.NotRunning

    def cancel(self) -> None:
        if self.is_running():
            self._proc.kill()

    def _on_stdout(self) -> None:
        self._stdout.extend(self._proc.readAllStandardOutput())

    def _on_stderr(self) -> None:
        self._stderr.extend(self._proc.readAllStandardError())

    def _on_finished(self, exit_code: int, _status) -> None:
        if exit_code != 0:
            detail = self._stderr.decode("utf-8", errors="replace").strip()
            self.failed.emit(
                f"manta_cli завершился с кодом {exit_code}"
                + (f":\n{detail}" if detail else "")
            )
            return
        try:
            result = parse_result(bytes(self._stdout))
        except (MantaParseError, TypeError, ValueError) as exc:
            self.failed.emit(str(exc))
            return
        self.finished_ok.emit(result)
