import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ChevronDown, Copy, Code2, Download, Plus, SlidersHorizontal, Trash2, Type, Upload } from "lucide-react";
import { toast } from "sonner";
import * as Bindings from "../../wailsjs/go/main/App";
import type { formula } from "../../wailsjs/go/models";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogClose } from "@/components/ui/dialog";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import FormulaEditor, { type FormulaEditorHandle } from "@/components/FormulaEditor";
import TokenPalette from "@/components/TokenPalette";
import HeroIcon from "@/components/HeroIcon";
import WeightRow, { sliderRange } from "@/components/WeightRow";
import { WEIGHT_META } from "@/lib/weights";
import { linearToExpression } from "@/lib/format";
import { heroName } from "@/lib/heroNames";
import { cn } from "@/lib/utils";

const EDITABLE = WEIGHT_META.filter((m) => m.key !== "networth" && !m.chipOnly);
const GROUPS = [...new Set(EDITABLE.map((m) => m.group))].map((g) => ({ group: g, items: EDITABLE.filter((m) => m.group === g) }));

type PreviewRow = { Hero: string; Name: string; Score: number };
type Preview = { error: string } | { rows: PreviewRow[] };

function ToolbarButton({ title, disabled, onClick, children }: { title: string; disabled?: boolean; onClick: () => void; children: React.ReactNode }) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button variant="ghost" size="icon-sm" disabled={disabled} onClick={onClick}>
          {children}
        </Button>
      </TooltipTrigger>
      <TooltipContent side="top">{title}</TooltipContent>
    </Tooltip>
  );
}

function CreatePresetDialog({ open, onOpenChange, onCreate }: { open: boolean; onOpenChange: (v: boolean) => void; onCreate: (name: string) => void }) {
  const [name, setName] = useState("");
  useEffect(() => {
    if (open) setName("");
  }, [open]);
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xs">
        <DialogHeader>
          <DialogTitle>Новая формула</DialogTitle>
          <DialogDescription>Введите название пресета формулы</DialogDescription>
        </DialogHeader>
        <Input
          autoFocus
          value={name}
          onChange={(e) => setName(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && name.trim()) onCreate(name.trim());
          }}
          placeholder="Название формулы"
        />
        <DialogFooter>
          <DialogClose asChild>
            <Button variant="outline">Отмена</Button>
          </DialogClose>
          <Button onClick={() => onCreate(name.trim())} disabled={!name.trim()}>
            Создать
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function PlayerPreview({ preview }: { preview: Preview }) {
  const rows = useMemo(() => ("error" in preview ? [] : [...preview.rows].sort((a, b) => b.Score - a.Score)), [preview]);
  if ("error" in preview) return <span className="text-xs text-destructive">Ошибка: {preview.error}</span>;
  if (rows.length === 0) return null;
  return (
    <div className="space-y-2 rounded-md border border-border bg-background/40 p-2">
      <label className="label-caps">Предпросмотр</label>
      <ScrollArea type="scroll" className="h-40">
        <div className="pr-1">
          {rows.map((r, i) => (
            <div
              key={`${r.Name}-${i}`}
              className={cn(
                "flex items-center gap-2 py-1 text-sm",
                i === 0 && "font-semibold text-gold",
                i < 3 && "font-medium"
              )}
            >
              <span className="w-5 shrink-0 text-right font-mono text-[11px] tabular-nums text-muted-foreground">{i + 1}</span>
              <HeroIcon hero={r.Hero} name={r.Name} />
              <span className="min-w-0 flex-1 truncate">{r.Name || heroName(r.Hero) || r.Hero}</span>
              <span className="font-mono text-[11px] tabular-nums text-muted-foreground">{r.Score.toFixed(2)}</span>
            </div>
          ))}
        </div>
      </ScrollArea>
    </div>
  );
}

