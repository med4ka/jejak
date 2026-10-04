import { Space_Grotesk, Archivo_Black, Work_Sans, JetBrains_Mono, Caveat } from "next/font/google";
import { cookies } from "next/headers";
import "./globals.css";
import Navbar from "./components/Navbar";
// PageTransition and the global provider moved to lib/ (Apple detail #5).
// MotionProvider is the client wrapper for MotionConfig
// reducedMotion="user" (context cannot cross a server component) → the
// whole app (Navbar, every page, modals, dashboard) collapses to
// opacity-only when a visitor requests reduced motion.
import PageTransition from "../lib/PageTransition";
import MotionProvider from "../lib/MotionProvider";
// ToastProvider: global toast system (Account settings, 2026-09-30):
// mounted once at the root so useToast() works in any component
// (dashboard/PengaturanTab etc.) without prop-drilling.
import { ToastProvider } from "./components/Toast";
// i18n: dictionaries are bundled JSON (no runtime fetch); the active
// locale comes from the NEXT_LOCALE cookie on every request.
import { I18nProvider } from "../lib/I18nProvider";
import { getLocale, DEFAULT_LOCALE } from "../lib/i18n";
import { loadMessages } from "../lib/serverI18n";
import idMessages from "../messages/id.json";
import enMessages from "../messages/en.json";
import deMessages from "../messages/de.json";

const MESSAGES = { id: idMessages, en: enMessages, de: deMessages };

// Per-locale document metadata: title/description/openGraph come from
// messages/<locale>.json (keys metadata.title / metadata.description) so the
// browser tab, link previews, and WhatsApp/Twitter/Discord scrapers follow
// the NEXT_LOCALE cookie instead of always shipping the Indonesian copy.
export async function generateMetadata() {
  const locale = getLocale((await cookies()).get("NEXT_LOCALE")?.value);
  const messages = loadMessages(locale);
  const title = messages.metadata?.title || "Jejak";
  const description = messages.metadata?.description || "Jejak: link-in-bio untuk kreator.";
  return {
    // metadataBase makes relative paths (e.g. /og-default.png) absolute in
    // meta tags: scrapers need absolute URLs. SITE_URL decides production.
    metadataBase: new URL(process.env.SITE_URL || "http://localhost:3000"),
    title,
    description,
    openGraph: {
      title,
      description,
      images: ["/og-default.png"],
      siteName: "Jejak",
      type: "website",
    },
  };
}

// DESIGN.md §2: Caveat (caption) is used in exactly 2 places: the link card
// caption and the profile header (§3). Do not use font-caption anywhere
// else.
const display = Space_Grotesk({ subsets: ["latin"], weight: ["500", "700"], variable: "--font-display" });
const blackhead = Archivo_Black({ subsets: ["latin"], weight: "400", variable: "--font-blackhead" });
const body = Work_Sans({ subsets: ["latin"], variable: "--font-body" });
const mono = JetBrains_Mono({ subsets: ["latin"], variable: "--font-mono" });
const caption = Caveat({ subsets: ["latin"], weight: ["600"], variable: "--font-caption" });

export default async function RootLayout({ children }) {
  const locale = getLocale((await cookies()).get("NEXT_LOCALE")?.value);
  const messages = MESSAGES[locale] || MESSAGES[DEFAULT_LOCALE];
  return (
    <html lang={locale} className={`${display.variable} ${blackhead.variable} ${body.variable} ${mono.variable} ${caption.variable}`}>
      <body className="min-h-screen bg-print-white font-body text-ink">
        <I18nProvider locale={locale} messages={messages}>
          <MotionProvider>
            <Navbar />
            <ToastProvider>
              <PageTransition>{children}</PageTransition>
            </ToastProvider>
          </MotionProvider>
        </I18nProvider>
      </body>
    </html>
  );
}
