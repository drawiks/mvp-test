from __future__ import annotations

from PyQt6.QtCore import Qt
from PyQt6.QtWidgets import QFrame, QGridLayout, QHBoxLayout, QLabel, QVBoxLayout, QWidget

from ..model import Player
from ..mvp import compute_score, score_breakdown, Weights

_TEAM_LABEL = {"radiant": "Radiant", "dire": "Dire"}
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


def _team_label(team: str) -> str:
    return _TEAM_LABEL.get(team, team.capitalize())


class MvpCard(QFrame):
    def __init__(self, title: str, medal: str, style_id: str, parent=None):
        super().__init__(parent)
        self.setObjectName(style_id)
        self.setMinimumHeight(190)

        layout = QVBoxLayout(self)
        layout.setContentsMargins(16, 14, 16, 14)
        layout.setSpacing(4)

        head = QHBoxLayout()
        self._medal = QLabel(medal)
        self._medal.setObjectName("cardMedal")
        self._title = QLabel(title)
        self._title.setObjectName("cardTitle")
        head.addWidget(self._medal)
        head.addWidget(self._title, 1)
        layout.addLayout(head)

        self._hero = QLabel("—")
        self._hero.setObjectName("cardHero")
        self._hero.setWordWrap(True)
        layout.addWidget(self._hero)

        self._team = QLabel("")
        self._team.setObjectName("cardStat")
        layout.addWidget(self._team)

        self._name = QLabel("—")
        self._name.setObjectName("cardName")
        self._name.setWordWrap(True)
        layout.addWidget(self._name)

        self._score = QLabel("–")
        self._score.setObjectName("cardScore")
        font = self._score.font()
        font.setBold(True)
        self._score.setFont(font)
        layout.addWidget(self._score, 0, Qt.AlignmentFlag.AlignLeft)

        grid = QGridLayout()
        grid.setHorizontalSpacing(14)
        self._stats: dict[str, QLabel] = {}
        for row, (key, caption) in enumerate(
            [("kda", "K/D/A"), ("level", "Уровень"), ("gpmxpm", "GPM / XPM")]
        ):
            cap = QLabel(caption)
            cap.setObjectName("cardStat")
            val = QLabel("—")
            val.setObjectName("cardStatValue")
            grid.addWidget(cap, row, 0)
            grid.addWidget(val, row, 1)
            self._stats[key] = val
        layout.addLayout(grid)
        layout.addStretch(1)

        self._set_empty()

    def _set_empty(self) -> None:
        self.setToolTip("Загрузите .dem реплей, чтобы рассчитать MVP")
        self._hero.setText("—")
        self._name.setText("Ожидание реплея")
        self._team.setText("")
        self._score.setText("–")
        for val in self._stats.values():
            val.setText("—")

    def show_player(self, player: Player | None, weights: Weights) -> None:
        if player is None:
            self._set_empty()
            return
        score = compute_score(player, weights)
        self._hero.setText(player.hero or "—")
        self._name.setText(player.name or "—")
        team = _team_label(player.team)
        self._team.setText(team)
        self._score.setText(f"{score:.2f}")
        self._stats["kda"].setText(f"{player.kills} / {player.deaths} / {player.assists}")
        self._stats["level"].setText(str(player.level))
        self._stats["gpmxpm"].setText(f"{player.gpm} / {player.xpm}")

        breakdown = score_breakdown(player, weights)
        lines = [f"Итоговый счёт: {score:.2f}", ""]
        for key, label in _BREAKDOWN_LABELS.items():
            lines.append(f"  {label:<22} {breakdown[key]:+10.3f}")
        self.setToolTip("\n".join(lines))


class MvpCardsRow(QWidget):
    def __init__(self, parent=None):
        super().__init__(parent)
        layout = QHBoxLayout(self)
        layout.setContentsMargins(0, 0, 0, 0)
        layout.setSpacing(12)
        self._winner_card = MvpCard("MVP — лучший в победившей", "🥇", "cardGold")
        self._winner2_card = MvpCard("2-й лучший в победившей", "🥈", "cardSilver")
        self._loser_card = MvpCard("Лучший в проигравшей", "🥉", "cardBronze")
        layout.addWidget(self._winner_card, 1)
        layout.addWidget(self._winner2_card, 1)
        layout.addWidget(self._loser_card, 1)

    def show_mvps(self, mvps: dict[str, Player | None], weights: Weights) -> None:
        self._winner_card.show_player(mvps.get("winner_top1"), weights)
        self._winner2_card.show_player(mvps.get("winner_top2"), weights)
        self._loser_card.show_player(mvps.get("loser_top1"), weights)

    def clear(self) -> None:
        for card in (self._winner_card, self._winner2_card, self._loser_card):
            card._set_empty()
