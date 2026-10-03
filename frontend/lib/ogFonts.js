// =====================================================================
// Google Fonts loader for the image-generation route (next/og → satori).
// The css2 API with a non-browser UA → Google serves TTF (not woff2:
// satori/opentype.js CANNOT read woff2, rendering would fail).
// Module-level cache: per (family,weight) only 1 Promise per process →
// the font is fetched once; if Google is down → resolves null and the route
// falls back to next/og's built-in default font ("sans serif" = Noto).
// =====================================================================

const cache = new Map();

export function googleFont(family, weight = 400) {
  const key = `${family}-${weight}`;
  if (cache.has(key)) {
    return cache.get(key);
  }
  const promise = (async () => {
    const css = await fetch(
      `https://fonts.googleapis.com/css2?family=${family.replace(/ /g, "+")}:wght@${weight}&display=swap`,
      { headers: { "User-Agent": "node" } }
    ).then((r) => {
      if (!r.ok) {
        throw new Error(`css ${r.status}`);
      }
      return r.text();
    });
    const match = css.match(/url\(([^)]+)\)/);
    if (!match) {
      throw new Error("font url tidak ditemukan");
    }
    const data = await fetch(match[1]).then((r) => {
      if (!r.ok) {
        throw new Error(`font ${r.status}`);
      }
      return r.arrayBuffer();
    });
    return { name: family, data, weight, style: "normal" };
  })().catch(() => null);
  cache.set(key, promise);
  return promise;
}
