from __future__ import annotations

from PyQt6.QtCore import QRect, Qt
from PyQt6.QtGui import QColor, QIcon, QPainter
from PyQt6.QtWidgets import (
    QAbstractItemView,
    QHeaderView,
    QStyleOptionViewItem,
    QStyledItemDelegate,
    QTableWidget,
    QTableWidgetItem,
)

from ..model import Player, Result
from ..mvp import compute_score, score_breakdown, select_mvps, Weights
from . import theme
from .hero_images import hero_images

_HEADERS = ["#", "Герой", "Игрок", "Ур.", "K", "D", "A", "LH", "GPM", "XPM", "NW", "Stun", "Heal", "T.Dmg", "Camps", "Runes", "FB", "Счёт"]

_MEDALS = {"winner_top1": ("1", theme.GOLD), "winner_top2": ("2", theme.SILVER), "loser_top1": ("3", theme.BRONZE)}

_TEAM_COLOR = {"radiant": QColor(theme.RADIANT), "dire": QColor(theme.DIRE)}
_ROW_TINT = {"radiant": QColor("#152419"), "dire": QColor("#261817")}
_HEADER_BG = {"radiant": QColor("#12301f"), "dire": QColor("#311a14")}

_HIGHLIGHT_BG = {
    "winner_top1": QColor("#38300e"),
    "winner_top2": QColor("#2a2e36"),
    "loser_top1": QColor("#33230f"),
}

_SCORE_MAX_ROLE = Qt.ItemDataRole.UserRole

_BREAKDOWN_LABELS = {
    "kills": "Kills",
    "deaths": "Deaths (база - N×w)",
    "assists": "Assists",
    "last_hits": "LastHits",
    "gpm": "GPM",
    "xpm": "XPM",
    "stun": "StunDuration",
    "healing": "Healing",
    "tower_damage": "TowerDamage",
    "camps": "CampsStacked",
    "runes": "RunePickups",
    "first_blood": "FirstBlood",
}

_PLAYER_ATTRS = {
    "stun": "stun_duration",
    "healing": "healing",
    "tower_damage": "tower_damage",
    "camps": "camps_stacked",
    "runes": "rune_pickups",
    "first_blood": "first_blood",
}

_FORMULA_COLS = {11: "stun", 12: "healing", 13: "tower_damage", 14: "camps", 15: "runes", 16: "first_blood"}

_FB_COL = 16
_SCORE_COL = 17


def fmt_num(value: float) -> str:
    if value >= 1_000_000:
        return f"{value / 1_000_000:.2f}M"
    if value >= 10_000:
        return f"{value / 1000:.1f}k"
    if value >= 1000:
        return f"{value / 1000:.2f}k"
    return f"{value:g}"


class ScoreDelegate(QStyledItemDelegate):
    def paint(self, painter: QPainter, option: QStyleOptionViewItem, index) -> None:
        painter.save()
        painter.setRenderHint(QPainter.RenderHint.Antialiasing)
        rect = option.rect
        background = index.data(Qt.ItemDataRole.BackgroundRole)
        if background is not None:
            painter.fillRect(rect, background)

        value = float(index.data(Qt.ItemDataRole.DisplayRole) or 0.0)
        max_value = float(index.data(_SCORE_MAX_ROLE) or 0.0)
        frac = value / max_value if max_value > 0 else 0.0

        bar_area = rect.adjusted(3, 0, -1, 0)
        bar_height = 8
        bar = QRect(
            bar_area.left(),
            rect.top() + (rect.height() - bar_height) // 2,
            bar_area.width() * 55 // 100,
            bar_height,
        )
        painter.setPen(Qt.PenStyle.NoPen)
        painter.setBrush(QColor(0, 0, 0, 70))
        painter.drawRoundedRect(bar, 4, 4)
        fill = int(bar.width() * max(0.0, min(frac, 1.0)))
        if fill > 0:
            painter.setBrush(QColor(theme.ACCENT))
            painter.drawRoundedRect(QRect(bar.left(), bar.top(), fill, bar.height()), 4, 4)

        painter.setPen(QColor(theme.TEXT_PRIMARY))
        font = option.font
        font.setBold(True)
        painter.setFont(font)
        text_rect = QRect(bar.right() + 4, rect.top(), rect.width() - (bar.right() - rect.left()) - 4, rect.height())
        painter.drawText(text_rect, Qt.AlignmentFlag.AlignLeft | Qt.AlignmentFlag.AlignVCenter, f"{value:.2f}")
        painter.restore()


