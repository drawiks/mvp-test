import { useMemo } from "react";
import type { formula, main } from "../../wailsjs/go/models";
import { heroName } from "@/lib/heroNames";
import { Progress } from "@/components/ui/progress";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import HeroIcon from "@/components/HeroIcon";
import { buildBreakdownTooltip, fmtNum } from "@/lib/format";
import { WEIGHT_META, type WeightMeta } from "@/lib/weights";
import { cn } from "@/lib/utils";

type Col =
  | { kind: "base"; header: string; key: keyof main.PlayerView; align?: "center" | "right"; render?: (v: main.PlayerView) => string }
  | { kind: "stat"; header: string; stat: string; align?: "center" | "right"; render?: (v: main.PlayerView, s: number) => string };

const BASE: Col[] = [
  { kind: "base", header: "Место", key: "PlayerID", align: "center" },
  { kind: "base", header: "Герой", key: "Hero" },
  { kind: "base", header: "Игрок", key: "Name" },
  { kind: "base", header: "Линия", key: "Lane", align: "center", render: (v) => v.Lane || "—" },
  { kind: "base", header: "Ур.", key: "Level", align: "center" },
];

const STAT_COLS: Col[] = [
  { kind: "base", header: "LH", key: "LastHits", align: "center", render: (v) => fmtNum(v.LastHits) },
  { kind: "base", header: "GPM", key: "GPM", align: "center" },
  { kind: "base", header: "XPM", key: "XPM", align: "center" },
  { kind: "base", header: "NW", key: "Networth", align: "center", render: (v) => fmtNum(v.Networth) },
  { kind: "stat", header: "Stun", stat: "stun_duration", align: "center", render: (_v, s) => s.toFixed(1) },
  { kind: "stat", header: "Heal", stat: "healing", align: "center", render: (_v, s) => fmtNum(s) },
  { kind: "stat", header: "T.Dmg", stat: "tower_damage", align: "center", render: (_v, s) => fmtNum(s) },
  { kind: "stat", header: "Camps", stat: "camps_stacked", align: "center" },
  { kind: "stat", header: "Runes", stat: "rune_pickups", align: "center" },
  { kind: "stat", header: "FB", stat: "first_blood", align: "center", render: (_v, s) => (s > 0 ? "Да" : "—") },
  { kind: "stat", header: "H.Dmg", stat: "hero_damage", align: "center", render: (_v, s) => fmtNum(s) },
  { kind: "stat", header: "D.Taken", stat: "damage_taken", align: "center", render: (_v, s) => fmtNum(s) },
  { kind: "stat", header: "Ward G", stat: "gold_spent_wards", align: "center" },
  { kind: "stat", header: "Smoke G", stat: "gold_spent_smoke", align: "center" },
  { kind: "stat", header: "Dust G", stat: "gold_spent_dust", align: "center" },
  { kind: "stat", header: "Buffs", stat: "buff_duration", align: "center", render: (_v, s) => `${Math.round(s)}` },
  { kind: "stat", header: "Save", stat: "save_duration", align: "center", render: (_v, s) => `${Math.round(s)}` },
  { kind: "stat", header: "Purge", stat: "purge_duration", align: "center", render: (_v, s) => `${Math.round(s)}` },
  { kind: "stat", header: "Shield", stat: "shield_duration", align: "center", render: (_v, s) => `${Math.round(s)}` },
  { kind: "stat", header: "Fear", stat: "fear_duration", align: "center", render: (_v, s) => s.toFixed(1) },
  { kind: "stat", header: "Silence", stat: "silence_duration", align: "center", render: (_v, s) => s.toFixed(1) },
  { kind: "stat", header: "Break", stat: "break_duration", align: "center", render: (_v, s) => s.toFixed(1) },
  { kind: "stat", header: "Disarm", stat: "disarm_duration", align: "center", render: (_v, s) => s.toFixed(1) },
  { kind: "stat", header: "HealVal", stat: "heal_value", align: "center", render: (_v, s) => fmtNum(s) },
  { kind: "stat", header: "Creeps", stat: "creeps_stacked", align: "center" },
];

const STAT_LABEL: Record<string, WeightMeta> = {};
for (const m of WEIGHT_META) STAT_LABEL[m.key] = m;

const ROLE_BG: Record<string, string> = {
  winner_top1: "bg-gold/10",
  loser_top1: "bg-silver/10",
  winner_top2: "bg-bronze/10",
};

function StatCell({
  v,
  col,
  weight,
  label,
}: {
  v: main.PlayerView;
  col: Extract<Col, { kind: "stat" }>;
  weight: number | undefined;
  label: string;
}) {
  const s = v.Stats?.[col.stat] ?? 0;
  const text = col.render ? col.render(v, s) : `${s}`;
  const tip = weight ? `${label}: ${s} × ${weight} = ${(s * weight).toFixed(3)}` : null;
  return <Tip tip={tip}>{text}</Tip>;
}

function Tip({ tip, children, className }: { tip: string | null; children: React.ReactNode; className?: string }) {
  if (!tip) return <span className={className}>{children}</span>;
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className={cn("tabular-nums", className)}>{children}</span>
      </TooltipTrigger>
      <TooltipContent side="top" className="font-mono text-[11px]">
        {tip}
      </TooltipContent>
    </Tooltip>
  );
}

