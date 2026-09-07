import { useEffect, useState } from "react";
import { heroIconURL } from "@/lib/hero";
import { cn } from "@/lib/utils";

export default function HeroIcon({ hero, name, className }: { hero: string; name?: string; className?: string }) {
  const [url, setUrl] = useState<string | null>(null);
  useEffect(() => {
    let alive = true;
    heroIconURL(hero).then((u) => alive && setUrl(u));
    return () => {
      alive = false;
    };
  }, [hero]);
  return (
    <span className={cn("flex size-6 shrink-0 items-center justify-center rounded bg-muted", className)}>
      {url ? (
        <img src={url} alt="" loading="lazy" className="size-full rounded object-cover" draggable={false} />
      ) : (
        <span className="text-[10px] font-bold text-muted-foreground">{(name ?? hero).slice(0, 1).toUpperCase()}</span>
      )}
    </span>
  );
}