"use client";

import { useEffect, useState } from "react";
import { motion } from "framer-motion";
import { Check, Copy, X } from "lucide-react";
import useCopyToClipboard from "../../lib/useCopyToClipboard";
import { SPRING } from "../../lib/animations";
import { useTranslation } from "../../lib/I18nProvider";

// Reusable copy button: click → clipboard, with INLINE feedback in the button
// for 2000ms (Apple-style detail #3):
// - success : icon Copy → Check, bg flash-yellow (#FFD23F), label
//            "Salin" → "Tersalin", gentle bounce scale 1 → 1.05 → 1
//            (SPRING of ~200ms per leg: the `bump` state is released after
//            200ms, so the return leg is also spring-driven; framer cannot mix
//            keyframes and a spring, which is why this two-state sequence is
//            how the spring is expressed)
// - failure : icon Copy → X, bg flash-coral (#FF5C3D), label "Gagal"
// A textarea fallback covers non-secure contexts where navigator.clipboard is
// unavailable (see lib/useCopyToClipboard.js).
//
// State colors are applied through an INLINE STYLE rather than by swapping the
// className string:
// - an inline style always overrides bg-* classes (Tailwind's CSS order is not
//   guaranteed between the caller's bg and the state bg),
// - the caller's size/radius/border survive the "Tersalin" state
//   (old bug: the feedback string replaced the full className, and the button
//   suddenly shrank to py-0.5 in the middle of the layout).
const DEFAULT_CLASS =
  "shrink-0 rounded-full border-2 border-ink bg-print-white px-3 py-0.5 text-xs font-bold text-ink transition-colors duration-150 hover:bg-paper-grey/60";

// Design tokens (do NOT change): #FFD23F = flash-yellow, #FF5C3D = flash-coral.
const STATE_BG = {
  success: "#FFD23F",
  failed: "#FF5C3D",
};

// Relative paths ("/r/…") are converted to absolute BEFORE copying: the
// clipboard cannot hold a relative path, and callers now pass same-origin
// paths (public-page redirect fix, 2026-09-30: callers previously sent the
// backend URL localhost:8081). Only strings starting with "/" are touched;
// non-URL text (API keys) and absolute URLs (https://jejak.app/u/…) are left
// as they are. The conversion happens on click (not during render) so that no
// window value is read in the SSR render.
function toCopyValue(value) {
  if (typeof value === "string" && value.startsWith("/") && typeof window !== "undefined") {
    return window.location.origin + value;
  }
  return value;
}

export default function CopyButton({ text, label, className = "" }) {
  const { t } = useTranslation();
  const { copied, copy } = useCopyToClipboard();
  const [bump, setBump] = useState(false);
  // Default label resolved here (not as a parameter default) so the
  // translation hook can be used; callers passing label keep their string.
  const restLabel = label ?? t("common.copy");

  // Bounce: on success the button scales up to 1.05 and returns to 1 after
  // 200ms: both legs are SPRING-driven (the way up reads as a "pop", the way
  // down as a soft landing). The "Tersalin" label stays for 2 seconds (from
  // the hook) and is not tied to this 200ms window.
  useEffect(() => {
    if (copied !== "success") {
      setBump(false);
      return undefined;
    }
    setBump(true);
    const t = setTimeout(() => setBump(false), 200);
    return () => clearTimeout(t);
  }, [copied]);

  const base = className !== "" ? className : DEFAULT_CLASS;
  const stateStyle =
    copied === "success"
      ? { backgroundColor: STATE_BG.success }
      : copied === "failed"
        ? { backgroundColor: STATE_BG.failed }
        : undefined;
  const Icon = copied === "success" ? Check : copied === "failed" ? X : Copy;

  return (
    <motion.button
      type="button"
      onClick={() => copy(toCopyValue(text))}
      animate={{ scale: bump ? 1.05 : 1 }}
      transition={SPRING}
      className={base}
      style={stateStyle}
      aria-live="polite"
    >
      <Icon className="mr-1 inline h-3.5 w-3.5" strokeWidth={3} aria-hidden="true" />
      {copied === "success"
        ? t("common.copied")
        : copied === "failed"
          ? t("common.failed")
          : restLabel}
    </motion.button>
  );
}
