"use client";

import { useEffect, useRef, useState } from "react";
import { AnimatePresence, motion } from "framer-motion";
import { SPRING } from "../../lib/animations";
import { useTranslation } from "../../lib/I18nProvider";

// Reusable confirmation modal (2026-09-30, account settings work): white
// panel + 2px ink border + offset instant-print shadow (DESIGN.md §5, same as
// ShareModal/EditLinkModal). Self-contained: AnimatePresence lives inside so
// that <ConfirmModal open={...}> can be rendered from anywhere without a
// wrapper.
//
// Props:
//   open         : visibility control
//   onClose      : close (Esc / backdrop / Cancel); ignored while busy
//   title        : dialog title (required; doubles as the accessible name via
//                   aria-label)
//   description  : optional explanation
//   confirmLabel : label of the confirm button
//   confirmVariant:  "default" (flash-yellow) | "danger" (flash-coral)
//   requireTyping: exact string that must be typed to enable the button
//                   (example: "HAPUS"); "" = no typing requirement
//   busy         : while true the buttons are locked (a request is in flight)
//   onConfirm    : called when the confirm button is pressed (async parents
//                   set busy themselves and close the modal on success)
//   children     : slot for extra content (e.g. a password confirmation
//                   input)
export default function ConfirmModal({
  open,
  onClose,
  title,
  description,
  confirmLabel,
  confirmVariant = "default",
  requireTyping = "",
  busy = false,
  onConfirm,
  children,
}) {
  const { t } = useTranslation();
  const panelRef = useRef(null);
  const [typed, setTyped] = useState("");
  // Default confirm label resolved here (not as a parameter default) so the
  // translation hook can be used; callers passing confirmLabel keep their
  // string as-is.
  const confirmText = confirmLabel ?? t("common.confirm");
  // "{requireTyping}" stays untranslated here on purpose: the value is
  // rendered inside its own <span>, so the sentence is split on the token and
  // the node is reinserted between the two halves.
  const [typingBefore, typingAfter] = t("forms.confirm.typingPrompt").split(
    "{requireTyping}"
  );

  // Reset the typing input every time the modal reopens: a leftover "HAPUS"
  // from the previous action must not re-enable the button immediately.
  useEffect(() => {
    if (!open) {
      setTyped("");
    }
  }, [open]);

  // Escape closes the modal except while busy: an already-sent request must
  // not be dismissed from the keyboard.
  useEffect(() => {
    if (!open) {
      return undefined;
    }
    function onKey(e) {
      if (e.key === "Escape" && !busy) {
        onClose();
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, busy, onClose]);

  // Focus trap: focus enters the panel on open and Tab cycles within it:
  // controls behind the modal remain unreachable by keyboard while it is open.
  useEffect(() => {
    if (!open) {
      return undefined;
    }
    const panel = panelRef.current;
    if (panel) {
      const first = panel.querySelector("input, button, [href], select, textarea");
      if (first) first.focus();
    }
    function onKey(e) {
      if (e.key !== "Tab" || !panelRef.current) return;
      const nodes = Array.from(
        panelRef.current.querySelectorAll(
          'input:not([disabled]), button:not([disabled]), [href], select:not([disabled]), textarea:not([disabled])'
        )
      ).filter((n) => n.offsetParent !== null);
      if (nodes.length === 0) return;
      const first = nodes[0];
      const last = nodes[nodes.length - 1];
      if (e.shiftKey && document.activeElement === first) {
        e.preventDefault();
        last.focus();
      } else if (!e.shiftKey && document.activeElement === last) {
        e.preventDefault();
        first.focus();
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open]);

  const typingLocked = requireTyping !== "" && typed !== requireTyping;
  const danger = confirmVariant === "danger";

  return (
    <AnimatePresence>
      {open && (
        <div className="fixed inset-0 z-[60] flex items-end justify-center sm:items-center sm:p-4">
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={SPRING}
            onClick={() => {
              if (!busy) onClose();
            }}
            aria-hidden="true"
            className="absolute inset-0 bg-ink/40 backdrop-blur-xs"
          />
          <motion.div
            ref={panelRef}
            role="dialog"
            aria-modal="true"
            aria-label={title}
            initial={{ opacity: 0, y: 24, scale: 0.98 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: 12, scale: 0.98 }}
            transition={SPRING}
            className="relative w-full max-w-md rounded-xl border-2 border-ink bg-white p-6 text-ink shadow-[6px_6px_0px_#1C1A12]"
          >
            <h2 className="font-display text-lg font-bold">{title}</h2>
            {description && <p className="mt-2 text-sm text-ink/70">{description}</p>}

            {children}

            {requireTyping !== "" && (
              <label className="mt-4 block text-sm font-medium">
                {typingBefore}
                <span className="font-mono font-bold">{requireTyping}</span>
                {typingAfter}
                <input
                  value={typed}
                  onChange={(e) => setTyped(e.target.value)}
                  disabled={busy}
                  autoComplete="off"
                  spellCheck={false}
                  className="mt-1 w-full rounded-xl border-2 border-ink bg-white px-3 py-2 font-mono text-sm focus:border-flash-yellow focus:ring-2 focus:ring-flash-yellow/30 focus:outline-hidden transition-all duration-150 disabled:opacity-60"
                />
              </label>
            )}

            <div className="mt-6 flex flex-wrap justify-end gap-2">
              <button
                type="button"
                onClick={onClose}
                disabled={busy}
                className="rounded-full border-2 border-ink bg-white px-4 py-2 text-sm font-bold text-ink transition-colors duration-150 hover:bg-paper-grey/60 disabled:cursor-not-allowed disabled:opacity-60"
              >
                {t("common.cancel")}
              </button>
              <button
                type="button"
                onClick={onConfirm}
                disabled={busy || typingLocked}
                className={`rounded-full border-2 border-ink px-4 py-2 text-sm font-bold transition-[filter] duration-150 hover:brightness-95 disabled:cursor-not-allowed disabled:opacity-60 ${
                  danger ? "bg-flash-coral text-white" : "bg-flash-yellow text-ink"
                }`}
              >
                {busy ? t("common.processing") : confirmText}
              </button>
            </div>
          </motion.div>
        </div>
      )}
    </AnimatePresence>
  );
}
