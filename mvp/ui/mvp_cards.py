from __future__ import annotations

from PyQt6.QtCore import Qt
from PyQt6.QtGui import QColor, QLinearGradient, QPainter
from PyQt6.QtWidgets import QFrame, QGridLayout, QHBoxLayout, QLabel, QVBoxLayout, QWidget

from ..model import Player
from ..mvp import compute_score, score_breakdown, Weights
from . import theme
from .hero_images import hero_images
from .icons import medal_pixmap

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

_CARD_BG = {
    "cardGold": theme.CARD_GOLD_BG,
    "cardSilver": theme.CARD_SILVER_BG,
    "cardBronze": theme.CARD_BRONZE_BG,
}

_MEDAL_COLOR = {
    "cardGold": theme.GOLD,
    "cardSilver": theme.SILVER,
    "cardBronze": theme.BRONZE,
}


def _team_label(team: str) -> str:
    return _TEAM_LABEL.get(team, team.capitalize())


class ScoreBar(QWidget):
    def __init__(self, color: str, parent=None):
        super().__init__(parent)
        self._color = QColor(color)
        self._frac = 0.0
        self.setFixedHeight(10)
        self.setMinimumWidth(60)

    def set_value(self, value: float, max_value: float) -> None:
        self._frac = value / max_value if max_value > 0 else 0.0
        self.update()

    def paintEvent(self, event) -> None:
        painter = QPainter(self)
        painter.setRenderHint(QPainter.RenderHint.Antialiasing)
        rect = self.rect().adjusted(0, 1, 0, -1)
        painter.setPen(Qt.PenStyle.NoPen)
        painter.setBrush(QColor(0, 0, 0, 80))
        painter.drawRoundedRect(rect, 4, 4)
        width = int(rect.width() * max(0.0, min(self._frac, 1.0)))
        if width > 0:
            fill = rect.adjusted(0, 0, -(rect.width() - width), 0)
            painter.setBrush(self._color)
            painter.drawRoundedRect(fill, 4, 4)


