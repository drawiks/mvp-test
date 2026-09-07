import { useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import { ChevronDown, SlidersHorizontal } from "lucide-react";
import * as Bindings from "../../wailsjs/go/main/App";
import type { formula, main } from "../../wailsjs/go/models";
import { Badge } from "@/components/ui/badge";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { buildFormulaPreview } from "@/lib/format";
import { WEIGHT_META } from "@/lib/weights";
import { heroName } from "@/lib/heroNames";
import HeroIcon from "@/components/HeroIcon";
import WeightRow, { sliderRange } from "@/components/WeightRow";
import { cn } from "@/lib/utils";

type Props = {
  presets: formula.Preset[];
  activeId: string;
  views: main.PlayerView[];
  onChange: () => Promise<void> | void;
};

const EDITABLE = WEIGHT_META.filter((m) => m.key !== "networth" && !m.chipOnly);
const GROUPS = [...new Set(EDITABLE.map((m) => m.group))].map((g) => ({
  group: g,
  items: EDITABLE.filter((m) => m.group === g),
}));

type Row = { label: string; value: number };

export default function WeightsPanel({ presets, activeId, views, onChange }: Props) {
  const active = presets.find((p) => p.id === activeId) ?? null;
  const [draft, setDraft] = useState<formula.Preset | null>(active);
  const [heroPlayerID, setHeroPlayerID] = useState<number | null>(null);
  const [breakdown, setBreakdown] = useState<Row[] | null>(null);
  const [breakdownError, setBreakdownError] = useState<string | null>(null);
  const saveTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const saveGen = useRef(0);

  const sortedViews = useMemo(() => [...views].sort((a, b) => b.Score - a.Score), [views]);

  useEffect(() => setDraft(active), [active]);

  useEffect(() => {
    if (sortedViews.length > 0 && !sortedViews.some((v) => v.PlayerID === heroPlayerID)) {
      setHeroPlayerID(sortedViews[0].PlayerID);
    }
  }, [sortedViews, heroPlayerID]);

  const weights = useMemo(() => (draft && draft.kind === "linear" ? (draft.weights ?? {}) : {}), [draft]);
  const range = useMemo(() => sliderRange(weights, EDITABLE), [weights]);
  const hero = useMemo(() => sortedViews.find((v) => v.PlayerID === heroPlayerID) ?? null, [sortedViews, heroPlayerID]);
  const isLinear = draft?.kind === "linear";
  const isBuiltin = draft?.id === "standard_v2" || draft?.id === "tutorial";

  const formulaText = useMemo(
    () => (isLinear ? buildFormulaPreview(weights, EDITABLE) : draft?.expression ?? ""),
    [isLinear, weights, draft]
  );

  const linearBreakdown = useMemo<Row[] | null>(() => {
    if (!isLinear || !hero) return null;
    const rows: Row[] = [];
    for (const m of EDITABLE) {
      const w = weights[m.key] ?? 0;
      if (w === 0) continue;
      const val = hero.Stats[m.key] ?? 0;
      rows.push({ label: m.label, value: val * w });
    }
    return rows.length ? rows.sort((a, b) => b.value - a.value) : null;
  }, [isLinear, hero, weights]);

  useEffect(() => {
    if (!draft) {
      setBreakdown(null);
      setBreakdownError(null);
      return;
    }
    if (isLinear) return;
    if (!hero || !draft.expression.trim()) {
      setBreakdown(null);
      setBreakdownError(null);
      return;
    }
    let cancelled = false;
    Bindings.EvalBreakdown(draft.expression, hero.PlayerID).then(
      (rows) => {
        if (cancelled) return;
        const r = rows
          .map((row) => ({ label: row.label, value: row.value }))
          .sort((a, b) => b.value - a.value);
        setBreakdown(r.length ? r : null);
        setBreakdownError(null);
      },
      (e) => {
        if (cancelled) return;
        setBreakdown(null);
        setBreakdownError(String(e));
      }
    );
    return () => {
      cancelled = true;
    };
  }, [draft, hero, isLinear]);

  if (!active) return null;

  const invalidate = () => {
    saveGen.current++;
  };

  const commit = (next: formula.Preset) => {
    setDraft(next);
    const gen = saveGen.current;
    if (saveTimer.current) clearTimeout(saveTimer.current);
    saveTimer.current = setTimeout(async () => {
      if (saveGen.current !== gen) return;
      try {
        await Bindings.UpsertPreset(next);
        await Bindings.Recompute();
        await onChange();
      } catch (e) {
        toast.error(String(e));
        await onChange();
      }
    }, 200);
  };

  const setWeight = (key: string, value: number) => {
    if (!draft) return;
    commit({ ...draft, weights: { ...draft.weights, [key]: value } });
  };

  const selectPreset = async (id: string) => {
    invalidate();
    try {
      await Bindings.SetActivePreset(id);
    } catch (e) {
      toast.error(String(e));
    }
    await onChange();
  };

  const breakdownRows = hero && (isLinear ? linearBreakdown : breakdown);

  return (
    <div className="flex h-full flex-col gap-3 rounded-lg border border-border bg-card p-3">
      <div className="flex items-center justify-between">
        <h2 className="label-caps flex items-center gap-1.5">
          <SlidersHorizontal className="size-3.5" /> Формула
        </h2>
        {active && (
          <Badge variant={active.kind === "expression" ? "outline" : "gold"}>
            {active.kind === "expression" ? "выражение" : "линейная"}
          </Badge>
        )}
      </div>

      <Select value={active.id} onValueChange={(v) => void selectPreset(v)}>
        <SelectTrigger className="w-full">
          <SelectValue placeholder="Выберите формулу" />
        </SelectTrigger>
        <SelectContent>
          {presets.map((p) => (
            <SelectItem key={p.id} value={p.id}>
              {p.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      <div className="rounded-md border border-border bg-background/60 px-2.5 py-1.5 font-mono text-[11px] text-muted-foreground break-all" title="Формула">
        {formulaText}
      </div>

      <Separator />

      {hero && (
        <Select value={String(hero.PlayerID)} onValueChange={(v) => setHeroPlayerID(Number(v))}>
          <SelectTrigger className="w-full">
            <SelectValue placeholder="Игрок" />
          </SelectTrigger>
          <SelectContent>
            {sortedViews.map((v) => (
              <SelectItem key={v.PlayerID} value={String(v.PlayerID)} className="gap-2">
                <HeroIcon hero={v.Hero} name={v.Name} />
                <span className="truncate">{v.Name || heroName(v.Hero) || v.Hero}</span>
                <span className="ml-auto font-mono text-[11px] tabular-nums text-muted-foreground">{v.Score.toFixed(2)}</span>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )}

      {isLinear && hero ? (
        <ScrollArea className="min-h-0 flex-1">
          {breakdownRows ? (
            <div className="space-y-1 pr-2">
              {breakdownRows.map((row, i) => (
                <div key={`${row.label}-${i}`} className="flex items-center gap-2 text-xs">
                  <span className="min-w-0 flex-1 truncate text-muted-foreground">{row.label}</span>
                  <span className={cn("font-mono tabular-nums", row.value >= 0 ? "text-foreground" : "text-destructive")}>
                    {row.value.toFixed(2)}
                  </span>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-xs text-muted-foreground">У выбранной формулы нет весов.</p>
          )}
        </ScrollArea>
      ) : (
        <ScrollArea className="min-h-0 flex-1">
          {breakdownError ? (
            <p className="text-xs text-destructive break-all">Ошибка: {breakdownError}</p>
          ) : breakdownRows ? (
            <div className="space-y-1 pr-2">
              {breakdownRows.map((row, i) => (
                <div key={`${row.label}-${i}`} className="flex items-center gap-2 text-xs">
                  <span className="min-w-0 flex-1 truncate font-mono text-muted-foreground" title={row.label}>
                    {row.label}
                  </span>
                  <span className={cn("font-mono tabular-nums", row.value >= 0 ? "text-foreground" : "text-destructive")}>
                    {row.value.toFixed(2)}
                  </span>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-xs text-muted-foreground">Выберите игрока для разбора.</p>
          )}
        </ScrollArea>
      )}

      {isLinear && hero && (
        <>
          <Separator />
          <ScrollArea className="max-h-56 shrink-0">
            <div className="space-y-3 pr-2">
              {GROUPS.map(({ group, items }) => (
                <Collapsible key={group} defaultOpen className="rounded-md border border-border bg-background/40">
                  <CollapsibleTrigger className="label-caps group flex w-full items-center gap-1 p-1.5">
                    <ChevronDown className="size-3.5 transition-transform group-data-[state=closed]:-rotate-90" />
                    {group}
                  </CollapsibleTrigger>
                  <CollapsibleContent className="space-y-2 p-2 pt-0">
                    {items.map((m) => (
                      <WeightRow
                        key={m.key}
                        meta={{ key: m.key, label: m.label, unit: m.unit }}
                        value={weights[m.key] ?? 0}
                        range={range}
                        onChange={(v) => setWeight(m.key, v)}
                      />
                    ))}
                  </CollapsibleContent>
                </Collapsible>
              ))}
            </div>
          </ScrollArea>
          {isBuiltin && (
            <Badge variant="secondary" className="text-[10px]">Встроенная формула — только просмотр</Badge>
          )}
        </>
      )}
    </div>
  );
}