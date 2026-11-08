import { useEffect, useMemo, useState } from "react";
import { List } from "lucide-react";
import type { main } from "../../wailsjs/go/models";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import HeroIcon from "@/components/HeroIcon";
import { heroName } from "@/lib/heroNames";
import { WEIGHT_META } from "@/lib/weights";

const PLAYER_TOKENS = WEIGHT_META.filter((token) => token.key !== "match_duration" && token.key !== "hero");

function rawValue(hero: main.PlayerView, key: string): string {
  if (key === "networth") return String(hero.Networth);
  const value = hero.Stats?.[key];
  return value === undefined ? "—" : String(value);
}

export default function PlayerDetailsPanel({ views }: { views: main.PlayerView[] }) {
  const sortedViews = useMemo(() => [...views].sort((a, b) => b.Score - a.Score), [views]);
  const [heroPlayerID, setHeroPlayerID] = useState<number | null>(null);

  useEffect(() => {
    if (sortedViews.length > 0 && !sortedViews.some((v) => v.PlayerID === heroPlayerID)) {
      setHeroPlayerID(sortedViews[0].PlayerID);
    }
  }, [sortedViews, heroPlayerID]);

  const hero = useMemo(
    () => sortedViews.find((v) => v.PlayerID === heroPlayerID) ?? null,
    [sortedViews, heroPlayerID]
  );

  return (
    <div className="flex h-full flex-col gap-3 rounded-lg border border-border bg-card p-3">
      <h2 className="label-caps flex items-center gap-1.5">
        <List className="size-3.5" /> Подробности
      </h2>

      {hero && (
        <Select value={String(hero.PlayerID)} onValueChange={(v) => setHeroPlayerID(Number(v))}>
          <SelectTrigger className="w-full">
            <SelectValue placeholder="Игрок" />
          </SelectTrigger>
          <SelectContent>
            {sortedViews.map((v) => (
              <SelectItem key={v.PlayerID} value={String(v.PlayerID)} className="gap-2">
                <HeroIcon hero={v.Hero} name={v.Name} />
                <span className="truncate">{v.Name || heroName(v.Hero) || v.Hero}</span>
                <span className="ml-auto font-mono text-[11px] tabular-nums text-muted-foreground">{v.Score.toFixed(2)}</span>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )}

      {hero ? (
        <ScrollArea className="min-h-0 flex-1">
          <table className="w-full table-fixed border-collapse text-[11px]">
            <thead className="sticky top-0 z-10 bg-card">
              <tr>
                <th className="label-caps w-2/3 border-b border-border px-2 py-1.5 text-left">Токен</th>
                <th className="label-caps w-1/3 border-b border-border px-2 py-1.5 text-right">Значение</th>
              </tr>
            </thead>
            <tbody>
              {PLAYER_TOKENS.map((token) => (
                <tr key={token.key} className="border-b border-border/40 odd:bg-background/30 hover:bg-accent/30">
                  <td className="truncate px-2 py-1 font-mono text-muted-foreground" title={token.key}>
                    {token.key}
                  </td>
                  <td className="px-2 py-1 text-right font-mono tabular-nums text-foreground">
                    {rawValue(hero, token.key)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </ScrollArea>
      ) : (
        <p className="text-xs text-muted-foreground">Выберите игрока.</p>
      )}
    </div>
  );
}
