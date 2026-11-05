import { useEffect, useRef, useState } from "react";
import { Bot, Copy, Eraser, Loader2, RefreshCw, Send, Square, Trash2 } from "lucide-react";
import { toast } from "sonner";
import * as Bindings from "../../wailsjs/go/main/App";
import type { main } from "../../wailsjs/go/models";
import { onEvent, offEvent } from "@/lib/wails";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import MarkdownMessage from "@/components/MarkdownMessage";
import { cn } from "@/lib/utils";

type ToolEvent = { name: string; args?: string; result?: string };
type ChatMessage = { role: "user" | "assistant"; text: string; tools: ToolEvent[] };

const DEFAULT_MODEL: Record<string, string> = {
  groq: "qwen/qwen3.8-27b",
  openrouter: "nvidia/nemotron-3-ultra-550b-a55b:free",
  ollama: "",
  llamacpp: "",
};

const DEFAULT_URL: Record<string, string> = {
  ollama: "http://127.0.0.1:11434/v1",
  llamacpp: "http://127.0.0.1:9931/v1",
};

const PROVIDER_LABEL: Record<string, string> = {
  groq: "GroqCloud",
  openrouter: "OpenRouter",
  ollama: "Ollama",
  llamacpp: "llama.cpp",
};

function localURL(settings: main.AssistantSettings | null, provider: string): string {
  if (!settings) return DEFAULT_URL[provider] ?? "";
  return provider === "ollama" ? settings.ollamaUrl || DEFAULT_URL.ollama : settings.llamaCppUrl || DEFAULT_URL.llamacpp;
}

type Props = {
  onChange: () => Promise<void> | void;
  onOpenFormulas: () => void;
  onApplyToEditor: (p: main.AssistantProposal) => void;
};

function limitsLabel(l: main.AssistantLimits | null, provider: string): string {
  if (!l || l.provider !== provider) return "";
  if (provider === "openrouter" && l.freeRequestsLimit > 0) {
    return `${l.freeRequestsLimit - l.freeRequestsRemaining}/${l.freeRequestsLimit} free`;
  }
  const parts: string[] = [];
  if (l.requestsLimit > 0) parts.push(`${Math.min(l.requestsRemaining, l.requestsLimit)}/${l.requestsLimit} RPD`);
  if (l.tokensLimit > 0) parts.push(`${Math.round(Math.min(l.tokensRemaining, l.tokensLimit) / 1000)}k/${Math.round(l.tokensLimit / 1000)}k TPM`);
  return parts.join(" · ");
}

