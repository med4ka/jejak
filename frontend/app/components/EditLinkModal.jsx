"use client";

import { useEffect, useState } from "react";
import { motion } from "framer-motion";
import SubmitButton from "./SubmitButton";
import { SPRING } from "../../lib/animations";
import { formatLocal, fromInputValue, minuteKey, toInputValue } from "../../lib/expiry";
import { detectEcommerce } from "../../lib/deeplink";

const input =
  "w-full rounded-xl border-2 border-ink bg-print-white px-3 py-2 text-sm text-ink placeholder:text-muted focus:border-flash-yellow focus:ring-2 focus:ring-flash-yellow/30 focus:outline-hidden transition-all duration-150";

// Verdict labels for the health status (hard-coded Indonesian, same policy
// as the other new copy in this task: the i18n dictionaries are off-limits).
function healthLabel(status) {
  if (status === "healthy") return "sehat ✓";
  if (status === "broken") return "rusak";
  if (status === "timeout") return "timeout";
  return "belum dicek";
}

// Edit link modal (Smart Link): set per-device destination URLs (iOS/Android)
// plus tags in a single save (endpoint PUT /api/links/{short_code},
// full-replace). Clearing the iOS/Android URL falls back to the primary URL;
// clearing tags removes them all. The modal shell follows QrModal: backdrop +
// panel fade/scale driven by SPRING (UI state → spring, Apple-style detail
// #1), Escape closes.
export default function EditLinkModal({ link, onClose, onSaved, st }) {
  const [iosUrl, setIosUrl] = useState("");
  const [androidUrl, setAndroidUrl] = useState("");
  const [tagInput, setTagInput] = useState("");
  // Expiry: the input holds LOCAL time (datetime-local). It is sent to the API
  // ONLY when it CHANGES (see onSubmit): an already-expired old value is never
  // re-sent, so saving device_rules/tags cannot trip the 422 rejection.
  const [expiresInput, setExpiresInput] = useState("");
  // Password (link protection): "" = keep whatever exists; typed value =
  // set/change; clearPassword = remove the password (sent as ""). The hash is
  // never returned by the API, so `link.has_password` only tells the mode.
  const [password, setPassword] = useState("");
  const [clearPassword, setClearPassword] = useState(false);
  // Fallback URL (health monitor): sent ONLY when it changed (trimmed).
  const [fallbackUrl, setFallbackUrl] = useState("");
  // Health status shown in the modal: seeded from the link, refreshed by
  // "Cek sekarang" (POST check-health) WITHOUT a parent refetch, so a typed
  // draft is never clobbered by a background list reload.
  const [healthStatus, setHealthStatus] = useState("unknown");
  const [healthMsg, setHealthMsg] = useState("");
  const [checking, setChecking] = useState(false);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  // Fields are re-seeded whenever [link] changes; the parent re-mounts this
  // modal with key={link.short_code} when a different link is edited.
  useEffect(() => {
    const rules = link.device_rules || {};
    setIosUrl(rules.ios || "");
    setAndroidUrl(rules.android || "");
    setTagInput(Array.isArray(link.tags) ? link.tags.join(", ") : "");
    setExpiresInput(toInputValue(link.expires_at));
    setPassword("");
    setClearPassword(false);
    setFallbackUrl(link.fallback_url || "");
    setHealthStatus(link.health_status || "unknown");
    setHealthMsg("");
    setError("");
  }, [link]);

  useEffect(() => {
    function onKey(e) {
      if (e.key === "Escape") {
        onClose();
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  async function onSubmit(e) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      const deviceRules = {};
      if (iosUrl.trim() !== "") {
        deviceRules.ios = iosUrl.trim();
      }
      if (androidUrl.trim() !== "") {
        deviceRules.android = androidUrl.trim();
      }
      const tags = tagInput
        .split(",")
        .map((t) => t.trim().toLowerCase())
        .filter((t, i, arr) => t !== "" && arr.indexOf(t) === i)
        .slice(0, 5);

      const body = { device_rules: deviceRules, tags };
      // expires_at is attached ONLY when it changed (compared per minute: the
      // input's second-level precision must not count as an edit). null =
      // clear (remove the expiry).
      const nextIso = fromInputValue(expiresInput);
      const origIso = link.expires_at || null;
      if (minuteKey(nextIso) !== minuteKey(origIso)) {
        body.expires_at = nextIso;
      }
      // Password: omitted = keep (so an untouched draft never resets the
      // existing password). "" = remove, value = set (backend validates the
      // 4-72 rule and bcrypt-hashes it).
      if (clearPassword) {
        body.password = "";
      } else if (password !== "") {
        if (password.length < 4 || password.length > 72) {
          throw new Error("Password harus 4-72 karakter.");
        }
        body.password = password;
      }
      // Fallback URL: attached ONLY when it changed (trimmed). "" = clear.
      const nextFallback = fallbackUrl.trim();
      if (nextFallback !== (link.fallback_url || "")) {
        if (nextFallback !== "") {
          let parsed;
          try {
            parsed = new URL(nextFallback);
          } catch {
            throw new Error("Fallback URL tidak valid.");
          }
          if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
            throw new Error("Fallback URL harus http:// atau https://");
          }
        }
        body.fallback_url = nextFallback;
      }

      const res = await fetch(`/api/links/${link.short_code}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      const text = await res.text();
      let data = {};
      try {
        data = JSON.parse(text);
      } catch {
        throw new Error(text || "Gagal menyimpan");
      }
      if (!res.ok) {
        throw new Error(data.error || text || "Gagal menyimpan");
      }
      onSaved();
      onClose();
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }

  // Manual health probe (health monitor): runs the same live HEAD/GET check
  // as the worker (rate-limited to 10/min per IP by the backend) and shows
  // the verdict inline. Deliberately does NOT call onSaved(): a parent
  // refetch would replace the `link` prop and reseed the form, discarding a
  // half-typed draft; the dashboard badge catches up on the next save/load.
  async function onCheckHealth() {
    setError("");
    setHealthMsg("");
    setChecking(true);
    try {
      const res = await fetch(`/api/links/${link.short_code}/check-health`, {
        method: "POST",
      });
      const text = await res.text();
      let data = {};
      try {
        data = JSON.parse(text);
      } catch {
        throw new Error(text || "Gagal mengecek");
      }
      if (!res.ok) {
        throw new Error(data.error || text || "Gagal mengecek");
      }
      const status = data.health_status || "unknown";
      setHealthStatus(status);
      setHealthMsg(`Hasil cek: ${healthLabel(status)}`);
    } catch (err) {
      setError(err.message);
    } finally {
      setChecking(false);
    }
  }

  return (
    <div className="pointer-events-none fixed inset-0 z-[60] flex items-end justify-center sm:items-start sm:px-4 sm:pt-[14vh]">
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        transition={SPRING}
        onClick={onClose}
        className="pointer-events-auto absolute inset-0 bg-ink/60"
        aria-hidden="true"
      />
      <motion.div
        role="dialog"
        aria-modal="true"
        aria-label={`Edit link /${link.short_code}`}
        initial={{ opacity: 0, y: 24, scale: 0.98 }}
        animate={{ opacity: 1, y: 0, scale: 1 }}
        exit={{ opacity: 0, y: 12, scale: 0.98 }}
        transition={SPRING}
        className="pointer-events-auto relative w-full rounded-t-2xl border-2 border-ink bg-print-white p-6 text-ink sm:max-w-[440px] sm:rounded-2xl"
      >
        <p className="font-mono text-sm font-bold">/{link.short_code}</p>
        <p className="mt-1 text-sm text-muted">URL utama: {link.original_url}</p>

        {/* Deep link status (read-only display, deep link feature): aktif =
            the destination is a supported e-commerce URL (the redirect
            handler opens the merchant app first on mobile); mati = plain
            web redirect. A per-link toggle would need a new DB column, which
            was out of scope (migration policy). */}
        {(() => {
          const d = detectEcommerce(link.original_url);
          return d ? (
            <p className="mt-2 inline-block rounded-full border-2 border-ink bg-flash-yellow px-3 py-1 text-xs font-bold text-ink">
              🛍️ Deep link: aktif — buka {d.name} di HP
            </p>
          ) : (
            <p className="mt-2 text-xs text-muted">Deep link: mati — redirect biasa ke web</p>
          );
        })()}

        {/* Health monitor (read-only status + manual probe): status badge
            follows the deep-link badge pattern; "Cek sekarang" triggers the
            live check (rate-limited server-side). */}
        <div className="mt-2 flex flex-wrap items-center gap-2 text-xs">
          <span
            className={`inline-block rounded-full border-2 border-ink px-3 py-1 font-bold ${
              healthStatus === "broken"
                ? "bg-flash-coral text-ink"
                : healthStatus === "timeout"
                  ? "bg-flash-yellow text-ink"
                  : healthStatus === "healthy"
                    ? "bg-emerald-200 text-ink"
                    : "bg-paper-grey text-muted"
            }`}
          >
            Kesehatan: {healthLabel(healthStatus)}
          </span>
          <button
            type="button"
            onClick={onCheckHealth}
            disabled={checking}
            className="rounded-full border-2 border-ink bg-print-white px-3 py-1 text-xs font-bold text-ink transition-opacity duration-150 hover:opacity-80 disabled:opacity-50"
          >
            {checking ? "Mengecek..." : "Cek sekarang"}
          </button>
        </div>
        {healthMsg !== "" && <p className="mt-1 text-xs text-muted">{healthMsg}</p>}

        <form onSubmit={onSubmit} className="mt-4 flex flex-col gap-3">
          <label className="text-sm font-medium">
            URL untuk iOS (opsional)
            <input
              value={iosUrl}
              onChange={(e) => setIosUrl(e.target.value)}
              placeholder="https://apps.apple.com/..."
              className={`${input} mt-1`}
            />
          </label>
          <label className="text-sm font-medium">
            URL untuk Android (opsional)
            <input
              value={androidUrl}
              onChange={(e) => setAndroidUrl(e.target.value)}
              placeholder="https://play.google.com/..."
              className={`${input} mt-1`}
            />
          </label>
          <label className="text-sm font-medium">
            Tag (koma, maks 5)
            <input
              value={tagInput}
              onChange={(e) => setTagInput(e.target.value)}
              placeholder="deeplink, promo"
              className={`${input} mt-1`}
            />
          </label>
          {/* Password (link protection): never auto-cleared. When one exists
              it stays until "Hapus password" is chosen explicitly; an empty
              field means "keep", never "remove". */}
          {link.has_password ? (
            <label className="text-sm font-medium">
              Password {clearPassword ? "(aktif — akan dihapus)" : "(aktif)"}
              <input
                type="password"
                autoComplete="new-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                disabled={clearPassword}
                minLength={4}
                maxLength={72}
                placeholder={clearPassword ? "Dihapus saat disimpan" : "Kosongkan untuk tetap, isi untuk ganti"}
                className={`${input} mt-1`}
              />
            </label>
          ) : (
            <label className="text-sm font-medium">
              Password baru (opsional)
              <input
                type="password"
                autoComplete="new-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                minLength={4}
                maxLength={72}
                placeholder="4-72 karakter"
                className={`${input} mt-1`}
              />
            </label>
          )}
          <div className="flex items-center gap-2 text-xs">
            {link.has_password && !clearPassword && (
              <button
                type="button"
                onClick={() => setClearPassword(true)}
                className="rounded-full border-2 border-ink bg-print-white px-3 py-1 font-bold text-ink transition-opacity duration-150 hover:opacity-80"
              >
                Hapus password
              </button>
            )}
            {clearPassword && (
              <>
                <span className="font-medium text-ink">Password akan dihapus saat disimpan.</span>
                <button
                  type="button"
                  onClick={() => setClearPassword(false)}
                  className="rounded-full border-2 border-ink bg-print-white px-3 py-1 font-bold text-ink transition-opacity duration-150 hover:opacity-80"
                >
                  Batal
                </button>
              </>
            )}
          </div>
          {/* Expiry (link management): LOCAL on screen, UTC in the API. No
              `min` here: a stale value (possibly already past) must not block
              saving the other fields; the 422 validation applies only when the
              value CHANGES. */}
          <label className="text-sm font-medium">
            Aktif sampai (opsional)
            <span className="relative mt-1 block">
              <input
                type="datetime-local"
                value={expiresInput}
                onChange={(e) => setExpiresInput(e.target.value)}
                className={input}
              />
              {expiresInput !== "" && (
                <button
                  type="button"
                  onClick={() => setExpiresInput("")}
                  aria-label="Hapus batas waktu"
                  title="Hapus batas waktu (aktif selamanya)"
                  className="absolute right-2 top-1/2 -translate-y-1/2 text-sm text-muted transition-colors duration-150 hover:text-ink"
                >
                  ✕
                </button>
              )}
            </span>
            <span className="mt-1 block text-xs font-normal text-muted">
              {link.expires_at
                ? `Sekarang: ${formatLocal(link.expires_at)}`
                : "Tanpa batas waktu: biarkan kosong untuk tetap aktif."}
            </span>
          </label>
          {/* Fallback URL (health monitor): destination used by the broken-
              link interstitial when the primary URL is down. Sent only when
              it changed; empty = no fallback (visitor gets a 503 instead). */}
          <label className="text-sm font-medium">
            Fallback URL (opsional)
            <input
              type="url"
              value={fallbackUrl}
              onChange={(e) => setFallbackUrl(e.target.value)}
              placeholder="https://contoh.com/halaman-cadangan"
              className={`${input} mt-1`}
            />
            <span className="mt-1 block text-xs font-normal text-muted">
              Kalau link utama rusak, user akan diarahkan ke URL ini.
            </span>
          </label>

          <SubmitButton
            isLoading={loading}
            loadingLabel="Menyimpan..."
            className={`rounded-full border-2 border-ink ${st.accent} px-4 py-2.5 text-sm font-bold transition-[filter] duration-150 hover:brightness-95`}
          >
            Simpan
          </SubmitButton>
        </form>

        {error !== "" && (
          <p className="mt-3 rounded-xl border-2 border-ink bg-paper-grey px-4 py-2.5 text-sm font-medium text-ink">{error}</p>
        )}
      </motion.div>
    </div>
  );
}