export default function FormulasPage({ presets, activeId, onChange }: { presets: formula.Preset[]; activeId: string; onChange: () => Promise<void> | void }) {
  const [activePreset, setActivePreset] = useState(presets.find((p) => p.id === activeId) ?? null);
  const [draftExpr, setDraftExpr] = useState("");
  const [variables, setVariables] = useState<formula.Variable[]>([]);
  const [preview, setPreview] = useState<Preview | null>(null);
  const [showWeightSliders, setShowWeightSliders] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const saveTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const saveGen = useRef(0);
  const editorRef = useRef<FormulaEditorHandle>(null);

  useEffect(() => {
    const p = presets.find((p) => p.id === activeId) ?? null;
    setActivePreset(p);
    if (p) {
      setDraftExpr(p.kind === "expression" ? p.expression : linearToExpression(p.weights ?? {}, EDITABLE));
      void Bindings.ListVariables().then(setVariables);
    }
  }, [activeId, presets]);

  useEffect(() => {
    const timer = setTimeout(() => {
      if (!draftExpr.trim()) return setPreview(null);
      void Bindings.EvalPreview(draftExpr).then(
        (rows) => setPreview({ rows: rows.map((r) => ({ Hero: r.Hero, Name: r.Name, Score: r.Score })) }),
        (e) => setPreview({ error: String(e) })
      );
    }, 150);
    return () => clearTimeout(timer);
  }, [draftExpr]);

  const invalidate = useCallback(() => {
    saveGen.current++;
  }, []);

  const commit = useCallback(
    (next: formula.Preset) => {
      setActivePreset(next);
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
    },
    [onChange]
  );

  const selectPreset = useCallback(
    async (id: string) => {
      try {
        await Bindings.SetActivePreset(id);
      } catch (e) {
        toast.error(String(e));
      }
      await onChange();
    },
    [onChange]
  );

  const handleSave = useCallback(async () => {
    if (!activePreset) return;
    try {
      const next: formula.Preset = { ...activePreset, kind: "expression", expression: draftExpr.trim() };
      invalidate();
      await Bindings.UpsertPreset(next);
      await Bindings.Recompute();
      await onChange();
      toast.success("Формула сохранена");
    } catch (e) {
      toast.error(String(e));
    }
  }, [activePreset, draftExpr, invalidate, onChange]);

  const handleCreate = useCallback(
    async (name: string) => {
      const trimmed = name.trim();
      if (!trimmed) return;
      setCreateOpen(false);
      const weights: Record<string, number> = {};
      for (const m of EDITABLE) weights[m.key] = m.key === "deaths" ? 0.1 : 0;
      try {
        await Bindings.UpsertPreset({ id: crypto.randomUUID(), name: trimmed, kind: "linear", weights, expression: "" });
        await onChange();
        toast.success(`Создан пресет «${trimmed}»`);
      } catch (e) {
        toast.error(String(e));
      }
    },
    [onChange]
  );

  const handleDuplicate = useCallback(
    async (p: formula.Preset | null) => {
      if (!p) return;
      try {
        await Bindings.UpsertPreset({ ...p, id: crypto.randomUUID(), name: `${p.name} (копия)` });
        await onChange();
        toast.success("Пресет продублирован");
      } catch (e) {
        toast.error(String(e));
      }
    },
    [onChange]
  );

  const handleRemove = useCallback(
    async (id: string) => {
      try {
        const ok = await Bindings.RemovePreset(id);
        if (ok) toast.success("Пресет удалён");
        await onChange();
      } catch (e) {
        toast.error(String(e));
      }
    },
    [onChange]
  );

  const handleImport = useCallback(async () => {
    try {
      const res = await Bindings.ImportPresetDialog();
      if (res && typeof res !== "boolean") toast.success(`Импортирован «${res.name}»`);
      await onChange();
    } catch (e) {
      toast.error(String(e));
    }
  }, [onChange]);

  const handleExport = useCallback(
    async (id: string) => {
      try {
        const ok = await Bindings.ExportPresetDialog(id);
        if (ok) toast.success("Экспортировано");
      } catch (e) {
        toast.error(String(e));
      }
    },
    []
  );

  const insertToken = useCallback((t: string) => {
    const h = editorRef.current;
    if (h) h.insert(t);
    else setDraftExpr((e) => `${e}${e ? " " : ""}${t}`);
  }, []);

  const isBuiltin = activePreset?.id === "standard_v2" || activePreset?.id === "tutorial";
  const isLinear = activePreset?.kind === "linear";

  const weights = useMemo(() => (activePreset?.kind === "linear" ? activePreset.weights ?? {} : {}), [activePreset]);
  const range = useMemo(() => sliderRange(weights, EDITABLE), [weights]);

  return (
    <div className="flex h-full min-h-0 flex-col gap-4 p-4">
      <div className="flex items-center justify-between">
        <h2 className="flex items-center gap-1.5 text-lg font-bold">
          <Type className="size-5" /> Формулы
        </h2>
      </div>

      <div className="flex min-h-0 flex-1 flex-col gap-4 md:flex-row">
        <aside className="flex w-[280px] min-h-0 shrink-0 flex-col">
          <div className="flex-1 min-h-0">
            <ScrollArea type="scroll" className="h-full">
              <div className="space-y-1 pr-1">
                {presets.map((p) => (
                  <button
                    key={p.id}
                    onClick={() => void selectPreset(p.id)}
                    title={p.name}
                    className={cn(
                      "w-full flex items-center gap-2 rounded-md border border-border bg-background/60 px-2 py-1.5 text-left transition-colors hover:bg-accent/50",
                      p.id === activeId && "border-gold/40 bg-gold/5"
                    )}
                  >
                    <span className="min-w-0 flex-1 truncate text-sm font-medium">{p.name}</span>
                    <Badge variant={p.kind === "expression" ? "outline" : "gold"} className="shrink-0">
                      {p.kind === "expression" ? "expr" : "лин."}
                    </Badge>
                  </button>
                ))}
              </div>
            </ScrollArea>
          </div>
          <Separator />
          <TooltipProvider delayDuration={200}>
            <div className="flex items-center gap-1 p-2">
              <ToolbarButton title="Новая формула…" onClick={() => setCreateOpen(true)}>
                <Plus className="size-3.5" />
              </ToolbarButton>
              <ToolbarButton title="Импортировать из файла" onClick={() => void handleImport()}>
                <Download className="size-3.5" />
              </ToolbarButton>
              <Separator orientation="vertical" className="mx-1 h-4" />
              <ToolbarButton title="Дублировать" disabled={!activePreset} onClick={() => void handleDuplicate(activePreset)}>
                <Copy className="size-3.5" />
              </ToolbarButton>
              <ToolbarButton title="Экспортировать в файл" disabled={!activePreset} onClick={() => void handleExport(activePreset!.id)}>
                <Upload className="size-3.5" />
              </ToolbarButton>
              <ToolbarButton title="Удалить" disabled={!activePreset || isBuiltin} onClick={() => void handleRemove(activePreset!.id)}>
                <Trash2 className="size-3.5 text-destructive" />
              </ToolbarButton>
            </div>
          </TooltipProvider>
        </aside>

        <div className="flex min-h-0 min-w-0 flex-1 flex-col gap-4 overflow-y-auto">
          {!activePreset ? (
            <div className="flex flex-1 items-center justify-center text-muted-foreground">Выберите формулу</div>
          ) : (
            <>
              <div className="flex items-center gap-2">
                <input
                  type="text"
                  value={activePreset.name}
                  onChange={(e) => {
                    const next = { ...activePreset!, name: e.target.value };
                    setActivePreset(next);
                    commit(next);
                  }}
                  className="flex-1 rounded-md border border-border bg-background px-2 py-1.5 text-sm"
                  disabled={isBuiltin}
                />
                <Badge variant={activePreset.kind === "expression" ? "outline" : "gold"}>
                  {activePreset.kind === "expression" ? "выражение" : "линейная"}
                </Badge>
                {isBuiltin && (
                  <Badge variant="secondary" className="text-[10px]">
                    Встроенная формула — только просмотр
                  </Badge>
                )}
              </div>

              {isLinear && (
                <div className="space-y-3">
                  <div className="flex items-center gap-2">
                    <Button variant="secondary" size="sm" onClick={() => setShowWeightSliders(true)}>
                      <SlidersHorizontal className="size-3.5 mr-1" /> Веса
                    </Button>
                    <Button variant={showWeightSliders ? "outline" : "secondary"} size="sm" onClick={() => setShowWeightSliders(false)}>
                      <Code2 className="size-3.5 mr-1" /> Код
                    </Button>
                  </div>

                  {showWeightSliders ? (
                    <div className="space-y-3 pr-2 max-h-80 overflow-auto">
                      {GROUPS.map(({ group, items }) => (
                        <Collapsible key={group} defaultOpen className="rounded-md border border-border bg-background/60">
                          <CollapsibleTrigger className="label-caps flex w-full items-center gap-1 p-2">
                            <ChevronDown className="size-3" />
                            {group}
                          </CollapsibleTrigger>
                          <CollapsibleContent className="space-y-2 p-2 pt-0">
                            {items.map((m) => (
                              <WeightRow
                                key={m.key}
                                meta={{ key: m.key, label: m.label, unit: m.unit }}
                                value={weights[m.key] ?? 0}
                                range={range}
                                onChange={(v) => commit({ ...activePreset!, weights: { ...weights, [m.key]: v } })}
                              />
                            ))}
                          </CollapsibleContent>
                        </Collapsible>
                      ))}
                    </div>
                  ) : (
                    <>
                      <FormulaEditor ref={editorRef} expression={draftExpr} onChange={setDraftExpr} height="220px" />
                      <div className="flex flex-wrap gap-1">
                        <TokenPalette variables={variables} onInsert={insertToken} />
                      </div>
                      {preview && <PlayerPreview preview={preview} />}
                    </>
                  )}
                </div>
              )}

              {!isLinear && (
                <>
                  <FormulaEditor ref={editorRef} expression={draftExpr} onChange={setDraftExpr} height="220px" readOnly={isBuiltin} />
                  {!isBuiltin && (
                    <div className="flex flex-wrap gap-1">
                      <TokenPalette variables={variables} onInsert={insertToken} />
                    </div>
                  )}
                  {preview && <PlayerPreview preview={preview} />}
                </>
              )}

              {!isBuiltin && (
                <div className="mt-auto flex justify-end gap-2 border-t border-border pt-4">
                  <Button variant="outline" onClick={() => setDraftExpr(activePreset.kind === "expression" ? activePreset.expression : linearToExpression(activePreset.weights ?? {}, EDITABLE))}>
                    Отменить
                  </Button>
                  <Button onClick={handleSave}>Сохранить</Button>
                </div>
              )}
            </>
          )}
        </div>
      </div>

      <CreatePresetDialog open={createOpen} onOpenChange={setCreateOpen} onCreate={(name) => void handleCreate(name)} />
    </div>
  );
}