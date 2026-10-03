"use client";

import { useEffect, useRef, useState } from "react";
import { motion } from "framer-motion";
import { QRCodeCanvas } from "qrcode.react";
import { Check, Download, ExternalLink, Share2 } from "lucide-react";
import useCopyToClipboard from "../../lib/useCopyToClipboard";
import { SPRING } from "../../lib/animations";
import { useTranslation } from "../../lib/I18nProvider";

// "Bagikan Halaman" pill button: trigger for ShareModal. Used in two places
// (Summary tab next to the greeting, Profile tab next to "Halaman
// publikmu"). Hover: lifts 2px while the shadow grows; tap: scale 0.97 with y
// returning to 0: both SPRING-driven (Apple-style detail #4: primary buttons
// use SPRING instead of a duration tween). Colors are hard-coded to Instant
// Print (not st.*): the button specification is border-2 ink + flash-yellow +
// hard shadow on every theme.
export function ShareButton({ onClick }) {
  const { t } = useTranslation();
  return (
    <motion.button
      type="button"
      onClick={onClick}
      whileHover={{ y: -2, transition: SPRING }}
      whileTap={{ scale: 0.97, y: 0, transition: SPRING }}
      className="inline-flex shrink-0 items-center gap-2 rounded-full border-2 border-ink bg-flash-yellow px-5 py-2.5 text-sm font-bold text-ink shadow-[4px_4px_0px_#1C1A12] transition-shadow duration-150 hover:shadow-[7px_7px_0px_#1C1A12] focus:outline-none focus-visible:ring-2 focus-visible:ring-flash-coral"
    >
      <Share2 size={16} aria-hidden="true" />
      {t("forms.share.title")}
    </motion.button>
  );
}

