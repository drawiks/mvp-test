from __future__ import annotations

BG = "#0e1116"
BG_ELEVATED = "#131820"
PANEL = "#161b22"
PANEL_ALT = "#1c222b"
BORDER = "#262d38"
BORDER_LIGHT = "#333b47"

TEXT_PRIMARY = "#e6edf3"
TEXT_SECONDARY = "#98a2b3"
TEXT_MUTED = "#6e7681"

ACCENT = "#f5c542"
ACCENT_HOVER = "#ffd95e"
ACCENT_TEXT = "#1d1706"
BLUE = "#4a9bff"

RADIANT = "#4ecb71"
DIRE = "#e15241"

GOLD = "#f5c542"
SILVER = "#c6ccd4"
BRONZE = "#d28a4a"

KILL = "#7bd88f"
DEATH = "#f09090"

CARD_GOLD_BG = "#26200e"
CARD_SILVER_BG = "#1f2228"
CARD_BRONZE_BG = "#261b12"


def _rgba(hex_color: str, alpha: int) -> str:
    r = int(hex_color[1:3], 16)
    g = int(hex_color[3:5], 16)
    b = int(hex_color[5:7], 16)
    return f"rgba({r},{g},{b},{alpha})"


def _grad(hex_color: str, a0: int, a1: int) -> str:
    return f"qlineargradient(x1:0, y1:0, x2:1, y2:0, stop:0 {_rgba(hex_color, a0)}, stop:1 {_rgba(hex_color, a1)})"


