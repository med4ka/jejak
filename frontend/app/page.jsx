"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { motion } from "framer-motion";
import { THEMES, themeStyles } from "../lib/themes";
import ThemeBackdrop from "./components/ThemeBackdrop";

// Homepage (Task 4 — dulu SELALU hardcode tema CLASSIC walau kreator login:
// bg-print-white, input border-ink bg-print-white text-ink,
// placeholder:text-muted, error box bg-paper-grey dst; akibatnya user yang
// set profil darkroom/glass/coral, pas buka halaman utama, wajahnya NABRAK
// dengan tema mereka — seolah "shortener bukan bagian dari brand aku".
// Sekarang homepage ikut tema kreator:
//   - guest / belum login → classic (default, aman).
//   - login               → profile.theme dari GET /api/profile di-render
//     persis seperti halaman publik /u: body[data-profile-theme] + 
//     ThemeBackdrop (gradien glass/grain halus di semua tema) + semua token
//     styling lewat st.* (card/border/teks/placeholder ikut panggung).
// Shortener tetap berfungsi identik — hanya wajahnya yang ikut tema.
export default function Home() {
  const [url, setUrl] = useState("");
  const [slug, setSlug] = useState("");
  const [tagInput, setTagInput] = useState("");
  const [shortUrl, setShortUrl] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [theme, setTheme] = useState("classic");
  const st = themeStyles(theme);

  // Ikut tema user login (guest → classic). Body data-profile-theme di-set
  // supaya globals.css (background glass/darkroom) + grain ikut; cleanup saat
  // unmount (sama pola DashboardClient/Navbar di /u).
  useEffect(() => {
    document.body.dataset.profileTheme = theme;
    const u = localStorage.getItem("jejak_username");
    if (!u) {
      return;
    }
    // Tema hanya dipakai kalau profil benar-benar login punya theme valid.
    fetch("/api/profile", { cache: "no-store" })
      .then(async (res) => {
        const text = await res.text();
        let d = {};
        try {
          d = JSON.parse(text);
        } catch {
          return;
        }
        if (res.ok && THEMES.includes(d.theme)) {
          setTheme(d.theme);
        }
      })
      .catch(() => {});
    return () => {
      delete document.body.dataset.profileTheme;
    };
  }, []);

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
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className={st.text}>
      <ThemeBackdrop theme={theme} />
      <p className={`inline-block ${st.radiusFull} ${st.borderW} ${st.border} ${st.chip} px-3 py-1 font-mono text-xs font-bold uppercase tracking-[0.15em] ${st.textMuted}`}>
        Link-in-bio untuk kreator
      </p>
      <h1 className={`mt-3 ${st.headingFont} text-4xl font-bold leading-tight ${st.text}`}>
        Satu halaman
        <br />
        untuk semua link kamu.
      </h1>
      <p className={`mt-2 text-sm ${st.textMuted}`}>
        Persingkat link di bawah — kalau kamu login, link otomatis masuk ke profil publikmu.
      </p>

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

      <p className="mt-6">
        <Link href="/links" className={`inline-block ${st.radiusFull} ${st.borderW} ${st.border} ${st.accent} px-5 py-2 text-sm font-bold`}>
          Lihat analytics →
        </Link>
      </p>
    </main>
  );
}
