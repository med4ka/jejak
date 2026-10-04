"use client";

import { useEffect } from "react";
import { motion } from "framer-motion";
import { Inbox } from "lucide-react";
import ThemeBackdrop from "./ThemeBackdrop";
import { themeStyles } from "../../lib/themes";
import { useTranslation } from "../../lib/I18nProvider";
import { detectBrand, isBrightColor, isDarkColor, rgba, ICON_MAP } from "../../lib/brands";

// DESIGN.md §6: "photo leaving the camera" entrance: opacity + a small
// translateY (8-12px), 50-80ms stagger per card, firm ease-out, ≤400ms total.
// NOT a spring overshoot, NOT a slide from the side or a dramatic zoom.
// Tap: scale-down to 0.97-0.98 (no bounce).
// The subtle ±1-2° rotation is computed deterministically from the index (the
// server sends `tilt` ready to use: Math.random during render would cause a
// hydration mismatch).
//
// Component-level comment: this component is the only one mounted on
// /u/[username] (and not on other pages), so the <body> theme effect (the
// layout applies a global bg-print-white) is installed here: the dataset
// carries whatever theme is active (classic/darkroom/coral/glass) and the CSS
// rule in globals.css decides the visuals (classic = print-white, coral =
// cream, darkroom = #121212, glass = 135° gradient). Dashboard and /app never
// set this attribute: decision of 2026-09-29: themes apply to public pages
// plus the navbar there. Cleanup runs on unmount: otherwise, visiting another
// page after a dark-themed /u/ would leave that page dark as well.
export default function ProfileLinks({ links, top, theme = "classic" }) {
  // NOTE: `t` is reserved for the translation function: the theme token
  // object from themeStyles is therefore named `ts`.
  const { t } = useTranslation();
  const ts = themeStyles(theme);

  useEffect(() => {
    document.body.dataset.profileTheme = theme;
    return () => {
      delete document.body.dataset.profileTheme;
    };
  }, [theme]);

  if (links.length === 0) {
    // Empty state (not a bare <p>): icon card plus two lines of text, styled
    // with the active theme tokens so it stays consistent across Classic/
    // Darkroom/Coral/Glass. Public page → no CTA (this is not the owner's
    // dashboard). ThemeBackdrop is rendered as well so the page texture stays
    // identical even before any link exists.
    return (
      <>
        <ThemeBackdrop />
        <div
          className={`flex flex-col items-center justify-center gap-0.5 px-6 py-12 text-center ${ts.radius} ${ts.borderW} ${ts.border} ${ts.card} ${ts.shadow}`}
        >
          <Inbox className={`mb-3 h-12 w-12 ${ts.text}`} strokeWidth={1.5} aria-hidden="true" />
          <p className={`font-display text-lg font-bold ${ts.text}`}>
            {t("forms.profileLinks.emptyTitle")}
          </p>
          <p className={`mt-1 text-sm ${ts.textMuted}`}>
            {t("forms.profileLinks.emptyDescription")}
          </p>
        </div>
      </>
    );
  }

  return (
    <>
      {/* Subtle grain behind the content (global texture, all themes). */}
      <ThemeBackdrop />
      {links.map((l, i) => {
        // AUTO-DETECT BRAND (2026-10-04): domain URL → warna + ikon lucide
        // (lib/brands.js: 60 brand). Box style memakai INLINE STYLE karena
        // warnanya datang dari data brand, bukan token tema. Domain tak
        // dikenal jatuh ke token ts.accent supaya ikut warna 11 tema.
        const brand = detectBrand(l.original_url);
        const IconComp = ICON_MAP[brand.icon] || ICON_MAP.link;
        let boxStyle;
        if (!brand.matched) {
          // Fallback: token ts.accent (bg solid konsisten dgn badge
          // Unggulan, kontras by design lintas tema), border transparan.
          boxStyle = { borderColor: "transparent" };
        } else if (isBrightColor(brand.color)) {
          // Warna terang (mis. Snapchat #FFFC00): tint 20% + ikon terang
          // tidak kontras di kartu putih → bg solid + ikon gelap.
          boxStyle = { backgroundColor: brand.color, borderColor: brand.color, color: "#1C1A12" };
        } else if (isDarkColor(brand.color)) {
          // Warna gelap (mis. X/Medium #000000): tint 20% + ikon hitam
          // hilang di tema gelap (darkroom) → bg solid + ikon putih +
          // rim putih 25% supaya box terpisah dari kartu gelap.
          boxStyle = {
            backgroundColor: brand.color,
            borderColor: "rgba(255,255,255,0.25)",
            color: "#FFFFFF",
          };
        } else {
          // Normal: tint 20% + border 55% + ikon berwarna brand.
          boxStyle = {
            backgroundColor: rgba(brand.color, 0.2),
            borderColor: rgba(brand.color, 0.55),
            color: brand.color,
          };
        }
        return (
        // Tilt wrapper (pure CSS, globals.css .tilt-wrap): the angle is held by
        // the --tilt custom property so it can be reduced automatically on
        // narrow screens (±2° on desktop → ±0.6° at ≤640px). framer-motion
        // inside still owns entrance/hover/tap: the two transforms are
        // independent.
        <div key={l.short_code} className="tilt-wrap" style={{ "--tilt": `${l.tilt}deg` }}>
          <motion.a
            // RELATIVE PATH (public-page redirect fix, 2026-09-30):
            // `${apiBase}/r/...` used to point straight at the backend:
            // localhost:8081 appeared in the production URL bar and was not
            // reachable from other devices. The link now goes through the page
            // origin → Next proxies to the backend (app/r/[code]/route.js), so
            // the host is displayed correctly and analytics are still recorded
            // with the visitor's IP/UA.
            href={`/r/${l.short_code}`}
            target="_blank"
            rel="noreferrer"
            initial={{ opacity: 0, y: 10 }}
            animate={{
              opacity: 1,
              y: 0,
            }}
            whileHover={{ y: -2 }}
            whileTap={{ scale: 0.98 }}
            transition={{ duration: 0.25, delay: i * 0.06, ease: "easeOut" }}
            className={`block ${l.is_featured ? ts.borderStrong : ts.borderW} ${ts.radius} ${ts.border} ${ts.card} ${ts.shadow} ${
              l.is_featured ? ts.featuredClass : ""
            } ${
              // Alternating 4px accent bar on the left (risoPrint only: the
              // other presets have no barPrimary, so this yields an empty
              // string and the result is identical).
              ts.barPrimary ? (i % 2 === 0 ? ts.barPrimary : ts.barSecondary) : ""
            }`}
          >
            {/* BRAND ICON BOX (32px mobile / 40px desktop, radius 8px): auto-
                detected dari domain link. bg/border/warna ikon dari data
                brand (inline style di atas); fallback = ts.accent. Motion
                whileHover scale 1.05 (spring) — hover PADA box, kartu tetap
                y:-2 lewat motion.a induk. aria-hidden: dekoratif, URL sudah
                terbaca sebagai teks. */}
            <div className="flex items-center gap-3 px-3 sm:px-4">
              <motion.div
                aria-hidden="true"
                whileHover={{ scale: 1.05 }}
                transition={{ type: "spring", stiffness: 400, damping: 17 }}
                className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-[8px] border md:h-10 md:w-10 ${brand.matched ? "" : ts.accent}`}
                style={boxStyle}
              >
                <IconComp className="h-4 w-4 md:h-5 md:w-5" strokeWidth={2} />
              </motion.div>
              <div className="min-w-0 flex-1">
                {/* px-3 on mobile, px-4 at ≥640px plus a tighter gap (mobile fix):
                    URL text and badges get breathing room at 320-375px without
                    changing the desktop layout. Padding now lives on the OUTER
                    row above; this column only splits URL row / slug row. */}
                <div className="flex items-center justify-between gap-2 pt-4 sm:gap-3">
                  <p className={`min-w-0 truncate text-sm font-medium ${ts.url || ""}`}>{l.original_url}</p>
                  {/* shrink-0 + whitespace-nowrap: the TRENDING badge and the count
                      are never truncated or wrapped onto a second line on narrow
                      screens. */}
                  <div className="flex shrink-0 items-center gap-2 whitespace-nowrap">
                    {l.is_featured && (
                      <span className={`${ts.radiusFull} ${ts.accent} px-2.5 py-0.5 text-[11px] font-bold uppercase tracking-wide`}>
                        {t("forms.profileLinks.featuredBadge")}
                      </span>
                    )}
                    {top === l.short_code && (
                        <span className={`${ts.radiusFull} ${ts.badge} px-2.5 py-0.5 text-[11px] font-bold uppercase tracking-wide`}>
                          {t("forms.profileLinks.trendingBadge")}
                        </span>
                      )}
                    <span className={`font-mono text-sm font-bold ${ts.countColor || ""}`}>{l.click_count}</span>
                  </div>
                </div>
                {/* Polaroid bottom margin: Caveat caption (§3, first slot in the
                    hierarchy). risoPrint ships its own ts.slug (JetBrains Mono; the
                    specification forbids handwritten fonts for that theme), so the
                    caption fallback is skipped. */}
                <p className={`pb-3 pt-1 ${ts.slug ? ts.slug : `font-caption text-lg leading-none ${ts.caption}`}`}>
                  {t("forms.profileLinks.shortCode", { shortCode: l.short_code })}
                </p>
              </div>
            </div>
          </motion.a>
        </div>
        );
      })}
    </>
  );
}
