"use client";

import { useEffect, useRef } from "react";
import { motion } from "framer-motion";
import { QRCodeCanvas } from "qrcode.react";
import CopyButton from "./CopyButton";
import { SPRING } from "../../lib/animations";
import { useTranslation } from "../../lib/I18nProvider";

// Per-link QR modal: generated client-side from the short URL (qrcode.react,
// no server request), PNG downloaded from the canvas. The modal shell follows
// AuthModal (backdrop + panel) but transitions with SPRING: a modal is UI
// state → a natural spring without overshoot (Apple-style detail #1). The QR
// carries a 2px ink frame per DESIGN.md instead of being shown as a bare code.
export default function QrModal({ shortUrl, shortCode, onClose, st }) {
  const { t } = useTranslation();
  const canvasRef = useRef(null);

  useEffect(() => {
    function onKey(e) {
      if (e.key === "Escape") {
        onClose();
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  function download() {
    const canvas = canvasRef.current;
    if (!canvas) {
      return;
    }
    const a = document.createElement("a");
    a.href = canvas.toDataURL("image/png");
    a.download = `jejak-${shortCode}.png`;
    a.click();
  }

  return (
    <div className="pointer-events-none fixed inset-0 z-[60] flex items-end justify-center sm:items-center sm:p-4">
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        transition={SPRING}
        onClick={onClose}
        className="pointer-events-auto absolute inset-0 bg-ink/60"
        aria-hidden="true"
      />
      <motion.div
        role="dialog"
        aria-modal="true"
        aria-label={t("forms.qr.dialogAriaLabel", { shortCode })}
        initial={{ opacity: 0, y: 24, scale: 0.98 }}
        animate={{ opacity: 1, y: 0, scale: 1 }}
        exit={{ opacity: 0, y: 12, scale: 0.98 }}
        transition={SPRING}
        className="pointer-events-auto relative w-full max-w-[320px] rounded-t-2xl border-2 border-ink bg-print-white p-6 text-center text-ink sm:rounded-2xl"
      >
        <p className="font-mono text-sm font-bold">
          {t("forms.qr.shortCode", { shortCode })}
        </p>
        <CopyButton
          text={shortUrl}
          label={t("forms.qr.copyUrlLabel")}
          className="mt-2 w-full rounded-full border-2 border-ink bg-print-white px-4 py-2 text-sm font-bold text-ink transition-colors duration-150 hover:bg-paper-grey/60"
        />
        <div className="mx-auto mt-3 w-fit rounded-xl border-2 border-ink bg-print-white p-3">
          <QRCodeCanvas ref={canvasRef} value={shortUrl} size={200} level="M" />
        </div>
        <button
          onClick={download}
          className={`mt-4 w-full rounded-full border-2 border-ink ${st.accent} px-4 py-2 text-sm font-bold transition-[filter] duration-150 hover:brightness-95`}
        >
          {t("forms.qr.downloadLabel")}
        </button>
      </motion.div>
    </div>
  );
}
