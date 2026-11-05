import { useEffect, useState } from "react";
import { KeyRound, RefreshCw, Settings as SettingsIcon, ShieldCheck, Trash2 } from "lucide-react";
import { toast } from "sonner";
import * as Bindings from "../../wailsjs/go/main/App";
import type { main } from "../../wailsjs/go/models";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { cn } from "@/lib/utils";

const PROVIDER_LABEL: Record<string, string> = {
  groq: "GroqCloud",
  openrouter: "OpenRouter",
  ollama: "Ollama",
  llamacpp: "llama.cpp",
};

const DEFAULT_URL: Record<string, string> = {
  ollama: "http://127.0.0.1:11434/v1",
  llamacpp: "http://127.0.0.1:9931/v1",
};

function Section({ title, icon, children }: { title: string; icon: React.ReactNode; children: React.ReactNode }) {
  return (
    <section className="rounded-lg border border-border bg-card/40 p-4">
      <h2 className="mb-3 flex items-center gap-2 text-sm font-bold">
        {icon} {title}
      </h2>
      {children}
    </section>
  );
}

export default function SettingsPage() {
  const [parserEngine, setParserEngine] = useState("odota");
  const [parserURL, setParserURL] = useState("");
  const [settings, setSettings] = useState<main.AssistantSettings | null>(null);
  const [groqKey, setGroqKey] = useState("");
  const [openRouterKey, setOpenRouterKey] = useState("");
  const [loadingModels, setLoadingModels] = useState(false);
  const [modelCount, setModelCount] = useState<number | null>(null);

  const load = async () => {
    const [engine, url, assistant] = await Promise.all([
      Bindings.GetParserEngine(),
      Bindings.GetParserURL(),
      Bindings.GetAssistantSettings(),
    ]);
    setParserEngine(engine);
    setParserURL(url);
    setSettings(assistant);
  };

  useEffect(() => {
    void load().catch((e) => toast.error(String(e)));
  }, []);

  const saveParser = async () => {
    try {
      await Bindings.SetParserEngine(parserEngine);
      if (parserEngine !== "manta") await Bindings.SetParserURL(parserURL);
      toast.success("Настройки парсера сохранены");
    } catch (e) {
      toast.error(String(e));
    }
  };

  const persistAssistant = async (withKeys: boolean) => {
    if (!settings) return;
    await Bindings.SaveAssistantSettings(settings, withKeys ? groqKey : "", withKeys ? openRouterKey : "");
    if (withKeys) {
      setGroqKey("");
      setOpenRouterKey("");
    }
    const fresh = await Bindings.GetAssistantSettings();
    setSettings(fresh);
    return fresh;
  };

  const saveAssistant = async () => {
    try {
      await persistAssistant(true);
      toast.success("Настройки ассистента сохранены");
    } catch (e) {
      toast.error(String(e));
    }
  };

  const testConnection = async () => {
    try {
      await persistAssistant(true);
      await Bindings.TestAssistantConnection();
      toast.success("Подключение работает");
    } catch (e) {
      toast.error(String(e));
    }
  };

  const refreshModels = async (provider: string, baseURL: string) => {
    setLoadingModels(true);
    try {
      const list = await Bindings.ListAssistantModels(provider, baseURL, true);
      setModelCount(list.length);
      toast.success(`Моделей найдено: ${list.length}`);
    } catch (e) {
      toast.error(String(e));
    } finally {
      setLoadingModels(false);
    }
  };

  const clearKey = async (provider: string) => {
    try {
      await Bindings.ClearAssistantKey(provider);
      const fresh = await Bindings.GetAssistantSettings();
      setSettings(fresh);
      toast.success("Ключ удалён");
    } catch (e) {
      toast.error(String(e));
    }
  };

  const provider = settings?.provider ?? "groq";
  const isLocal = provider === "ollama" || provider === "llamacpp";
  const providerLabel = PROVIDER_LABEL[provider] ?? provider;
  const localURL = provider === "ollama"
    ? settings?.ollamaUrl || DEFAULT_URL.ollama
    : settings?.llamaCppUrl || DEFAULT_URL.llamacpp;

  return (
    <div className="flex-1 overflow-y-auto p-4">
      <div className="mx-auto flex max-w-3xl flex-col gap-4">
        <h1 className="flex items-center gap-2 text-lg font-bold">
          <SettingsIcon className="size-5" /> Настройки
        </h1>

        <Section title="Парсер реплеев" icon={<RefreshCw className="size-4" />}>
          <div className="flex flex-wrap items-end gap-2">
            <div className="space-y-1">
              <label className="label-caps">Движок</label>
              <Select value={parserEngine} onValueChange={setParserEngine}>
                <SelectTrigger className="h-8 w-56"><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="odota">odota-cli (HTTP)</SelectItem>
                  <SelectItem value="manta">manta-cli (локально)</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="min-w-72 flex-1 space-y-1">
              <label className="label-caps">URL парсера</label>
              <Input
                value={parserURL}
                onChange={(e) => setParserURL(e.target.value)}
                disabled={parserEngine === "manta"}
                placeholder={parserEngine === "manta" ? "не используется" : "http://localhost:5600"}
                className="h-8"
              />
            </div>
            <Button size="sm" onClick={() => void saveParser()}>Сохранить</Button>
          </div>
          {parserEngine === "manta" && (
            <p className="mt-2 text-[11px] text-muted-foreground">.dem парсится локально, Docker не нужен.</p>
          )}
        </Section>

        <Section title="Ассистент" icon={<KeyRound className="size-4" />}>
          <div className="mb-3 flex flex-wrap items-center gap-2 rounded-md border border-border bg-background/50 p-2 text-xs">
            <span className="label-caps">Сейчас:</span>
            <Badge variant="outline">{providerLabel}</Badge>
            <span className="font-mono text-muted-foreground">{settings?.model || "модель не выбрана"}</span>
            <span className="text-muted-foreground">Провайдер и модель меняются на вкладке «Ассистент».</span>
          </div>

          {settings && (
            <div className="space-y-3">
              {provider === "groq" && (
                <div className="space-y-1">
                  <label className="label-caps">API-ключ GroqCloud {settings.hasGroqKey && <span className="text-radiant">сохранён</span>}</label>
                  <div className="flex flex-wrap gap-2">
                    <Input type="password" value={groqKey} onChange={(e) => setGroqKey(e.target.value)} placeholder="вставьте API-ключ" className="h-8 max-w-md flex-1" />
                    {settings.hasGroqKey && <Button variant="ghost" size="sm" className="text-destructive" onClick={() => void clearKey("groq")}><Trash2 className="mr-1.5 size-3.5" /> Удалить</Button>}
                  </div>
                </div>
              )}
              {provider === "openrouter" && (
                <div className="space-y-1">
                  <label className="label-caps">API-ключ OpenRouter {settings.hasOpenRouterKey && <span className="text-radiant">сохранён</span>}</label>
                  <div className="flex flex-wrap gap-2">
                    <Input type="password" value={openRouterKey} onChange={(e) => setOpenRouterKey(e.target.value)} placeholder="вставьте API-ключ" className="h-8 max-w-md flex-1" />
                    {settings.hasOpenRouterKey && <Button variant="ghost" size="sm" className="text-destructive" onClick={() => void clearKey("openrouter")}><Trash2 className="mr-1.5 size-3.5" /> Удалить</Button>}
                  </div>
                  <p className="rounded-md border border-amber-500/30 bg-amber-500/10 p-2 text-[11px] text-amber-200">
                    Free-модели OpenRouter: часть маршрутов может использовать данные для обучения. Имена игроков по умолчанию маскируются.
                  </p>
                </div>
              )}
              {isLocal && (
                <div className="space-y-1">
                  <label className="label-caps">URL {providerLabel}</label>
                  <Input
                    value={provider === "ollama" ? settings.ollamaUrl : settings.llamaCppUrl}
                    onChange={(e) => setSettings({
                      ...settings,
                      ...(provider === "ollama" ? { ollamaUrl: e.target.value } : { llamaCppUrl: e.target.value }),
                    })}
                    placeholder={DEFAULT_URL[provider]}
                    className="h-8"
                  />
                  <p className="text-[11px] text-radiant">Локальный сервер: данные матча и формулы не покидают компьютер.</p>
                  {provider === "llamacpp" && (
                    <p className="text-[11px] text-amber-300">
                      Для tool calling запускайте llama-server с `--jinja`; для стабильного id используйте `--alias`.
                    </p>
                  )}
                </div>
              )}

              <label className="flex items-center gap-2 text-[11px] text-muted-foreground">
                <input
                  type="checkbox"
                  checked={settings.sendPlayerNames}
                  onChange={(e) => setSettings({ ...settings, sendPlayerNames: e.target.checked })}
                />
                Отправлять имена игроков и Steam ID (по умолчанию маскируются)
              </label>

              <div className="flex flex-wrap gap-2 border-t border-border pt-3">
                <Button size="sm" onClick={() => void saveAssistant()}>Сохранить</Button>
                <Button variant="outline" size="sm" onClick={() => void testConnection()}>Проверить подключение</Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => void refreshModels(provider, isLocal ? localURL : "")}
                  disabled={loadingModels}
                >
                  <RefreshCw className={cn("mr-1.5 size-3.5", loadingModels && "animate-spin")} />
                  Обновить модели
                </Button>
                {modelCount !== null && <span className="self-center text-[11px] text-muted-foreground">Найдено: {modelCount}</span>}
              </div>
            </div>
          )}
        </Section>

        <div className="flex items-center gap-2 text-[11px] text-muted-foreground">
          <ShieldCheck className="size-3.5" />
          API-ключи хранятся локально в защищённом виде и не попадают в репозиторий.
        </div>
      </div>
    </div>
  );
}
