"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { THEMES, themeStyles } from "../../lib/themes";
import ThemeBackdrop from "../components/ThemeBackdrop";
import ShortenForm from "../components/ShortenForm";

// Halaman /app — isi SAMA PERSIS dengan homepage lama (dipindah utuh pada
// refactor TAHAP B). Homepage "/" sekarang hanya redirect ke sini.
//
// Task 4 — dulu SELALU hardcode tema CLASSIC walau kreator login:
// bg-print-white, input border-ink bg-print-white text-ink,
// placeholder:text-muted, error box bg-paper-grey dst; akibatnya user yang
// set profil darkroom/glass/coral, pas buka halaman utama, wajahnya NABRAK
// dengan tema mereka — seolah "shortener bukan bagian dari brand aku".
// Sekarang halaman ini ikut tema kreator:
//   - guest / belum login → classic (default, aman).
//   - login               → profile.theme dari GET /api/profile di-render
//     persis seperti halaman publik /u: body[data-profile-theme] + 
//     ThemeBackdrop (gradien glass/grain halus di semua tema) + semua token
//     styling lewat st.* (card/border/teks/placeholder ikut panggung).
// Shortener tetap berfungsi identik — hanya wajahnya yang ikut tema.
// Form-nya sendiri sekarang diekstrak ke components/ShortenForm.jsx.
export default function AppHome() {
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

      <ShortenForm st={st} />

      <p className="mt-6">
        <Link href="/links" className={`inline-block ${st.radiusFull} ${st.borderW} ${st.border} ${st.accent} px-5 py-2 text-sm font-bold`}>
          Lihat analytics →
        </Link>
      </p>
    </main>
  );
}
