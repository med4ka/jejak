"use client";

// =====================================================================
// LANGUAGE SWITCHER: pill trigger + dropdown for the 3 locales (id/en/de).
// The active locale comes from the I18nProvider context (not local state):
// picking an option calls setLocale() from lib/i18n, which writes the
// NEXT_LOCALE cookie and reloads the page, so the server re-renders the
// tree with the new dictionary: nothing has to be synced after the click.
// Option labels are ALWAYS shown in their own native language (never run
// through t()), and the aria-label uses a tiny native-name map instead of
// a messages/*.json key (a "switch language" string would itself need a
// translation in every locale).
// =====================================================================
import { useEffect, useRef, useState } from "react";
import { AnimatePresence, motion } from "framer-motion";
import { Languages } from "lucide-react";
import { SPRING } from "../../lib/animations";
import { setLocale } from "../../lib/i18n";
import { useTranslation } from "../../lib/I18nProvider";

const OPTIONS = [
  { locale: "id", short: "ID", label: "Bahasa Indonesia" },
  { locale: "en", short: "EN", label: "English" },
  { locale: "de", short: "DE", label: "Deutsch" },
];

// Shared with the mobile drawer's inline language expand (same native-name
// options, no dictionary needed): the drawer renders these rows itself
// instead of mounting a second dropdown.
export const LANGUAGE_OPTIONS = OPTIONS;

// aria-label keyed by the ACTIVE locale (native names, no dictionary key).
const ARIA = { id: "Bahasa", en: "Language", de: "Sprache" };
export const LANGUAGE_ARIA = ARIA;

export default function LanguageSwitcher() {
  const { locale } = useTranslation();
  const [open, setOpen] = useState(false);
  const rootRef = useRef(null);

  // Outside click + Escape close the dropdown; both listeners are attached
  // only while it is open and removed on close/unmount (same pattern as the
  // navbar's Fitur and account dropdowns).
  useEffect(() => {
    if (!open) return undefined;
    function onDoc(e) {
      if (rootRef.current && !rootRef.current.contains(e.target)) {
        setOpen(false);
      }
    }
    function onKey(e) {
      if (e.key === "Escape") setOpen(false);
    }
    document.addEventListener("mousedown", onDoc);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDoc);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  const active = OPTIONS.find((o) => o.locale === locale) || OPTIONS[0];

  return (
    <div className="relative" ref={rootRef}>
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-label={ARIA[locale] || ARIA.id}
        className="border-2 border-ink bg-white rounded-full px-3 py-2 inline-flex items-center gap-1.5 text-sm font-bold text-ink transition-colors duration-150 hover:bg-ink/5"
      >
        <Languages className="h-4 w-4" strokeWidth={2.5} aria-hidden="true" />
        {active.short}
      </button>

      <AnimatePresence>
        {open && (
          <motion.div
            role="menu"
            initial={{ opacity: 0, scale: 0.96 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0, scale: 0.96 }}
            transition={SPRING}
            className="absolute right-0 top-full z-50 mt-2 w-48 origin-top-right bg-white border-2 border-ink shadow-[4px_4px_0_#1C1A12] rounded-[12px] p-2"
          >
            {OPTIONS.map((option) => (
              <button
                key={option.locale}
                type="button"
                role="menuitem"
                onClick={() => {
                  setOpen(false);
                  setLocale(option.locale);
                }}
                className={`px-3 py-2 rounded-[8px] hover:bg-ink/5 w-full text-left text-ink${
                  option.locale === locale ? " bg-flash-yellow font-bold" : ""
                }`}
              >
                {option.label}
              </button>
            ))}
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
