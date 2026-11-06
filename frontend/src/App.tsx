import { useEffect, useState } from "react";
import { Download, FolderOpen, Loader2, Settings } from "lucide-react";
import * as Bindings from "../wailsjs/go/main/App";
import type { formula, main } from "../wailsjs/go/models";
import { onEvent, offEvent } from "@/lib/wails";
import type { MatchInfo } from "@/components/InfoBar";
import HomePage from "@/pages/HomePage";
import FormulasPage from "@/pages/FormulasPage";
import AssistantPage from "@/pages/AssistantPage";
import SettingsPage from "@/pages/SettingsPage";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Progress } from "@/components/ui/progress";
import { Toaster } from "@/components/ui/sonner";
import { cn } from "@/lib/utils";

type PendingFormula = { name: string; description: string; expression: string; nonce: number };

const NAV: { id: "home" | "formulas" | "assistant" | "settings"; label: string }[] = [
  { id: "home", label: "Главная" },
  { id: "formulas", label: "Формулы" },
  { id: "assistant", label: "Ассистент" },
  { id: "settings", label: "Настройки" },
];

export default function App() {
  const [views, setViews] = useState<main.PlayerView[] | null>(null);
  const [presets, setPresets] = useState<formula.Preset[]>([]);
  const [activeId, setActiveId] = useState("");
  const [status, setStatus] = useState("idle");
  const [error, setError] = useState("");
  const [showUpdates, setShowUpdates] = useState(false);
  const [loading, setLoading] = useState(true);
  const [matchInfo, setMatchInfo] = useState<MatchInfo | null>(null);
  const [page, setPage] = useState<"home" | "formulas" | "assistant" | "settings">("home");
  const [pendingFormula, setPendingFormula] = useState<PendingFormula | null>(null);
  const [appVersion, setAppVersion] = useState("");
  const [update, setUpdate] = useState<main.UpdateInfo | null>(null);
  const [showUpdate, setShowUpdate] = useState(false);
  const [checkingUpdate, setCheckingUpdate] = useState(false);
  const [updateChecked, setUpdateChecked] = useState(false);
  const [updateProgress, setUpdateProgress] = useState<{ stage: string; percent: number } | null>(null);
  const [updateError, setUpdateError] = useState("");

  const refreshPresets = async () => {
    const [list, active, ver] = await Promise.all([
      Bindings.ListPresets(),
      Bindings.ActivePresetID(),
      Bindings.GetAppVersion(),
    ]);
    setPresets(list);
    setActiveId(active);
    setAppVersion(ver);
  };

  const checkForUpdates = async (manual = false) => {
    setCheckingUpdate(true);
    if (manual) setUpdateError("");
    try {
      const info = await Bindings.CheckForUpdates();
      setAppVersion(info.currentVersion);
      setUpdate(info.available ? info : null);
      setUpdateChecked(true);
    } catch (e) {
      if (manual) setUpdateError(String(e));
    } finally {
      setCheckingUpdate(false);
    }
  };

  const startUpdate = async () => {
    setUpdateError("");
    setUpdateProgress({ stage: "download", percent: 0 });
    try {
      await Bindings.InstallUpdate();
    } catch (e) {
      setUpdateError(String(e));
      setUpdateProgress(null);
    }
  };

  useEffect(() => {
    void (async () => {
      try {
        await refreshPresets();
        void checkForUpdates();
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
    onEvent("updateProgress", (payload: { stage?: string; percent?: number } | null) => {
      setUpdateProgress({ stage: payload?.stage ?? "download", percent: payload?.percent ?? 0 });
    });
    onEvent("updateDone", (payload: { ok?: boolean; error?: string } | null) => {
      if (!payload?.ok) {
        setUpdateError(payload?.error ?? "Не удалось обновить приложение");
        setUpdateProgress(null);
      }
    });
    return () => {
      offEvent("parseStart");
      offEvent("parseDone");
      offEvent("resultUpdated");
      offEvent("matchInfo");
      offEvent("updateProgress");
      offEvent("updateDone");
    };
  }, []);

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
          {update?.available && (
            <Button variant="outline" size="sm" onClick={() => setShowUpdate(true)} className="h-7">
              <Download className="mr-1.5 size-3.5" /> Обновить v{update.latestVersion}
            </Button>
          )}
          {views && (
            <Button variant="outline" size="sm" onClick={() => void Bindings.ChooseReplay()} className="h-7">
              <FolderOpen className="mr-1.5 size-3.5" /> Открыть
            </Button>
          )}
          <Button variant="ghost" size="icon" className="size-7" onClick={() => setShowUpdates((s) => !s)} title="Версия и обновления">
            <Settings className="size-4" />
          </Button>
        </div>
      </header>

      {showUpdates && (
        <div className="flex flex-wrap items-center gap-2 border-b border-border bg-background/60 px-4 py-2">
          <span className="text-xs text-muted-foreground">Версия:</span>
          <span className="font-mono text-xs">{appVersion || "dev"}</span>
          <Button variant="outline" size="sm" onClick={() => void checkForUpdates(true)} disabled={checkingUpdate}>
            {checkingUpdate && <Loader2 className="mr-1.5 size-3.5 animate-spin" />}
            Проверить обновления
          </Button>
          {update?.available ? (
            <Button size="sm" onClick={() => setShowUpdate(true)}>
              Доступна v{update.latestVersion}
            </Button>
          ) : updateChecked ? (
            <span className="text-[11px] text-muted-foreground">Обновлений нет</span>
          ) : null}
          {updateError && <span className="text-[11px] text-destructive">{updateError}</span>}
        </div>
      )}

      {loading ? (
        <div className="flex flex-1 items-center justify-center text-muted-foreground">
          <Loader2 className="mr-2 size-4 animate-spin" /> Загрузка…
        </div>
      ) : page === "formulas" ? (
        <FormulasPage
          presets={presets}
          activeId={activeId}
          onChange={refreshPresets}
          pendingFormula={pendingFormula}
          onPendingConsumed={() => setPendingFormula(null)}
        />
      ) : page === "assistant" ? (
        <AssistantPage
          onChange={refreshPresets}
          onOpenFormulas={() => setPage("formulas")}
          onApplyToEditor={(p) => {
            setPendingFormula({ ...p, nonce: Date.now() });
            setPage("formulas");
          }}
        />
      ) : page === "settings" ? (
        <SettingsPage />
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
      <Dialog open={showUpdate} onOpenChange={setShowUpdate}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Обновление v{update?.latestVersion || ""}</DialogTitle>
            <DialogDescription>Текущая версия: {appVersion || "dev"}</DialogDescription>
          </DialogHeader>
          {update?.notes && (
            <div className="max-h-56 overflow-auto rounded-md border border-border bg-muted/30 p-2 text-[11px] whitespace-pre-wrap text-muted-foreground">
              {update.notes}
            </div>
          )}
          {updateProgress && (
            <div className="space-y-1">
              <Progress value={updateProgress.percent} />
              <div className="flex justify-between text-[11px] text-muted-foreground">
                <span>
                  {updateProgress.stage === "download"
                    ? "Скачивание…"
                    : updateProgress.stage === "verify"
                      ? "Проверка…"
                      : "Установка…"}
                </span>
                <span>{updateProgress.percent}%</span>
              </div>
            </div>
          )}
          {updateError && (
            <div className="rounded-md border border-destructive/40 bg-destructive/10 p-2 text-left text-sm text-destructive">
              {updateError}
            </div>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowUpdate(false)} disabled={!!updateProgress}>
              Позже
            </Button>
            <Button onClick={() => void startUpdate()} disabled={!!updateProgress}>
              {updateProgress ? "Обновление…" : "Обновить и перезапустить"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <Toaster position="bottom-right" />
    </div>
  );
}