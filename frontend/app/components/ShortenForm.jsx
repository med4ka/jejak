"use client";

import { useState } from "react";
import { motion } from "framer-motion";
import SubmitButton from "./SubmitButton";
import CopyButton from "./CopyButton";
import { fromInputValue, minInputValue } from "../../lib/expiry";
import { sameOriginShortUrl } from "../../lib/shortlink";
import { detectEcommerce } from "../../lib/deeplink";
import { useTranslation } from "../../lib/I18nProvider";

// Reusable shorten form (extracted from the homepage during the phase A
// refactor). A pure code move: behavior and appearance unchanged: URL input,
// custom slug, tags, Shorten button, error box, and result display.
//
// - `st`   : result of themeStyles(theme) passed in by the parent, so the
//            form is styled with the theme of the page that renders it (the
//            form does not hold a theme of its own).
// - onSuccess(data): optional callback receiving the API response after a
//            successful shorten: for follow-up needs (e.g. refreshing the
//            dashboard list).
export default function ShortenForm({ st, onSuccess }) {
  const { t } = useTranslation();
  const [url, setUrl] = useState("");
  const [slug, setSlug] = useState("");
  const [tagInput, setTagInput] = useState("");
  // Optional expiry (link management): the datetime-local input holds LOCAL
  // time; it is sent as ISO UTC (fromInputValue): the backend stores pure
  // UTC.
  const [expiresAt, setExpiresAt] = useState("");
  // Optional password (link protection): empty = no password. The backend
  // rule is 4-72 chars (also mirrored by minLength/maxLength below, which
  // apply only when the field is non-empty: the field is not `required`).
  const [password, setPassword] = useState("");
  const [shortUrl, setShortUrl] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  // min-w-0: lets the input inside the flex row shrink below its intrinsic
  // width (320px audit: without it, input plus button can widen until they
  // trigger horizontal scrolling).
  const inputThemed = `w-full min-w-0 ${st.radius} ${st.borderW} ${st.border} ${st.card} px-4 py-2.5 text-sm ${st.text} ${st.placeholder} focus:border-flash-yellow focus:ring-2 focus:ring-flash-yellow/30 focus:outline-hidden transition-all duration-150`;

  async function onSubmit(e) {
    e.preventDefault();
    setError("");
    setShortUrl("");
    setLoading(true);
    try {
      // Pre-split the comma-separated input on the client (the server
      // re-validates and normalizes on its own: the authoritative limit
      // always lives in the backend).
      const tags = tagInput
        .split(",")
        .map((t) => t.trim().toLowerCase())
        .filter((t, i, arr) => t !== "" && arr.indexOf(t) === i)
        .slice(0, 5);
      const res = await fetch("/api/shorten", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          url,
          slug: slug.trim() === "" ? undefined : slug.trim(),
          tags,
          // local → UTC; empty means no expiry (the field is omitted entirely).
          expires_at: fromInputValue(expiresAt) || undefined,
          // Omitted entirely when empty: the backend only hashes a non-empty
          // password (bcrypt, 4-72 chars).
          password: password !== "" ? password : undefined,
        }),
      });
      const data = await res.json();
      if (!res.ok) {
        // The error message must state WHAT went wrong and WHAT to do about it.
        // The server sends specific messages (invalid URL, duplicate slug,
        // etc.): this fallback exists only for responses that carry none.
        throw new Error(data.error || t("errors.shorten.createFailed"));
      }
      // The proxy response contains a short URL built by the backend from its
      // own request Host (http://localhost:8081/r/…). It is forced to the
      // current origin so the link that is displayed/copied opens the Next
      // proxy rather than the backend directly (public-page redirect fix,
      // 2026-09-30).
      setShortUrl(sameOriginShortUrl(data.shortUrl, null));
      // The password is write-only (only a bcrypt hash reaches the DB): drop
      // it from the form right after a successful shorten.
      setPassword("");
      // Track anonymous links for later claiming: ONLY while logged out.
      // Logged-in users are assigned the link server-side, so no tracking is
      // needed. Capped at the 50 most recent entries (oldest dropped);
      // read/write failures are ignored silently.
      if (!localStorage.getItem("jejak_username")) {
        try {
          const code = data.shortUrl.split("/").pop();
          const raw = localStorage.getItem("jejak_unclaimed_links");
          const parsed = raw ? JSON.parse(raw) : [];
          const list = Array.isArray(parsed) ? parsed : [];
          if (code && !list.includes(code)) {
            list.push(code);
          }
          localStorage.setItem("jejak_unclaimed_links", JSON.stringify(list.slice(-50)));
        } catch {
          // storage unavailable or corrupt: the shorten still succeeds and
          // claiming is skipped
        }
      }
      if (onSuccess) {
        onSuccess(data);
      }
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }

  // Instant e-commerce verdict while typing (deep link feature): shows the
  // platform badge the moment a Shopee/Tokopedia/... URL lands in the field.
  // Copy is intentionally hard-coded Indonesian (like EditLinkModal): the
  // i18n dictionaries were off-limits for this task.
  const detected = detectEcommerce(url);

  return (
    <>
      <form onSubmit={onSubmit} className="flex flex-col gap-2">
        <div className="flex gap-2">
          <input
            type="url"
            required
            aria-label={t("forms.shorten.urlAriaLabel")}
            placeholder={t("forms.shorten.urlPlaceholder")}
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            className={inputThemed}
          />
          <SubmitButton
            isLoading={loading}
            loadingLabel={t("forms.shorten.loadingLabel")}
            className={`shrink-0 ${st.radiusFull} ${st.borderW} ${st.border} ${st.accent} px-5 py-3 text-sm font-bold transition-[filter] duration-150 hover:brightness-95`}
          >
            {t("forms.shorten.submitLabel")}
          </SubmitButton>
        </div>
        <input
          aria-label={t("forms.shorten.slugAriaLabel")}
          placeholder={t("forms.shorten.slugPlaceholder")}
          value={slug}
          onChange={(e) => setSlug(e.target.value)}
          maxLength={30}
          className={inputThemed}
        />
        <input
          aria-label={t("forms.shorten.tagAriaLabel")}
          placeholder={t("forms.shorten.tagPlaceholder")}
          value={tagInput}
          onChange={(e) => setTagInput(e.target.value)}
          className={inputThemed}
        />
        {/* Optional expiry: datetime-local (values are local time) with
            min = now + 1h: the backend rejects values inside that window
            (422). The ✕ button clears the limit. */}
        <div className="flex gap-2">
          <input
            type="datetime-local"
            aria-label={t("forms.shorten.expiryAriaLabel")}
            min={minInputValue()}
            value={expiresAt}
            onChange={(e) => setExpiresAt(e.target.value)}
            className={`${inputThemed} min-w-0`}
          />
          {expiresAt !== "" && (
            <button
              type="button"
              onClick={() => setExpiresAt("")}
              title={t("common.clearExpiry")}
              aria-label={t("common.clearExpiry")}
              className={`shrink-0 ${st.radius} ${st.borderW} ${st.border} ${st.card} px-3 text-sm ${st.text} transition-opacity duration-150 hover:opacity-70`}
            >
              ✕
            </button>
          )}
        </div>
        {/* Optional password (link protection): visitors see a password
            form before the redirect. Omitted when empty; 4-72 chars. */}
        <input
          type="password"
          aria-label="Password link (opsional)"
          placeholder="Password link (opsional, 4-72 karakter)"
          autoComplete="new-password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          minLength={4}
          maxLength={72}
          className={inputThemed}
        />
        <p className={`text-xs ${st.textMuted}`}>
          {t("forms.shorten.helperText")}
        </p>
      </form>

      {/* Deep link badge: rendered as soon as the typed URL matches a
          supported platform (mirrors what the redirect handler will do on
          mobile: open the merchant app first, web as fallback). */}
      {detected && !shortUrl && (
        <div
          role="status"
          className={`${st.radius} ${st.borderW} ${st.border} ${st.accent} px-4 py-2.5 text-sm font-bold`}
        >
          🛍️ {detected.name} terdeteksi — link akan buka app di HP
        </div>
      )}

      {error !== "" && (
        <div className={`${st.radius} ${st.borderW} ${st.border} ${st.card} px-4 py-2.5 text-sm font-medium ${st.text}`}>
          {error}
        </div>
      )}

      {shortUrl !== "" && (
        <div className={`${st.radius} ${st.borderW} ${st.border} ${st.card} px-4 py-3 ${st.text}`}>
          <div className="flex items-center gap-2">
            <a href={shortUrl} target="_blank" rel="noreferrer" className="min-w-0 flex-1 truncate font-mono text-sm underline">
              {shortUrl}
            </a>
            <CopyButton
              text={shortUrl}
              label={t("common.copy")}
              className={`shrink-0 rounded-full ${st.borderW} ${st.border} ${st.card} px-4 py-3.5 text-xs font-bold transition-colors duration-150 hover:opacity-90`}
            />
          </div>
          <p className={`text-sm ${st.textMuted}`}>{t("forms.shorten.resultHint")}</p>
        </div>
      )}
    </>
  );
}
