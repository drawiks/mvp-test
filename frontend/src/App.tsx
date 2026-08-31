import { useEffect, useState } from "react";
import { FolderOpen, Loader2, Settings, Variable } from "lucide-react";
import * as Bindings from "../wailsjs/go/main/App";
import type { formula, main } from "../wailsjs/go/models";
import { onEvent, offEvent } from "@/lib/wails";
import MvpCards from "@/components/MvpCards";
import StatsTable from "@/components/StatsTable";
import WeightsPanel from "@/components/WeightsPanel";
import ManageVariables from "@/components/ManageVariables";
import InfoBar, { type MatchInfo } from "@/components/InfoBar";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Toaster } from "@/components/ui/sonner";
import { GradientHeading } from "@/components/ui/gradient-heading";
import { Badge } from "@/components/ui/badge";

export default function App() {
  const [views, setViews] = useState<main.PlayerView[] | null>(null);
  const [presets, setPresets] = useState<formula.Preset[]>([]);
  const [activeId, setActiveId] = useState("");
  const [status, setStatus] = useState("idle"); // idle | parsing | error
  const [error, setError] = useState("");
  const [showSettings, setShowSettings] = useState(false);
  const [parserURL, setParserURL] = useState("");
  const [loading, setLoading] = useState(true);
  const [matchInfo, setMatchInfo] = useState<MatchInfo | null>(null);

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

  // Native OS drag & drop (Wails): absolute paths arrive here for any drop
  // over the window; filter to replay/extensions and parse the first match.
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

  const activePreset = presets.find((p) => p.id === activeId) ?? null;

  return (
    <div
      className="flex h-screen flex-col overflow-hidden bg-background text-foreground"
      style={{ ["--wails-drop-target" as string]: "drop" }}
      data-drop-zone
    >
      <header className="flex items-center gap-4 border-b border-border bg-card/60 px-4 py-2.5">
        <div className="flex items-end gap-2">
          <span className="text-lg">⚔️</span>
          <GradientHeading variant="gold" size="sm" weight="semi" className="leading-none">
            MVP-Test
          </GradientHeading>
          <span className="mb-0.5 text-[10px] font-medium text-muted-foreground">by drawiks</span>
        </div>
        <div className="ml-auto flex items-center gap-3">
          <Badge
            variant={status === "parsing" ? "outline" : status === "error" ? "destructive" : "radiant"}
            className={status === "parsing" ? "border-sky-500/30 bg-sky-500/15 text-sky-300" : ""}
          >
            {status === "parsing" && <Loader2 className="size-3 animate-spin" />}
            {status === "parsing" ? "Парсим реплей…" : status === "error" ? "Ошибка" : views ? "Готово" : "Ожидание реплея"}
          </Badge>
          {views && (
            <Button variant="outline" size="sm" onClick={() => void Bindings.ChooseReplay()} className="h-8">
              <FolderOpen className="mr-1.5 size-3.5" /> Открыть
            </Button>
          )}
          <ManageVariables views={views}>
            <Button variant="ghost" size="sm" className="h-8">
              <Variable className="mr-1.5 size-3.5" /> Переменные
            </Button>
          </ManageVariables>
          <Button variant="ghost" size="icon" className="size-8" onClick={() => setShowSettings((s) => !s)}>
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
      ) : views && views.length > 0 ? (
        <>
          <InfoBar info={matchInfo} />
          <main className="flex min-h-0 flex-1 gap-4 p-4">
            <section className="flex min-h-0 min-w-0 flex-1 flex-col gap-4">
              <MvpCards views={views} preset={activePreset} />
              {error && <div className="rounded-lg border border-destructive/40 bg-destructive/10 p-2 text-sm text-destructive">{error}</div>}
              <div className="min-h-0 flex-1">
                <StatsTable views={views} preset={activePreset} />
              </div>
            </section>
            <aside className="w-88 min-h-0 shrink-0">
              <WeightsPanel presets={presets} activeId={activeId} onChange={refreshPresets} />
            </aside>
          </main>
        </>
      ) : (
        <main className="flex flex-1 items-center justify-center">
          <div className="w-full max-w-md text-center">
            <GradientHeading variant="gold" size="lg" weight="semi" className="mb-3">
              MVP-Test
            </GradientHeading>
            {error && <div className="mb-4 rounded-lg border border-destructive/40 bg-destructive/10 p-2 text-sm text-destructive">{error}</div>}
            <div className="flex justify-center gap-3">
              <Button size="lg" onClick={() => void Bindings.ChooseReplay()} disabled={status === "parsing"}>
                {status === "parsing" ? <Loader2 className="mr-2 size-4 animate-spin" /> : <FolderOpen className="mr-2 size-4" />}
                Открыть реплей
              </Button>
            </div>
            <p className="mt-4 text-xs text-muted-foreground">
              Горячая клавиша: Ctrl+O · или перетащите файл в окно
            </p>
          </div>
        </main>
      )}
      <Toaster position="bottom-right" />
    </div>
  );
}