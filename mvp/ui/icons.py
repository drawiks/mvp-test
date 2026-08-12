from __future__ import annotations

from collections import OrderedDict

from PyQt6.QtCore import QPointF, QRectF, Qt
from PyQt6.QtGui import QColor, QFont, QLinearGradient, QPainter, QPen, QPixmap

from . import theme

_MAX_ICON_CACHE = 50

_cache: OrderedDict[tuple, QPixmap] = OrderedDict()

_PLACE_COLORS = {
    "gold": (theme.GOLD, "#8a6d1c"),
    "silver": (theme.SILVER, "#6d7681"),
    "bronze": (theme.BRONZE, "#7a4c20"),
}


def _cache_get(key: tuple) -> QPixmap | None:
    pm = _cache.get(key)
    if pm is not None:
        _cache.move_to_end(key)
    return pm


def _cache_put(key: tuple, pm: QPixmap) -> None:
    if key in _cache:
        _cache.move_to_end(key)
    else:
        _cache[key] = pm
        if len(_cache) > _MAX_ICON_CACHE:
            _cache.popitem(last=False)


def medal_pixmap(place: str, size: tuple[int, int] = (26, 26)) -> QPixmap:
    key = ("medal", place, size)
    cached = _cache_get(key)
    if cached is not None:
        return cached
    width, height = size
    pm = QPixmap(width, height)
    pm.fill(Qt.GlobalColor.transparent)
    painter = QPainter(pm)
    painter.setRenderHint(QPainter.RenderHint.Antialiasing)

    main_color, edge_color = _PLACE_COLORS.get(place, _PLACE_COLORS["bronze"])
    cx = width / 2
    cy = height * 0.56
    radius = min(width, height) * 0.34

    painter.setPen(Qt.PenStyle.NoPen)
    painter.setBrush(QColor(main_color))
    painter.drawRoundedRect(QRectF(cx - radius * 0.3, height * 0.08, radius * 0.6, radius * 0.42), 2, 2)

    grad = QLinearGradient(cx, cy - radius, cx, cy + radius)
    grad.setColorAt(0.0, QColor(main_color).lighter(118))
    grad.setColorAt(1.0, QColor(main_color).darker(112))
    painter.setPen(QPen(QColor(edge_color), max(1.0, radius * 0.12)))
    painter.setBrush(grad)
    painter.drawEllipse(QPointF(cx, cy), radius, radius)

    number = {"gold": "1", "silver": "2", "bronze": "3"}.get(place, "?")
    painter.setPen(QColor("#1b1610"))
    font = QFont()
    font.setWeight(QFont.Weight.ExtraBold)
    font.setPixelSize(int(radius * 1.5))
    painter.setFont(font)
    painter.drawText(
        QRectF(cx - radius, cy - radius, radius * 2, radius * 2),
        Qt.AlignmentFlag.AlignCenter,
        number,
    )
    painter.end()
    _cache_put(key, pm)
    return pm


def folder_pixmap(size: tuple[int, int] = (32, 32)) -> QPixmap:
    key = ("folder", size)
    cached = _cache_get(key)
    if cached is not None:
        return cached
    width, height = size
    pm = QPixmap(width, height)
    pm.fill(Qt.GlobalColor.transparent)
    painter = QPainter(pm)
    painter.setRenderHint(QPainter.RenderHint.Antialiasing)
    painter.setPen(Qt.PenStyle.NoPen)

    w = width
    h = height

    painter.setBrush(QColor(0, 0, 0, 70))
    painter.drawRoundedRect(QRectF(w * 0.08, h * 0.22, w * 0.88, h * 0.62), h * 0.08, h * 0.08)

    grad = QLinearGradient(0, 0, 0, h)
    grad.setColorAt(0.0, QColor(theme.ACCENT_HOVER))
    grad.setColorAt(1.0, QColor(theme.ACCENT))
    painter.setBrush(grad)

    painter.drawRoundedRect(QRectF(w * 0.06, h * 0.16, w * 0.36, h * 0.2), h * 0.07, h * 0.07)
    painter.drawRoundedRect(QRectF(w * 0.06, h * 0.3, w * 0.82, h * 0.56), h * 0.08, h * 0.08)

    painter.end()
    _cache_put(key, pm)
    return pm
