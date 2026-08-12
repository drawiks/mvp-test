from __future__ import annotations

import json
from pathlib import Path

from PyQt6.QtGui import QFontDatabase

ASSETS_DIR = Path(__file__).resolve().parents[2] / "assets"
FONTS_DIR = ASSETS_DIR / "fonts"
HEROES_JSON = ASSETS_DIR / "heroes.json"

FONT_FAMILY = "Inter"


def register_fonts() -> bool:
    loaded = False
    if FONTS_DIR.is_dir():
        for font_file in sorted(FONTS_DIR.glob("*.ttf")):
            font_id = QFontDatabase.addApplicationFont(str(font_file))
            if font_id >= 0:
                loaded = True
    return loaded


def hero_image_names() -> dict[int, str]:
    try:
        data = json.loads(HEROES_JSON.read_text(encoding="utf-8"))
    except (OSError, ValueError):
        return {}
    return {int(hid): name for hid, name in data.items()}
