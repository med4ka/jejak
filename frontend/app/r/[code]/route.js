import { NextResponse } from "next/server";

const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

// Proxy redirect /r/{code} -> Go /r/{code} (same pattern as app/api/*).
//
// Why it is needed: public pages previously rendered hrefs pointing straight at
// the backend (http://localhost:8081/r/...): in production the URL bar showed
// "localhost:8081", other devices could not reach that address, and analytics
// were recorded from the server IP instead of the visitor's. With this proxy
// the browser only knows the page origin (localhost:3000 / the production
// domain) while the backend still receives the request, so redirects and
// analytics keep working exactly as before.
//
// 3 headers MUST be forwarded (if one is missing, analytics lie):
//   1. user-agent      → device classification (classifyDevice) & unique fingerprint
//   2. referer         → referrer_domain + source bucket (search/social/chat/etc.)
//   3. x-forwarded-for → the visitor's real IP (isUniqueClick = hash(IP|UA));
//      without it every click is attributed to the Next server IP.
//
// redirect: "manual" is CRUCIAL: without it fetch follows the 302 to the target
// URL and what reaches the browser is the target page content (status 200):
// not a navigation. The browser must receive the 302 + Location itself.
//
// THE BACKEND RESPONSE IS FORWARDED AS-IS, except errors originating from
// Jejak (404/410/5xx), which are re-rendered as a Jejak-style fallback page:
// a plain-text error ("URL not found") would only confuse visitors. The body of
// a 302 is NEVER inspected: whether the target URL is alive or dead is outside
// Jejak's control.

const CODE_RE = /^[A-Za-z0-9_-]{1,30}$/;

// DESIGN.md §1 tokens are used inline (the fallback HTML is not part of the
// Next CSS bundle, so Tailwind classes do not apply here).
const C = {
  paper: "#FAFAF7", // print-white
  ink: "#1C1A12",
  yellow: "#FFD23F", // flash-yellow
  muted: "rgba(28,26,18,0.75)",
};

const ICONS = {
  searchX:
    '<path d="M11 5a7 7 0 1 1-4.95 12.05L3 20"/><circle cx="11" cy="11" r="7"/><path d="M8.5 8.5l5 5M13.5 8.5l-5 5"/>',
  clock:
    '<circle cx="12" cy="12" r="9"/><path d="M12 7v5.5l3.5 2"/>',
  alert:
    '<path d="M10.3 4.3 2.6 17.6A2 2 0 0 0 4.3 20.6h15.4a2 2 0 0 0 1.7-3L13.7 4.3a2 2 0 0 0-3.4 0Z"/><path d="M12 9.5v4M12 17h.01"/>',
};

const FALLBACK = {
  404: {
    icon: ICONS.searchX,
    title: "Link tidak ditemukan.",
    text: "Link ini belum ada atau mungkin salah ketik. Buka lagi halaman profil pemiliknya untuk cek daftar link.",
  },
  410: {
    icon: ICONS.clock,
    title: "Link sudah kadaluarsa.",
    text: "Link ini sudah lewat masa berlakunya dan tidak bisa dibuka lagi.",
  },
  "5xx": {
    icon: ICONS.alert,
    title: "Sedang ada masalah. Coba lagi.",
    text: "Server Jejak sedang sibuk atau belum terhubung. Coba beberapa saat lagi, ya.",
  },
};

