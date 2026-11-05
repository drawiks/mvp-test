import { useEffect, useRef, useState } from "react";
import { Bot, Copy, Eraser, Loader2, RefreshCw, Send, Square, Trash2 } from "lucide-react";
import { toast } from "sonner";
import * as Bindings from "../../wailsjs/go/main/App";
import type { main } from "../../wailsjs/go/models";
import { onEvent, offEvent } from "@/lib/wails";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { cn } from "@/lib/utils";

type ToolEvent = { name: string; args?: string; result?: string };
type ChatMessage = { role: "user" | "assistant"; text: string; tools: ToolEvent[] };

const DEFAULT_MODEL: Record<string, string> = {
  groq: "qwen/qwen3.8-27b",
  openrouter: "nvidia/nemotron-3-ultra-550b-a55b:free",
};

type Props = {
  onChange: () => Promise<void> | void;
  onOpenFormulas: () => void;
};

export default function AssistantPage({ onChange, onOpenFormulas }: Props) {
  const [settings, setSettings] = useState<main.AssistantSettings | null>(null);
  const [models, setModels] = useState<main.AssistantModel[]>([]);
  const [groqKey, setGroqKey] = useState("");
  const [openRouterKey, setOpenRouterKey] = useState("");
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState("");
  const [streaming, setStreaming] = useState(false);
  const [loadingModels, setLoadingModels] = useState(false);
  const [error, setError] = useState("");
  const [proposal, setProposal] = useState<main.AssistantProposal | null>(null);
  const bottomRef = useRef<HTMLDivElement>(null);

  const reloadModels = async (provider: string, refresh = false) => {
    setLoadingModels(true);
    try {
      const list = await Bindings.ListAssistantModels(provider, refresh);
      setModels(list);
      setSettings((s) => {
        if (!s) return s;
        if (list.some((m) => m.id === s.model)) return s;
        return { ...s, model: DEFAULT_MODEL[provider] ?? list[0]?.id ?? s.model };
      });
    } catch (e) {
      if (refresh) setError(String(e));
    } finally {
      setLoadingModels(false);
    }
  };

  useEffect(() => {
    void (async () => {
      try {
        const s = await Bindings.GetAssistantSettings();
        setSettings(s);
        await reloadModels(s.provider);
      } catch (e) {
        setError(String(e));
      }
    })();

    onEvent("assistantDelta", (payload: { text?: string } | null) => {
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
      if (payload) setProposal(payload);
    });
    onEvent("assistantDone", (payload: { ok?: boolean; error?: string } | null) => {
      setStreaming(false);
      if (!payload?.ok) setError(payload?.error ?? "Ошибка ассистента");
    });
    return () => {
      offEvent("assistantDelta");
      offEvent("assistantTool");
      offEvent("assistantToolResult");
      offEvent("assistantProposal");
      offEvent("assistantDone");
    };
  }, []);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, streaming]);

  const saveSettings = async () => {
    if (!settings) return;
    try {
      await Bindings.SaveAssistantSettings(settings, groqKey, openRouterKey);
      setGroqKey("");
      setOpenRouterKey("");
      const s = await Bindings.GetAssistantSettings();
      setSettings(s);
      await reloadModels(s.provider, true);
      toast.success("Настройки ассистента сохранены");
    } catch (e) {
      setError(String(e));
    }
  };

  const testConnection = async () => {
    try {
      await Bindings.TestAssistantConnection();
      toast.success("Подключение работает");
      setError("");
    } catch (e) {
      setError(String(e));
      toast.error(String(e));
    }
  };

  const changeProvider = async (provider: string) => {
    if (!settings) return;
    const next = { ...settings, provider, model: DEFAULT_MODEL[provider] ?? "" };
    setSettings(next);
    setError("");
    await reloadModels(provider);
  };

  const send = async () => {
    const text = input.trim();
    if (!text || streaming) return;
    setInput("");
    setError("");
    setProposal(null);
    setMessages((prev) => [...prev, { role: "user", text, tools: [] }]);
    setStreaming(true);
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
    setError("");
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
  const keySet = settings?.provider === "openrouter" ? settings?.hasOpenRouterKey : settings?.hasGroqKey;

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3 p-4">
      <div className="flex items-center justify-between">
        <h2 className="flex items-center gap-2 text-lg font-bold">
          <Bot className="size-5" /> Ассистент формулы
        </h2>
        <div className="flex items-center gap-2">
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
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1">
            <label className="label-caps">Бесплатная модель</label>
            <Select
              value={settings.model}
              onValueChange={(v) => setSettings({ ...settings, model: v })}
            >
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
          </div>
          <div className="flex items-end gap-2">
            <Button variant="outline" size="sm" onClick={() => void reloadModels(settings.provider, true)} disabled={loadingModels}>
              <RefreshCw className={cn("mr-1.5 size-3.5", loadingModels && "animate-spin")} /> Обновить
            </Button>
          </div>

          <div className="space-y-1 md:col-span-3">
            <label className="label-caps">
              API-ключ {settings.provider === "groq" ? "GroqCloud" : "OpenRouter"}
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
            <label className="flex items-center gap-2 pt-1 text-[11px] text-muted-foreground">
              <input
                type="checkbox"
                checked={settings.sendPlayerNames}
                onChange={(e) => setSettings({ ...settings, sendPlayerNames: e.target.checked })}
              />
              Отправлять имена игроков и Steam ID (по умолчанию маскируются)
            </label>
          </div>
        </div>
      )}

      {error && (
        <div className="rounded-md border border-destructive/40 bg-destructive/10 p-2 text-sm text-destructive">
          {error}
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
                  "max-w-[85%] rounded-lg px-3 py-2 text-sm whitespace-pre-wrap",
                  m.role === "user" ? "bg-gold/15 text-foreground" : "border border-border bg-card"
                )}>
                  {m.text}
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
          <div className="flex items-center justify-between">
            <div>
              <div className="text-sm font-semibold">{proposal.name}</div>
              {proposal.description && <div className="text-[11px] text-muted-foreground">{proposal.description}</div>}
            </div>
            <div className="flex gap-2">
              <Button variant="outline" size="sm" onClick={() => void navigator.clipboard.writeText(proposal.expression)}>
                <Copy className="mr-1.5 size-3.5" /> Копировать
              </Button>
              <Button size="sm" onClick={() => void saveProposal()}>Сохранить как формулу</Button>
            </div>
          </div>
          <pre className="max-h-32 overflow-auto rounded border border-border bg-background p-2 font-mono text-[11px] whitespace-pre-wrap break-all">
            {proposal.expression}
          </pre>
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
