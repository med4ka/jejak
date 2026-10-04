"use client";

import { useEffect, useState } from "react";
import { motion } from "framer-motion";
import SubmitButton from "./SubmitButton";
import { SPRING } from "../../lib/animations";
import { formatLocal, fromInputValue, minuteKey, toInputValue } from "../../lib/expiry";

const input =
  "w-full rounded-xl border-2 border-ink bg-print-white px-3 py-2 text-sm text-ink placeholder:text-muted focus:border-flash-yellow focus:ring-2 focus:ring-flash-yellow/30 focus:outline-hidden transition-all duration-150";

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