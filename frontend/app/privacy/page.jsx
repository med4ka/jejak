import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { getServerTranslation } from "../../lib/serverI18n";

export const metadata = {
  title: "Kebijakan Privasi: Jejak",
  description: "Kebijakan privasi dan data pengguna Jejak.",
};

// Stub page (footer audit). Placeholder content: official copy to follow;
// the point is that the footer Legal link is not dead / does not 404.
export default async function PrivacyPage() {
  const { t } = await getServerTranslation();
  return (
    <main className="mx-auto w-full max-w-2xl px-4 pb-16 pt-24 sm:px-6">
      <article className="rounded-xl border-2 border-ink bg-print-white p-6 shadow-[4px_4px_0px_#1C1A12] sm:p-8">
        <h1 className="font-display text-3xl font-bold text-ink">
          {t("privacy.title")}
        </h1>
        <div className="mt-5 space-y-4 text-base leading-relaxed text-ink">
          <p>{t("privacy.paragraph1")}</p>
          <p>{t("privacy.paragraph2")}</p>
          <p className="text-sm text-muted">{t("privacy.lastUpdated")}</p>
        </div>
        <Link
          href="/"
          className="mt-8 inline-flex items-center gap-2 rounded-full border-2 border-ink bg-transparent px-6 py-3 text-sm font-bold text-ink transition-colors duration-150 hover:bg-paper-grey/60"
        >
          <ArrowLeft className="h-4 w-4" strokeWidth={2.5} />
          {t("privacy.backToHome")}
        </Link>
      </article>
    </main>
  );
}
