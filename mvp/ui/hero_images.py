from __future__ import annotations

import hashlib
import os
import re
import urllib.request
from pathlib import Path

from PyQt6.QtCore import QObject, QRunnable, QThreadPool, Qt, pyqtSignal
from PyQt6.QtGui import QColor, QFont, QLinearGradient, QPainter, QPixmap

from .resources import hero_image_names

_BASE_URL = "https://cdn.cloudflare.steamstatic.com/apps/dota2/images/dota_react/heroes/"
_VARIANTS = {"portrait": "", "icon": "icons/"}

_CACHE_ROOT = (
    Path(os.environ.get("XDG_CACHE_HOME") or (Path.home() / ".cache"))
    / "mvp-calculator"
    / "heroes"
)

_SLUG_RE = re.compile(r"[^A-Za-z0-9_]+")


class _Signals(QObject):
    ready = pyqtSignal(int, str)


class _FetchTask(QRunnable):
    def __init__(self, hero_id: int, variant: str, url: str, path: Path, signals: _Signals):
        super().__init__()
        self._hero_id = hero_id
        self._variant = variant
        self._url = url
        self._path = path
        self._signals = signals

    def run(self) -> None:
        try:
            request = urllib.request.Request(self._url, headers={"User-Agent": "Mozilla/5.0"})
            with urllib.request.urlopen(request, timeout=15) as response:
                data = response.read()
            self._path.parent.mkdir(parents=True, exist_ok=True)
            tmp = self._path.with_name(self._path.name + ".part")
            tmp.write_bytes(data)
            os.replace(tmp, self._path)
        except Exception:
            pass
        self._signals.ready.emit(self._hero_id, self._variant)


class HeroImages(QObject):
    loaded = pyqtSignal(int, str)

    def __init__(self, parent: QObject | None = None):
        super().__init__(parent)
        self._names = hero_image_names()
        self._pixmaps: dict[tuple, QPixmap] = {}
        self._in_flight: set[tuple[int, str]] = set()
        self._pool = QThreadPool(self)
        self._pool.setMaxThreadCount(4)
        self._signals = _Signals()
        self._signals.ready.connect(self._handle_ready)

    def request(self, hero_id: int, hero_name: str = "", variant: str = "portrait") -> None:
        name = self._slug(hero_id, hero_name)
        key = (hero_id, variant)
        if name is None or key in self._in_flight:
            return
        path = self._cache_path(hero_id, variant, name)
        if path.exists():
            self._signals.ready.emit(hero_id, variant)
            return
        url = _BASE_URL + _VARIANTS[variant] + name + ".png"
        self._in_flight.add(key)
        self._pool.start(_FetchTask(hero_id, variant, url, path, self._signals))

    def pixmap(self, hero_id: int, variant: str = "portrait", size: tuple[int, int] = (0, 0)) -> QPixmap:
        pm = self._pixmaps.get((hero_id, variant))
        if pm is None:
            name = self._slug(hero_id, "")
            if name is not None:
                path = self._cache_path(hero_id, variant, name)
                if path.exists():
                    loaded = QPixmap(str(path))
                    if not loaded.isNull():
                        pm = loaded
                        self._pixmaps[(hero_id, variant)] = pm
        if pm is None:
            return self._fallback(hero_id, size)
        width, height = size
        if width > 0 and height > 0:
            pm = pm.scaled(width, height, Qt.AspectRatioMode.KeepAspectRatio, Qt.TransformationMode.SmoothTransformation)
        return pm

    def has_image(self, hero_id: int, variant: str = "portrait") -> bool:
        if (hero_id, variant) in self._pixmaps:
            return True
        name = self._slug(hero_id, "")
        return name is not None and self._cache_path(hero_id, variant, name).exists()

    def cropped(self, hero_id: int, target_size: tuple[int, int]) -> QPixmap:
        pm = self._pixmaps.get((hero_id, "portrait"))
        if pm is None:
            name = self._slug(hero_id, "")
            if name is not None:
                path = self._cache_path(hero_id, "portrait", name)
                if path.exists():
                    loaded = QPixmap(str(path))
                    if not loaded.isNull():
                        pm = loaded
                        self._pixmaps[(hero_id, "portrait")] = pm
        if pm is None:
            return self._fallback(hero_id, target_size)
        tw, th = target_size
        src_ar = pm.width() / pm.height()
        target_ar = tw / th if th else 1.0
        if src_ar > target_ar:
            crop_w = int(pm.height() * target_ar)
            x0 = max(0, (pm.width() - crop_w) // 2)
            cropped = pm.copy(x0, 0, crop_w, pm.height())
        else:
            crop_h = int(pm.width() / target_ar)
            y0 = max(0, (pm.height() - crop_h) // 2)
            cropped = pm.copy(0, y0, pm.width(), crop_h)
        return cropped.scaled(
            max(1, tw),
            max(1, th),
            Qt.AspectRatioMode.IgnoreAspectRatio,
            Qt.TransformationMode.SmoothTransformation,
        )

    def _handle_ready(self, hero_id: int, variant: str) -> None:
        self._in_flight.discard((hero_id, variant))
        self.loaded.emit(hero_id, variant)

    def _slug(self, hero_id: int, hero_name: str = "") -> str | None:
        name = self._names.get(hero_id)
        if not name and hero_name:
            name = _SLUG_RE.sub("_", hero_name.strip()).lower().strip("_")
        return name or None

    @staticmethod
    def _cache_path(hero_id: int, variant: str, name: str) -> Path:
        return _CACHE_ROOT / variant / f"{name}.png"

    def _fallback(self, hero_id: int, size: tuple[int, int]) -> QPixmap:
        width = max(size[0] or 64, 1)
        height = max(size[1] or 64, 1)
        key = ("fb", hero_id, width, height)
        pm = self._pixmaps.get(key)
        if pm is not None:
            return pm
        seed = int(hashlib.md5(str(hero_id).encode()).hexdigest()[:6], 16)
        hue = seed % 360
        c1 = QColor.fromHsv(hue, 110, 58)
        c2 = QColor.fromHsv((hue + 40) % 360, 170, 32)
        pm = QPixmap(width, height)
        pm.fill(Qt.GlobalColor.transparent)
        painter = QPainter(pm)
        painter.setRenderHint(QPainter.RenderHint.Antialiasing)
        grad = QLinearGradient(0, 0, width, height)
        grad.setColorAt(0, c1)
        grad.setColorAt(1, c2)
        painter.setBrush(grad)
        painter.setPen(Qt.PenStyle.NoPen)
        painter.drawRoundedRect(0, 0, width, height, 6, 6)
        initial = (self._slug(hero_id, "") or "?").lstrip("_")[:1].upper()
        painter.setPen(QColor(255, 255, 255, 210))
        font = QFont()
        font.setWeight(QFont.Weight.DemiBold)
        font.setPixelSize(int(width * 0.62))
        painter.setFont(font)
        painter.drawText(pm.rect(), Qt.AlignmentFlag.AlignCenter, initial)
        painter.end()
        self._pixmaps[key] = pm
        return pm


hero_images = HeroImages()