// Share modal for the public page /u/{username}: (1) copy the link (the
// useCopyToClipboard hook: "Tersalin" feedback for 2 seconds), (2) 200x200 QR
// in ink on print-white plus PNG download, (3) open Lihat Profil in a new tab.
// The shell follows the other modals (blur backdrop + Escape + backdrop
// click); the panel fades and scales with SPRING (a modal is UI state →
// spring, Apple-style detail #1).
export default function ShareModal({ username, onClose }) {
  const { t } = useTranslation();
  const [url, setUrl] = useState("");
  const { copied, copy } = useCopyToClipboard();
  const panelRef = useRef(null);
  const canvasRef = useRef(null);

  // The URL is built on the client (window.location.origin) so that dev and
  // prod behave identically: kept in state rather than read during render
  // because this component only mounts on the client.
  useEffect(() => {
    setUrl(`${window.location.origin}/u/${username}`);
  }, [username]);

  // Escape close + FOCUS TRAP: focus moves into the panel on open, Tab and
  // Shift+Tab cycle within the panel only, and focus returns to the trigger
  // when the modal closes.
  useEffect(() => {
    const prevFocus = document.activeElement;
    const panel = panelRef.current;
    const selector = 'button:not([disabled]), a[href], input:not([disabled]), [tabindex]:not([tabindex="-1"])';
    const first = panel?.querySelector(selector);
    first?.focus();

    function onKey(e) {
      if (e.key === "Escape") {
        onClose();
        return;
      }
      if (e.key !== "Tab" || !panel) {
        return;
      }
      const els = Array.from(panel.querySelectorAll(selector)).filter(
        (el) => el.offsetParent !== null
      );
      if (els.length === 0) {
        return;
      }
      const firstEl = els[0];
      const lastEl = els[els.length - 1];
      if (e.shiftKey && document.activeElement === firstEl) {
        e.preventDefault();
        lastEl.focus();
      } else if (!e.shiftKey && document.activeElement === lastEl) {
        e.preventDefault();
        firstEl.focus();
      }
    }
    window.addEventListener("keydown", onKey);
    return () => {
      window.removeEventListener("keydown", onKey);
      if (prevFocus && typeof prevFocus.focus === "function") {
        prevFocus.focus();
      }
    };
  }, [onClose]);

  function downloadQR() {
    const canvas = canvasRef.current;
    if (!canvas) {
      return;
    }
    const a = document.createElement("a");
    a.href = canvas.toDataURL("image/png");
    a.download = `jejak-u-${username}.png`;
    a.click();
  }

  return (
    <div className="pointer-events-none fixed inset-0 z-[60] flex items-center justify-center p-4">
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        transition={SPRING}
        onClick={onClose}
        className="pointer-events-auto absolute inset-0 bg-ink/40 backdrop-blur-sm"
        aria-hidden="true"
      />
      <motion.div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="share-modal-title"
        initial={{ opacity: 0, scale: 0.95 }}
        animate={{ opacity: 1, scale: 1 }}
        exit={{ opacity: 0, scale: 0.95 }}
        transition={SPRING}
        className="pointer-events-auto relative w-full max-w-md rounded-xl border-2 border-ink bg-white p-6 shadow-[6px_6px_0px_#1C1A12]"
      >
        <h2
          id="share-modal-title"
          className="font-display text-lg font-bold text-ink"
        >
          {t("forms.share.title")}
        </h2>
        <p className="mt-1 text-sm text-ink/60">
          {t("forms.share.description")}
        </p>

        {/* 1: Copy link */}
        <div className="mt-4 flex items-center gap-2">
          <input
            id="share-url"
            readOnly
            value={url}
            onFocus={(e) => e.target.select()}
            className="min-w-0 flex-1 rounded-xl border-2 border-ink bg-print-white px-3 py-2 font-mono text-xs text-ink focus:border-flash-yellow focus:outline-none"
            aria-label={t("forms.share.urlAriaLabel")}
          />
          <button
            type="button"
            onClick={() => copy(url)}
            aria-live="polite"
            className={`inline-flex shrink-0 items-center gap-1 rounded-full border-2 border-ink px-4 py-2 text-xs font-bold text-ink transition-colors duration-150 ${
              copied === "success"
                ? "bg-flash-yellow"
                : copied === "failed"
                  ? "bg-flash-coral"
                  : "bg-white hover:bg-paper-grey/60"
            }`}
          >
            {copied === "success" && (
              <Check className="h-3.5 w-3.5" strokeWidth={3} aria-hidden="true" />
            )}
            {copied === "success"
              ? t("common.copied")
              : copied === "failed"
                ? t("common.failed")
                : t("common.copy")}
          </button>
        </div>

        {/* 2: QR code */}
        <div className="mt-4 flex flex-col items-center">
          <div className="rounded-xl border-2 border-ink bg-print-white p-3">
            <QRCodeCanvas
              ref={canvasRef}
              value={url || `https://jejak.app/u/${username}`}
              size={200}
              level="M"
              fgColor="#1C1A12"
              bgColor="#FAFAF7"
            />
          </div>
          <button
            type="button"
            onClick={downloadQR}
            className="mt-3 inline-flex items-center gap-1.5 rounded-full border-2 border-ink bg-white px-4 py-2 text-xs font-bold text-ink transition-colors duration-150 hover:bg-paper-grey/60"
          >
            <Download size={14} aria-hidden="true" />
            {t("common.downloadQr")}
          </button>
        </div>

        {/* 3: Preview: opens /u/{username} in a new tab */}
        <a
          href={`/u/${username}`}
          target="_blank"
          rel="noopener noreferrer"
          className="mt-4 inline-flex w-full items-center justify-center gap-2 rounded-full border-2 border-ink bg-white px-4 py-2.5 text-sm font-bold text-ink transition-transform duration-150 hover:-translate-y-0.5"
        >
          <ExternalLink size={16} aria-hidden="true" />
          {t("common.viewProfile")}
        </a>
      </motion.div>
    </div>
  );
}
