import { fmtTime } from "@/lib/format";
import { cn } from "@/lib/utils";

export interface MatchInfo {
  matchId: number;
  durationSec: number;
  radiantScore: number;
  direScore: number;
  winner: "radiant" | "dire" | "";
  presetName: string;
}

function Stat({ label, children, className }: { label: string; children: React.ReactNode; className?: string }) {
  return (
    <div className={cn("flex items-baseline gap-1.5", className)}>
      <span className="label-caps">{label}</span>
      <span className="text-xs font-semibold tabular-nums text-foreground">{children}</span>
    </div>
  );
}

export default function InfoBar({ info }: { info: MatchInfo | null }) {
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
        <span className="text-xs font-semibold text-gold">{info.presetName}</span>
      </div>
    </div>
  );
}