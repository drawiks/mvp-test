import { useEffect, useState } from "react";
import { Crown, Medal, Swords } from "lucide-react";
import type { formula, main } from "../../wailsjs/go/models";
import { heroImageURL } from "@/lib/hero";
import { heroName } from "@/lib/heroNames";
import { dominantColor, withAlpha } from "@/lib/color";
import { Progress } from "@/components/ui/progress";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { buildBreakdownTooltip, fmtScore } from "@/lib/format";
import { WEIGHT_META } from "@/lib/weights";
import { cn } from "@/lib/utils";

const ROLE_META: Record<
  string,
  {
    label: string;
    icon: typeof Crown;
    bar: string;
    accent: string;
    textAccent: string;
    place: string;
  }
> = {
  winner_top1: {
    label: "MVP матча",
    icon: Crown,
    bar: "bg-gold",
    accent: "bg-gold",
    textAccent: "text-gold",
    place: "01",
  },
  loser_top1: {
    label: "Лучший в проигравшей команде",
    icon: Swords,
    bar: "bg-silver",
    accent: "bg-silver",
    textAccent: "text-silver",
    place: "02",
  },
  winner_top2: {
    label: "2-й лучший в победившей команде",
    icon: Medal,
    bar: "bg-bronze",
    accent: "bg-bronze",
    textAccent: "text-bronze",
    place: "03",
  },
};

type MvpCardsProps = {
  views: main.PlayerView[];
  preset: formula.Preset | null;
};

/** Top-3 podium cards with hero art, mirrors the old MvpCardsRow. */
export default function MvpCards({ views, preset }: MvpCardsProps) {
  const mvps = Object.keys(ROLE_META)
    .map((role) => ({ role, v: views.find((v) => v.MvpRole === role) ?? null }))
    .filter((x) => x.v) as { role: string; v: main.PlayerView }[];
  if (mvps.length === 0) return null;

  const maxScore = Math.max(...mvps.map((x) => x.v.Score), 0.0001);

  return (
    <div className="grid grid-cols-3 gap-3">
      {mvps.map(({ role, v }) => {
        const meta = ROLE_META[role] ?? ROLE_META.winner_top1;
        const isWinner = role === "winner_top1";
        return (
          <MvpCard key={role} v={v} meta={meta} maxScore={maxScore} preset={preset} featured={isWinner} />
        );
      })}
    </div>
  );
}

function MvpCard({
  v,
  meta,
  maxScore,
  preset,
  featured,
}: {
  v: main.PlayerView;
  meta: (typeof ROLE_META)["winner_top1"];
  maxScore: number;
  preset: formula.Preset | null;
  featured: boolean;
}) {
  const Icon = meta.icon;
  const [art, setArt] = useState<string | null>(null);
  const [tint, setTint] = useState<string | null>(null);
  useEffect(() => {
    let alive = true;
    heroImageURL(v.Hero).then((u) => {
      if (!alive) return;
      setArt(u);
      if (u) {
        dominantColor(u).then((c) => alive && setTint(c));
      }
    });
    return () => {
      alive = false;
    };
  }, [v.Hero]);

  const weights = preset?.kind === "linear" ? (preset.weights ?? {}) : undefined;
  const breakdown =
    weights && WEIGHT_META.length
      ? buildBreakdownTooltip(v.Stats ?? {}, weights, WEIGHT_META)
      : "";

  return (
    <div
      className={cn(
        "relative flex h-full flex-col gap-2 overflow-hidden rounded-lg border border-border bg-card py-4 pr-4 pl-5",
        featured && "bg-gold/[0.03]"
      )}
    >
      <span className={cn("absolute inset-y-0 left-0 w-[3px]", meta.accent)} />

      {/* hero art: edge-to-edge right band, tinted by the hero's colors */}
      {art && (
        <div className="pointer-events-none absolute inset-y-0 right-0 w-[58%] overflow-hidden">
          <img src={art} alt="" className="h-full w-full scale-125 object-cover object-center" draggable={false} />
          <div
            className="absolute inset-0"
            style={{
              background: `linear-gradient(90deg, ${withAlpha(tint, 0.92)} 0%, ${withAlpha(tint, 0.68)} 32%, transparent 64%)`,
            }}
          />
          <div className="absolute inset-0 bg-gradient-to-t from-background/70 via-transparent to-transparent" />
        </div>
      )}

      <div className="relative z-10 flex items-center gap-1.5">
        <span className={cn("font-mono text-[11px] font-bold", meta.textAccent)}>{meta.place}</span>
        <Icon className={cn("size-3.5", meta.textAccent)} />
        <span className="label-caps truncate">{meta.label}</span>
      </div>

      <div className="relative z-10 mt-1 min-w-0">
        <div className={cn("truncate text-lg leading-tight font-bold", meta.textAccent)}>{heroName(v.Hero) || "-"}</div>
        <div className="flex items-center gap-2">
          <span className={cn("text-[10px] font-bold tracking-wide uppercase", v.Team === "radiant" ? "text-radiant" : "text-dire")}>
            {v.Team === "radiant" ? "Radiant" : "Dire"}
          </span>
          <span className="truncate text-sm font-medium text-foreground">{v.Name || "-"}</span>
        </div>
      </div>

      <TooltipProvider delayDuration={200}>
        <Tooltip>
          <TooltipTrigger asChild>
            <div className="relative z-10 mt-auto space-y-1.5">
              <div className="flex items-end justify-between gap-2">
                <span className="font-mono text-3xl leading-none font-bold tabular-nums">{fmtScore(v.Score)}</span>
                <span className="text-xs tabular-nums text-muted-foreground">
                  K/D/A {v.Kills}/{v.Deaths}/{v.Assists}
                </span>
              </div>
              <Progress value={(v.Score / maxScore) * 100} color={meta.bar} className="h-1" />
              <div className="flex justify-between text-[11px] text-muted-foreground">
                <span>Ур. {v.Level}</span>
                <span className="tabular-nums">
                  {v.GPM} GPM / {v.XPM} XPM
                </span>
              </div>
            </div>
          </TooltipTrigger>
          {breakdown && (
            <TooltipContent side="top" className="max-w-72 whitespace-pre-line font-mono text-[11px]">
              {breakdown}
            </TooltipContent>
          )}
        </Tooltip>
      </TooltipProvider>
    </div>
  );
}