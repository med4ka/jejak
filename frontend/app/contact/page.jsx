import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { getServerTranslation } from "../../lib/serverI18n";

export const metadata = {
  title: "Kontak: Jejak",
  description: "Hubungi tim Jejak.",
};

// Stub page (footer audit). Placeholder content: official copy to follow;
// the point is that the footer Company link is not dead / does not 404.
export default async function ContactPage() {
  const { t } = await getServerTranslation();
  const [beforeEmail, afterEmail] = t("contact.paragraph2").split("{email}");
  return (
    <main className="mx-auto w-full max-w-2xl px-4 pb-16 pt-24 sm:px-6">
      <article className="rounded-xl border-2 border-ink bg-print-white p-6 shadow-[4px_4px_0px_#1C1A12] sm:p-8">
        <h1 className="font-display text-3xl font-bold text-ink">
          {t("contact.title")}
        </h1>
        <div className="mt-5 space-y-4 text-base leading-relaxed text-ink">
          <p>{t("contact.paragraph1")}</p>
          <p>
            {beforeEmail}
            <a
              href="mailto:halo@jejak.app"
              className="font-bold underline underline-offset-2"
            >
              halo@jejak.app
            </a>
            {afterEmail}
          </p>
          <p className="text-sm text-muted">{t("contact.responseTime")}</p>
        </div>
        <Link
          href="/"
          className="mt-8 inline-flex items-center gap-2 rounded-full border-2 border-ink bg-transparent px-6 py-3 text-sm font-bold text-ink transition-colors duration-150 hover:bg-paper-grey/60"
        >
          <ArrowLeft className="h-4 w-4" strokeWidth={2.5} />
          {t("contact.backToHome")}
        </Link>
      </article>
    </main>
  );
}
