import { useState } from "react";
import { Input } from "@/components/ui/input";
import { Slider } from "@/components/ui/slider";
import type { WeightMeta } from "@/lib/weights";

/** Max absolute weight in the set, used to size all sliders in that set consistently. */
export function sliderRange(weights: Record<string, number>, editable: WeightMeta[]) {
  const maxAbs = Math.max(...editable.map((m) => Math.abs(weights[m.key] ?? 0)), 0);
  const scale = Math.max(2, Math.ceil(maxAbs * 1.5));
  return Math.min(scale, 100);
}

type Meta = { key: string; label: string; unit: string };

/** One labeled weight slider with a click-to-edit exact value field. */
export default function WeightRow({
  meta,
  value,
  range,
  onChange,
}: {
  meta: Meta;
  value: number;
  range: number;
  onChange: (v: number) => void;
}) {
  const [editing, setEditing] = useState<string | null>(null);

  return (
    <div className="flex items-center gap-2">
      <span className="w-24 shrink-0 truncate text-xs text-muted-foreground" title={meta.label}>
        {meta.label}
      </span>
      <Slider value={[value]} min={-range} max={range} step={0.05} onValueChange={([v]) => onChange(v)} className="flex-1" />
      {editing !== null ? (
        <Input
          autoFocus
          className="h-6 w-16 shrink-0 px-1 text-right font-mono text-[11px] tabular-nums"
          defaultValue={editing}
          onBlur={(e) => {
            const v = parseFloat(e.target.value.replace(",", "."));
            onChange(Number.isFinite(v) ? v : value);
            setEditing(null);
          }}
          onKeyDown={(e) => {
            if (e.key === "Enter") (e.target as HTMLInputElement).blur();
            if (e.key === "Escape") setEditing(null);
          }}
        />
      ) : (
        <button
          className="h-6 w-16 shrink-0 rounded border border-border bg-background px-1 text-right font-mono text-[11px] tabular-nums transition-colors hover:border-primary/50 hover:text-foreground"
          onClick={() => setEditing(String(value))}
          title="Точное значение"
        >
          {value}
        </button>
      )}
    </div>
  );
}
