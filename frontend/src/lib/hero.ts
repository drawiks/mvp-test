import * as Bindings from "../../wailsjs/go/main/App";
import { normalizeHeroName } from "@/lib/heroNames";

const cache = new Map<string, Promise<string | null>>();

/** Resolve hero portrait/promo image URL via the backend, cached. */
export function heroImageURL(hero: string): Promise<string | null> {
  if (!hero) return Promise.resolve(null);
  const key = normalizeHeroName(hero);
  if (!cache.has(key)) {
    cache.set(key, Bindings.HeroImageURL(key).then((u) => u || null));
  }
  return cache.get(key)!;
}

/**
 * Classic square hero icon from the public Steam CDN (same asset folder the
 * old python UI used: heroes/icons/<hero>.png). Cached per hero.
 */
export function heroIconURL(hero: string): Promise<string | null> {
  if (!hero) return Promise.resolve(null);
  const key = normalizeHeroName(hero);
  if (!cache.has(`icon:${key}`)) {
    cache.set(
      `icon:${key}`,
      Promise.resolve(
        `https://cdn.cloudflare.steamstatic.com/apps/dota2/images/dota_react/heroes/icons/${key}.png`
      )
    );
  }
  return cache.get(`icon:${key}`)!;
}