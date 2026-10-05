import { MessageCircle } from "lucide-react";
import WhatsAppTool from "../../components/WhatsAppTool";
import { getServerTranslation } from "../../../lib/serverI18n";

export async function generateMetadata() {
  const { t } = await getServerTranslation();
  return {
    title: t("tools.whatsapp.metadataTitle"),
    description: t("tools.whatsapp.metadataDescription"),
  };
}

// WhatsApp click-to-chat builder page (deep link tools,  ​2026-10-04).
// Prefill via query string (?phone=…&message=…): makes shareable templates
// (and screenshots) without changing the form contract.
export default async function WhatsAppToolPage({ searchParams }) {
  const { t } = await getServerTranslation();
  const sp = await searchParams;
  const initialPhone = typeof sp?.phone === "string" ? sp.phone.slice(0, 30) : "";
  const initialMessage = typeof sp?.message === "string" ? sp.message.slice(0, 500) : "";
  return (
    <main className="mx-auto w-full max-w-2xl px-4 pb-16 pt-24 sm:px-6">
      <article className="rounded-xl border-2 border-ink bg-print-white p-6 shadow-[4px_4px_0px_#1C1A12] sm:p-8">
        <span className="inline-flex items-center gap-2 rounded-full border-2 border-ink bg-[#25D366] px-4 py-1.5 text-xs font-bold text-ink">
          <MessageCircle className="h-3.5 w-3.5" strokeWidth={2.5} />
          {t("tools.whatsapp.badge")}
        </span>
        <h1 className="mt-4 font-display text-3xl font-bold text-ink">
          {t("tools.whatsapp.heading")}
        </h1>
        <p className="mt-3 text-base leading-relaxed text-ink">
          {t("tools.whatsapp.descriptionPart1")}{" "}
          <span className="font-mono font-bold">{t("tools.whatsapp.waMe")}</span>{" "}
          {t("tools.whatsapp.descriptionPart2")}
        </p>
        <div className="mt-6">
          <WhatsAppTool initialPhone={initialPhone} initialMessage={initialMessage} />
        </div>
      </article>
    </main>
  );
}
