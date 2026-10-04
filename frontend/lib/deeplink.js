// E-commerce URL detection for the shorten-form badge and the edit-modal
// deep-link status (deep link feature, 2026-10-04).
//
// This MIRRORS backend/internal/deeplink: the browser needs an instant,
// offline verdict while the user types (a round-trip per keystroke would be
// silly), and the Go side keeps its own copy because the redirect handler
// must decide server-side anyway. Keep the two in sync: same domains, same
// suffix-safe host matching (a raw "url.includes('shopee.co.id')" would let
// "shopee.co.id.evil.com" pass), same http(s)-only rule.

const PLATFORMS = [
  { id: "shopee", name: "Shopee", domains: ["shopee.co.id", "shopee.sg", "shopee.my", "shopee.th", "shopee.vn", "shopee.com.tw"] },
  { id: "tokopedia", name: "Tokopedia", domains: ["tokopedia.com"] },
  { id: "tiktok", name: "TikTok Shop", domains: ["tiktok.com"] },
  { id: "lazada", name: "Lazada", domains: ["lazada.co.id", "lazada.com"] },
  { id: "blibli", name: "Blibli", domains: ["blibli.com"] },
  { id: "bukalapak", name: "Bukalapak", domains: ["bukalapak.com"] },
];

function matchHost(host, domain) {
  return host === domain || host.endsWith("." + domain);
}

// detectEcommerce returns { id, name } for a supported e-commerce URL, or
// null. Never throws: input is a raw text field.
export function detectEcommerce(raw) {
  if (typeof raw !== "string" || raw.trim() === "") return null;
  let u;
  try {
    u = new URL(raw.trim());
  } catch {
    return null;
  }
  if (u.protocol !== "http:" && u.protocol !== "https:") return null;
  const host = u.hostname.toLowerCase().replace(/^www\./, "");
  for (const p of PLATFORMS) {
    if (p.domains.some((d) => matchHost(host, d))) {
      return { id: p.id, name: p.name };
    }
  }
  return null;
}