export default function AssistantPage({ onChange, onOpenFormulas, onApplyToEditor }: Props) {
  const [settings, setSettings] = useState<main.AssistantSettings | null>(null);
  const [models, setModels] = useState<main.AssistantModel[]>([]);
  const [groqKey, setGroqKey] = useState("");
  const [openRouterKey, setOpenRouterKey] = useState("");
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState("");
  const [streaming, setStreaming] = useState(false);
  const [loadingModels, setLoadingModels] = useState(false);
  const [error, setError] = useState("");
  const [errorDetail, setErrorDetail] = useState("");
  const [proposal, setProposal] = useState<main.AssistantProposal | null>(null);
  const [diff, setDiff] = useState<main.FormulaDiff | null>(null);
  const [limits, setLimits] = useState<main.AssistantLimits | null>(null);
  const [waitInfo, setWaitInfo] = useState<{ seconds: number; reason: string } | null>(null);
  const bottomRef = useRef<HTMLDivElement>(null);

  const refreshLimits = async () => {
    try {
      setLimits(await Bindings.GetAssistantLimits());
    } catch {
      // Limits are advisory; ignore transient failures.
    }
  };

  const showProposal = async (p: main.AssistantProposal) => {
    setProposal(p);
    try {
      const current = await Bindings.GetActiveFormulaExpression();
      setDiff(await Bindings.DiffFormula(current, p.expression));
    } catch {
      setDiff(null);
    }
  };

  const reloadModels = async (provider: string, baseURL = "", refresh = false) => {
    setLoadingModels(true);
    try {
      const list = await Bindings.ListAssistantModels(provider, baseURL, refresh);
      setModels(list);
      setSettings((s) => {
        if (!s) return s;
        if (list.some((m) => m.id === s.model)) return s;
        return { ...s, model: DEFAULT_MODEL[provider] ?? list[0]?.id ?? s.model };
      });
    } catch (e) {
      setError(String(e));
    } finally {
      setLoadingModels(false);
    }
  };

  const persist = async (next: main.AssistantSettings, keys?: { groq?: string; openrouter?: string }) => {
    await Bindings.SaveAssistantSettings(next, keys?.groq ?? "", keys?.openrouter ?? "");
    const fresh = await Bindings.GetAssistantSettings();
    setSettings(fresh);
    return fresh;
  };

  useEffect(() => {
    void (async () => {
      try {
        const s = await Bindings.GetAssistantSettings();
        setSettings(s);
        await reloadModels(s.provider, localURL(s, s.provider));
        await refreshLimits();
        const history = await Bindings.GetAssistantHistory();
        setMessages(
          (history.messages ?? []).map((m) => ({
            role: m.role === "assistant" ? ("assistant" as const) : ("user" as const),
            text: m.text,
            tools: [],
          }))
        );
        if (history.proposal) await showProposal(history.proposal);
      } catch (e) {
        setError(String(e));
      }
    })();

    onEvent("assistantWait", (payload: { seconds?: number; reason?: string } | null) => {
      setWaitInfo({ seconds: payload?.seconds ?? 0, reason: payload?.reason ?? "" });
    });
    onEvent("assistantDelta", (payload: { text?: string } | null) => {
      setWaitInfo(null);
      const text = payload?.text ?? "";
      if (!text) return;
      setMessages((prev) => {
        const next = [...prev];
        const last = next[next.length - 1];
        if (last?.role === "assistant") {
          next[next.length - 1] = { ...last, text: last.text + text };
        } else {
          next.push({ role: "assistant", text, tools: [] });
        }
        return next;
      });
    });
    onEvent("assistantTool", (payload: { name?: string; args?: string } | null) => {
      const tool: ToolEvent = { name: payload?.name ?? "tool", args: payload?.args };
      setMessages((prev) => {
        const next = [...prev];
        const last = next[next.length - 1];
        if (last?.role === "assistant") {
          next[next.length - 1] = { ...last, tools: [...last.tools, tool] };
        } else {
          next.push({ role: "assistant", text: "", tools: [tool] });
        }
        return next;
      });
    });
    onEvent("assistantToolResult", (payload: { name?: string; result?: string } | null) => {
      setMessages((prev) => {
        const next = [...prev];
        const last = next[next.length - 1];
        if (last?.role !== "assistant") return prev;
        const tools = last.tools.map((t, i) =>
          i === last.tools.length - 1 && t.name === (payload?.name ?? t.name)
            ? { ...t, result: payload?.result ?? "" }
            : t
        );
        next[next.length - 1] = { ...last, tools };
        return next;
      });
    });
    onEvent("assistantProposal", (payload: main.AssistantProposal | null) => {
      if (payload) void showProposal(payload);
    });
    onEvent("assistantLimits", (payload: main.AssistantLimits | null) => {
      if (payload) setLimits(payload);
    });
    onEvent("assistantDone", (payload: { ok?: boolean; error?: string; detail?: string } | null) => {
      setStreaming(false);
      setWaitInfo(null);
      if (!payload?.ok) {
        setError(payload?.error ?? "Ошибка ассистента");
        setErrorDetail(payload?.detail ?? "");
      }
      void refreshLimits();
    });
    return () => {
      offEvent("assistantDelta");
      offEvent("assistantTool");
      offEvent("assistantToolResult");
      offEvent("assistantProposal");
      offEvent("assistantLimits");
      offEvent("assistantWait");
      offEvent("assistantDone");
    };
  }, []);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, streaming]);

  const saveSettings = async () => {
    if (!settings) return;
    try {
      await persist(settings, { groq: groqKey, openrouter: openRouterKey });
      setGroqKey("");
      setOpenRouterKey("");
      await reloadModels(settings.provider, localURL(settings, settings.provider));
      await refreshLimits();
      toast.success("Настройки ассистента сохранены");
    } catch (e) {
      setError(String(e));
    }
  };

  const testConnection = async () => {
    if (!settings) return;
    try {
      await persist(settings);
      await Bindings.TestAssistantConnection();
      await refreshLimits();
      toast.success("Подключение работает");
      setError("");
    } catch (e) {
      setError(String(e));
      toast.error(String(e));
    }
  };

  const changeProvider = async (provider: string) => {
    if (!settings) return;
    const baseURL = provider === "ollama" ? settings.ollamaUrl : provider === "llamacpp" ? settings.llamaCppUrl : "";
    const list = await Bindings.ListAssistantModels(provider, baseURL, false).catch(() => [] as main.AssistantModel[]);
    setModels(list);
    const model = DEFAULT_MODEL[provider] || list[0]?.id || "";
    const next = { ...settings, provider, model };
    setSettings(next);
    setError("");
    if (!model) {
      setError("Модели не найдены. Проверьте URL и запустите локальный сервер.");
      return;
    }
    try {
      await persist(next);
      await refreshLimits();
    } catch (e) {
      setError(String(e));
    }
  };

  const changeModel = async (model: string) => {
    if (!settings) return;
    const next = { ...settings, model };
    setSettings(next);
    try {
      await persist(next);
    } catch (e) {
      setError(String(e));
    }
  };

  const send = async () => {
    const text = input.trim();
    if (!text || streaming || !settings) return;
    try {
      await persist(settings);
    } catch (e) {
      setError(String(e));
      return;
    }
    setInput("");
    setError("");
    setErrorDetail("");
    setProposal(null);
    setDiff(null);
    setMessages((prev) => [...prev, { role: "user", text, tools: [] }]);
    setStreaming(true);
    setWaitInfo(null);
    try {
      await Bindings.SendAssistantMessage(text);
    } catch (e) {
      setStreaming(false);
      setError(String(e));
    }
  };

  const clear = async () => {
    await Bindings.ClearAssistantHistory();
    setMessages([]);
    setProposal(null);
    setDiff(null);
    setError("");
    setErrorDetail("");
  };

  const saveProposal = async () => {
    if (!proposal) return;
    try {
      await Bindings.UpsertPreset({
        id: crypto.randomUUID(),
        name: proposal.name || "Формула от ассистента",
        kind: "expression",
        expression: proposal.expression,
        weights: {},
      });
      await onChange();
      toast.success("Формула сохранена");
      onOpenFormulas();
    } catch (e) {
      toast.error(String(e));
    }
  };

  const currentModel = models.find((m) => m.id === settings?.model);
  const isLocal = settings?.provider === "ollama" || settings?.provider === "llamacpp";
  const keySet = settings?.provider === "openrouter" ? settings?.hasOpenRouterKey : settings?.hasGroqKey;
  const providerLabel = PROVIDER_LABEL[settings?.provider ?? ""] ?? "Ассистент";
  const limitText = isLocal ? "" : limitsLabel(limits, settings?.provider ?? "");
  const limitLow =
    !isLocal &&
    limits &&
    ((limits.requestsLimit > 0 && limits.requestsRemaining <= limits.requestsLimit * 0.1) ||
      (limits.freeRequestsLimit > 0 && limits.freeRequestsRemaining <= limits.freeRequestsLimit * 0.1));

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3 p-4">
      <div className="flex items-center justify-between gap-3">
        <div className="flex min-w-0 items-center gap-2">
          <h2 className="flex shrink-0 items-center gap-2 text-lg font-bold">
            <Bot className="size-5" /> Ассистент формулы
          </h2>
          {settings && (
            <div className="flex min-w-0 items-center gap-1.5 text-[11px]">
              <Badge variant="outline">{providerLabel}</Badge>
              <span className="max-w-56 truncate font-mono text-muted-foreground" title={settings.model}>
                {currentModel?.name ?? settings.model}
              </span>
              <Badge variant={isLocal ? "outline" : "gold"} className="text-[9px]">{isLocal ? "LOCAL" : "FREE"}</Badge>
              {limitText && (
                <span className={cn("font-mono tabular-nums", limitLow ? "text-destructive" : "text-muted-foreground")}>
                  {limitText}
                </span>
              )}
            </div>
          )}
        </div>
        <div className="flex shrink-0 items-center gap-2">
          {streaming && (
            <Button variant="outline" size="sm" onClick={() => void Bindings.CancelAssistant()}>
              <Square className="mr-1.5 size-3.5" /> Остановить
            </Button>
          )}
          <Button variant="outline" size="sm" onClick={() => void clear()}>
            <Eraser className="mr-1.5 size-3.5" /> Очистить
          </Button>
        </div>
      </div>

      {settings && (
        <div className="grid gap-2 rounded-lg border border-border bg-card/40 p-3 md:grid-cols-[180px_1fr_auto]">
          <div className="space-y-1">
            <label className="label-caps">Провайдер</label>
            <Select value={settings.provider} onValueChange={(v) => void changeProvider(v)}>
              <SelectTrigger className="h-8 w-full"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="groq">GroqCloud</SelectItem>
                <SelectItem value="openrouter">OpenRouter</SelectItem>
                <SelectItem value="ollama">Ollama (локально)</SelectItem>
                <SelectItem value="llamacpp">llama.cpp (локально)</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1">
            <label className="label-caps">{isLocal ? "Локальная модель" : "Бесплатная модель"}</label>
            <Select value={settings.model} onValueChange={(v) => void changeModel(v)}>
              <SelectTrigger className="h-8 w-full">
                <SelectValue placeholder="Выберите модель" />
              </SelectTrigger>
              <SelectContent>
                {models.map((m) => (
                  <SelectItem key={m.id} value={m.id}>
                    {m.name} · {Math.round(m.contextLength / 1000)}k
                  </SelectItem>
                ))}
                {!models.some((m) => m.id === settings.model) && settings.model && (
                  <SelectItem value={settings.model}>{settings.model}</SelectItem>
                )}
              </SelectContent>
            </Select>
            {currentModel?.note && <p className="text-[10px] text-muted-foreground">{currentModel.note}</p>}
            {currentModel?.privacy && <p className="text-[10px] text-muted-foreground">{currentModel.privacy}</p>}
            {isLocal && currentModel && !currentModel.supportsTools && (
              <p className="text-[10px] text-destructive">Модель не поддерживает tool calling.</p>
            )}
            {settings.provider === "llamacpp" && (
              <p className="text-[10px] text-amber-300">
                Запускайте llama-server с `--jinja`; для стабильного id используйте `--alias`.
              </p>
            )}
          </div>
          {!isLocal && (
            <div className="flex items-end gap-2">
              <Button variant="outline" size="sm" onClick={() => void reloadModels(settings.provider, "", true)} disabled={loadingModels}>
                <RefreshCw className={cn("mr-1.5 size-3.5", loadingModels && "animate-spin")} /> Обновить
              </Button>
              <Button variant="ghost" size="sm" onClick={() => void refreshLimits()}>Лимиты</Button>
            </div>
          )}

          <div className="space-y-1 md:col-span-3">
            {isLocal ? (
              <div className="flex flex-wrap items-end gap-2">
                <div className="min-w-72 flex-1 space-y-1">
                  <label className="label-caps">URL сервера</label>
                  <Input
                    value={settings.provider === "ollama" ? settings.ollamaUrl : settings.llamaCppUrl}
                    onChange={(e) =>
                      setSettings({
                        ...settings,
                        ...(settings.provider === "ollama"
                          ? { ollamaUrl: e.target.value }
                          : { llamaCppUrl: e.target.value }),
                      })
                    }
                    placeholder={DEFAULT_URL[settings.provider]}
                    className="h-8"
                  />
                </div>
                <Button size="sm" onClick={() => void saveSettings()}>Сохранить</Button>
                <Button variant="outline" size="sm" onClick={() => void testConnection()}>Проверить</Button>
                <Button variant="outline" size="sm" onClick={() => void reloadModels(settings.provider, localURL(settings, settings.provider), true)} disabled={loadingModels}>
                  <RefreshCw className={cn("mr-1.5 size-3.5", loadingModels && "animate-spin")} /> Модели
                </Button>
              </div>
            ) : (
              <div className="space-y-1">
                <label className="label-caps">
                  API-ключ {providerLabel}
                  {keySet && <span className="ml-2 normal-case text-radiant">сохранён</span>}
                </label>
                <div className="flex flex-wrap gap-2">
                  <Input
                    type="password"
                    value={settings.provider === "groq" ? groqKey : openRouterKey}
                    onChange={(e) => (settings.provider === "groq" ? setGroqKey(e.target.value) : setOpenRouterKey(e.target.value))}
                    placeholder={keySet ? "оставьте пустым, чтобы не менять" : "вставьте API-ключ"}
                    className="h-8 max-w-md flex-1"
                  />
                  <Button size="sm" onClick={() => void saveSettings()}>Сохранить</Button>
                  <Button variant="outline" size="sm" onClick={() => void testConnection()}>Проверить</Button>
                  {keySet && (
                    <Button
                      variant="ghost"
                      size="sm"
                      className="text-destructive"
                      onClick={async () => {
                        await Bindings.ClearAssistantKey(settings.provider);
                        const s = await Bindings.GetAssistantSettings();
                        setSettings(s);
                        toast.success("Ключ удалён");
                      }}
                    >
                      <Trash2 className="mr-1.5 size-3.5" /> Удалить ключ
                    </Button>
                  )}
                </div>
              </div>
            )}
            <label className="flex items-center gap-2 pt-1 text-[11px] text-muted-foreground">
              <input
                type="checkbox"
                checked={settings.sendPlayerNames}
                onChange={(e) => {
                  const next = { ...settings, sendPlayerNames: e.target.checked };
                  setSettings(next);
                  void persist(next).catch((err) => setError(String(err)));
                }}
              />
              Отправлять имена игроков и Steam ID (по умолчанию маскируются)
            </label>
            {settings.provider === "openrouter" && (
              <div className="rounded-md border border-amber-500/30 bg-amber-500/10 p-2 text-[11px] text-amber-200">
                OpenRouter free-модели: часть маршрутов может использовать данные для обучения. Не отправляйте секретные формулы; имена по умолчанию маскируются.
              </div>
            )}
            {isLocal && (
              <div className="rounded-md border border-radiant/30 bg-radiant/10 p-2 text-[11px] text-radiant">
                Локальный сервер: данные матча и формулы не покидают компьютер.
              </div>
            )}
          </div>
        </div>
      )}

      {waitInfo && (
        <div className="rounded-md border border-amber-500/30 bg-amber-500/10 p-2 text-sm text-amber-200">
          Лимит Groq: ждём {waitInfo.seconds} с… {waitInfo.reason === "tokens" ? "(токены)" : "(запросы)"}
        </div>
      )}

      {error && (
        <div className="space-y-1 rounded-md border border-destructive/40 bg-destructive/10 p-2 text-sm text-destructive">
          <div>{error}</div>
          {errorDetail && (
            <details className="text-[11px] text-destructive/80">
              <summary className="cursor-pointer">Детали провайдера</summary>
              <pre className="mt-1 max-h-32 overflow-auto whitespace-pre-wrap break-all">{errorDetail}</pre>
            </details>
          )}
          <Button variant="outline" size="sm" onClick={() => void clear()}>Очистить историю</Button>
        </div>
      )}

      <div className="min-h-0 flex-1 overflow-y-auto rounded-lg border border-border bg-background/40 p-3">
        {messages.length === 0 ? (
          <div className="flex h-full flex-col items-center justify-center gap-2 text-center text-sm text-muted-foreground">
            <Bot className="size-8 opacity-50" />
            <p>Спросите про текущую формулу, рейтинг игроков или попросите предложить улучшение.</p>
            <p className="max-w-md text-[11px]">
              Ассистент читает матч и формулы через инструменты. Числа не выдумываются: они приходят из приложения.
            </p>
          </div>
        ) : (
          <div className="space-y-3">
            {messages.map((m, i) => (
              <div key={i} className={cn("flex", m.role === "user" ? "justify-end" : "justify-start")}>
                <div className={cn(
                  "max-w-[85%] rounded-lg px-3 py-2 text-sm",
                  m.role === "user" ? "whitespace-pre-wrap bg-gold/15 text-foreground" : "border border-border bg-card"
                )}>
                  {m.role === "assistant" ? <MarkdownMessage text={m.text} /> : m.text}
                  {m.tools.length > 0 && (
                    <div className="mt-2 space-y-1 border-t border-border/60 pt-2">
                      {m.tools.map((t, j) => (
                        <details key={j} className="text-[10px] text-muted-foreground">
                          <summary className="cursor-pointer font-mono">{t.name}{t.result ? " ✓" : "…"}</summary>
                          {t.args && <pre className="mt-1 max-h-24 overflow-auto whitespace-pre-wrap break-all">{t.args}</pre>}
                          {t.result && <pre className="mt-1 max-h-24 overflow-auto whitespace-pre-wrap break-all">{t.result}</pre>}
                        </details>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            ))}
            {streaming && !messages.some((m) => m.role === "assistant" && m.text) && (
              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                <Loader2 className="size-3.5 animate-spin" /> Ассистент думает…
              </div>
            )}
            <div ref={bottomRef} />
          </div>
        )}
      </div>

      {proposal && (
        <div className="space-y-2 rounded-lg border border-gold/40 bg-gold/5 p-3">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div>
              <div className="text-sm font-semibold">{proposal.name}</div>
              {proposal.description && <div className="text-[11px] text-muted-foreground">{proposal.description}</div>}
            </div>
            <div className="flex flex-wrap gap-2">
              <Button variant="outline" size="sm" onClick={() => void navigator.clipboard.writeText(proposal.expression)}>
                <Copy className="mr-1.5 size-3.5" /> Копировать
              </Button>
              <Button variant="outline" size="sm" onClick={() => onApplyToEditor(proposal)}>
                Применить в редактор
              </Button>
              <Button size="sm" onClick={() => void saveProposal()}>Сохранить как формулу</Button>
            </div>
          </div>
          {diff && (
            <div className="grid gap-2 md:grid-cols-2">
              <div>
                <div className="label-caps mb-1">Сейчас</div>
                <pre className="max-h-32 overflow-auto rounded border border-border bg-background p-2 font-mono text-[11px] whitespace-pre-wrap break-all">
                  {diff.current || "—"}
                </pre>
              </div>
              <div>
                <div className="label-caps mb-1">Предложено</div>
                <pre className="max-h-32 overflow-auto rounded border border-border bg-background p-2 font-mono text-[11px] whitespace-pre-wrap break-all">
                  {diff.proposed}
                </pre>
              </div>
              {diff.added.length > 0 && (
                <div className="text-[11px] md:col-span-2">
                  <span className="text-radiant">Добавлено: </span>
                  {diff.added.join(" · ")}
                </div>
              )}
              {diff.removed.length > 0 && (
                <div className="text-[11px] md:col-span-2">
                  <span className="text-destructive">Удалено: </span>
                  {diff.removed.join(" · ")}
                </div>
              )}
            </div>
          )}
          {!diff && (
            <pre className="max-h-32 overflow-auto rounded border border-border bg-background p-2 font-mono text-[11px] whitespace-pre-wrap break-all">
              {proposal.expression}
            </pre>
          )}
        </div>
      )}

      <div className="flex gap-2">
        <Input
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              void send();
            }
          }}
          placeholder="Спросите ассистента…"
          disabled={streaming}
        />
        <Button onClick={() => void send()} disabled={streaming || !input.trim()}>
          <Send className="mr-1.5 size-3.5" /> Отправить
        </Button>
      </div>
    </div>
  );
}
