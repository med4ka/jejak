"use client";

import Link from "next/link";
import { motion } from "framer-motion";
import {
  ArrowRight,
  Camera,
  ChartColumnIncreasing,
  Link2,
  MonitorSmartphone,
  Plus,
  QrCode,
} from "lucide-react";
import { themeStyles } from "../lib/themes";
import ThemeBackdrop from "./components/ThemeBackdrop";

// =====================================================================
// LANDING PAGE - INSTANT PRINT (TAHAP C)
// =====================================================================
// Homepage "/" dulu (TAHAP B) cuma redirect kosong ke /app. Sekarang jadi
// landing page marketing beneran: hero + 4 fitur (reuse lucide yang sudah
// dipakai: Camera, Plus sudah ada di Navbar/Dashboard; sisanya ikon lucide
// standar yang namanya confirmed ada di node_modules 0.469) + preview
// contoh halaman profil + CTA.
//
// DESAIN (DESIGN.md): landing = tema "classic" SELALU (netral, publik
// brand, BUKAN ikut tema kreator yang login). Ini halaman statis murni
// (TIDAK ada fetch): contoh link pakai data dummy supaya tampil sama di
// setiap render (no hydration mismatch).
//
// Format tampilan profil publik yang dicontohkan di sini MIRIP halaman
// /u/[username]: avatar lingkaran + @username + daftar card link dengan
// border print-tebal + aksen flash-yellow. Tapi karena ini landing page
// publik, isinya dummy statik (bukan data kreator beneran).
// =====================================================================
export default function Home() {
  const st = themeStyles("classic");

  const exampleLinks = [
    { short_code: "youtube", original_url: "https://youtube.com/@kreator", click_count: 284 },
    { short_code: "shopee", original_url: "https://shopee.co.id/kreator", click_count: 156 },
    { short_code: "ig", original_url: "https://instagram.com/kreator", click_count: 98 },
    { short_code: "tiktok", original_url: "https://tiktok.com/@kreator", click_count: 210 },
  ];

  const features = [
    {
      icon: Link2,
      title: "Custom Slug",
      desc: "Tentukan sendiri kode pendek biar brand-nya kelihatan, bukan random string.",
    },
    {
      icon: MonitorSmartphone,
      title: "Smart Link",
      desc: "Satu link, tampilan berbeda per device: arahkan visitor HP/desktop ke tujuan yang pas.",
    },
    {
      icon: QrCode,
      title: "QR Code",
      desc: "Cetak link kamu jadi QR yang siap dipindai di poster, kemasan, atau kartu nama.",
    },
    {
      icon: ChartColumnIncreasing,
      title: "Analytics",
      desc: "Pantau click, unique visitor, dan performa tiap link - semua di satu panggung.",
    },
  ];

  return (
    <main className={`min-h-screen ${st.page}`}>
      <ThemeBackdrop theme="classic" />

      {/* ===== HERO ===== */}
      <section className="relative mx-auto flex max-w-3xl flex-col items-center px-6 pb-16 pt-20 text-center">
        <motion.p
          initial={{ opacity: 0, y: 8 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.25, ease: "easeOut" }}
          className={`inline-block ${st.chip} ${st.radiusFull} ${st.borderW} ${st.border} px-3 py-1 font-mono text-xs font-bold uppercase tracking-[0.15em]`}
        >
          LINK-IN-BIO UNTUK KREATOR
        </motion.p>
        <motion.h1
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, ease: "easeOut", delay: 0.06 }}
          className={`mt-6 ${st.headingFont} text-4xl font-bold leading-tight sm:text-5xl`}
        >
          Satu halaman,
          <br />
          semua link kamu.
        </motion.h1>
        <motion.p
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, ease: "easeOut", delay: 0.12 }}
          className={`mt-4 max-w-xl text-base ${st.textMuted}`}
        >
          Persingkat semua link sosial-mu jadi satu halaman yang bisa dibagikan -
          dengan smart redirect per device dan analytics lengkap di baliknya.
        </motion.p>
        <motion.div
          initial={{ opacity: 0, y: 8 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.25, ease: "easeOut", delay: 0.18 }}
          className="mt-8"
        >
          <Link
            href="/app"
            className={`inline-flex items-center gap-2 ${st.accent} ${st.radiusFull} ${st.borderW} ${st.border} px-7 py-3 text-sm font-bold`}
          >
            Mulai Sekarang
            <ArrowRight className="h-4 w-4" strokeWidth={2.5} />
          </Link>
        </motion.div>
      </section>

      {/* ===== CONTOH: preview halaman profil publik ===== */}
      <section className={`border-t-2 ${st.border}`}>
        <div className="mx-auto max-w-3xl px-6 py-14">
          <h2 className={`${st.heading} ${st.headingFont} ${st.radiusFull} ${st.chip} inline-block ${st.borderW} ${st.border} px-4 py-1.5 font-mono text-xs font-bold uppercase tracking-[0.15em]`}>
            Contoh halaman profil kamu
          </h2>
          <div className={`mt-6 ${st.radiusLarge} ${st.borderW} ${st.border} ${st.card} p-6`}>
            {/* header profil */}
            <div className="flex items-center gap-4">
              <div className={`relative flex h-16 w-16 items-center justify-center ${st.avatar} ${st.radiusFull} ${st.borderW} ${st.border} text-2xl font-bold`}>
                <Camera className={`absolute -bottom-1 -right-1 h-5 w-5 ${st.radiusFull} bg-print-white p-0.5`} strokeWidth={2.5} />
                K
              </div>
              <div>
                <p className={`text-xs ${st.textMuted}`}>@kreator</p>
                <p className={`${st.headingFont} text-xl font-bold`}>Kreator Digital</p>
              </div>
            </div>

            {/* daftar contoh link */}
            <div className="mt-5 flex flex-col gap-3">
              {exampleLinks.map((l) => (
                <div
                  key={l.short_code}
                  className={`flex items-center justify-between gap-3 ${st.card} ${st.borderW} ${st.border} border-2 px-4 py-3`}
                >
                  <div>
                    <p className={`font-mono text-sm font-bold`}>/{l.short_code}</p>
                    <p className={`truncate text-xs ${st.textMuted}`}>{l.original_url}</p>
                  </div>
                  <span className="font-mono text-sm font-bold">{l.click_count}</span>
                </div>
              ))}
            </div>
          </div>
          <p className={`mt-4 text-center text-xs ${st.textMuted}`}>
            Contoh tampilan profil publik kreator.
          </p>
        </div>
      </section>

      {/* ===== FITUR ===== */}
      <section className={`border-t-2 ${st.border}`}>
        <div className="mx-auto max-w-3xl px-6 py-14">
          <h2 className={`${st.headingFont} text-2xl font-bold ${st.heading}`}>
            Kenapa Jejak?
          </h2>
          <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-2">
            {features.map((f) => (
              <div
                key={f.title}
                className={`${st.card} ${st.borderW} ${st.border} ${st.radius} p-5`}
              >
                <div className={`flex h-10 w-10 items-center justify-center ${st.accent} ${st.radius}`}>
                  <f.icon className="h-5 w-5" strokeWidth={2.5} />
                </div>
                <h3 className={`mt-3 ${st.headingFont} text-base font-bold`}>{f.title}</h3>
                <p className={`mt-1 text-sm ${st.textMuted}`}>{f.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* ===== CTA PENUTUP ===== */}
      <section className={`border-t-2 ${st.border}`}>
        <div className="mx-auto flex max-w-3xl flex-col items-center px-6 py-16 text-center">
          <h2 className={`${st.headingFont} text-2xl font-bold ${st.heading}`}>
            Tertarik? Daftar gratis sekarang.
          </h2>
          <p className={`mt-2 max-w-xl text-base ${st.textMuted}`}>
            Ngga perlu card, ngga perlu ribet: langsung persingkat link kamu, bikin
            halaman link-in-bio, dan lihat analytics-nya.
          </p>
          <Link
            href="/app"
            className={`mt-8 inline-flex items-center gap-2 ${st.accent} ${st.radiusFull} ${st.borderW} ${st.border} px-7 py-3 text-sm font-bold`}
          >
            Daftar Gratis
            <Plus className="h-4 w-4" strokeWidth={2.5} />
          </Link>
        </div>
      </section>
    </main>
  );
}
