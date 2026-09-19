"use client";

import { useEffect } from "react";
import { motion } from "framer-motion";
import ThemeBackdrop from "./ThemeBackdrop";
import { themeStyles } from "../../lib/themes";

// DESIGN.md §6 — entrance "foto keluar dari kamera": opacity + translateY
// kecil (8-12px), stagger 50-80ms per card, ease-out tegas, ≤400ms.
// BUKAN spring overshoot, BUKAN slide dari samping / zoom dramatis.
// Tap: scale-down 0.97-0.98 (bukan bounce).
// Rotasi acak halus ±1-2° dihitung deterministik dari index (server kirim
// `tilt` siap pakai — Math.random di render bikin hydration mismatch).
//
// Command comment: komponen ini satu-satunya yang BERDIRI di /u/[username]
// (bukan di halaman lain), jadi efek tema untuk <body> (layout punya
// bg-print-white global) dipasang di sini: dataset berisi theme APA PUN
// (classic/darkroom/coral/glass), rule CSS di globals.css memutuskan
// visualnya. Classic/coral tidak punya rule → body tetap print-white.
// Cleanup pas unmount — kalau tidak, mengunjungi halaman lain setelah /u/
// bertema gelap bakal tetap gelap.
export default function ProfileLinks({ links, top, apiBase, theme = "classic" }) {
  const t = themeStyles(theme);

  useEffect(() => {
    document.body.dataset.profileTheme = theme;
    return () => {
      delete document.body.dataset.profileTheme;
    };
  }, [theme]);

  if (links.length === 0) {
    return <p className={`text-sm ${t.caption}`}>Kreator ini belum punya link.</p>;
  }

  return (
    <>
      {/* Blob glass di belakang konten — tema lain null. */}
      <ThemeBackdrop theme={theme} />
      {links.map((l, i) => (
        <motion.a
          key={l.short_code}
          href={`${apiBase}/r/${l.short_code}`}
          target="_blank"
          rel="noreferrer"
          initial={{ opacity: 0, y: 10 }}
          animate={{
            opacity: 1,
            y: 0,
            // Rotasi deterministik dari server (tilt, ±1-2° dari index) —
            // Math.random di render bikin hydration mismatch.
            rotate: l.tilt,
          }}
          whileHover={{ y: -2 }}
          whileTap={{ scale: 0.98 }}
          transition={{ duration: 0.25, delay: i * 0.06, ease: "easeOut" }}
          className={`block ${l.is_featured ? t.borderStrong : t.borderW} ${t.radius} ${t.border} ${t.card} ${
            l.is_featured ? t.featuredClass : ""
          }`}
        >
          <div className="flex items-center justify-between gap-3 px-4 pt-4">
            <p className="min-w-0 truncate text-sm font-medium">{l.original_url}</p>
            <div className="flex shrink-0 items-center gap-2">
              {l.is_featured && (
                <span className={`${t.radiusFull} ${t.accent} px-2.5 py-0.5 text-[11px] font-bold uppercase tracking-wide`}>
                  ★ Unggulan
                </span>
              )}
              {top === l.short_code && (
                  <span className={`${t.radiusFull} ${t.badge} px-2.5 py-0.5 text-[11px] font-bold uppercase tracking-wide`}>
                  Trending
                </span>
              )}
              <span className="font-mono text-sm font-bold">{l.click_count}</span>
            </div>
          </div>
          {/* Margin bawah Polaroid: caption Caveat (§3, lokasi ke-1). */}
          <p className={`px-4 pb-3 pt-1 font-caption text-lg leading-none ${t.caption}`}>
            /{l.short_code}
          </p>
        </motion.a>
      ))}
    </>
  );
}