function fallbackPage(kind) {
  const { icon, title, text } = FALLBACK[kind];
  const arrow =
    '<path d="M4 12h15M13 6l6 6-6 6"/>';
  return `<!DOCTYPE html>
<html lang="id">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>${title}: Jejak</title>
<style>
  * { box-sizing: border-box; }
  html, body { margin: 0; padding: 0; }
  body {
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 24px;
    background: ${C.paper};
    color: ${C.ink};
    font-family: "Work Sans", "Segoe UI", system-ui, -apple-system, sans-serif;
    -webkit-font-smoothing: antialiased;
  }
  .card {
    width: 100%;
    max-width: 27rem;
    text-align: center;
    background: ${C.paper};
    border: 2px solid ${C.ink};
    border-radius: 16px;
    box-shadow: 4px 4px 0 ${C.ink};
    padding: 40px 28px 36px;
  }
  .brand {
    display: inline-block;
    font-family: "Space Grotesk", "Segoe UI", system-ui, sans-serif;
    font-size: 11px;
    font-weight: 700;
    letter-spacing: .18em;
    text-transform: uppercase;
    border: 2px solid ${C.ink};
    border-radius: 9999px;
    padding: 4px 12px;
    margin-bottom: 26px;
  }
  .icon { display: block; margin: 0 auto; }
  h1 {
    font-family: "Space Grotesk", "Segoe UI", system-ui, sans-serif;
    font-size: 30px;
    line-height: 1.15;
    font-weight: 700;
    margin: 22px 0 0;
  }
  p { font-size: 16px; line-height: 1.55; color: ${C.muted}; margin: 12px 0 0; }
  .cta {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    margin-top: 28px;
    padding: 12px 28px;
    background: ${C.yellow};
    color: ${C.ink};
    border: 2px solid ${C.ink};
    border-radius: 9999px;
    box-shadow: 4px 4px 0 ${C.ink};
    font-size: 15px;
    font-weight: 700;
    text-decoration: none;
    transition: transform .15s ease;
  }
  .cta:hover { transform: translateY(-2px); }
  .cta:active { transform: translateY(0); }
  .cta:focus-visible { outline: 3px solid ${C.ink}; outline-offset: 3px; }
</style>
</head>
<body>
  <main class="card">
    <span class="brand">Jejak</span>
    <svg class="icon" width="56" height="56" viewBox="0 0 24 24" fill="none"
      stroke="${C.ink}" stroke-width="1.5" stroke-linecap="round"
      stroke-linejoin="round" aria-hidden="true">${icon}</svg>
    <h1>${title}</h1>
    <p>${text}</p>
    <a class="cta" href="/">Balik ke Beranda
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="${C.ink}"
        stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"
        aria-hidden="true">${arrow}</svg>
    </a>
  </main>
</body>
</html>`;
}

function htmlResponse(kind, status) {
  return new NextResponse(fallbackPage(kind), {
    status,
    headers: {
      "Content-Type": "text/html; charset=utf-8",
      "Cache-Control": "no-store",
    },
  });
}

function clientForward(req) {
  const headers = new Headers();
  for (const name of ["user-agent", "referer", "cookie", "accept", "accept-language"]) {
    const v = req.headers.get(name);
    if (v) headers.set(name, v);
  }
  // Prefer an XFF already written by an upstream proxy; if none exists, use the
  // peer IP known to Next (req.ip). If both are empty the header is not sent
  // (the backend falls back to RemoteAddr = the Next server: a degradation,
  // not an error).
  const peer = req.headers.get("x-forwarded-for") || req.ip || req.headers.get("x-real-ip");
  if (peer) headers.set("x-forwarded-for", peer);
  return headers;
}

async function proxy(req, code) {
  if (!CODE_RE.test(code || "")) {
    return new NextResponse("Kode link tidak valid", {
      status: 400,
      headers: { "Content-Type": "text/plain; charset=utf-8" },
    });
  }

  let goRes;
  try {
    goRes = await fetch(`${GO_API_URL}/r/${code}`, {
      headers: clientForward(req),
      redirect: "manual",
      cache: "no-store",
    });
  } catch {
    // Backend unreachable → Jejak 5xx page (status 502).
    return htmlResponse("5xx", 502);
  }

  // 302 (carrying Location) and 410/404/5xx are handled per status. All other
  // statuses (301, 307, 400, etc.) pass through unchanged.
  if (goRes.status === 404) return htmlResponse("404", 404);
  if (goRes.status === 410) return htmlResponse("410", 410);
  if (goRes.status >= 500) return htmlResponse("5xx", goRes.status);

  // The response passes through as-is: 302 (with Location), 400, 429, ...
  // The status is never swallowed or altered.
  const out = new Headers();
  for (const name of ["location", "content-type", "cache-control", "set-cookie"]) {
    const v = goRes.headers.get(name);
    if (v) out.set(name, v);
  }
  const body =
    goRes.status === 204 || goRes.status === 304 ? null : await goRes.arrayBuffer();
  return new NextResponse(body, { status: goRes.status, headers: out });
}

export async function GET(req, props) {
  const params = await props.params;
  return proxy(req, params?.code);
}

// HEAD is used by some crawlers/monitors: identical path and headers.
export async function HEAD(req, props) {
  const params = await props.params;
  return proxy(req, params?.code);
}