class StatsTable(QTableWidget):
    def __init__(self, parent=None):
        super().__init__(parent)
        self.setColumnCount(len(_HEADERS))
        self.setHorizontalHeaderLabels(_HEADERS)
        self.setAlternatingRowColors(False)
        self.setSelectionBehavior(QAbstractItemView.SelectionBehavior.SelectRows)
        self.setSelectionMode(QAbstractItemView.SelectionMode.SingleSelection)
        self.setEditTriggers(QAbstractItemView.EditTrigger.NoEditTriggers)
        self.setSortingEnabled(False)
        self.setHorizontalScrollMode(QAbstractItemView.ScrollMode.ScrollPerPixel)
        self.verticalHeader().setVisible(False)
        self.verticalHeader().setDefaultSectionSize(38)
        self.horizontalHeader().setMinimumSectionSize(46)
        self.horizontalHeader().setSectionResizeMode(QHeaderView.ResizeMode.ResizeToContents)
        self.horizontalHeader().setSectionResizeMode(1, QHeaderView.ResizeMode.Fixed)
        self.horizontalHeader().resizeSection(1, 170)
        self.horizontalHeader().setSectionResizeMode(2, QHeaderView.ResizeMode.Fixed)
        self.horizontalHeader().resizeSection(2, 200)
        self.horizontalHeader().setSectionResizeMode(_SCORE_COL, QHeaderView.ResizeMode.Fixed)
        self.horizontalHeader().resizeSection(_SCORE_COL, 100)
        self.setItemDelegateForColumn(_SCORE_COL, ScoreDelegate(self))

        hero_images.loaded.connect(self._on_hero_loaded)
        self._rows_heroes: dict[int, int] = {}

    def set_result(self, result: Result, weights: Weights) -> None:
        mvps = select_mvps(result, weights)
        highlight_ids = {id(p): key for key, p in mvps.items() if p is not None}
        radiant = sorted(
            (p for p in result.players if p.team == "radiant"),
            key=lambda p: compute_score(p, weights),
            reverse=True,
        )
        dire = sorted(
            (p for p in result.players if p.team == "dire"),
            key=lambda p: compute_score(p, weights),
            reverse=True,
        )
        max_score = max((compute_score(p, weights) for p in result.players), default=0.0)
        if max_score <= 0:
            max_score = 1.0

        self._rows_heroes = {}
        self.setRowCount(0)
        row = 0
        row = self._append_team(result, "radiant", radiant, weights, highlight_ids, max_score, row)
        row = self._append_team(result, "dire", dire, weights, highlight_ids, max_score, row)
        self.resizeRowsToContents()

    def _append_team(
        self,
        result: Result,
        team: str,
        players: list[Player],
        weights: Weights,
        highlight_ids: dict[int, str],
        max_score: float,
        row: int,
    ) -> int:
        header_row = row
        row += 1
        kills = sum(p.kills for p in players)
        header_item = QTableWidgetItem(f"  {team.upper()}   ·   {kills} kill(ов)")
        header_item.setBackground(_HEADER_BG[team])
        header_item.setForeground(_TEAM_COLOR[team])
        font = header_item.font()
        font.setBold(True)
        font.setPixelSize(12)
        header_item.setFont(font)
        self.insertRow(header_row)
        self.setItem(header_row, 0, header_item)
        self.setSpan(header_row, 0, 1, self.columnCount())

        for player in players:
            self.insertRow(row)
            self._fill_row(row, team, player, weights, highlight_ids, max_score)
            self._rows_heroes[row] = player.hero_id
            row += 1
        return row

    def _fill_row(
        self,
        row: int,
        team: str,
        player: Player,
        weights: Weights,
        highlight_ids: dict[int, str],
        max_score: float,
    ) -> None:
        score = compute_score(player, weights)
        key = highlight_ids.get(id(player))
        bg = _HIGHLIGHT_BG[key] if key else _ROW_TINT[team]

        icon = QIcon(hero_images.pixmap(player.hero_id, "icon", (24, 24)))
        hero_images.request(player.hero_id, player.hero, "icon")

        medal = _MEDALS.get(key)
        values = [
            medal[0] if medal else str(player.player_id),
            player.hero or "-",
            player.name or "-",
            str(player.level),
            str(player.kills),
            str(player.deaths),
            str(player.assists),
            fmt_num(player.last_hits),
            str(player.gpm),
            str(player.xpm),
            fmt_num(player.networth),
            f"{player.stun_duration:.1f}",
            fmt_num(player.healing),
            fmt_num(player.tower_damage),
            str(player.camps_stacked),
            str(player.rune_pickups),
            "Да" if player.first_blood else "-",
            f"{score:.2f}",
        ]
        align = [Qt.AlignmentFlag.AlignCenter] * len(values)
        align[1] = Qt.AlignmentFlag.AlignLeft | Qt.AlignmentFlag.AlignVCenter
        align[2] = Qt.AlignmentFlag.AlignLeft | Qt.AlignmentFlag.AlignVCenter

        for col, text in enumerate(values):
            item = QTableWidgetItem(text)
            item.setBackground(bg)
            item.setTextAlignment(align[col])
            if col == 1:
                item.setIcon(icon)
                item.setToolTip(player.hero)
            elif col == 2:
                item.setToolTip(player.name)
            if col == 0 and medal:
                item.setForeground(QColor(medal[1]))
                medal_font = item.font()
                medal_font.setBold(True)
                item.setFont(medal_font)
            elif col == 5:
                item.setForeground(QColor(theme.DEATH))
            elif col == 4:
                item.setForeground(QColor(theme.KILL))
            elif col == _FB_COL and player.first_blood:
                item.setForeground(QColor(theme.GOLD))
            elif col == _SCORE_COL:
                item.setData(_SCORE_MAX_ROLE, max_score)
                item.setToolTip(self._breakdown_tooltip(player, weights, score))
            if col in _FORMULA_COLS:
                item.setToolTip(self._stat_tooltip(player, weights, _FORMULA_COLS[col]))
            font = item.font()
            font.setPixelSize(12)
            item.setFont(font)
            self.setItem(row, col, item)

    @staticmethod
    def _stat_tooltip(player: Player, weights: Weights, key: str) -> str:
        label = _BREAKDOWN_LABELS[key]
        value = getattr(player, _PLAYER_ATTRS[key])
        w = getattr(weights, key)
        return f"{label}: {value:g} × {w:g} = {value * w:+.3f}"

    @staticmethod
    def _breakdown_tooltip(player: Player, weights: Weights, score: float) -> str:
        breakdown = score_breakdown(player, weights)
        lines = [f"{player.hero or '-'} · {player.name or '-'}", f"Итоговый счёт: {score:.2f}", ""]
        for key, label in _BREAKDOWN_LABELS.items():
            lines.append(f"  {label:<22} {breakdown[key]:+10.3f}")
        return "\n".join(lines)

    def _on_hero_loaded(self, hero_id: int, variant: str) -> None:
        if variant != "icon":
            return
        for row, hid in self._rows_heroes.items():
            if hid == hero_id:
                item = self.item(row, 1)
                if item is not None:
                    item.setIcon(QIcon(hero_images.pixmap(hero_id, "icon", (24, 24))))
