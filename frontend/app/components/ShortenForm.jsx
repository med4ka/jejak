"use client";

import { useState } from "react";
import { motion } from "framer-motion";

// Form shorten reusable (diekstrak dari homepage pada refactor TAHAP A).
// Murni pindah kode — perilaku & tampilan identik: input URL, custom slug,
// tags, tombol Shorten, kotak error, dan tampilan hasil.
//
// - `st`   : hasil themeStyles(theme) dari parent, supaya styling form ikut
//            tema halaman yang memakainya (form tidak menyimpan tema sendiri).
// - onSuccess(data): opsional, dipanggil dengan respons API setelah shorten
//            berhasil — buat kebutuhan lanjutan (mis. refresh list dashboard).
export default function ShortenForm({ st, onSuccess }) {
  const [url, setUrl] = useState("");
  const [slug, setSlug] = useState("");
  const [tagInput, setTagInput] = useState("");
  const [shortUrl, setShortUrl] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const inputThemed = `w-full ${st.radius} ${st.borderW} ${st.border} ${st.card} px-4 py-2.5 text-sm ${st.text} ${st.placeholder} focus:outline-none`;

  async function onSubmit(e) {
    e.preventDefault();
    setError("");
    setShortUrl("");
    setLoading(true);
    try {
      // Pre-parse koma jadi array di client (server validasi ulang +
      // normalisasi sendiri — batas final selalu di backend).
      const tags = tagInput
        .split(",")
        .map((t) => t.trim().toLowerCase())
        .filter((t, i, arr) => t !== "" && arr.indexOf(t) === i)
        .slice(0, 5);
      const res = await fetch("/api/shorten", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ url, slug: slug.trim() === "" ? undefined : slug.trim(), tags }),
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error || "Gagal membuat short URL");
      }
      setShortUrl(data.shortUrl);
      // Lacak link anonim untuk klaim nanti: HANYA kalau belum login.
      // Yang login langsung ke-assign di server, tidak perlu tracking.
      // Cap 50 terbaru (buang paling lama); gagal baca/tulis abaikan diam-diam.
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
          // storage tidak tersedia/korup — shorten tetap sukses, klaim dilewati
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

  return (
    <>
      <form onSubmit={onSubmit} className="mt-6 flex flex-col gap-2">
        <div className="flex gap-2">
          <input
            type="url"
            required
            placeholder="https://..."
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            className={inputThemed}
          />
          <motion.button
            type="submit"
            disabled={loading}
            whileTap={{ scale: 0.97 }}
            transition={{ duration: 0.2, ease: "easeOut" }}
            className={`shrink-0 ${st.radiusFull} ${st.borderW} ${st.border} ${st.accent} px-5 py-2.5 text-sm font-bold transition-[filter] duration-150 hover:brightness-95 disabled:opacity-50`}
          >
            {loading ? "..." : "Shorten"}
          </motion.button>
        </div>
        <input
          placeholder="Custom slug (opsional, 3-30: huruf/angka/_/-)"
          value={slug}
          onChange={(e) => setSlug(e.target.value)}
          maxLength={30}
          className={inputThemed}
        />
        <input
          placeholder="Tag (opsional, pisah koma — mis. youtube, promo)"
          value={tagInput}
          onChange={(e) => setTagInput(e.target.value)}
          className={inputThemed}
        />
      </form>

      {error !== "" && (
        <div className={`mt-3 ${st.radius} ${st.borderW} ${st.border} ${st.card} px-4 py-2.5 text-sm font-medium ${st.text}`}>
          {error}
        </div>
      )}

      {shortUrl !== "" && (
        <div className={`mt-3 ${st.radius} ${st.borderW} ${st.border} ${st.card} px-4 py-3 ${st.text}`}>
          <a href={shortUrl} target="_blank" rel="noreferrer" className="font-mono text-sm underline">
            {shortUrl}
          </a>
          <p className={`text-sm ${st.textMuted}`}>Klik untuk membuka redirect-nya.</p>
        </div>
      )}
    </>
  );
}
