import { useCallback, useMemo } from "react";
import { Button } from "@/components/ui/button";
import { WEIGHT_META } from "@/lib/weights";
import { cn } from "@/lib/utils";

type Props = {
  onInsert: (token: string) => void;
  className?: string;
};

const GROUPS = ["Бой", "Дебаффы", "Баффы", "Фарм", "Саппорт", "Утилити"] as const;

export default function TokenPalette({ onInsert, className }: Props) {
  const insert = useCallback((t: string) => onInsert(t), [onInsert]);

  const groups = useMemo(() => {
    const metaByGroup: Record<string, typeof WEIGHT_META[0][]> = {};
    for (const m of WEIGHT_META) {
      if (m.chipOnly) continue;
      (metaByGroup[m.group] ??= []).push(m);
    }
    return GROUPS.filter((g) => metaByGroup[g]?.length).map((group) => ({
      group,
      items: metaByGroup[group]!,
    }));
  }, []);

  return (
    <div className={cn("flex flex-col gap-3", className)}>
      {groups.map(({ group, items }) => (
        <div key={group} className="space-y-1">
          <span className="label-caps">{group}</span>
          <div className="flex flex-wrap gap-1">
            {items.map((m) => (
              <Button
                key={m.key}
                variant="outline"
                size="sm"
                className="h-6 px-2 font-mono text-[10px]"
                title={m.label}
                onClick={() => insert(m.key)}
              >
                {m.key}
              </Button>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}
