"use client";

import { useEffect, useRef, useState } from "react";
import { motion } from "framer-motion";
import { SPRING } from "../../lib/animations";
import { useTranslation } from "../../lib/I18nProvider";

// Reusable submit button with a loading state: 16x16 spinner plus the
// (loadingLabel) label, shown for at least 300ms so very fast requests do not
// flicker, and guarded against double-clicks (disabled while isLoading).
// Accepts the complete styling className from the parent (each theme's own
// accent). Motion (Apple-style detail #4): hover lifts -2px, tap scales 0.97
// with y returning to 0 (forced back down while pressed): both SPRING, not a
// duration tween.
export default function SubmitButton({
  isLoading,
  loadingLabel,
  className = "",
  disabled,
  children,
}) {
  const { t } = useTranslation();
  const [showLoading, setShowLoading] = useState(false);
  const appearedAt = useRef(0);

  useEffect(() => {
    if (isLoading) {
      appearedAt.current = Date.now();
      setShowLoading(true);
      return undefined;
    }
    if (!showLoading) {
      return undefined;
    }
    const elapsed = Date.now() - appearedAt.current;
    const remaining = Math.max(0, 300 - elapsed);
    const t = setTimeout(() => setShowLoading(false), remaining);
    return () => clearTimeout(t);
  }, [isLoading, showLoading]);

  return (
    <motion.button
      type="submit"
      disabled={disabled || isLoading}
      whileHover={{ y: -2, transition: SPRING }}
      whileTap={{ scale: 0.97, y: 0, transition: SPRING }}
      aria-busy={isLoading || undefined}
      className={`inline-flex items-center justify-center gap-2 disabled:cursor-not-allowed disabled:opacity-70 ${className}`}
    >
      {showLoading && (
        <svg
          className="h-4 w-4 shrink-0 animate-spin"
          xmlns="http://www.w3.org/2000/svg"
          fill="none"
          viewBox="0 0 24 24"
          aria-hidden="true"
        >
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
          <path
            className="opacity-75"
            fill="currentColor"
            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
          />
        </svg>
      )}
      {showLoading ? (loadingLabel ?? t("common.processing")) : children}
    </motion.button>
  );
}
