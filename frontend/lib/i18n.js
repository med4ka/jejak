// i18n: client-side dictionary approach (3 locales: id default, en, de).
// Cookie NEXT_LOCALE is the single source of truth for the active locale;
// it predates this module (old setup) and is read on the server in
// app/layout.jsx and on the client here.

export const LOCALES = ["id", "en", "de"];
export const DEFAULT_LOCALE = "id";

// Accepts "en", "EN", "en-US", "id-ID", ... -> returns a supported locale
// or null when nothing matches (callers fall back to DEFAULT_LOCALE).
export function normalizeLocale(value) {
  if (!value || typeof value !== "string") return null;
  const v = value.trim().toLowerCase();
  return LOCALES.find((l) => v === l || v.startsWith(`${l}-`)) || null;
}

function readCookie(name) {
  if (typeof document === "undefined") return null;
  const match = document.cookie.match(new RegExp(`(?:^|;\\s*)${name}=([^;]*)`));
  return match ? decodeURIComponent(match[1]) : null;
}

// Server: pass cookies().get("NEXT_LOCALE")?.value (app/layout.jsx).
// Client: call with no argument: reads document.cookie.
// Missing/unsupported cookie -> DEFAULT_LOCALE ("id").
export function getLocale(cookieValue) {
  const raw = cookieValue !== undefined && cookieValue !== null ? cookieValue : readCookie("NEXT_LOCALE");
  return normalizeLocale(raw) || DEFAULT_LOCALE;
}

// Persist the choice for one year and reload so the server re-renders the
// tree with the new dictionary (layout reads the cookie on every request).
export function setLocale(locale) {
  const next = normalizeLocale(locale) || DEFAULT_LOCALE;
  if (typeof document !== "undefined") {
    document.cookie = `NEXT_LOCALE=${encodeURIComponent(next)}; path=/; max-age=31536000; samesite=lax`;
  }
  if (typeof window !== "undefined") window.location.reload();
  return next;
}
