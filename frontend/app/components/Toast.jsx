"use client";

import { createContext, useCallback, useContext, useRef, useState } from "react";
import { AnimatePresence, motion } from "framer-motion";
import { CheckCircle2, XCircle } from "lucide-react";
import { SPRING } from "../../lib/animations";

// Global toast system (2026-09-30, account settings work): previously there
// was none: each component had its own inline banner (see notice/error in
// DashboardClient) that could not be reused across components. Toasts cover
// actions whose outcome is short and binary (success/failure): save email,
// change password, log out all sessions, delete account: without moving form
// focus. Mounted once in app/layout.jsx and consumed through useToast.
const ToastContext = createContext(null);

// useToast returns { success(msg), error(msg) } and is meant to be called only
// inside <ToastProvider>; outside a provider it deliberately throws (a
// forgotten provider must surface quickly instead of silently no-op'ing).
export function useToast() {
  const ctx = useContext(ToastContext);
  if (!ctx) {
    throw new Error("useToast harus dipakai di dalam <ToastProvider> (pasang di app/layout.jsx)");
  }
  return ctx;
}

let nextId = 0;

export function ToastProvider({ children }) {
  const [toasts, setToasts] = useState([]);
  const dismiss = useCallback((id) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  const push = useCallback(
    (variant, message) => {
      const id = ++nextId;
      setToasts((prev) => [...prev, { id, variant, message }]);
      // Auto-dismiss after 3 seconds; dismissing an id that is already gone is
      // a no-op, so an earlier manual removal stays safe (there is no close
      // button: toasts are short-lived, so an extra click would be noise).
      setTimeout(() => dismiss(id), 3000);
      return id;
    },
    [dismiss]
  );

  const toast = {
    success: (msg) => push("success", msg),
    error: (msg) => push("error", msg),
  };

  return (
    <ToastContext.Provider value={toast}>
      {children}
      {/* aria-live polite: screen readers announce the toast without cutting
          off the current utterance; the bottom-right position (bottom center
          on mobile) keeps it clear of the form's primary action buttons.
          z-[70] sits above modals (z-[60]). */}
      <div
        aria-live="polite"
        className="pointer-events-none fixed inset-x-4 bottom-4 z-[70] flex flex-col items-center gap-2 sm:inset-x-auto sm:right-6 sm:items-end"
      >
        <AnimatePresence initial={false}>
          {toasts.map((t) => (
            <motion.div
              key={t.id}
              layout
              initial={{ opacity: 0, y: 16, scale: 0.98 }}
              animate={{ opacity: 1, y: 0, scale: 1 }}
              exit={{ opacity: 0, y: 8, scale: 0.98 }}
              transition={SPRING}
              role="status"
              className={`pointer-events-auto flex max-w-sm items-center gap-2 rounded-xl border-2 border-ink px-4 py-3 text-sm font-bold shadow-[4px_4px_0px_#1C1A12] ${
                t.variant === "error" ? "bg-flash-coral text-white" : "bg-flash-yellow text-ink"
              }`}
            >
              {t.variant === "error" ? (
                <XCircle size={16} aria-hidden="true" className="shrink-0" />
              ) : (
                <CheckCircle2 size={16} aria-hidden="true" className="shrink-0" />
              )}
              <span className="min-w-0 break-words">{t.message}</span>
            </motion.div>
          ))}
        </AnimatePresence>
      </div>
    </ToastContext.Provider>
  );
}
