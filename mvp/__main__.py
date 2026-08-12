from __future__ import annotations

import logging
import os
import sys
import tempfile
import traceback
from pathlib import Path

from PyQt6.QtCore import QCoreApplication, QTimer
from PyQt6.QtWidgets import QApplication

from mvp import __version__
from mvp.ui.main_window import MainWindow
from mvp.ui.resources import register_fonts

_REPLAY_SUFFIXES = (".dem", ".dem.bz2", ".dem.zst")

_LOG_PATH = Path(tempfile.gettempdir()) / "mvp_calculator.log"


def _log_unhandled(exc_type, exc, tb) -> None:
    with open(_LOG_PATH, "a", encoding="utf-8") as fh:
        fh.write("".join(traceback.format_exception(exc_type, exc, tb)) + "\n")
    sys.__excepthook__(exc_type, exc, tb)


def main() -> int:
    QCoreApplication.setOrganizationName("drawiks")
    QCoreApplication.setApplicationName("mvp-calculator")
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(levelname)s %(name)s: %(message)s",
        handlers=[logging.FileHandler(_LOG_PATH, encoding="utf-8"), logging.StreamHandler()],
    )
    app = QApplication(sys.argv)
    app.setApplicationDisplayName("MVP Calculator")
    sys.excepthook = _log_unhandled
    register_fonts()
    window = MainWindow()
    window.show()

    replay = next(
        (arg for arg in sys.argv[1:] if arg.lower().endswith(_REPLAY_SUFFIXES)),
        None,
    )
    if replay:
        QTimer.singleShot(0, lambda: window.load_replay(Path(replay)))

    return app.exec()


if __name__ == "__main__":
    sys.exit(main())
