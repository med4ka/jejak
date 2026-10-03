import Link from "next/link";
import { SearchX, ArrowRight } from "lucide-react";
import { getServerTranslation } from "../lib/serverI18n";

export const metadata = {
  title: "Tidak ditemukan: Jejak",
  description: "Halaman yang kamu cari tidak ada.",
};

// Custom 404 (not the Next.js default): Instant Print design system:
// print-white, ink, hard-shadow yellow pill + ghost button, Space Grotesk
// headline / Work Sans subline. The fade + y-8 animation comes from the
// global PageTransition in layout.jsx (the layout itself stays untouched).
// Responsive: w-full on mobile (buttons stack at full width), a row from sm
// upward.
export default function NotFound() {
  const { t } = getServerTranslation();
  return (
    <main className="flex min-h-screen flex-col items-center justify-center bg-print-white px-4 py-24 text-center sm:px-6">
      <div className="w-full max-w-md">
        <SearchX
          className="mx-auto h-16 w-16 text-ink"
          strokeWidth={1.5}
          aria-hidden="true"
        />
        <h1 className="mt-6 font-display text-3xl font-bold leading-tight text-ink sm:text-4xl">
          {t("notFound.title")}
        </h1>
        <p className="mt-3 text-base text-muted">{t("notFound.subtitle")}</p>
        <div className="mt-8 flex flex-col gap-3 sm:flex-row sm:justify-center">
          <Link
            href="/"
            className="inline-flex w-full items-center justify-center gap-2 rounded-full border-2 border-ink bg-flash-yellow px-7 py-3 text-sm font-bold text-ink shadow-[4px_4px_0px_#1C1A12] transition-transform duration-150 hover:-translate-y-0.5 active:translate-y-0"
          >
            {t("notFound.goHomeButton")}
            <ArrowRight className="h-4 w-4" strokeWidth={2.5} aria-hidden="true" />
          </Link>
          <Link
            href="/app"
            className="inline-flex w-full items-center justify-center gap-2 rounded-full border-2 border-ink bg-transparent px-7 py-3 text-sm font-bold text-ink transition-colors duration-150 hover:bg-paper-grey/60"
          >
            {t("notFound.checkLinksButton")}
          </Link>
        </div>
      </div>
    </main>
  );
}