class MvpCard(QFrame):
    def __init__(self, title: str, place: str, style_id: str, min_height: int, parent=None):
        super().__init__(parent)
        self._hero_id = 0
        self.setObjectName(style_id)
        self.setMinimumHeight(min_height)

        layout = QVBoxLayout(self)
        layout.setContentsMargins(16, 12, 16, 12)
        layout.setSpacing(3)

        head = QHBoxLayout()
        self._medal = QLabel()
        self._medal.setPixmap(medal_pixmap(place, (26, 26)))
        self._title = QLabel(title)
        self._title.setObjectName("cardTitle")
        head.addWidget(self._medal)
        head.addWidget(self._title, 1)
        layout.addLayout(head)

        self._hero = QLabel("-")
        self._hero.setObjectName("cardHero")
        layout.addWidget(self._hero)

        self._team = QLabel("")
        self._team.setVisible(False)
        layout.addWidget(self._team)

        self._name = QLabel("-")
        self._name.setObjectName("cardName")
        layout.addWidget(self._name)

        self._score = QLabel("-")
        self._score.setObjectName(f"cardScore{style_id[4:]}")
        layout.addWidget(self._score)

        self._bar = ScoreBar(_MEDAL_COLOR[style_id])
        layout.addWidget(self._bar)

        grid = QGridLayout()
        grid.setHorizontalSpacing(14)
        self._stats: dict[str, QLabel] = {}
        for row, (key, caption) in enumerate(
            [("kda", "K/D/A"), ("level", "Уровень"), ("gpmxpm", "GPM / XPM")]
        ):
            cap = QLabel(caption)
            cap.setObjectName("cardStat")
            val = QLabel("-")
            val.setObjectName("cardStatValue")
            grid.addWidget(cap, row, 0)
            grid.addWidget(val, row, 1)
            self._stats[key] = val
        layout.addLayout(grid)
        layout.addStretch(1)

        hero_images.loaded.connect(self._on_hero_loaded)
        self._set_empty()

    def paintEvent(self, event) -> None:
        super().paintEvent(event)
        if not self._hero_id:
            return
        band_width = max(120, self.width() * 42 // 100)
        pm = hero_images.cropped(self._hero_id, (band_width, self.height()))
        painter = QPainter(self)
        painter.setRenderHint(QPainter.RenderHint.SmoothPixmapTransform)
        painter.setOpacity(0.85)
        x = self.width() - pm.width() - 6
        y = max(4, (self.height() - pm.height()) // 2)
        painter.drawPixmap(x, y, pm)
        painter.setOpacity(1.0)

        bg = QColor(_CARD_BG.get(self.objectName(), theme.PANEL))
        span = max(1, self.width())
        grad = QLinearGradient(0, 0, span, 0)
        grad.setColorAt(0.0, bg)
        grad.setColorAt(0.55, bg)
        end = QColor(bg)
        end.setAlpha(0)
        grad.setColorAt(0.78, end)
        grad.setColorAt(1.0, end)
        painter.setBrush(grad)
        painter.setPen(Qt.PenStyle.NoPen)
        painter.drawRect(self.rect())
        painter.end()

    def _on_hero_loaded(self, hero_id: int, variant: str) -> None:
        if hero_id == self._hero_id and variant == "portrait":
            self.update()

    def _set_empty(self) -> None:
        self._hero_id = 0
        self.setToolTip("Загрузите .dem реплей, чтобы рассчитать MVP")
        self._hero.setText("-")
        self._name.setText("Ожидание реплея")
        self._team.setText("")
        self._team.setVisible(False)
        self._score.setText("-")
        self._bar.set_value(0.0, 1.0)
        for val in self._stats.values():
            val.setText("-")

    def show_player(self, player: Player | None, weights: Weights, max_score: float) -> None:
        if player is None:
            self._set_empty()
            return
        score = compute_score(player, weights)
        self._hero_id = player.hero_id
        hero_images.request(player.hero_id, player.hero, "portrait")
        self._hero.setText(player.hero or "-")
        self._name.setText(player.name or "-")
        team = _team_label(player.team)
        self._team.setText(team)
        self._team.setObjectName("teamBadgeRadiant" if player.team == "radiant" else "teamBadgeDire")
        self._team.setVisible(True)
        self._score.setText(f"{score:.2f}")
        self._bar.set_value(score, max_score)
        self._stats["kda"].setText(f"{player.kills} / {player.deaths} / {player.assists}")
        self._stats["level"].setText(str(player.level))
        self._stats["gpmxpm"].setText(f"{player.gpm} / {player.xpm}")

        breakdown = score_breakdown(player, weights)
        lines = [f"Итоговый счёт: {score:.2f}", ""]
        for key, label in _BREAKDOWN_LABELS.items():
            lines.append(f"  {label:<22} {breakdown[key]:+10.3f}")
        self.setToolTip("\n".join(lines))
        self.update()


class MvpCardsRow(QWidget):
    def __init__(self, parent=None):
        super().__init__(parent)
        layout = QHBoxLayout(self)
        layout.setContentsMargins(0, 0, 0, 0)
        layout.setSpacing(12)
        self._winner_card = MvpCard("MVP - лучший в победившей", "gold", "cardGold", 205)
        self._winner2_card = MvpCard("2-й лучший в победившей", "silver", "cardSilver", 180)
        self._loser_card = MvpCard("Лучший в проигравшей", "bronze", "cardBronze", 180)
        layout.addWidget(self._winner_card, 1)
        layout.addWidget(self._winner2_card, 1)
        layout.addWidget(self._loser_card, 1)

    def show_mvps(self, mvps: dict[str, Player | None], weights: Weights) -> None:
        players = [mvps.get("winner_top1"), mvps.get("winner_top2"), mvps.get("loser_top1")]
        valid = [compute_score(p, weights) for p in players if p is not None]
        max_score = max(valid) if valid else 1.0
        self._winner_card.show_player(mvps.get("winner_top1"), weights, max_score)
        self._winner2_card.show_player(mvps.get("winner_top2"), weights, max_score)
        self._loser_card.show_player(mvps.get("loser_top1"), weights, max_score)

    def clear(self) -> None:
        for card in (self._winner_card, self._winner2_card, self._loser_card):
            card._set_empty()
