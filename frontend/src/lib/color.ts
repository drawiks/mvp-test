/**
 * Dominant color of a hero image, dark enough to sit behind card text.
 * Draws the image into a 16x16 canvas and averages alpha-weighted channels.
 */
export function dominantColor(url: string): Promise<string | null> {
  return new Promise((resolve) => {
    const img = new Image();
    img.crossOrigin = "anonymous";
    img.onload = () => {
      try {
        const c = document.createElement("canvas");
        c.width = c.height = 16;
        const ctx = c.getContext("2d");
        if (!ctx) return resolve(null);
        ctx.drawImage(img, 0, 0, 16, 16);
        const { data } = ctx.getImageData(0, 0, 16, 16);
        let r = 0;
        let g = 0;
        let b = 0;
        let n = 0;
        for (let i = 0; i < data.length; i += 4) {
          const a = data[i + 3] / 255;
          r += data[i] * a;
          g += data[i + 1] * a;
          b += data[i + 2] * a;
          n += a;
        }
        if (!n) return resolve(null);
        r = (r / n) * 0.45;
        g = (g / n) * 0.45;
        b = (b / n) * 0.45;
        resolve(`rgb(${r | 0}, ${g | 0}, ${b | 0})`);
      } catch {
        resolve(null);
      }
    };
    img.onerror = () => resolve(null);
    img.src = url;
  });
}

/** Append alpha to an "rgb(r, g, b)" string, yielding "rgba(r, g, b, a)". */
export function withAlpha(color: string | null, alpha: number): string {
  if (!color) return "transparent";
  const m = color.match(/\(([^)]+)\)/);
  if (!m) return "transparent";
  return `rgba(${m[1]}, ${alpha})`;
}