import { useEffect, useState } from "react";
import { FolderOpen, Loader2, Settings } from "lucide-react";
import * as Bindings from "../wailsjs/go/main/App";
import type { formula, main } from "../wailsjs/go/models";
import { onEvent, offEvent } from "@/lib/wails";
import type { MatchInfo } from "@/components/InfoBar";
import HomePage from "@/pages/HomePage";
import FormulasPage from "@/pages/FormulasPage";
import VariablesPage from "@/pages/VariablesPage";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Toaster } from "@/components/ui/sonner";
import { cn } from "@/lib/utils";

const NAV: { id: "home" | "formulas" | "variables"; label: string }[] = [
  { id: "home", label: "Главная" },
  { id: "formulas", label: "Формулы" },
  { id: "variables", label: "Переменные" },
];

export default function App() {
  const [views, setViews] = useState<main.PlayerView[] | null>(null);
  const [presets, setPresets] = useState<formula.Preset[]>([]);
  const [activeId, setActiveId] = useState("");
  const [status, setStatus] = useState("idle");
  const [error, setError] = useState("");
  const [showSettings, setShowSettings] = useState(false);
  const [parserURL, setParserURL] = useState("");
  const [loading, setLoading] = useState(true);
  const [matchInfo, setMatchInfo] = useState<MatchInfo | null>(null);
  const [page, setPage] = useState<"home" | "formulas" | "variables">("home");

  const refreshPresets = async () => {
    const [list, active, url] = await Promise.all([
      Bindings.ListPresets(),
      Bindings.ActivePresetID(),
      Bindings.GetParserURL(),
    ]);
    setPresets(list);
    setActiveId(active);
    setParserURL(url);
  };

  useEffect(() => {
    void (async () => {
      try {
        await refreshPresets();
      } finally {
        setLoading(false);
      }
    })();
    onEvent("parseStart", () => {
      setStatus("parsing");
      setError("");
      setMatchInfo(null);
    });
    onEvent("parseDone", (payload: { ok?: boolean; error?: string }) => {
      setStatus(payload?.ok ? "idle" : "error");
      if (!payload?.ok) setError(payload?.error ?? "Неизвестная ошибка");
    });
    onEvent("resultUpdated", (payload: main.PlayerView[] | null) => {
      setViews(payload ?? []);
    });
    onEvent("matchInfo", (payload: MatchInfo | null) => {
      setMatchInfo(payload ?? null);
    });
    return () => {
      offEvent("parseStart");
      offEvent("parseDone");
      offEvent("resultUpdated");
      offEvent("matchInfo");
    };
  }, []);

  const saveURL = async () => {
    try {
      await Bindings.SetParserURL(parserURL);
      setShowSettings(false);
    } catch (e) {
      setError(String(e));
    }
  };

  useEffect(() => {
    if (typeof window === "undefined" || !window.runtime?.OnFileDrop) return;
    window.runtime.OnFileDrop((_x, _y, paths) => {
      const target = (paths ?? []).find((p) => /\.(dem|json|ndjson|bz2|zst)$/i.test(p));
      if (target) void Bindings.ParseReplay(target);
    }, true);
    return () => {
      if (typeof window !== "undefined" && window.runtime?.OnFileDropOff) {
        window.runtime.OnFileDropOff();
      }
    };
  }, []);

  const statusDot =
    status === "parsing" ? "bg-sky-400 animate-pulse" : status === "error" ? "bg-destructive" : "bg-radiant";
  const statusLabel =
    status === "parsing" ? "Парсим реплей…" : status === "error" ? "Ошибка" : views ? "Готово" : "Ожидание реплея";

  return (
    <div
      className="flex h-screen flex-col overflow-hidden bg-background text-foreground"
      style={{ ["--wails-drop-target" as string]: "drop" }}
      data-drop-zone
    >
      <header className="flex items-center gap-6 border-b border-border bg-card/40 px-4">
        <div className="flex items-center gap-2.5 py-3">
          <span className="flex size-6 items-center justify-center rounded border border-gold/40 font-mono text-[11px] font-bold text-gold">
            M
          </span>
          <div className="flex flex-col leading-none">
            <span className="text-[13px] font-bold">MVP-Test</span>
            <span className="text-[9px] font-medium tracking-wide text-muted-foreground uppercase">by drawiks</span>
          </div>
        </div>

        <nav className="flex h-full items-center gap-1 self-stretch">
          {NAV.map((n) => (
            <button
              key={n.id}
              onClick={() => setPage(n.id)}
              className={cn(
                "relative flex h-full items-center px-2 text-[12.5px] font-medium text-muted-foreground transition-colors hover:text-foreground",
                page === n.id && "text-foreground"
              )}
            >
              {n.label}
              {page === n.id && <span className="absolute right-0 bottom-0 left-0 h-[2px] bg-gold" />}
            </button>
          ))}
        </nav>

        <div className="ml-auto flex items-center gap-3 py-2.5">
          <span className="flex items-center gap-1.5 text-[11.5px] text-muted-foreground">
            <span className={cn("size-1.5 rounded-full", statusDot)} />
            {statusLabel}
          </span>
          {views && (
            <Button variant="outline" size="sm" onClick={() => void Bindings.ChooseReplay()} className="h-7">
              <FolderOpen className="mr-1.5 size-3.5" /> Открыть
            </Button>
          )}
          <Button variant="ghost" size="icon" className="size-7" onClick={() => setShowSettings((s) => !s)}>
            <Settings className="size-4" />
          </Button>
        </div>
      </header>

      {showSettings && (
        <div className="flex items-center gap-2 border-b border-border bg-background/60 px-4 py-2">
          <label className="text-xs text-muted-foreground">URL парсера:</label>
          <Input value={parserURL} onChange={(e) => setParserURL(e.target.value)} className="h-8 w-80" placeholder="http://localhost:5600" />
          <Button size="sm" onClick={() => void saveURL()}>
            Сохранить
          </Button>
        </div>
      )}

      {loading ? (
        <div className="flex flex-1 items-center justify-center text-muted-foreground">
          <Loader2 className="mr-2 size-4 animate-spin" /> Загрузка…
        </div>
      ) : page === "formulas" ? (
        <FormulasPage presets={presets} activeId={activeId} onChange={refreshPresets} />
      ) : page === "variables" ? (
        <VariablesPage views={views} />
      ) : views && views.length > 0 ? (
        <HomePage views={views} presets={presets} activeId={activeId} matchInfo={matchInfo} onChange={refreshPresets} />
      ) : (
        <main className="flex flex-1 items-center justify-center">
          <div className="w-full max-w-sm text-center">
            <span className="mx-auto mb-4 flex size-11 items-center justify-center rounded-lg border border-gold/40 font-mono text-base font-bold text-gold">
              M
            </span>
            <h1 className="mb-1 text-xl font-bold">MVP-Test</h1>
            <p className="mb-5 text-xs text-muted-foreground">Откройте реплей, чтобы рассчитать MVP матча</p>
            {error && (
              <div className="mb-4 rounded-md border border-destructive/40 bg-destructive/10 p-2 text-left text-sm text-destructive">
                {error}
              </div>
            )}
            <Button size="lg" onClick={() => void Bindings.ChooseReplay()} disabled={status === "parsing"}>
              {status === "parsing" ? <Loader2 className="mr-2 size-4 animate-spin" /> : <FolderOpen className="mr-2 size-4" />}
              Открыть реплей
            </Button>
            <p className="mt-4 text-[11px] text-muted-foreground">Ctrl+O · или перетащите файл в окно</p>
          </div>
        </main>
      )}
      <Toaster position="bottom-right" />
    </div>
  );
}