from __future__ import annotations

import sys
from pathlib import Path

from PyQt6.QtCore import QCoreApplication, QTimer
from PyQt6.QtWidgets import QApplication

from mvp import __version__
from mvp.ui.main_window import MainWindow
from mvp.ui.resources import register_fonts

_REPLAY_SUFFIXES = (".dem", ".dem.bz2", ".dem.zst")


def main() -> int:
    QCoreApplication.setOrganizationName("drawiks")
    QCoreApplication.setApplicationName("mvp-calculator")
    app = QApplication(sys.argv)
    app.setApplicationDisplayName("MVP Calculator")
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