def build_stylesheet() -> str:
    return f"""
* {{
    font-family: "Inter", "Segoe UI", "Ubuntu", sans-serif;
    font-size: 13px;
}}

QMainWindow, QDialog {{
    background-color: {BG};
}}

QWidget {{
    color: {TEXT_PRIMARY};
    background: transparent;
}}

QMainWindow::separator {{ background: {BORDER}; width: 1px; }}

QMenuBar {{ background: {BG}; color: {TEXT_SECONDARY}; }}
QMenuBar::item {{ padding: 5px 10px; background: transparent; }}
QMenuBar::item:selected {{ background: {PANEL_ALT}; color: {TEXT_PRIMARY}; }}
QMenu {{
    background-color: {PANEL};
    border: 1px solid {BORDER};
    padding: 4px;
}}
QMenu::item {{ padding: 6px 26px 6px 18px; color: {TEXT_SECONDARY}; border-radius: 4px; }}
QMenu::item:selected {{ background: {PANEL_ALT}; color: {TEXT_PRIMARY}; }}
QMenu::separator {{ height: 1px; background: {BORDER}; margin: 4px 6px; }}

QLabel#pageTitle {{ font-size: 24px; font-weight: 800; color: {TEXT_PRIMARY}; }}
QLabel#pageSubtitle {{ color: {TEXT_SECONDARY}; font-size: 12px; }}
QLabel#sectionTitle {{ color: {TEXT_SECONDARY}; font-size: 13px; font-weight: 700; }}

QFrame#card {{ background-color: {PANEL}; border: 1px solid {BORDER}; border-radius: 12px; }}
QFrame#cardGold {{ background-color: {CARD_GOLD_BG}; border: 1px solid {GOLD}; border-radius: 14px; }}
QFrame#cardSilver {{ background-color: {CARD_SILVER_BG}; border: 1px solid {SILVER}; border-radius: 12px; }}
QFrame#cardBronze {{ background-color: {CARD_BRONZE_BG}; border: 1px solid {BRONZE}; border-radius: 12px; }}

QLabel#cardTitle {{ color: {TEXT_SECONDARY}; font-size: 12px; font-weight: 600; }}
QLabel#cardHero {{ color: {ACCENT}; font-size: 19px; font-weight: 800; }}
QLabel#cardName {{ color: {TEXT_PRIMARY}; font-size: 15px; font-weight: 600; }}
QLabel#cardScore {{ font-size: 34px; font-weight: 800; }}
QLabel#cardScoreGold {{ color: {GOLD}; }}
QLabel#cardScoreSilver {{ color: {SILVER}; }}
QLabel#cardScoreBronze {{ color: {BRONZE}; }}
QLabel#cardStat {{ color: {TEXT_SECONDARY}; font-size: 12px; }}
QLabel#cardStatValue {{ color: {TEXT_PRIMARY}; font-size: 12px; font-weight: 600; }}

QLabel#medalGold {{ color: {GOLD}; }}
QLabel#medalSilver {{ color: {SILVER}; }}
QLabel#medalBronze {{ color: {BRONZE}; }}

QLabel#teamBadgeRadiant {{
    background: {_rgba(RADIANT, 28)};
    color: {RADIANT};
    border: 1px solid {_rgba(RADIANT, 70)};
    border-radius: 5px;
    padding: 2px 9px;
    font-size: 11px;
    font-weight: 700;
}}
QLabel#teamBadgeDire {{
    background: {_rgba(DIRE, 28)};
    color: {DIRE};
    border: 1px solid {_rgba(DIRE, 70)};
    border-radius: 5px;
    padding: 2px 9px;
    font-size: 11px;
    font-weight: 700;
}}

QLabel#matchInfoCaption {{ color: {TEXT_MUTED}; font-size: 11px; font-weight: 600; }}
QLabel#matchBadge {{
    border-radius: 7px;
    padding: 5px 14px;
    font-weight: 700;
    font-size: 13px;
}}
QLabel#matchBadgeRadiant {{ background: {_rgba(RADIANT, 30)}; color: {RADIANT}; }}
QLabel#matchBadgeDire {{ background: {_rgba(DIRE, 30)}; color: {DIRE}; }}

QFrame#dropZone {{
    border: 2px dashed {BORDER_LIGHT};
    color: {TEXT_SECONDARY};
    font-size: 14px;
    background: transparent;
}}
QFrame#dropZoneActive {{
    border-color: {ACCENT};
    color: {ACCENT};
    background: {_rgba(ACCENT, 12)};
}}

QPushButton {{
    background-color: {PANEL_ALT};
    border: 1px solid {BORDER_LIGHT};
    border-radius: 8px;
    padding: 8px 16px;
    font-weight: 600;
    color: {TEXT_PRIMARY};
}}
QPushButton:hover {{ background-color: #232a35; border-color: {BORDER_LIGHT}; }}
QPushButton:pressed {{ background-color: {BG_ELEVATED}; }}
QPushButton:disabled {{ color: {TEXT_MUTED}; background-color: {BG_ELEVATED}; border-color: {BORDER}; }}
QPushButton#accent {{ background-color: {ACCENT}; color: {ACCENT_TEXT}; border: none; }}
QPushButton#accent:hover {{ background-color: {ACCENT_HOVER}; }}
QPushButton#accent:disabled {{ background-color: #5c5426; color: #8a8260; }}
QPushButton#ghost {{
    background: transparent;
    border: 1px solid {BORDER_LIGHT};
    color: {TEXT_SECONDARY};
}}
QPushButton#ghost:hover {{ color: {TEXT_PRIMARY}; border-color: {ACCENT}; }}
QPushButton#ghost:disabled {{ color: {TEXT_MUTED}; border-color: {BORDER}; }}

QFrame#weightSection {{
    background-color: {PANEL};
    border: 1px solid {BORDER};
    border-radius: 9px;
}}
QLabel#weightSectionTitle {{ color: {TEXT_SECONDARY}; font-size: 12px; font-weight: 700; }}
QLabel#weightHint {{ color: {TEXT_MUTED}; font-size: 11px; }}
QLabel#formulaText {{ color: {TEXT_SECONDARY}; font-size: 11px; }}
QLabel#previewScore {{ color: {ACCENT}; font-weight: 700; }}

QPushButton#chip {{
    background-color: {PANEL};
    border: 1px solid {BORDER_LIGHT};
    border-radius: 12px;
    padding: 3px 10px;
    font-size: 12px;
    font-weight: 600;
    color: {TEXT_SECONDARY};
}}
QPushButton#chip:hover {{ background-color: {_rgba(ACCENT, 18)}; border-color: {ACCENT}; color: {TEXT_PRIMARY}; }}

QComboBox {{
    background-color: {BG_ELEVATED};
    border: 1px solid {BORDER_LIGHT};
    border-radius: 7px;
    padding: 5px 10px;
    color: {TEXT_PRIMARY};
}}
QComboBox:focus {{ border-color: {ACCENT}; }}
QComboBox::drop-down {{ border: none; width: 22px; }}
QComboBox QAbstractItemView {{
    background-color: {PANEL};
    border: 1px solid {BORDER_LIGHT};
    selection-background-color: {PANEL_ALT};
    color: {TEXT_PRIMARY};
}}

QListWidget {{
    background-color: {PANEL};
    border: 1px solid {BORDER};
    border-radius: 9px;
    padding: 4px;
}}
QListWidget::item {{ padding: 8px 10px; border-radius: 6px; color: {TEXT_SECONDARY}; }}
QListWidget::item:selected {{ background-color: {_rgba(ACCENT, 18)}; color: {TEXT_PRIMARY}; }}
QListWidget::item:hover {{ background-color: {PANEL_ALT}; }}

QPlainTextEdit {{
    background-color: {BG_ELEVATED};
    border: 1px solid {BORDER_LIGHT};
    border-radius: 8px;
    padding: 8px;
    color: {TEXT_PRIMARY};
    selection-background-color: {BLUE};
    font-family: "Consolas", "Courier New", monospace;
    font-size: 13px;
}}
QPlainTextEdit:focus {{ border-color: {ACCENT}; }}

QDoubleSpinBox {{
    background-color: {BG_ELEVATED};
    border: 1px solid {BORDER_LIGHT};
    border-radius: 6px;
    padding: 4px 6px;
    color: {TEXT_PRIMARY};
    selection-background-color: {BLUE};
}}
QDoubleSpinBox:focus {{ border-color: {ACCENT}; }}
QDoubleSpinBox::up-button, QDoubleSpinBox::down-button {{
    background: transparent;
    border: none;
    width: 16px;
}}
QDoubleSpinBox::up-arrow {{
    width: 0; height: 0;
    border-left: 4px solid transparent;
    border-right: 4px solid transparent;
    border-bottom: 5px solid {TEXT_SECONDARY};
}}
QDoubleSpinBox::down-arrow {{
    width: 0; height: 0;
    border-left: 4px solid transparent;
    border-right: 4px solid transparent;
    border-top: 5px solid {TEXT_SECONDARY};
}}

QTableWidget {{
    background-color: {PANEL};
    alternate-background-color: {PANEL_ALT};
    gridline-color: {BORDER};
    border: 1px solid {BORDER};
    border-radius: 10px;
    selection-background-color: {_rgba(BLUE, 40)};
    selection-color: {TEXT_PRIMARY};
}}
QHeaderView::section {{
    background-color: {BG_ELEVATED};
    color: {TEXT_SECONDARY};
    border: none;
    border-bottom: 1px solid {BORDER_LIGHT};
    padding: 7px 8px;
    font-weight: 700;
}}
QTableWidget::item {{ padding: 4px 8px; border: none; }}

QScrollBar:vertical {{
    background: transparent;
    width: 10px;
    margin: 2px;
}}
QScrollBar::handle:vertical {{
    background: {BORDER_LIGHT};
    border-radius: 5px;
    min-height: 24px;
}}
QScrollBar::handle:vertical:hover {{ background: {TEXT_MUTED}; }}
QScrollBar::add-line:vertical, QScrollBar::sub-line:vertical {{ height: 0; }}
QScrollBar::add-page:vertical, QScrollBar::sub-page:vertical {{ background: transparent; }}
QScrollBar:horizontal {{ background: transparent; height: 10px; margin: 2px; }}
QScrollBar::handle:horizontal {{ background: {BORDER_LIGHT}; border-radius: 5px; min-width: 24px; }}
QScrollBar::handle:horizontal:hover {{ background: {TEXT_MUTED}; }}
QScrollBar::add-line:horizontal, QScrollBar::sub-line:horizontal {{ width: 0; }}
QScrollBar::add-page:horizontal, QScrollBar::sub-page:horizontal {{ background: transparent; }}

QStatusBar {{ color: {TEXT_SECONDARY}; background-color: {BG_ELEVATED}; border-top: 1px solid {BORDER}; }}
QStatusBar::item {{ border: none; }}

QToolTip {{
    background-color: {PANEL_ALT};
    color: {TEXT_PRIMARY};
    border: 1px solid {BORDER_LIGHT};
    border-radius: 6px;
    padding: 6px 8px;
}}

QSplitter::handle {{ background: transparent; }}
QSplitter::handle:hover {{ background: {_rgba(ACCENT, 40)}; }}

QScrollArea {{ background: transparent; border: none; }}
QScrollArea > QWidget > QWidget {{ background: transparent; }}
"""


STYLE_SHEET = build_stylesheet()
