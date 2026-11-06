import { fmtTime } from "@/lib/format";
import { cn } from "@/lib/utils";
import type { formula } from "../../wailsjs/go/models";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

export interface MatchInfo {
  matchId: number;
  durationSec: number;
  radiantScore: number;
  direScore: number;
  winner: "radiant" | "dire" | "";
  presetName: string;
}

type Props = {
  info: MatchInfo | null;
  presets: formula.Preset[];
  activeId: string;
  onPresetChange: (id: string) => void;
};

function Stat({ label, children, className }: { label: string; children: React.ReactNode; className?: string }) {
  return (
    <div className={cn("flex items-baseline gap-1.5", className)}>
      <span className="label-caps">{label}</span>
      <span className="text-xs font-semibold tabular-nums text-foreground">{children}</span>
    </div>
  );
}

export default function InfoBar({ info, presets, activeId, onPresetChange }: Props) {
  if (!info) return null;
  return (
    <div className="flex flex-wrap items-center gap-x-5 gap-y-1.5 border-b border-border bg-card/40 px-4 py-2">
      <Stat label="Матч">{info.matchId || "-"}</Stat>
      <div className="hidden h-3 w-px bg-border sm:block" />
      <Stat label="Время">{fmtTime(info.durationSec)}</Stat>
      <div className="hidden h-3 w-px bg-border sm:block" />
      <div className="flex items-center gap-2">
        <span className="label-caps">Победа</span>
        <span className={cn("text-xs font-bold", info.winner === "radiant" ? "text-radiant" : "text-dire")}>
          {info.winner === "radiant" ? "Radiant" : "Dire"}
        </span>
        <span className="text-xs tabular-nums text-muted-foreground">
          (<span className="text-radiant">{info.radiantScore}</span> : <span className="text-dire">{info.direScore}</span>)
        </span>
      </div>
      <div className="ml-auto flex items-center gap-1.5">
        <span className="label-caps">Формула</span>
        {presets.length > 0 ? (
          <Select value={activeId} onValueChange={onPresetChange}>
            <SelectTrigger className="h-7 w-56 border-border bg-background/60 text-xs">
              <SelectValue placeholder={info.presetName} />
            </SelectTrigger>
            <SelectContent>
              {presets.map((preset) => (
                <SelectItem key={preset.id} value={preset.id}>
                  {preset.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        ) : (
          <span className="text-xs font-semibold text-gold">{info.presetName}</span>
        )}
      </div>
    </div>
  );
}