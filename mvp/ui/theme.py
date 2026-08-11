from __future__ import annotations

STYLE_SHEET = """
* {
    font-family: "Segoe UI", "Ubuntu", sans-serif;
}

QMainWindow, QWidget {
    background-color: #15171b;
    color: #d8dbe0;
}

QFrame#card {
    background-color: #1f2229;
    border: 1px solid #2e323b;
    border-radius: 10px;
}

QFrame#cardGold { background-color: #2a2413; border: 2px solid #c9a227; }
QFrame#cardSilver { background-color: #23252c; border: 2px solid #8f98a3; }
QFrame#cardBronze { background-color: #291f16; border: 2px solid #a9702f; }

QLabel#cardTitle {
    color: #9aa1ab;
    font-size: 12px;
    font-weight: 600;
    letter-spacing: 1px;
}

QLabel#cardMedal { font-size: 30px; }

QLabel#cardHero {
    color: #f5c542;
    font-size: 20px;
    font-weight: 700;
}

QLabel#cardName { color: #e8e8e8; font-size: 15px; font-weight: 600; }

QLabel#cardScore { font-size: 34px; font-weight: 800; }
QLabel#cardScoreGold { color: #f5c542; }
QLabel#cardScoreSilver { color: #c6ccd4; }
QLabel#cardScoreBronze { color: #d28a4a; }

QLabel#cardStat { color: #9aa1ab; font-size: 12px; }
QLabel#cardStatValue { color: #e8e8e8; font-size: 12px; font-weight: 600; }

QLabel#matchInfoValue {
    color: #cfe9ff;
    font-size: 13px;
    font-weight: 600;
}
QLabel#matchInfoCaption {
    color: #7d8590;
    font-size: 11px;
    text-transform: uppercase;
}

QLabel#pageTitle { font-size: 22px; font-weight: 800; }
QLabel#pageSubtitle { color: #7d8590; font-size: 12px; }

QLabel#dropHint {
    color: #9aa1ab;
    font-size: 13px;
    border: 2px dashed #3a4049;
    border-radius: 10px;
    padding: 14px;
}
QLabel#dropHintActive {
    color: #cfe9ff;
    border-color: #4a9bff;
    background-color: #16202e;
}

QPushButton {
    background-color: #2a2e36;
    border: 1px solid #3a4049;
    border-radius: 6px;
    padding: 8px 16px;
    font-weight: 600;
}
QPushButton:hover { background-color: #333842; }
QPushButton:pressed { background-color: #23262d; }
QPushButton:disabled { color: #6a707a; background-color: #22252b; }
QPushButton#accent {
    background-color: #f5c542;
    color: #201c0d;
    border: none;
}
QPushButton#accent:hover { background-color: #ffd95e; }
QPushButton#accent:disabled { background-color: #5c5426; color: #8a8260; }

QGroupBox {
    border: 1px solid #2e323b;
    border-radius: 8px;
    margin-top: 12px;
    padding-top: 8px;
    font-weight: 700;
}
QGroupBox::title {
    subcontrol-origin: margin;
    left: 10px;
    padding: 0 4px;
    color: #cfe9ff;
}

QDoubleSpinBox {
    background-color: #17191e;
    border: 1px solid #333842;
    border-radius: 5px;
    padding: 4px 6px;
}
QDoubleSpinBox:focus { border-color: #4a9bff; }

QTableWidget {
    background-color: #1a1d23;
    alternate-background-color: #20242c;
    gridline-color: #2a2e36;
    border: 1px solid #2e323b;
    border-radius: 8px;
    selection-background-color: #2c3646;
}
QHeaderView::section {
    background-color: #22252c;
    color: #9aa1ab;
    border: none;
    border-bottom: 1px solid #333842;
    padding: 6px 8px;
    font-weight: 700;
}
QTableWidget::item { padding: 4px 8px; }

QScrollBar:vertical {
    background: transparent;
    width: 10px;
    margin: 0;
}
QScrollBar::handle:vertical {
    background: #3a4049;
    border-radius: 5px;
    min-height: 24px;
}
QScrollBar::add-line:vertical, QScrollBar::sub-line:vertical { height: 0; }

QStatusBar { color: #9aa1ab; background-color: #181a1f; border-top: 1px solid #262a31; }
QStatusBar::item { border: none; }

QToolTip {
    background-color: #20242c;
    color: #e8e8e8;
    border: 1px solid #3a4049;
    padding: 6px;
}
"""
