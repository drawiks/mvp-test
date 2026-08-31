import { useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import { ChevronDown, Code2, Plus, SlidersHorizontal, Trash2 } from "lucide-react";
import * as Bindings from "../../wailsjs/go/main/App";
import type { formula } from "../../wailsjs/go/models";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Slider } from "@/components/ui/slider";
import { Textarea } from "@/components/ui/textarea";
import { buildFormulaPreview, linearToExpression } from "@/lib/format";
import { WEIGHT_META } from "@/lib/weights";
import { cn } from "@/lib/utils";

type Props = {
  presets: formula.Preset[];
  activeId: string;
  onChange: () => Promise<void> | void;
};

// networth is not a scoring token in the engine - hide it from the editor.
const EDITABLE = WEIGHT_META.filter((m) => m.key !== "networth");
const GROUPS = ["Бой", "Фарм", "Утилити", "Защита"].map((g) => ({ group: g, items: EDITABLE.filter((m) => m.group === g) }));

function sliderRange(weights: Record<string, number>) {
  const maxAbs = Math.max(...EDITABLE.map((m) => Math.abs(weights[m.key] ?? 0)), 0);
  const scale = Math.max(2, Math.ceil(maxAbs * 1.5));
  return Math.min(scale, 100);
}

export default function WeightsPanel({ presets, activeId, onChange }: Props) {
  const active = presets.find((p) => p.id === activeId) ?? null;
  const [draft, setDraft] = useState<formula.Preset | null>(active);
  const saveTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Re-seed the draft whenever the active preset changes from outside.
  useEffect(() => setDraft(active), [active]);

  const weights = useMemo(
    () => (draft && draft.kind === "linear" ? (draft.weights ?? {}) : {}),
    [draft]
  );
  const range = useMemo(() => sliderRange(weights), [weights]);
  const formulaPreview = useMemo(
    () => (draft?.kind === "linear" ? buildFormulaPreview(weights, EDITABLE) : draft?.expression ?? ""),
    [draft, weights]
  );

  const commit = (next: formula.Preset) => {
    setDraft(next);
    if (saveTimer.current) clearTimeout(saveTimer.current);
    saveTimer.current = setTimeout(async () => {
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

  if (!active) return null;

  const setWeight = (key: string, value: number) => {
    if (!draft) return;
    commit({ ...draft, weights: { ...draft.weights, [key]: value } });
  };

  const selectPreset = async (id: string) => {
    try {
      await Bindings.SetActivePreset(id);
    } catch (e) {
      toast.error(String(e));
    }
    await onChange();
  };

  return (
    <div className="flex h-full flex-col gap-3 rounded-xl border border-border bg-card p-3">
      <div className="flex items-center justify-between">
        <h2 className="flex items-center gap-1.5 text-[13px] font-bold uppercase tracking-wide text-muted-foreground">
          <SlidersHorizontal className="size-3.5" /> Формула
        </h2>
        <Badge variant={draft?.kind === "expression" ? "outline" : "gold"}>
          {draft?.kind === "expression" ? "выражение" : "линейная"}
        </Badge>
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

      <div className="rounded-md border border-border bg-background/60 px-2.5 py-1.5 font-mono text-[11px] text-muted-foreground" title="Формула">
        {formulaPreview}
      </div>

      <ScrollArea className="min-h-0 flex-1">
        <div className="space-y-3 pr-2">
          {draft?.kind === "linear" ? (
            GROUPS.map(({ group, items }) => (
              <Collapsible key={group} defaultOpen>
                <CollapsibleTrigger className="group flex w-full items-center gap-1 text-[11px] font-bold tracking-widest text-muted-foreground uppercase">
                  <ChevronDown className="size-3.5 transition-transform group-data-[state=closed]:-rotate-90" />
                  {group}
                </CollapsibleTrigger>
                <CollapsibleContent className="space-y-2 pt-2">
                  {items.map((m) => (
                    <WeightRow key={m.key} meta={{ key: m.key, label: m.label, unit: m.unit }} value={weights[m.key] ?? 0} range={range} onChange={(v) => setWeight(m.key, v)} />
                  ))}
                </CollapsibleContent>
              </Collapsible>
            ))
          ) : draft ? (
            <div className="space-y-2">
              <p className="rounded-md bg-background/60 p-2 font-mono text-[11px] leading-relaxed break-all text-foreground">
                {draft.expression}
              </p>
              <EditorButton preset={draft} onChange={onChange} />
            </div>
          ) : null}
        </div>
      </ScrollArea>

      <Separator />

      <div className="grid grid-cols-2 gap-2">
        {draft?.kind === "linear" && (
          <EditorButton
            preset={draft}
            onChange={onChange}
            trigger={<Button variant="secondary" size="sm"><Code2 className="size-3.5" /> Редактор</Button>}
          />
        )}
        <ManageButton presets={presets} activeId={activeId} onChange={onChange} />
      </div>
    </div>
  );
}

function WeightRow({
  meta,
  value,
  range,
  onChange,
}: {
  meta: { key: string; label: string; unit: string };
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
      <Slider
        value={[value]}
        min={-range}
        max={range}
        step={0.05}
        onValueChange={([v]) => onChange(v)}
        className="flex-1"
      />
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
          {meta.unit === "÷" ? `${value}` : value}
        </button>
      )}
    </div>
  );
}

function EditorButton({
  preset,
  onChange,
  trigger,
}: {
  preset: formula.Preset;
  onChange: () => Promise<void> | void;
  trigger?: React.ReactNode;
}) {
  const [open, setOpen] = useState(false);
  const [text, setText] = useState("");
  const [variables, setVariables] = useState<formula.Variable[]>([]);
  const [preview, setPreview] = useState<{ error: string } | { rows: { Hero: string; Score: number }[] } | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (open) {
      setText(preset.kind === "expression" ? preset.expression : linearToExpression(preset.weights ?? {}, EDITABLE));
      void Bindings.ListVariables().then(setVariables);
    }
  }, [open, preset]);

  const validate = async (expr: string) => {
    if (!expr.trim()) return setPreview(null);
    setBusy(true);
    try {
      const rows = await Bindings.EvalPreview(expr);
      setPreview({ rows: rows.map((r) => ({ Hero: r.Hero, Score: r.Score })) });
    } catch (e) {
      setPreview({ error: String(e) });
    } finally {
      setBusy(false);
    }
  };

  const save = async () => {
    try {
      const next: formula.Preset = { ...preset, kind: "expression", expression: text.trim() };
      await Bindings.UpsertPreset(next);
      await Bindings.Recompute();
      await onChange();
      setOpen(false);
      toast.success("Формула сохранена");
    } catch (e) {
      toast.error(String(e));
    }
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>{trigger ?? <Button variant="secondary" size="sm"><Code2 className="size-3.5" /> Редактор формулы</Button>}</DialogTrigger>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>Редактор формулы</DialogTitle>
          <DialogDescription>
            Используйте токены статов (kills, assists, stun_duration…), операторы + − * / и функции max/min/abs/round.
          </DialogDescription>
        </DialogHeader>
        <Textarea value={text} onChange={(e) => setText(e.target.value)} rows={6} className="font-mono text-[12px]" />
        <div className="flex flex-wrap gap-1">
          {EDITABLE.map((m) => (
            <Button key={m.key} variant="outline" size="sm" className="h-6 px-1.5 font-mono text-[10px]" onClick={() => setText((t) => `${t} ${t ? "+ " : ""}${m.key}*`)}>
              {m.key}
            </Button>
          ))}
        </div>
        {variables.length > 0 && (
          <div className="flex flex-col gap-1">
            <span className="text-[11px] font-semibold text-muted-foreground">Пользовательские переменные</span>
            <div className="flex flex-wrap gap-1">
              {variables.map((v) => (
                <Button
                  key={v.id}
                  variant="outline"
                  size="sm"
                  className="h-6 px-1.5 font-mono text-[10px]"
                  title={`${v.name} = ${v.expression}`}
                  onClick={() => setText((t) => `${t} ${t ? "+ " : ""}${v.name}`)}
                >
                  {v.name}
                </Button>
              ))}
            </div>
          </div>
        )}
        <Separator />
        <div className="min-h-8 text-xs">
          {preview && "error" in preview ? (
            <span className="text-destructive">Ошибка: {preview.error}</span>
          ) : preview ? (
            <div className="font-mono tabular-nums">
              {preview.rows.slice(0, 5).map((r) => (
                <div key={r.Hero} className="flex justify-between text-muted-foreground">
                  <span>{r.Hero}</span>
                  <span className="text-foreground">{r.Score.toFixed(2)}</span>
                </div>
              ))}
            </div>
          ) : null}
        </div>
        <DialogFooter>
          <Button variant="ghost" onClick={() => void validate(text)} disabled={busy}>
            Проверить
          </Button>
          <Button onClick={() => void save()}>Сохранить</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function ManageButton({ presets, activeId, onChange }: { presets: formula.Preset[]; activeId: string; onChange: () => Promise<void> | void }) {
  const [open, setOpen] = useState(false);
  const [name, setName] = useState("");

  const create = async () => {
    const trimmed = name.trim();
    if (!trimmed) return toast.error("Введите название");
    try {
      const weights: Record<string, number> = {};
      for (const m of EDITABLE) weights[m.key] = m.key === "deaths" || m.key === "deaths_base" ? 0.1 : 0;
      await Bindings.UpsertPreset({ id: crypto.randomUUID(), name: trimmed, kind: "linear", weights, expression: "" });
      setName("");
      await onChange();
      toast.success(`Создан пресет «${trimmed}»`);
    } catch (e) {
      toast.error(String(e));
    }
  };

  const duplicate = async (p: formula.Preset) => {
    try {
      await Bindings.UpsertPreset({ ...p, id: crypto.randomUUID(), name: `${p.name} (копия)` });
      await onChange();
      toast.success("Пресет продублирован");
    } catch (e) {
      toast.error(String(e));
    }
  };

  const remove = async (id: string) => {
    try {
      const ok = await Bindings.RemovePreset(id);
      if (ok) toast.success("Пресет удалён");
      await onChange();
    } catch (e) {
      toast.error(String(e));
    }
  };

  const importPreset = async () => {
    try {
      const res = await Bindings.ImportPresetDialog();
      if (res && typeof res !== "boolean") toast.success(`Импортирован «${res.name}»`);
      await onChange();
    } catch (e) {
      toast.error(String(e));
    }
  };

  const exportPreset = async (id: string) => {
    try {
      const ok = await Bindings.ExportPresetDialog(id);
      if (ok) toast.success("Экспортировано");
    } catch (e) {
      toast.error(String(e));
    }
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="ghost" size="sm">
          <Plus className="size-3.5" /> Управление
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Управление формулами</DialogTitle>
        </DialogHeader>
        <div className="flex items-center gap-2">
          <Input placeholder="Название новой формулы" value={name} onChange={(e) => setName(e.target.value)} onKeyDown={(e) => e.key === "Enter" && void create()} />
          <Button onClick={() => void create()}>
            <Plus className="size-4" /> Создать
          </Button>
        </div>
        <Separator />
        <ScrollArea className="max-h-80">
          <div className="space-y-1">
            {presets.map((p) => (
              <div key={p.id} className={cn("flex items-center gap-2 rounded-md border border-border bg-background/60 px-2 py-1.5", p.id === activeId && "border-gold/40")}>
                <span className="min-w-0 flex-1 truncate text-sm font-medium" title={p.name}>
                  {p.name}
                  {p.id === activeId && <Badge variant="gold" className="ml-2">активна</Badge>}
                </span>
                <Badge variant="outline">{p.kind === "expression" ? "expr" : "лин."}</Badge>
                <Button variant="ghost" size="icon-sm" title="Продублировать" onClick={() => void duplicate(p)}>
                  <Plus className="size-3.5" />
                </Button>
                <Button variant="ghost" size="icon-sm" title="Экспорт" onClick={() => void exportPreset(p.id)}>
                  <Code2 className="size-3.5" />
                </Button>
                <Button variant="ghost" size="icon-sm" title="Удалить" disabled={p.id === "standard" || p.id === "standard_v2"} onClick={() => void remove(p.id)}>
                  <Trash2 className="size-3.5 text-destructive" />
                </Button>
              </div>
            ))}
          </div>
        </ScrollArea>
        <DialogFooter>
          <Button variant="outline" onClick={() => void importPreset()}>
            Импорт
          </Button>
          <Button onClick={() => setOpen(false)}>Закрыть</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}