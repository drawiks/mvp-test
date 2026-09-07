export interface WeightMeta {
  key: string;
  label: string;
  group: string;
  unit: string;
  chipOnly?: boolean;
}

export const WEIGHT_META: WeightMeta[] = [
  { key: "kills", label: "Убийства", group: "Бой", unit: "×" },
  { key: "deaths", label: "Смерти", group: "Бой", unit: "×" },
  { key: "assists", label: "Ассисты", group: "Бой", unit: "×" },
  { key: "first_blood", label: "Фёрст блад", group: "Бой", unit: "×" },
  { key: "hero_damage", label: "Урон героям", group: "Бой", unit: "÷" },
  { key: "damage_taken", label: "Полученный урон", group: "Бой", unit: "÷" },
  { key: "stun_duration", label: "Станы (сек)", group: "Бой", unit: "×" },
  { key: "courier_kills", label: "Курьеры убиты", group: "Бой", unit: "×" },
  { key: "fear_duration", label: "Страх (сек)", group: "Дебаффы", unit: "×" },
  { key: "roots_duration", label: "Руты (сек)", group: "Дебаффы", unit: "×" },
  { key: "leash_duration", label: "Лиш (сек)", group: "Дебаффы", unit: "×" },
  { key: "trap_duration", label: "Ловушки (сек)", group: "Дебаффы", unit: "×" },
  { key: "taunt_duration", label: "Тонт (сек)", group: "Дебаффы", unit: "×" },
  { key: "silence_duration", label: "Сайленс (сек)", group: "Дебаффы", unit: "×" },
  { key: "break_duration", label: "Брейк (сек)", group: "Дебаффы", unit: "×" },
  { key: "disarm_duration", label: "Дизарм (сек)", group: "Дебаффы", unit: "×" },
  { key: "save_duration", label: "Сейвы (сек)", group: "Баффы", unit: "×" },
  { key: "purge_duration", label: "Развеи (сек)", group: "Баффы", unit: "×" },
  { key: "shield_duration", label: "Щиты (сек)", group: "Баффы", unit: "×" },
  { key: "buff_duration", label: "Буффы (сек)", group: "Баффы", unit: "×" },
  { key: "buff_stats_duration", label: "Св-ва буффов (сек)", group: "Баффы", unit: "×" },
  { key: "invisibility_duration", label: "Инвиз (сек)", group: "Баффы", unit: "×" },
  { key: "buff_haste_duration", label: "Хаст (сек)", group: "Баффы", unit: "×" },
  { key: "healing", label: "Хил", group: "Баффы", unit: "÷" },
  { key: "heal_duration", label: "Время хила (сек)", group: "Баффы", unit: "×" },
  { key: "heal_value", label: "Объём хила", group: "Баффы", unit: "÷" },
  { key: "last_hits", label: "Ластхиты", group: "Фарм", unit: "÷" },
  { key: "gpm", label: "GPM", group: "Фарм", unit: "÷" },
  { key: "xpm", label: "XPM", group: "Фарм", unit: "÷" },
  { key: "tower_damage", label: "Урон по вышкам", group: "Фарм", unit: "÷" },
  { key: "networth", label: "Нетворс", group: "Фарм", unit: "÷", chipOnly: true },
  { key: "match_duration", label: "Длительность матча", group: "Фарм", unit: "×", chipOnly: true },
  { key: "gold_spent_wards", label: "Варды (голд)", group: "Саппорт", unit: "÷" },
  { key: "gold_spent_smoke", label: "Смоуки (голд)", group: "Саппорт", unit: "÷" },
  { key: "gold_spent_dust", label: "Дасты (голд)", group: "Саппорт", unit: "÷" },
  { key: "wisdoms_captured", label: "Мудрость", group: "Саппорт", unit: "×" },
  { key: "watchers_captured", label: "Смотровые", group: "Саппорт", unit: "×" },
  { key: "lotuses_gathered", label: "Лотосы", group: "Саппорт", unit: "×" },
  { key: "camps_stacked", label: "Стаки", group: "Утилити", unit: "×" },
  { key: "creeps_stacked", label: "Крипы застаканы", group: "Утилити", unit: "×" },
  { key: "rune_pickups", label: "Руны", group: "Утилити", unit: "×" },
  { key: "time_dead", label: "Время смерти (сек)", group: "Утилити", unit: "÷" },
  { key: "position", label: "Позиция (1–5)", group: "Роль", unit: "×", chipOnly: true },
];

export const WEIGHT_TOKEN_GROUPS = ["Бой", "Дебаффы", "Баффы", "Фарм", "Саппорт", "Утилити", "Роль"] as const;

export const WEIGHT_TOKENS = WEIGHT_META.filter((m) => m.key !== "networth");

export function unitHint(meta: WeightMeta, weight: number): string {
  return `${meta.unit === "÷" ? "1/" : ""}${weight}`;
}