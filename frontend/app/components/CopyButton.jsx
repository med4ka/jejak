"use client";

import { useEffect, useRef, useState } from "react";
import { AnimatePresence, motion } from "framer-motion";

// Tombol salin reusable: klik -> clipboard, toast "Tersalin!" fixed di BAWAH
// (navbar kapsul di atas, jadi tidak bentrok), fade+slide 200ms easeOut,
// auto-dismiss ~1.8 detik lalu fade keluar. Fallback textarea untuk konteks
// non-secure di mana navigator.clipboard tidak tersedia.
export default function CopyButton({ text, label = "Salin", className = "" }) {
  const [copied, setCopied] = useState(false);
  const timer = useRef(null);

  useEffect(() => () => clearTimeout(timer.current), []);

  async function copy() {
    try {
      await navigator.clipboard.writeText(text);
    } catch {
      const ta = document.createElement("textarea");
      ta.value = text;
      document.body.appendChild(ta);
      ta.select();
      document.execCommand("copy");
      ta.remove();
    }
    setCopied(true);
    clearTimeout(timer.current);
    timer.current = setTimeout(() => setCopied(false), 1800);
  }

  const style =
    className !== ""
      ? className
      : "shrink-0 rounded-full border-2 border-ink bg-print-white px-3 py-0.5 text-xs font-bold text-ink transition-colors duration-150 hover:bg-paper-grey/60";

  return (
    <>
      <button onClick={copy} className={style}>
        {label}
      </button>
      <AnimatePresence>
        {copied && (
          <div className="pointer-events-none fixed inset-x-0 bottom-6 z-[70] flex justify-center">
            <motion.div
              initial={{ opacity: 0, y: 12 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: 8 }}
              transition={{ duration: 0.2, ease: "easeOut" }}
              className="rounded-full border-2 border-ink bg-ink px-4 py-2 text-sm font-bold text-print-white"
            >
              Tersalin!
            </motion.div>
          </div>
        )}
      </AnimatePresence>
    </>
  );
}
