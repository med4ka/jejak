"use client";

import { useEffect, useRef, useState } from "react";

// Reusable copy-to-clipboard hook: `copy(text)` writes to the clipboard
// (navigator.clipboard, textarea fallback for non-secure contexts), then
// sets the feedback status "success" | "failed" for 2000ms before resetting
// to null. Returns `{ copied, copy }`: consumers use `copied` to swap the
// button label ("Tersalin"/"Gagal").
export default function useCopyToClipboard() {
  const [copied, setCopied] = useState(null);
  const timer = useRef(null);

  useEffect(() => () => clearTimeout(timer.current), []);

  async function copy(text) {
    let ok = false;
    try {
      await navigator.clipboard.writeText(text);
      ok = true;
    } catch {
      try {
        const ta = document.createElement("textarea");
        ta.value = text;
        ta.setAttribute("readonly", "");
        ta.style.position = "fixed";
        ta.style.opacity = "0";
        document.body.appendChild(ta);
        ta.select();
        ok = document.execCommand("copy");
        ta.remove();
      } catch {
        ok = false;
      }
    }
    setCopied(ok ? "success" : "failed");
    clearTimeout(timer.current);
    timer.current = setTimeout(() => setCopied(null), 2000);
  }

  return { copied, copy };
}
