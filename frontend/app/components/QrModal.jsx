"use client";

import { useEffect, useRef } from "react";
import { motion } from "framer-motion";
import { QRCodeCanvas } from "qrcode.react";
import CopyButton from "./CopyButton";

// Modal QR per link: generate client-side dari short URL (qrcode.react,
// tanpa request server), download PNG dari canvas. Shell modal mengikuti
// pola AuthModal (backdrop + fade/scale easeOut 200ms, tanpa bounce).
// Bingkai QR pakai border ink 2px sesuai DESIGN.md (bukan QR polos).
export default function QrModal({ shortUrl, shortCode, onClose, st }) {
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
        transition={{ duration: 0.2, ease: "easeOut" }}
        onClick={onClose}
        className="pointer-events-auto absolute inset-0 bg-ink/60"
        aria-hidden="true"
      />
      <motion.div
        role="dialog"
        aria-modal="true"
        aria-label={`QR Code ${shortCode}`}
        initial={{ opacity: 0, y: 24, scale: 0.98 }}
        animate={{ opacity: 1, y: 0, scale: 1 }}
        exit={{ opacity: 0, y: 12, scale: 0.98 }}
        transition={{ duration: 0.2, ease: "easeOut" }}
        className="pointer-events-auto relative w-full max-w-[320px] rounded-t-2xl border-2 border-ink bg-print-white p-6 text-center text-ink sm:rounded-2xl"
      >
        <p className="font-mono text-sm font-bold">/{shortCode}</p>
        <CopyButton
          text={shortUrl}
          label="Salin URL"
          className="mt-2 w-full rounded-full border-2 border-ink bg-print-white px-4 py-2 text-sm font-bold text-ink transition-colors duration-150 hover:bg-paper-grey/60"
        />
        <div className="mx-auto mt-3 w-fit rounded-xl border-2 border-ink bg-print-white p-3">
          <QRCodeCanvas ref={canvasRef} value={shortUrl} size={200} level="M" />
        </div>
        <button
          onClick={download}
          className={`mt-4 w-full rounded-full border-2 border-ink ${st.accent} px-4 py-2 text-sm font-bold transition-[filter] duration-150 hover:brightness-95`}
        >
          Download PNG
        </button>
      </motion.div>
    </div>
  );
}
