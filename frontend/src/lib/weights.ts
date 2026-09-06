export interface WeightMeta {
  key: string;
  label: string;
  group: string;
  unit: string;
  /** True for tokens usable in expressions/variables but not as linear weights. */
  chipOnly?: boolean;
}

/** Coefficient editor schema, mirror of the Go runtime key list. */
export const WEIGHT_META: WeightMeta[] = [
  { key: "kills", label: "Убийства", group: "Бой", unit: "×" },
  { key: "deaths", label: "Смерти", group: "Бой", unit: "×" },
  { key: "deaths_base", label: "База (смерти)", group: "Бой", unit: "×" },
  { key: "assists", label: "Ассисты", group: "Бой", unit: "×" },
  { key: "first_blood", label: "Фёрст блад", group: "Бой", unit: "×" },
  { key: "hero_damage", label: "Урон героям", group: "Бой", unit: "÷" },
  { key: "stun_duration", label: "Станы (сек)", group: "Бой", unit: "×" },
  { key: "fear_duration", label: "Страх (сек)", group: "Бой", unit: "×" },
  { key: "roots_duration", label: "Руты (сек)", group: "Бой", unit: "×" },
  { key: "leash_duration", label: "Лиш (сек)", group: "Бой", unit: "×" },
  { key: "trap_duration", label: "Ловушки (сек)", group: "Бой", unit: "×" },
  { key: "taunt_duration", label: "Тонт (сек)", group: "Бой", unit: "×" },
  { key: "silence_duration", label: "Сайленс (сек)", group: "Бой", unit: "×" },
  { key: "break_duration", label: "Брейк (сек)", group: "Бой", unit: "×" },
  { key: "disarm_duration", label: "Дизарм (сек)", group: "Бой", unit: "×" },
  { key: "last_hits", label: "Ластхиты", group: "Фарм", unit: "÷" },
  { key: "gpm", label: "GPM", group: "Фарм", unit: "÷" },
  { key: "xpm", label: "XPM", group: "Фарм", unit: "÷" },
  { key: "networth", label: "Нетворс", group: "Фарм", unit: "÷" },
  { key: "camps_stacked", label: "Стаки", group: "Фарм", unit: "×" },
  { key: "rune_pickups", label: "Руны", group: "Фарм", unit: "×" },
  { key: "tower_damage", label: "Урон по вышкам", group: "Фарм", unit: "÷" },
  { key: "healing", label: "Хил", group: "Утилити", unit: "÷" },
  { key: "heal_duration", label: "Время хила (сек)", group: "Утилити", unit: "×" },
  { key: "heal_value", label: "Объём хила", group: "Утилити", unit: "÷" },
  { key: "save", label: "Сейвы", group: "Утилити", unit: "×" },
  { key: "purge", label: "Развеи", group: "Утилити", unit: "×" },
  { key: "shield_uptime", label: "Щиты", group: "Утилити", unit: "×" },
  { key: "buffs_duration", label: "Буффы (сек)", group: "Утилити", unit: "×" },
  { key: "gold_spent_wards", label: "Варды (голд)", group: "Утилити", unit: "÷" },
  { key: "gold_spent_smoke", label: "Смоуки (голд)", group: "Утилити", unit: "÷" },
  { key: "gold_spent_dust", label: "Дасты (голд)", group: "Утилити", unit: "÷" },
  { key: "gold_lost", label: "Потеряно голды", group: "Утилити", unit: "÷" },
  { key: "damage_taken", label: "Полученный урон", group: "Защита", unit: "÷" },
  { key: "wisdoms_captured", label: "Мудрость", group: "Фарм", unit: "×" },
  { key: "watchers_captured", label: "Смотровые", group: "Фарм", unit: "×" },
  { key: "lotuses_gathered", label: "Лотосы", group: "Фарм", unit: "×" },
  { key: "courier_kills", label: "Убийства курьеров", group: "Бой", unit: "×" },
  { key: "match_duration", label: "Длительность матча", group: "Фарм", unit: "×", chipOnly: true },
];

/** Every scoring token usable as a chip (excludes display-only networth). */
export const WEIGHT_TOKENS = WEIGHT_META.filter((m) => m.key !== "networth");

/** Division-ish stats scale by dividing; display helper only. */
export function unitHint(meta: WeightMeta, weight: number): string {
  return `${meta.unit === "÷" ? "1/" : ""}${weight}`;
}