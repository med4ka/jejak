"use client";

import { useEffect, useState } from "react";
import { motion } from "framer-motion";

const input =
  "w-full rounded-xl border-2 border-ink bg-print-white px-3 py-2 text-sm text-ink placeholder:text-muted focus:outline-none";

// Modal edit link (Smart Link): atur URL tujuan per-device (iOS/Android) +
// tags, dalam 1 simpanan (endpoint PUT /api/links/{short_code}, full-replace).
// Kosongkan URL iOS/Android = pakai URL utama seperti biasa. Kosongkan tag =
// hapus semua tag. Shell modal mengikuti pola QrModal/AuthModal: backdrop +
// fade/scale easeOut 200ms, Escape menutup.
export default function EditLinkModal({ link, onClose, onSaved, st }) {
  const [iosUrl, setIosUrl] = useState("");
  const [androidUrl, setAndroidUrl] = useState("");
  const [tagInput, setTagInput] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  // Utama: render ulang via key={link.short_code} di parent kalau ganti link.
  useEffect(() => {
    const rules = link.device_rules || {};
    setIosUrl(rules.ios || "");
    setAndroidUrl(rules.android || "");
    setTagInput(Array.isArray(link.tags) ? link.tags.join(", ") : "");
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

      const res = await fetch(`/api/links/${link.short_code}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ device_rules: deviceRules, tags }),
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
        transition={{ duration: 0.2, ease: "easeOut" }}
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
        transition={{ duration: 0.2, ease: "easeOut" }}
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

          <motion.button
            type="submit"
            disabled={loading}
            whileTap={{ scale: 0.97 }}
            transition={{ duration: 0.2, ease: "easeOut" }}
            className={`rounded-full border-2 border-ink ${st.accent} px-4 py-2.5 text-sm font-bold transition-[filter] duration-150 hover:brightness-95 disabled:opacity-50`}
          >
            {loading ? "..." : "Simpan"}
          </motion.button>
        </form>

        {error !== "" && (
          <p className="mt-3 rounded-xl border-2 border-ink bg-paper-grey px-4 py-2.5 text-sm font-medium text-ink">{error}</p>
        )}
      </motion.div>
    </div>
  );
}