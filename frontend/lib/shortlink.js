// SAME-ORIGIN short URL helpers (public-page redirect fix, 2026-09-30).
//
// Previously hrefs were built by the backend (GO_API_URL /
// NEXT_PUBLIC_API_URL = http://localhost:8081), so links on /u/[username]
// pointed straight at the backend: the production URL bar showed
// "localhost:8081" and other devices could not reach it. All links now use
// the /r/{code} path on the page's own ORIGIN: Next proxies to the backend
// through app/r/[code]/route.js, so the displayed host is always the host
// being visited (dev :3000, production domain).

// Relative path: used as href (the browser resolves it against the page
// origin).
export function shortPath(code) {
  return `/r/${code}`;
}

// Absolute same-origin URL: for consumers that need a complete string: text
// copied to the clipboard and QR codes (a QR/clipboard cannot hold a
// relative path). window exists only in the browser; on the server (SSR) it
// falls back to the path: this value is never rendered to the DOM during
// hydration, so it is safe.
export function absoluteShortUrl(code) {
  if (typeof window === "undefined") return shortPath(code);
  return window.location.origin + shortPath(code);
}

// Normalizes a shortUrl from the backend API (still absolute
// "http://<host-be>/r/x" because the backend builds it from the request
// Host) into a same-origin URL. Used by the shorten form and bulk-import
// results, which only accept complete URLs.
export function sameOriginShortUrl(raw, fallbackCode) {
  if (typeof raw === "string" && raw !== "") {
    const m = raw.match(/\/r\/([A-Za-z0-9_-]{1,30})/);
    if (m) return absoluteShortUrl(m[1]);
  }
  return absoluteShortUrl(fallbackCode);
}