export default function StatsTable({
  views,
  preset,
}: {
  views: main.PlayerView[];
  preset: formula.Preset | null;
}) {
  const weights = preset?.kind === "linear" ? (preset.weights ?? {}) : undefined;
  const rows = useMemo(() => {
    const radiant = views.filter((v) => v.Team === "radiant").sort((a, b) => b.Score - a.Score);
    const dire = views.filter((v) => v.Team === "dire").sort((a, b) => b.Score - a.Score);
    return { radiant, dire };
  }, [views]);
  const maxScore = useMemo(() => Math.max(...views.map((v) => v.Score), 0.0001), [views]);

  const row = (v: main.PlayerView, place: number) => (
    <tr key={v.SteamID} className={cn("border-b border-border/60 transition-colors hover:bg-accent/40", ROLE_BG[v.MvpRole] ?? (v.IsWinner ? "bg-radiant/[0.02]" : "bg-dire/[0.02]"))}>
      <td className={cn("px-2 py-1.5 text-center font-bold", v.MvpRole === "winner_top1" && "text-gold", v.MvpRole === "loser_top1" && "text-silver", v.MvpRole === "winner_top2" && "text-bronze")}>
        {place}
      </td>
      <td className="px-2 py-1.5">
        <div className="flex items-center gap-2">
          <HeroIcon hero={v.Hero} name={v.Name} />
          <span className="font-medium text-foreground">{heroName(v.Hero) || "—"}</span>
        </div>
      </td>
      <td className="px-2 py-1.5 text-muted-foreground">{v.Name || "—"}</td>
      <td className="px-2 py-1.5 text-center text-muted-foreground">{v.Lane || "—"}</td>
      <td className="px-2 py-1.5 text-center text-muted-foreground">{v.Level}</td>
      <td className="px-2 py-1.5 text-center font-semibold text-kill">{v.Kills}</td>
      <td className="px-2 py-1.5 text-center font-semibold text-death">{v.Deaths}</td>
      <td className="px-2 py-1.5 text-center font-semibold text-foreground">{v.Assists}</td>
      {STAT_COLS.map((col) =>
        col.kind === "base" ? (
          <td key={col.header} className="px-2 py-1.5 text-center tabular-nums text-muted-foreground">
            {col.render ? col.render(v) : String(v[col.key as keyof main.PlayerView])}
          </td>
        ) : (
          <td key={col.header} className="px-2 py-1.5">
            <StatCell v={v} col={col} weight={weights?.[col.stat]} label={STAT_LABEL[col.stat]?.label ?? col.stat} />
          </td>
        )
      )}
      <td className="px-2 py-1.5">
        <Tooltip>
          <TooltipTrigger asChild>
            <div className="flex w-24 items-center gap-2">
              <Progress value={(v.Score / maxScore) * 100} color="bg-gold" className="h-1 flex-1" />
              <span className="w-14 text-right font-mono text-xs font-bold tabular-nums text-gold">{v.Score.toFixed(2)}</span>
            </div>
          </TooltipTrigger>
          {weights && (
            <TooltipContent side="left" className="max-w-72 whitespace-pre-line font-mono text-[11px]">
              {buildBreakdownTooltip(v.Stats ?? {}, weights, WEIGHT_META)}
            </TooltipContent>
          )}
        </Tooltip>
      </td>
    </tr>
  );

  const section = (team: "radiant" | "dire", players: main.PlayerView[]) => (
    <>
      <tr>
        <td colSpan={BASE.length + 3 + STAT_COLS.length + 1} className="h-7 bg-secondary px-3">
          <span className={cn("label-caps", team === "radiant" ? "text-radiant" : "text-dire")}>
            {team === "radiant" ? "Radiant" : "Dire"}
          </span>
          <span className="ml-2 text-[11px] text-muted-foreground">
            · {players.reduce((acc, p) => acc + p.Kills, 0)} kills
          </span>
        </td>
      </tr>
      {players.map((v, i) => row(v, i + 1))}
    </>
  );

  return (
    <div className="h-full overflow-auto rounded-lg border border-border bg-card">
      <TooltipProvider delayDuration={150}>
        <table className="w-full min-w-[2100px] border-collapse text-xs">
          <thead className="sticky top-0 z-10">
            <tr className="bg-popover">
              {BASE.map((c) => (
                <th key={c.header} className="label-caps border-b border-border px-2 py-2 text-left">
                  {c.header}
                </th>
              ))}
              {["K", "D", "A"].map((h) => (
                <th key={h} className="label-caps border-b border-border px-2 py-2 text-center">
                  {h}
                </th>
              ))}
              {STAT_COLS.map((c) => (
                <th
                  key={c.header}
                  className="label-caps border-b border-border px-2 py-2 text-center hover:text-foreground"
                  title={c.kind === "stat" ? STAT_LABEL[c.stat]?.label : undefined}
                >
                  {c.header}
                </th>
              ))}
              <th className="label-caps border-b border-border px-2 py-2 text-center">Счёт</th>
            </tr>
          </thead>
          <tbody>
            {section("radiant", rows.radiant)}
            {section("dire", rows.dire)}
          </tbody>
        </table>
      </TooltipProvider>
    </div>
  );
}