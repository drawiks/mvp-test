export const fmt = (v: number, digits = 0): string =>
  v.toLocaleString("ru-RU", { maximumFractionDigits: digits });

export const fmtScore = (v: number): string =>
  v.toLocaleString("ru-RU", { minimumFractionDigits: 2, maximumFractionDigits: 2 });

export const fmtPercent = (v: number): string =>
  v.toLocaleString("ru-RU", { maximumFractionDigits: 1 }) + "%";

/** Seconds to m:ss (replays usually under an hour). */
export const fmtTime = (sec: number): string => {
  const s = Math.max(0, Math.round(sec));
  const m = Math.floor(s / 60);
  const r = s % 60;
  return `${m}:${String(r).padStart(2, "0")}`;
};

/** Compact human number: 1.2M / 3.4k / plain. */
export const fmtNum = (v: number): string => {
  const abs = Math.abs(v);
  if (abs >= 1_000_000) return `${(v / 1_000_000).toFixed(2)}M`;
  if (abs >= 10_000) return `${(v / 1_000).toFixed(1)}k`;
  if (abs >= 1000) return `${(v / 1000).toFixed(2)}k`;
  return `${v}`;
};

import type { WeightMeta } from "./weights";

/** Build a human-readable formula preview line: "Счёт = 0.3·Убийства − 0.1·Смерти …" */
export const buildFormulaPreview = (weights: Record<string, number>, meta: WeightMeta[]): string => {
  const parts: string[] = [];
  for (const m of meta) {
    const w = weights[m.key];
    if (!w) continue;
    const sign = w > 0 ? (parts.length === 0 ? "" : " + ") : " − ";
    const abs = Math.abs(w);
    parts.push(`${sign}${abs !== 1 ? `${abs}` : ""}${m.label}`);
  }
  return parts.length === 0 ? "Счёт = 0" : `Счёт = ${parts.join("")}`;
};

/**
 * Per-stat contribution tooltip: "Урон героям: 12500 × 0.1 = 1250.0"
 */
export const statContribution = (
  statValue: number,
  weight: number | undefined,
  label: string
): string => {
  if (weight === undefined || weight === 0) return `${label}: ${statValue}`;
  return `${label}: ${statValue} × ${weight} = ${(statValue * weight).toFixed(3)}`;
};

/**
 * Breakdown tooltip for score: list of label → contribution.
 */
export const buildBreakdownTooltip = (
  stats: Record<string, number>,
  weights: Record<string, number>,
  meta: WeightMeta[]
): string => {
  const lines: string[] = [];
  for (const m of meta) {
    const val = stats[m.key];
    if (val === undefined) continue;
    const w = weights[m.key] ?? 0;
    if (w === 0) continue;
    const contrib = val * w;
    lines.push(`${m.label}: ${val} × ${w} = ${contrib > 0 ? "+" : ""}${contrib.toFixed(3)}`);
  }
  return lines.join("\n");
};

/** Build an expression string from a linear weight set. */
export const linearToExpression = (weights: Record<string, number>, meta: WeightMeta[]): string => {
  const parts: string[] = [];
  for (const m of meta) {
    const w = weights[m.key] ?? 0;
    if (w === 0) continue;
    parts.push(`${m.key}*${w}`);
  }
  return parts.join("+");
};