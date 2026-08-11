from __future__ import annotations

from PyQt6.QtCore import Qt
from PyQt6.QtGui import QColor
from PyQt6.QtWidgets import QAbstractItemView, QHeaderView, QTableWidget, QTableWidgetItem

from ..model import Player, Result
from ..mvp import compute_score, ranked_players, select_mvps, Weights

_HEADERS = [
    "#", "Команда", "Герой", "Игрок", "Счёт",
    "K/D/A", "Ур.", "LH", "GPM", "XPM",
    "Stun", "Heal", "T.Dmg", "Camps", "Runes", "FB",
]

_TEAM_COLOR = {"radiant": QColor("#7bd88f"), "dire": QColor("#f09090")}
_HIGHLIGHT = {
    "winner_top1": QColor("#4a3d12"),
    "winner_top2": QColor("#33373f"),
    "loser_top1": QColor("#3a2a18"),
}


class StatsTable(QTableWidget):
    def __init__(self, parent=None):
        super().__init__(parent)
        self.setColumnCount(len(_HEADERS))
        self.setHorizontalHeaderLabels(_HEADERS)
        self.setAlternatingRowColors(True)
        self.setSelectionBehavior(QAbstractItemView.SelectionBehavior.SelectRows)
        self.setSelectionMode(QAbstractItemView.SelectionMode.SingleSelection)
        self.setEditTriggers(QAbstractItemView.EditTrigger.NoEditTriggers)
        self.setSortingEnabled(False)
        self.verticalHeader().setVisible(False)
        header = self.horizontalHeader()
        header.setSectionResizeMode(QHeaderView.ResizeMode.ResizeToContents)
        header.setSectionResizeMode(3, QHeaderView.ResizeMode.Stretch)
        header.setSectionResizeMode(2, QHeaderView.ResizeMode.Stretch)

    def set_result(self, result: Result, weights: Weights) -> None:
        ranked = ranked_players(result, weights)
        mvps = select_mvps(result, weights)
        highlight_ids = {
            id(p): key for key, p in mvps.items() if p is not None
        }
        self.setRowCount(len(ranked))
        for row, player in enumerate(ranked):
            bg = None
            key = highlight_ids.get(id(player))
            if key:
                bg = _HIGHLIGHT[key]
            values = self._row_values(player, weights)
            for col, value in enumerate(values):
                item = QTableWidgetItem(value)
                if bg is not None:
                    item.setBackground(bg)
                item.setTextAlignment(Qt.AlignmentFlag.AlignCenter)
                if col == 2:
                    item.setTextAlignment(Qt.AlignmentFlag.AlignLeft)
                if col == 1:
                    item.setForeground(_TEAM_COLOR.get(player.team, QColor("#cccccc")))
                if col == 4:
                    font = item.font()
                    font.setBold(True)
                    item.setFont(font)
                self.setItem(row, col, item)
        self.resizeRowsToContents()

    @staticmethod
    def _row_values(player: Player, weights: Weights) -> list[str]:
        fb = "Да" if player.first_blood else "Нет"
        return [
            str(player.player_id),
            ("Radiant" if player.team == "radiant" else "Dire"),
            player.hero or "—",
            player.name or "—",
            f"{compute_score(player, weights):.2f}",
            f"{player.kills} / {player.deaths} / {player.assists}",
            str(player.level),
            str(player.last_hits),
            str(player.gpm),
            str(player.xpm),
            f"{player.stun_duration:.1f}",
            f"{player.healing:.0f}",
            f"{player.tower_damage:.0f}",
            str(player.camps_stacked),
            str(player.rune_pickups),
            fb,
        ]
