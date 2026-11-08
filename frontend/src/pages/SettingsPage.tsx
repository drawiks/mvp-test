import { useEffect, useState } from "react";
import { KeyRound, RefreshCw, Settings as SettingsIcon, ShieldCheck, Trash2 } from "lucide-react";
import { toast } from "sonner";
import * as Bindings from "../../wailsjs/go/main/App";
import type { main } from "../../wailsjs/go/models";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

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
  const [settings, setSettings] = useState<main.AssistantSettings | null>(null);
  const [googleKeys, setGoogleKeys] = useState<main.GoogleKeyInfo[]>([]);
  const [newGoogleKey, setNewGoogleKey] = useState("");
  const [loadingModels, setLoadingModels] = useState(false);
  const [modelCount, setModelCount] = useState<number | null>(null);

  const load = async () => {
    const [assistant, gkeys] = await Promise.all([
      Bindings.GetAssistantSettings(),
      Bindings.GetGoogleKeys(),
    ]);
    setSettings(assistant);
    setGoogleKeys(gkeys);
  };

  useEffect(() => {
    void load().catch((e) => toast.error(String(e)));
  }, []);

  const saveAssistant = async () => {
    if (!settings) return;
    try {
      await Bindings.SaveAssistantSettings(settings);
      setSettings(await Bindings.GetAssistantSettings());
      toast.success("Настройки ассистента сохранены");
    } catch (e) {
      toast.error(String(e));
    }
  };

  const testConnection = async () => {
    try {
      await saveAssistant();
      await Bindings.TestAssistantConnection();
      toast.success("Подключение работает");
    } catch (e) {
      toast.error(String(e));
    }
  };

  const refreshModels = async () => {
    setLoadingModels(true);
    try {
      const list = await Bindings.ListAssistantModels(true);
      setModelCount(list.length);
      toast.success(`Моделей найдено: ${list.length}`);
    } catch (e) {
      toast.error(String(e));
    } finally {
      setLoadingModels(false);
    }
  };

  const addGoogleKey = async () => {
    try {
      await Bindings.AddGoogleKey(newGoogleKey);
      setNewGoogleKey("");
      setGoogleKeys(await Bindings.GetGoogleKeys());
      setSettings(await Bindings.GetAssistantSettings());
      toast.success("Google-ключ добавлен");
    } catch (e) {
      toast.error(String(e));
    }
  };

  const removeGoogleKey = async (index: number) => {
    try {
      await Bindings.RemoveGoogleKey(index);
      setGoogleKeys(await Bindings.GetGoogleKeys());
      setSettings(await Bindings.GetAssistantSettings());
      toast.success("Google-ключ удалён");
    } catch (e) {
      toast.error(String(e));
    }
  };

  const testGoogleKeys = async () => {
    try {
      await Bindings.TestGoogleKeys();
      setGoogleKeys(await Bindings.GetGoogleKeys());
      toast.success("Google-ключи проверены");
    } catch (e) {
      setGoogleKeys(await Bindings.GetGoogleKeys().catch(() => []));
      toast.error(String(e));
    }
  };

  return (
    <div className="flex-1 overflow-y-auto p-4">
      <div className="mx-auto flex max-w-3xl flex-col gap-4">
        <h1 className="flex items-center gap-2 text-lg font-bold">
          <SettingsIcon className="size-5" /> Настройки
        </h1>

        <Section title="Google AI Studio" icon={<KeyRound className="size-4" />}>
          <div className="mb-3 flex flex-wrap items-center gap-2 rounded-md border border-border bg-background/50 p-2 text-xs">
            <span className="label-caps">Модель:</span>
            <span className="font-mono text-muted-foreground">{settings?.model || "не выбрана"}</span>
            <span className="text-muted-foreground">Модель меняется на вкладке «Ассистент».</span>
          </div>

          <div className="space-y-2">
            <label className="label-caps">Ключи Google AI Studio</label>
            {googleKeys.map((k) => (
              <div key={k.index} className="flex flex-wrap items-center gap-2 rounded-md border border-border bg-background/50 p-2 text-xs">
                <span className="font-mono">{k.masked}</span>
                <Badge variant={k.status === "ok" ? "outline" : "secondary"}>{k.status}</Badge>
                {k.active && <Badge variant="gold">активный</Badge>}
                {k.lastError && <span className="max-w-64 truncate text-muted-foreground" title={k.lastError}>{k.lastError}</span>}
                <Button variant="ghost" size="sm" className="ml-auto text-destructive" onClick={() => void removeGoogleKey(k.index)}>
                  <Trash2 className="mr-1 size-3.5" /> Удалить
                </Button>
              </div>
            ))}
            <div className="flex flex-wrap gap-2">
              <Input
                type="password"
                value={newGoogleKey}
                onChange={(e) => setNewGoogleKey(e.target.value)}
                placeholder="AIza…"
                className="h-8 max-w-md flex-1"
              />
              <Button size="sm" onClick={() => void addGoogleKey()} disabled={!newGoogleKey.trim()}>Добавить ключ</Button>
              <Button variant="outline" size="sm" onClick={() => void testGoogleKeys()} disabled={googleKeys.length === 0}>Проверить ключи</Button>
            </div>
            <p className="rounded-md border border-amber-500/30 bg-amber-500/10 p-2 text-[11px] text-amber-200">
              Лимиты Gemini считаются на Google Cloud project. Ключи из одного проекта не увеличивают квоту — используйте ключи из разных проектов. Free-tier запросы могут использоваться для улучшения продуктов.
            </p>
          </div>

          {settings && (
            <div className="mt-3 space-y-3">
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
                <Button variant="outline" size="sm" onClick={() => void refreshModels()} disabled={loadingModels}>
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
