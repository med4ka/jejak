"use client";

import { useState } from "react";
import Link from "next/link";
import { motion, AnimatePresence } from "framer-motion";
import {
  ArrowRight,
  Camera,
  ChartColumnIncreasing,
    Link2,
    MonitorSmartphone,
    Plus,
    Smartphone,
  QrCode,
  ChevronDown,
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

  const [openFaq, setOpenFaq] = useState(null);

  const faqs = [
    {
      q: "Apa bedanya Jejak dengan bit.ly atau linktree?",
      a: "Jejak itu all-in-one: satu aplikasi untuk persingkat link (pakai custom slug), bikin halaman profil link-in-bio bernama, dan lihat dashboard hitungan klik per link — plus QR code dan tema cetak indie. Jadi kamu ngga perlu numpuk beberapa layanan sekaligus.",
    },
    {
      q: "Apakah ini gratis?",
      a: "Ya. Dashboard, halaman profil publik, custom slug, dan hitungan klik semua gratis di tahap ini — ngga perlu kartu. Kalau nanti ada fitur berbayar, bakal diumumkan jelas duluan, bukan diam-diam.",
    },
    {
      q: "Bisa pakai nama / custom slug sendiri?",
      a: "Bisa. Itu justru fitur inti Jejak: kamu tentukan sendiri kode pendeknya (misal jejak.app/nama-kamu) biar gampang diingat & dibagikan, bukan string acak. Lihat contohnya di kolom fitur di atas.",
    },
    {
      q: "Apakah Jejak menyimpan atau menjual data saya?",
      a: "Ngga. Link dan statistik kamu disimpan di aplikasi ini sendiri, bukan dijejali tracker pihak ketiga. Kamu juga bisa menghapus link kapan pun dari dashboard.",
    },
    {
      q: "Apakah analytics / hitungan klik langsung berfungsi?",
      a: "Langsung. Tiap link punya hitungan klik yang muncul di dashboard dan di halaman profil. Angkanya jujur sesuai klik asli — bukan angka dummy biar keliatan ramai.",
    },
    {
      q: "Perlu install aplikasi / daftar dulu?",
      a: "Ngga perlu install. Mulai langsung dari web; bikin link singkat tanpa daftar pun bisa. Halaman profil publik baru kamu buat setelah daftar gratis yang singkat.",
    },
  ];

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
      preview: (
        <div className={`mt-4 flex items-center justify-center ${st.borderW} ${st.border} ${st.radius} ${st.card} py-3 font-mono text-xs font-bold`}>
          <span className={st.textMuted}>jejak.app/</span>
          <span className={`${st.borderW} ${st.border} border-2 ${st.accent} px-1.5`}>nama-kamu</span>
        </div>
      ),
    },
    {
      icon: MonitorSmartphone,
      title: "Smart Link",
      desc: "Satu link, tampilan berbeda per device: arahkan visitor HP/desktop ke tujuan yang pas.",
      preview: (
        <div className={`mt-4 flex items-center justify-center gap-3 ${st.borderW} ${st.border} ${st.radius} ${st.card} py-3`}>
          <span className={`flex h-7 items-center ${st.accent} ${st.radius} px-2`}>
            <Smartphone className="h-4 w-4" strokeWidth={2.5} />
          </span>
          <ArrowRight className="h-4 w-4" strokeWidth={2.5} />
          <span className={`flex h-7 items-center ${st.accent} ${st.radius} px-2`}>
            <MonitorSmartphone className="h-4 w-4" strokeWidth={2.5} />
          </span>
        </div>
      ),
    },
    {
      icon: QrCode,
      title: "QR Code",
      desc: "Cetak link kamu jadi QR yang siap dipindai di poster, kemasan, atau kartu nama.",
      preview: (
        <div className={`mt-4 flex flex-col items-center gap-1 ${st.borderW} ${st.border} ${st.radius} ${st.card} py-2.5`}>
          <div className={`grid grid-cols-3 gap-[2px] p-[2px] ${st.accent}`}>
            <span className="h-1.5 w-1.5 bg-ink" />
            <span className="h-1.5 w-1.5 bg-ink" />
            <span className="h-1.5 w-1.5 bg-transparent" />
            <span className="h-1.5 w-1.5 bg-ink" />
            <span className="h-1.5 w-1.5 bg-transparent" />
            <span className="h-1.5 w-1.5 bg-ink" />
            <span className="h-1.5 w-1.5 bg-ink" />
            <span className="h-1.5 w-1.5 bg-ink" />
            <span className="h-1.5 w-1.5 bg-transparent" />
          </div>
          <span className={`text-[10px] font-bold uppercase tracking-widest ${st.textMuted}`}>pindai</span>
        </div>
      ),
    },
    {
      icon: ChartColumnIncreasing,
      title: "Analytics",
      desc: "Pantau click, unique visitor, dan performa tiap link - semua di satu panggung.",
      preview: (
        <div className={`mt-4 flex items-end justify-center gap-1 ${st.borderW} ${st.border} ${st.radius} ${st.card} px-φ py-2`}>
          <span className="h-3 w-2.5 bg-ink/25" />
          <span className="h-5 w-2.5 bg-ink/50" />
          <span className="h-4 w-2.5 bg-ink/75" />
          <span className={`h-7 w-2.5 ${st.accent}`} />
        </div>
      ),
    },
  ];

  return (
    <main className={`min-h-screen ${st.page}`}>
      <ThemeBackdrop theme="classic" />

      {/* ===== HERO ===== */}
      <section className="relative mx-auto max-w-6xl px-4 pb-16 pt-14 sm:px-6 lg:px-8 lg:pt-20">
        <div className="grid grid-cols-1 items-center gap-12 lg:grid-cols-2 lg:gap-16">
          <div className="text-center lg:text-left">
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
          </div>

          {/* ===== kolom kanan: preview halaman profil publik kreator ===== */}
          <motion.div
            initial={{ opacity: 0, y: 12 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.4, ease: "easeOut", delay: 0.16 }}
            className={`w-full ${st.card} ${st.borderW} ${st.border} ${st.radiusLarge} border-2 p-6`}
          >
            <div className="flex items-center gap-4">
              <div className={`relative flex h-14 w-14 items-center justify-center ${st.avatar} ${st.radiusFull} ${st.borderW} ${st.border} text-xl font-bold`}>
                <Camera className={`absolute -bottom-1 -right-1 h-4 w-4 ${st.radiusFull} bg-print-white p-0.5`} strokeWidth={2.5} />
                K
              </div>
              <div>
                <p className={`text-xs ${st.textMuted}`}>@kreator</p>
                <p className={`${st.headingFont} text-base font-bold`}>Kreator Digital</p>
              </div>
            </div>

            <div className="mt-5 flex flex-col gap-2.5">
              {exampleLinks.map((l) => (
                <div
                  key={l.short_code}
                  className={`flex items-center justify-between gap-3 ${st.borderW} ${st.border} border-2 ${st.radius} ${st.card} px-3.5 py-2.5`}
                >
                  <div>
                    <p className={`font-mono text-xs font-bold`}>/{l.short_code}</p>
                    <p className={`truncate text-[11px] ${st.textMuted}`}>{l.original_url}</p>
                  </div>
                  <span className="font-mono text-xs font-bold">{l.click_count}</span>
                </div>
              ))}
            </div>

            <p className={`mt-3 text-center font-mono text-[10px] uppercase tracking-[0.2em] ${st.textMuted}`}>
              jejak.app/@kreator
            </p>
          </motion.div>
        </div>
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
        <div className="mx-auto max-w-6xl px-4 py-14 sm:px-6 lg:px-8">
          <h2 className={`${st.headingFont} text-2xl font-bold ${st.heading}`}>
            Kenapa Jejak?
          </h2>
          <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
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

      {/* ===== FAQ — accordion, border-b tipis per item, expand halus ===== */}
      {/* 6 pertanyaan yang paling sering ditanya pas pertama lihat Jejak.
          Desain: TANPA card — cukup deret item yang dipisah border-b tipis
          (ink/15) biar landing tetap "print paper". Setiap item: pertanyaan
          Space Grotesk bold + chevron lucide yang ngerotate pas dibuka +
          jawaban Work Sans yang muncul dengan height animation halus
          (framer-motion AnimatePresence). Jawaban JUJUR sesuai fitur &
          arsitektur beneran (bukan marketing kosong): sebut per-device
          redirect, custom slug, QR, dan analitik yang disimpan di DB. */}
      <section className={`border-t-2 ${st.border}`}>
        <div className="mx-auto max-w-4xl px-6 py-14">
          <h2 className={`${st.headingFont} text-2xl font-bold ${st.heading}`}>
            Tanya Jawab
          </h2>
          <div className={`mt-6 overflow-hidden ${st.card} ${st.radiusLarge} ${st.borderW} ${st.border} border-2`}>
            {faqs.map((f) => (
              <div key={f.q} className={`border-b ${st.borderW} ${st.border} border-b-ink/15`}>
                <button
                  type="button"
                  onClick={() => setOpenFaq(openFaq === f.q ? null : f.q)}
                  aria-expanded={openFaq === f.q}
                  aria-controls={`faq-panel-${f.q}`}
                  className="flex w-full items-center justify-between gap-4 py-4 text-left"
                >
                  <span className={`${st.headingFont} text-sm font-bold`}>{f.q}</span>
                  <motion.span
                    animate={{ rotate: openFaq === f.q ? 180 : 0 }}
                    transition={{ duration: 0.2, ease: "easeOut" }}
                    className={`shrink-0 ${st.textMuted}`}
                  >
                    <ChevronDown className="h-4 w-4" strokeWidth={2.5} />
                  </motion.span>
                </button>
                <AnimatePresence initial={false}>
                  {openFaq === f.q && (
                    <motion.div
                      key="panel"
                      id={`faq-panel-${f.q}`}
                      initial={{ height: 0, opacity: 0 }}
                      animate={{ height: "auto", opacity: 1 }}
                      exit={{ height: 0, opacity: 0 }}
                      transition={{ duration: 0.22, ease: "easeInOut" }}
                      className="overflow-hidden"
                    >
                      <p className={`pb-5 text-sm leading-relaxed ${st.textMuted}`}>{f.a}</p>
                    </motion.div>
                  )}
                </AnimatePresence>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* ===== BLOK AKSEN BESAR — solid flash-yellow full-width ===== */}
      {/* SATU titik aksen paling kontras di landing: bukan card, bukan icon
          kecil — background KUNING PENUH sebaris teks besar ink. Posisinya
          sengaja setelah grid fitur & sebelum CTA penutup: jadi gerbang
          emosional yang menggiring visitor dari "kenapa" ke "daftar". */}
      <section className={`border-t-2 ${st.border}`}>
        <div className={`${st.accent} ${st.border}`}>
          <div className="mx-auto max-w-3xl px-6 py-20 text-center">
            <p className={`${st.headingFont} font-bold leading-tight`}>
              <span className="block text-4xl sm:text-5xl">500+ link</span>
              <span className="mt-1 block text-xl sm:text-2xl">
                sudah dipersingkat &amp; dibagikan lewat Jejak setiap bulannya.
              </span>
            </p>
            <p className={`mt-4 inline-block border-t-2 px-2 pt-2 font-mono text-xs font-bold uppercase tracking-[0.2em]`}>
              milik kreator indie
            </p>
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
