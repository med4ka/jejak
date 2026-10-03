"use client";

// I18nProvider: carries the locale + message dictionary from the server
// layout down to any client component. Dictionaries are plain JSON objects
// imported by app/layout.jsx (messages/<locale>.json), so no fetch happens
// at runtime; switching locale = set cookie + full reload.

import { createContext, useContext, useMemo } from "react";
import { DEFAULT_LOCALE } from "./i18n";
import { translate } from "./translate";

const I18nContext = createContext({ locale: DEFAULT_LOCALE, messages: {} });

export function I18nProvider({ locale, messages, children }) {
  const value = useMemo(
    () => ({ locale: locale || DEFAULT_LOCALE, messages: messages || {} }),
    [locale, messages]
  );
  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

// t(key) -> translated string, or the key itself when the lookup misses.
// t(key, vars) optionally fills {name} placeholders (see lib/translate.js).
export function useTranslation() {
  const { locale, messages } = useContext(I18nContext);
  const t = (key, vars) => translate(messages, key, vars);
  return { t, locale };
}
