"use client";

import { useState } from "react";
import { MessageCircle } from "lucide-react";
import SubmitButton from "./SubmitButton";
import CopyButton from "./CopyButton";
import { sameOriginShortUrl } from "../../lib/shortlink";
import { useTranslation } from "../../lib/I18nProvider";

const input =
  "w-full rounded-xl border-2 border-ink bg-print-white px-3 py-2 text-sm text-ink placeholder:text-muted focus:border-flash-yellow focus:ring-2 focus:ring-flash-yellow/30 focus:outline-hidden transition-all duration-150";

// WhatsApp click-to-chat builder (deep link tools, 2026-10-04). Generates a
// wa.me link with a prefilled message and (by default) an auto-shortened
// Jejak link so every click is counted like any other short link (device,
// referrer, analytics).
export default function WhatsAppTool({ initialPhone = "", initialMessage = "" }) {
  const { t } = useTranslation();
  const [phone, setPhone] = useState(initialPhone);
  const [message, setMessage] = useState(initialMessage);
  const [shorten, setShorten] = useState(true);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [result, setResult] = useState(null); // { wa_link, short_url? }
  const [addingBio, setAddingBio] = useState(false);
  const [inBio, setInBio] = useState(false);
  const [bioError, setBioError] = useState("");

  // Runes (not UTF-16 units) so the counter matches the server's []rune cap.
  const messageLen = [...message].length;

  async function onSubmit(e) {
    e.preventDefault();
    setError("");
    setResult(null);
    setInBio(false);
    setBioError("");
    setLoading(true);
    try {
      const res = await fetch("/api/tools/whatsapp-link", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ phone, message, shorten }),
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error || t("tools.whatsapp.errorDefault"));
      }
      setResult(data);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }

  // "Tambah ke Bio": reuses POST /api/shorten so the link lands in the
  // creator's dashboard (= the public /u/[username] bio list) with the
  // session's ownership. With shorten=true an earlier row already exists;
  // this button then effectively confirms/saves it under the account
  // (an extra identical row for an anonymous first attempt is the accepted
  // trade-off instead of a hidden claim mechanism).
  async function addToBio() {
    if (!result || inBio || addingBio) return;
    setBioError("");
    setAddingBio(true);
    try {
      const res = await fetch("/api/shorten", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ url: result.wa_link, tags: ["whatsapp"] }),
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error || t("tools.whatsapp.bioErrorDefault"));
      }
      setResult({ ...result, short_url: data.shortUrl });
      setInBio(true);
    } catch (err) {
      setBioError(err.message);
    } finally {
      setAddingBio(false);
    }
  }

  // Backend builds the short URL from ITS own host (:8082): force the same
  // origin as the page, exactly like ShortenForm does.
  const displayUrl = !result
    ? ""
    : result.short_url
      ? sameOriginShortUrl(result.short_url, null)
      : result.wa_link;

  return (
    <form onSubmit={onSubmit} className="flex flex-col gap-4">
      <label className="text-sm font-medium text-ink">
        {t("tools.whatsapp.phoneLabel")}
        <input
          value={phone}
          onChange={(e) => setPhone(e.target.value)}
          placeholder={t("tools.whatsapp.phonePlaceholder")}
          inputMode="tel"
          autoComplete="tel"
          required
          className={`${input} mt-1`}
        />
        <span className="mt-1 block text-xs font-normal text-muted">
          {t("tools.whatsapp.phoneHint")}
        </span>
      </label>

      <label className="text-sm font-medium text-ink">
        {t("tools.whatsapp.messageLabel")}
        <textarea
          value={message}
          onChange={(e) => setMessage(e.target.value)}
          placeholder={t("tools.whatsapp.messagePlaceholder")}
          rows={4}
          maxLength={500}
          className={`${input} mt-1 resize-y`}
        />
        <span
          className={`mt-1 block text-right text-xs ${messageLen >= 500 ? "font-bold text-[#FF5C3D]" : "text-muted"}`}
        >
          {messageLen}/500
        </span>
      </label>

      {/* Preview: a WhatsApp-style chat bubble so the user sees what the
          recipient will get before generating anything. */}
      <div className="rounded-xl border-2 border-ink bg-paper-grey p-4">
        <p className="mb-3 text-xs font-bold uppercase tracking-wide text-muted">{t("tools.whatsapp.previewTitle")}</p>
        <div className="flex items-end gap-2">
          <span
            aria-hidden="true"
            className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-[#25D366] text-white"
          >
            <MessageCircle className="h-4 w-4" strokeWidth={2.5} />
          </span>
          <div className="max-w-[85%] rounded-2xl rounded-tl-sm border-2 border-ink bg-[#DCF8C6] px-4 py-2.5 text-sm text-ink">
            {message.trim() !== "" ? (
              message
            ) : (
              <span className="italic opacity-60">{t("tools.whatsapp.previewEmpty")}</span>
            )}
            <span className="mt-1 block text-right text-[10px] opacity-50">{t("tools.whatsapp.previewTime")}</span>
          </div>
        </div>
      </div>

      <label className="flex cursor-pointer items-center gap-3 text-sm font-medium text-ink">
        <input
          type="checkbox"
          checked={shorten}
          onChange={(e) => setShorten(e.target.checked)}
          className="h-4 w-4 accent-[#25D366]"
        />
        {t("tools.whatsapp.shortenCheckbox")}
      </label>

      <SubmitButton
        isLoading={loading}
        loadingLabel={t("tools.whatsapp.loadingLabel")}
        className="rounded-full border-2 border-ink bg-[#25D366] px-5 py-3 text-sm font-bold text-ink transition-[filter] duration-150 hover:brightness-95"
      >
        {t("tools.whatsapp.submitLabel")}
      </SubmitButton>

      {error !== "" && (
        <div role="alert" className="rounded-xl border-2 border-ink bg-print-white px-4 py-2.5 text-sm font-medium text-ink">
          {error}
        </div>
      )}

      {result && (
        <div className="flex flex-col gap-3 rounded-xl border-2 border-ink bg-print-white p-4 shadow-[4px_4px_0px_#1C1A12]">
          <p className="text-xs font-bold uppercase tracking-wide text-muted">{t("tools.whatsapp.resultTitle")}</p>
          <div className="flex items-center gap-2">
            <a
              href={displayUrl}
              target="_blank"
              rel="noreferrer"
              className="min-w-0 flex-1 truncate font-mono text-sm font-bold text-ink underline"
            >
              {displayUrl}
            </a>
            <CopyButton
              text={displayUrl}
              label={t("tools.whatsapp.copyLink")}
              className="shrink-0 rounded-full border-2 border-ink bg-flash-yellow px-4 py-2 text-xs font-bold text-ink transition-[filter] duration-150 hover:brightness-95"
            />
          </div>
          {result.short_url && result.short_url !== result.wa_link && (
            <p className="break-all font-mono text-xs text-muted">
              {t("tools.whatsapp.originalWaLink", { link: result.wa_link })}
            </p>
          )}
          <div className="flex flex-wrap items-center gap-2">
            <button
              type="button"
              onClick={addToBio}
              disabled={inBio || addingBio}
              className="rounded-full border-2 border-ink bg-print-white px-4 py-2 text-xs font-bold text-ink transition-colors duration-150 hover:bg-paper-grey disabled:cursor-not-allowed disabled:opacity-60"
            >
              {inBio
                ? t("tools.whatsapp.addedToBio")
                : addingBio
                  ? t("tools.whatsapp.addingBio")
                  : t("tools.whatsapp.addToBio")}
            </button>
            <span className="text-xs text-muted">{t("tools.whatsapp.openHint")}</span>
          </div>
          {bioError !== "" && <p className="text-xs font-medium text-[#FF5C3D]">{bioError}</p>}
        </div>
      )}
    </form>
  );
}
