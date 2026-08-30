import { useEffect, useState } from "react";
import { heroImageURL } from "@/lib/hero";

/** Hero portrait with an inline letter fallback while it loads. */
export default function HeroAvatar({ hero, name, size = 34 }: { hero: string; name?: string; size?: number }) {
  const [url, setUrl] = useState<string | null>(null);
  useEffect(() => {
    let alive = true;
    heroImageURL(hero).then((u) => alive && setUrl(u));
    return () => {
      alive = false;
    };
  }, [hero]);
  return (
    <div className="relative flex shrink-0 items-center justify-center overflow-hidden rounded-md bg-slate-800" style={{ width: size, height: size }}>
      {url ? (
        <img src={url} width={size} height={size} alt={name ?? hero} loading="lazy" className="object-cover" />
      ) : (
        <span className="text-xs font-bold text-slate-400">{(name ?? hero).slice(0, 1).toUpperCase()}</span>
      )}
    </div>
  );
}