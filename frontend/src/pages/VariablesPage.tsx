import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Check, Copy, Download, Plus, Trash2, Upload, Variable as VariableIcon, X } from "lucide-react";
import { toast } from "sonner";
import * as Bindings from "../../wailsjs/go/main/App";
import type { formula, main } from "../../wailsjs/go/models";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { Badge } from "@/components/ui/badge";
import FormulaEditor, { type FormulaEditorHandle } from "@/components/FormulaEditor";
import TokenPalette from "@/components/TokenPalette";
import { heroName } from "@/lib/heroNames";
import { cn } from "@/lib/utils";

const FUNCS_HINT = "+ − * / ** · условия if/elif/else (if position == 1 { kills } else { kills / 4 }) · сравнения == != < > <= >= · логика && || ! · функции max( ) min( ) abs( ) round( )";

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

export default function VariablesPage({ views }: { views: main.PlayerView[] | null }) {
  const [variables, setVariables] = useState<formula.Variable[]>([]);
  const [draft, setDraft] = useState<formula.Variable | null>(null);
  const [isNew, setIsNew] = useState(false);
  const [expr, setExpr] = useState("");
  const [error, setError] = useState("");
  const [value, setValue] = useState<number | null>(null);
  const [heroId, setHeroId] = useState("");
  const testBase = useRef<Record<string, number> | null>(null);
  const editorRef = useRef<FormulaEditorHandle>(null);
  const saveTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const saveGen = useRef(0);

  const refresh = useCallback(async () => {
    setVariables(await Bindings.ListVariables());
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const loaded = !!views && views.length > 0;
  const heroes = useMemo(() => (views ?? []).slice(), [views]);

  useEffect(() => {
    if (!loaded) {
      void Bindings.TestPlayerStats().then((s) => {
        testBase.current = s;
      });
    }
  }, [loaded]);

  useEffect(() => {
    if (heroes.length && !heroes.some((h) => String(h.SteamID) === heroId)) {
      setHeroId(String(heroes[0].SteamID));
    }
  }, [heroes, heroId]);

  const invalidate = useCallback(() => {
    saveGen.current++;
  }, []);

  const reset = useCallback(() => {
    setDraft(null);
    setIsNew(false);
    setExpr("");
    setError("");
    setValue(null);
  }, []);

  const startNew = useCallback(() => {
    setDraft({ id: crypto.randomUUID(), name: "", expression: "" });
    setIsNew(true);
    setExpr("");
    setError("");
    setValue(null);
    if (heroes.length) setHeroId(String(heroes[0].SteamID));
  }, [heroes]);

  const selectVar = useCallback(
    (v: formula.Variable) => {
      setDraft(v);
      setIsNew(false);
      setExpr(v.expression);
      setError("");
      setValue(null);
      if (heroes.length) setHeroId(String(heroes[0].SteamID));
    },
    [heroes]
  );

  const commit = useCallback(
    (next: formula.Variable) => {
      setDraft(next);
      const gen = saveGen.current;
      if (saveTimer.current) clearTimeout(saveTimer.current);
      saveTimer.current = setTimeout(async () => {
        if (saveGen.current !== gen) return;
        try {
          await Bindings.UpsertVariable(next);
          await Bindings.Recompute();
          await refresh();
        } catch (e) {
          toast.error(String(e));
          await refresh();
        }
      }, 200);
    },
    [refresh]
  );

  const evaluate = useCallback(
    async (expression: string) => {
      let base: Record<string, number>;
      if (loaded) {
        const p = heroes.find((h) => String(h.SteamID) === heroId);
        if (!p) return;
        base = p.Stats;
      } else {
        base = testBase.current ?? {};
      }
      try {
        const v = await Bindings.EvalVariable(draft?.name ?? "", expression, base);
        setError("");
        setValue(v);
      } catch (e) {
        setError(String(e));
        setValue(null);
      }
    },
    [loaded, heroes, heroId, draft]
  );

  useEffect(() => {
    if (!expr.trim()) {
      setError("");
      setValue(null);
      return;
    }
    const t = setTimeout(() => void evaluate(expr), 250);
    return () => clearTimeout(t);
  }, [expr, heroId, loaded, evaluate]);

  const insertToken = useCallback((token: string) => {
    const h = editorRef.current;
    if (h) h.insert(token);
    else setExpr((e) => `${e}${e ? (e.endsWith(" ") ? "" : " ") : ""}${token}`);
  }, []);

  const handleSave = useCallback(async () => {
    if (!draft) return;
    invalidate();
    const payload: formula.Variable = { ...draft, name: draft.name.trim(), expression: expr.trim() };
    try {
      await Bindings.UpsertVariable(payload);
      await Bindings.Recompute();
      await refresh();
      setDraft(payload);
      setIsNew(false);
      toast.success(isNew ? `Создана переменная «${payload.name}»` : "Переменная сохранена");
    } catch (e) {
      toast.error(String(e));
    }
  }, [draft, expr, isNew, invalidate, refresh]);

  const remove = useCallback(async () => {
    if (!draft) return;
    try {
      await Bindings.RemoveVariable(draft.id);
      await refresh();
      reset();
      toast.success(`Переменная «${draft.name}» удалена`);
    } catch (e) {
      toast.error(String(e));
    }
  }, [draft, refresh, reset]);

  const exportVariable = useCallback(async () => {
    if (!draft) return;
    try {
      const ok = await Bindings.ExportVariableDialog(draft.id);
      if (ok) toast.success("Переменная экспортирована");
    } catch (e) {
      toast.error(String(e));
    }
  }, [draft]);

  const importVariable = useCallback(async () => {
    try {
      const res = await Bindings.ImportVariableDialog();
      if (res && typeof res !== "boolean") toast.success(`Импортирована «${res.name}»`);
      await refresh();
    } catch (e) {
      toast.error(String(e));
    }
  }, [refresh]);

  const duplicate = useCallback(async () => {
    if (!draft) return;
    try {
      await Bindings.UpsertVariable({ ...draft, id: crypto.randomUUID(), name: `${draft.name} (копия)` });
      await refresh();
      toast.success("Переменная продублирована");
    } catch (e) {
      toast.error(String(e));
    }
  }, [draft, refresh]);

  return (
    <div className="flex h-full min-h-0 flex-col gap-4 p-4">
      <div className="flex items-center justify-between">
        <h2 className="flex items-center gap-1.5 text-lg font-bold">
          <VariableIcon className="size-5" /> Переменные
        </h2>
      </div>

      <div className="flex min-h-0 flex-1 flex-col gap-4 md:flex-row">
        <aside className="flex w-[280px] min-h-0 shrink-0 flex-col">
          <div className="flex-1 min-h-0">
            <ScrollArea type="scroll" className="h-full">
              <div className="space-y-1 pr-1">
                {variables.map((v) => (
                  <button
                    key={v.id}
                    onClick={() => selectVar(v)}
                    title={`${v.name} = ${v.expression}`}
                    className={cn(
                      "w-full flex items-center gap-2 rounded-md border border-border bg-background/60 px-2 py-1.5 text-left transition-colors hover:bg-accent/50",
                      !isNew && draft?.id === v.id && "border-gold/40 bg-gold/5"
                    )}
                  >
                    <span className="min-w-0 flex-1 truncate text-sm font-medium">{v.name}</span>
                    <Badge variant="outline" className="shrink-0">
                      переменная
                    </Badge>
                  </button>
                ))}
                {variables.length === 0 && (
                  <p className="px-2 py-3 text-center text-xs text-muted-foreground">Переменных пока нет</p>
                )}
              </div>
            </ScrollArea>
          </div>
          <Separator />
          <TooltipProvider delayDuration={200}>
            <div className="flex items-center gap-1 p-2">
              <ToolbarButton title="Новая переменная" onClick={startNew}>
                <Plus className="size-3.5" />
              </ToolbarButton>
              <ToolbarButton title="Импортировать из файла" onClick={() => void importVariable()}>
                <Download className="size-3.5" />
              </ToolbarButton>
              <Separator orientation="vertical" className="mx-1 h-4" />
              <ToolbarButton title="Дублировать" disabled={!draft || isNew} onClick={() => void duplicate()}>
                <Copy className="size-3.5" />
              </ToolbarButton>
              <ToolbarButton title="Экспортировать в файл" disabled={!draft || isNew} onClick={() => void exportVariable()}>
                <Upload className="size-3.5" />
              </ToolbarButton>
              <ToolbarButton title="Удалить" disabled={!draft || isNew} onClick={() => void remove()}>
                <Trash2 className="size-3.5 text-destructive" />
              </ToolbarButton>
            </div>
          </TooltipProvider>
        </aside>

        <div className="flex min-h-0 min-w-0 flex-1 flex-col gap-4 overflow-y-auto">
          {!draft ? (
            <div className="flex flex-1 items-center justify-center text-muted-foreground">
              {variables.length ? "Выберите переменную" : "Переменных пока нет"}
            </div>
          ) : (
            <>
              <div className="flex items-center gap-2">
                <input
                  type="text"
                  value={draft.name}
                  onChange={(e) => {
                    const next = { ...draft, name: e.target.value };
                    setDraft(next);
                    if (!isNew) commit(next);
                  }}
                  className="flex-1 rounded-md border border-border bg-background px-2 py-1.5 text-sm"
                  placeholder="Название переменной"
                />
                <Badge variant="outline">{isNew ? "новая" : "переменная"}</Badge>
              </div>

              <FormulaEditor ref={editorRef} expression={expr} onChange={setExpr} height="220px" />

              <div className="flex flex-wrap gap-1">
                <TokenPalette variables={variables} onInsert={insertToken} />
              </div>

              <p className="text-[10px] text-muted-foreground">{FUNCS_HINT}</p>

              <div className="space-y-2 rounded-md border border-border bg-background/40 p-2">
                <label className="label-caps">Предпросмотр</label>
                {loaded && (
                  <Select value={heroId} onValueChange={(v) => setHeroId(v)}>
                    <SelectTrigger className="h-8">
                      <SelectValue placeholder="Выберите героя" />
                    </SelectTrigger>
                    <SelectContent>
                      {heroes.map((h) => (
                        <SelectItem key={h.SteamID} value={String(h.SteamID)}>
                          {heroName(h.Hero) || h.Hero} — {h.Name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                )}
                <div className="min-h-6 text-xs">
                  {error ? (
                    <span className="flex items-center gap-1 text-destructive">
                      <X className="size-3.5" /> Ошибка: {error}
                    </span>
                  ) : value !== null && !Number.isNaN(value) ? (
                    <span className="flex items-center gap-1 font-mono text-foreground tabular-nums">
                      <Check className="size-3.5 text-emerald-500" />
                      {draft.name.trim() || "variable"} = {value.toLocaleString("ru-RU", { maximumFractionDigits: 2 })}
                      <span className="text-muted-foreground">({loaded ? "реальные данные" : "тестовые данные"})</span>
                    </span>
                  ) : (
                    <span className="text-muted-foreground">Введите выражение для проверки</span>
                  )}
                </div>
              </div>

              <div className="mt-auto flex justify-end gap-2 border-t border-border pt-4">
                <Button variant="outline" onClick={reset}>
                  Отменить
                </Button>
                <Button onClick={() => void handleSave()}>Сохранить</Button>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
}