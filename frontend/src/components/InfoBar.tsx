import { Gamepad2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { fmtTime } from "@/lib/format";

export interface MatchInfo {
  matchId: number;
  durationSec: number;
  radiantScore: number;
  direScore: number;
  winner: "radiant" | "dire" | "";
  presetName: string;
}

export default function InfoBar({ info }: { info: MatchInfo | null }) {
  if (!info) return null;
  return (
    <div className="flex flex-wrap items-center gap-x-4 gap-y-1 border-b border-border bg-card/60 px-4 py-1.5 text-xs text-muted-foreground">
      <span className="flex items-center gap-1.5">
        <Gamepad2 className="size-3.5" />
        <span className="font-semibold text-foreground">Матч</span> {info.matchId ? info.matchId : "-"}
      </span>
      <span>
        Длительность <span className="font-semibold tabular-nums text-foreground">{fmtTime(info.durationSec)}</span>
      </span>
      <span className="flex items-center gap-1.5">
        <Badge variant={info.winner === "radiant" ? "radiant" : "dire"} className="normal-case">
          Поб. {info.winner === "radiant" ? "Radiant" : "Dire"}
        </Badge>
        <span className="tabular-nums">
          <span className="text-radiant font-bold">{info.radiantScore}</span>
          <span className="mx-1 text-foreground">:</span>
          <span className="text-dire font-bold">{info.direScore}</span>
        </span>
        <span className="text-muted-foreground">(убийства)</span>
      </span>
      <span className="ml-auto tabular-nums">
        Формула: <span className="font-semibold text-gold">{info.presetName}</span>
      </span>
    </div>
  );
}