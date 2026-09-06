import { useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import { Check, Download, Pencil, Plus, Trash2, Variable as VariableIcon, X } from "lucide-react";
import * as Bindings from "../../wailsjs/go/main/App";
import type { formula, main } from "../../wailsjs/go/models";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { WEIGHT_TOKENS } from "@/lib/weights";
import { heroName } from "@/lib/heroNames";

type Props = {
  views: main.PlayerView[] | null;
  children?: React.ReactNode;
};

// Reusable token chips: all scoring tokens (exclude display-only networth) +
// the operators/functions are a fixed hint line.
const TOKENS = WEIGHT_TOKENS;
const FUNCS_HINT = "+ − * / ** · функции max( ) min( ) abs( ) round( ) · max(deaths,1) защитит от деления на ноль";

export default function ManageVariables({ views, children }: Props) {
  const [open, setOpen] = useState(false);
  const [variables, setVariables] = useState<formula.Variable[]>([]);

  useEffect(() => {
    if (open) void refresh();
  }, [open]);

  const refresh = async () => {
    setVariables(await Bindings.ListVariables());
  };

  const remove = async (v: formula.Variable) => {
    try {
      await Bindings.RemoveVariable(v.id);
      await refresh();
      toast.success(`Переменная «${v.name}» удалена`);
    } catch (e) {
      toast.error(String(e));
    }
  };

  const exportVariable = async (v: formula.Variable) => {
    try {
      const ok = await Bindings.ExportVariableDialog(v.id);
      if (ok) toast.success("Переменная экспортирована");
    } catch (e) {
      toast.error(String(e));
    }
  };

  const importVariable = async () => {
    try {
      const res = await Bindings.ImportVariableDialog();
      if (res && typeof res !== "boolean") toast.success(`Импортирована «${res.name}»`);
      await refresh();
    } catch (e) {
      toast.error(String(e));
    }
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>{children}</DialogTrigger>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Пользовательские переменные</DialogTitle>
          <DialogDescription className="hidden" />
        </DialogHeader>

        <VariableEditor onSaved={refresh} views={views} />

        <ScrollArea className="max-h-80">
          <div className="space-y-1">
            {variables.map((v) => (
              <div key={v.id} className="flex items-start gap-2 rounded-md border border-border bg-background/60 px-2 py-1.5">
                <div className="min-w-0 flex-1">
                  <div className="truncate text-sm font-medium" title={v.name}>
                    {v.name}
                  </div>
                  <div className="truncate font-mono text-[11px] text-muted-foreground" title={v.expression}>
                    = {v.expression}
                  </div>
                </div>
                <Button variant="ghost" size="icon-sm" title="Экспорт" onClick={() => void exportVariable(v)}>
                  <Download className="size-3.5" />
                </Button>
                <VariableEditor trigger={<Button variant="ghost" size="icon-sm" title="Редактировать"><Pencil className="size-3.5" /></Button>} onSaved={refresh} views={views} variable={v} />
                <Button variant="ghost" size="icon-sm" title="Удалить" onClick={() => void remove(v)}>
                  <Trash2 className="size-3.5 text-destructive" />
                </Button>
              </div>
            ))}
            {variables.length === 0 && <p className="px-2 py-3 text-center text-xs text-muted-foreground">Переменных пока нет</p>}
          </div>
        </ScrollArea>

        <div className="flex justify-end">
          <Button variant="outline" onClick={() => void importVariable()}>
            Импорт
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}

type EditorProps = {
  views: main.PlayerView[] | null;
  onSaved: () => Promise<void> | void;
  variable?: formula.Variable;
  trigger?: React.ReactNode;
};

function VariableEditor({ views, onSaved, variable, trigger }: EditorProps) {
  const [openForm, setOpenForm] = useState(false);
  const [name, setName] = useState("");
  const [expr, setExpr] = useState("");
  const [error, setError] = useState("");
  const [value, setValue] = useState<number | null>(null);
  const [heroId, setHeroId] = useState<string>("");
  const testBase = useRef<Record<string, number> | null>(null);

  const loaded = !!views && views.length > 0;
  const heroes = useMemo(() => (views ?? []).slice(), [views]);

  useEffect(() => {
    if (openForm) {
      setName(variable?.name ?? "");
      setExpr(variable?.expression ?? "");
      setError("");
      setValue(null);
      // Select the first hero by default when a replay is loaded.
      setHeroId(heroes.length ? String(heroes[0].SteamID) : "");
      if (!loaded) void Bindings.TestPlayerStats().then((s) => { testBase.current = s; });
    }
  }, [openForm, variable, heroes, loaded]);

  const evaluate = async (expression: string) => {
    let base: Record<string, number>;
    if (loaded) {
      const p = heroes.find((h) => String(h.SteamID) === heroId);
      if (!p) return;
      base = p.Stats;
    } else {
      base = testBase.current ?? {};
    }
    try {
      const v = await Bindings.EvalVariable(name.trim(), expression, base);
      setError("");
      setValue(v);
    } catch (e) {
      setError(String(e));
      setValue(null);
    }
  };

  const insertToken = (token: string) => {
    const next = `${expr}${expr ? (expr.endsWith(" ") ? "" : " ") : ""}${token}`;
    setExpr(next);
    void evaluate(next);
  };

  const save = async () => {
    const trimmed = name.trim();
    if (!trimmed) return toast.error("Введите название");
    if (error) return toast.error("Исправьте ошибки в выражении");
    const payload: formula.Variable = { ...(variable ?? {}), id: variable?.id ?? crypto.randomUUID(), name: trimmed, expression: expr.trim() };
    try {
      await Bindings.UpsertVariable(payload);
      await onSaved();
      setOpenForm(false);
      toast.success(variable ? "Переменная сохранена" : `Создана переменная «${trimmed}»`);
    } catch (e) {
      toast.error(String(e));
    }
  };

  // Debounced live validation while typing the expression.
  useEffect(() => {
    if (!openForm || expr.trim() === "") {
      setError("");
      setValue(null);
      return;
    }
    const t = setTimeout(() => void evaluate(expr), 250);
    return () => clearTimeout(t);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [expr, heroId, openForm, loaded]);

  return (
    <Dialog open={openForm} onOpenChange={setOpenForm}>
      <DialogTrigger asChild>{trigger ?? <Button><Plus className="size-4" /> Создать</Button>}</DialogTrigger>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-1.5">
            <VariableIcon className="size-4" /> {variable ? "Редактировать переменную" : "Новая переменная"}
          </DialogTitle>
          <DialogDescription className="hidden" />
        </DialogHeader>

        <div className="space-y-3">
          <div className="grid gap-1">
            <label className="text-[11px] font-semibold text-muted-foreground">Название</label>
            <Input placeholder="напр. my_variable" value={name} onChange={(e) => setName(e.target.value)} />
          </div>

          <div className="grid gap-1">
            <label className="text-[11px] font-semibold text-muted-foreground">Выражение</label>
            <Textarea value={expr} onChange={(e) => setExpr(e.target.value)} rows={3} className="font-mono text-[12px]" placeholder="напр. time_dead / max(deaths, 1)" />
          </div>

          <div className="flex flex-wrap gap-1">
            {TOKENS.map((m) => (
              <Button key={m.key} variant="outline" size="sm" className="h-6 px-1.5 font-mono text-[10px]" title={m.label} onClick={() => insertToken(m.key)}>
                {m.key}
              </Button>
            ))}
          </div>
          <p className="text-[10px] text-muted-foreground">{FUNCS_HINT}</p>

          {/* Error / value preview */}
          <div className="grid gap-1 rounded-md border border-border bg-background/40 p-2">
            <label className="text-[11px] font-semibold text-muted-foreground">Проверка</label>
            {loaded && (
              <Select value={heroId} onValueChange={(v) => { setHeroId(v); }}>
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
                <span className="flex items-center gap-1 text-destructive"><X className="size-3.5" /> Ошибка: {error}</span>
              ) : value !== null && !Number.isNaN(value) ? (
                <span className="flex items-center gap-1 font-mono text-foreground tabular-nums">
                  <Check className="size-3.5 text-emerald-500" />
                  {name.trim() || "variable"} = {value.toLocaleString("ru-RU", { maximumFractionDigits: 2 })}
                  <span className="text-muted-foreground">({loaded ? "реальные данные" : "тестовые данные"})</span>
                </span>
              ) : (
                <span className="text-muted-foreground">Введите выражение для проверки</span>
              )}
            </div>
          </div>
        </div>

        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={() => setOpenForm(false)}>Отмена</Button>
          <Button onClick={() => void save()}>Сохранить</Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
