// Server-side translation helper for Server Components that export
// `metadata` (terms/privacy/contact/not-found) and therefore cannot be
// turned into client components. Same dictionaries and same lookup rules
// as the client provider: only the cookie is read here via next/headers.

import { cookies } from "next/headers";
import { getLocale, DEFAULT_LOCALE } from "./i18n";
import { translate } from "./translate";
import idMessages from "../messages/id.json";
import enMessages from "../messages/en.json";
import deMessages from "../messages/de.json";

const MESSAGES = { id: idMessages, en: enMessages, de: deMessages };

// Full dictionary for a locale (falls back to the default). Used by
// generateMetadata in app/layout.jsx and by getServerTranslation below.
export function loadMessages(locale) {
  return MESSAGES[locale] || MESSAGES[DEFAULT_LOCALE];
}

export async function getServerTranslation() {
  const cookieStore = await cookies();
  const locale = getLocale(cookieStore.get("NEXT_LOCALE")?.value);
  const messages = loadMessages(locale);
  return {
    locale,
    t: (key, vars) => translate(messages, key, vars),
  };
}
