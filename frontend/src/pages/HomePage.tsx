import { useMemo } from "react";
import MvpCards from "@/components/MvpCards";
import StatsTable from "@/components/StatsTable";
import WeightsPanel from "@/components/WeightsPanel";
import InfoBar, { type MatchInfo } from "@/components/InfoBar";
import type { formula, main } from "../../wailsjs/go/models";

type Props = {
  views: main.PlayerView[];
  presets: formula.Preset[];
  activeId: string;
  matchInfo: MatchInfo | null;
  onChange: () => Promise<void> | void;
};

export default function HomePage({ views, presets, activeId, matchInfo, onChange }: Props) {
  const activePreset = useMemo(() => presets.find((p) => p.id === activeId) ?? null, [presets, activeId]);

  return (
    <>
      <InfoBar info={matchInfo} />
      <main className="flex min-h-0 flex-1 gap-4 p-4">
        <section className="flex min-h-0 min-w-0 flex-1 flex-col gap-4">
          <MvpCards views={views} preset={activePreset} />
          <div className="min-h-0 flex-1">
            <StatsTable views={views} preset={activePreset} />
          </div>
        </section>
        <aside className="w-88 min-h-0 shrink-0">
          <WeightsPanel presets={presets} activeId={activeId} views={views} onChange={onChange} />
        </aside>
      </main>
    </>
  );
}
