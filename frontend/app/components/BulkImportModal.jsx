"use client";

import { useEffect, useMemo, useState } from "react";
import { motion } from "framer-motion";
import SubmitButton from "./SubmitButton";
import CopyButton from "./CopyButton";
import { SPRING } from "../../lib/animations";
import { sameOriginShortUrl } from "../../lib/shortlink";
import { useTranslation } from "../../lib/I18nProvider";
import { translateError } from "../../lib/errors";

const input =
  "w-full rounded-xl border-2 border-ink bg-print-white px-3 py-2 text-sm text-ink placeholder:text-muted focus:border-flash-yellow focus:ring-2 focus:ring-flash-yellow/30 focus:outline-hidden transition-all duration-150";

// Bulk import modal (link management, 2026-09-30): paste 10-100 URLs (one per
// line) → POST /api/links/bulk. The per-line ✓/✗ preview is computed on the
// CLIENT (valid URL + duplicate) for instant feedback; the server still
// re-validates (100KB cap, 10-100 range, validRemoteURL) and has the final
// say: a 201 response carries created[] + errors[] per line. The modal shell
// follows EditLinkModal (backdrop + SPRING + Escape).
export default function BulkImportModal({ st, onClose, onImported }) {
  const { t } = useTranslation();
  const [text, setText] = useState("");
  const [tagInput, setTagInput] = useState("");
  const [result, setResult] = useState(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    function onKey(e) {
      if (e.key === "Escape") {
        onClose();
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  // Preview for each non-empty line: valid / not a URL / duplicate. Empty
  // lines are skipped rather than treated as errors (pasting from a
  // spreadsheet routinely leaves blank lines). The rejection reason is kept
  // as a message KEY here and translated only at render time.
  const preview = useMemo(() => {
    const seen = new Set();
    const out = [];
    text.split(/\r?\n/).forEach((raw, i) => {
      const url = raw.trim();
      if (url === "") return;
      let ok = true;
      let reasonKey = "";
      try {
        const u = new URL(url);
        if (u.protocol !== "http:" && u.protocol !== "https:") {
          ok = false;
          reasonKey = "errors.bulkImport.reasonNotHttp";
        }
      } catch {
        ok = false;
        reasonKey = "errors.bulkImport.reasonNotUrl";
      }
      if (ok && seen.has(url)) {
        ok = false;
        reasonKey = "errors.bulkImport.reasonDuplicate";
      }
      if (ok) seen.add(url);
      out.push({ line: i + 1, url, ok, reasonKey });
    });
    return out;
  }, [text]);

  const totalLines = preview.length; // entri (non-kosong): server pakai ini
  const validCount = preview.filter((p) => p.ok).length;
  const canSubmit = !loading && totalLines >= 10 && totalLines <= 100 && validCount > 0;

  async function onSubmit(e) {
    e.preventDefault();
    setError("");
    setResult(null);
    setLoading(true);
    try {
      // Send ALL non-empty entries (not only the valid ones): server line
      // numbers then align with `preview`, so errors can be pinned to their
      // original line.
      const urls = preview.map((p) => p.url);
      const res = await fetch("/api/links/bulk", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ urls, tag: tagInput.trim() === "" ? undefined : tagInput.trim() }),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        // A 400 from the server still carries errors[] when available: show
        // them in detail.
        if (Array.isArray(data.errors) && data.errors.length > 0) {
          setResult({ ...data, created: data.created || [] });
          return;
        }
        throw new Error(translateError(data, t, t("errors.bulkImport.importFailed")));
      }
      setResult(data);
      // The list is NOT refreshed here: the import result is shown first;
      // "Buka Link Saya" is what calls onImported (refresh + modal close).
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }

  const created = result && Array.isArray(result.created) ? result.created : [];
  const failed = result && Array.isArray(result.errors) ? result.errors : [];
  const done = result !== null && result.summary && result.summary.created > 0;

  // ORIGINAL textarea line number for one response entry: the server counts
  // array index+1, and the preview maps that onto non-empty line numbers.
  function realLine(lineNo) {
    const p = preview[lineNo - 1];
    return p ? p.line : lineNo;
  }

  return (
    <div className="pointer-events-none fixed inset-0 z-[60] flex items-end justify-center sm:items-start sm:px-4 sm:pt-[10vh]">
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        transition={SPRING}
        onClick={onClose}
        className="pointer-events-auto absolute inset-0 bg-ink/60"
        aria-hidden="true"
      />
      <motion.div
        role="dialog"
        aria-modal="true"
        aria-label={t("forms.bulkImport.title")}
        initial={{ opacity: 0, y: 24, scale: 0.98 }}
        animate={{ opacity: 1, y: 0, scale: 1 }}
        exit={{ opacity: 0, y: 12, scale: 0.98 }}
        transition={SPRING}
        className="pointer-events-auto relative max-h-[85vh] w-full overflow-y-auto rounded-t-2xl border-2 border-ink bg-print-white p-6 text-ink sm:max-w-[560px] sm:rounded-2xl"
      >
        <p className="font-mono text-sm font-bold">{t("forms.bulkImport.title")}</p>
        <p className="mt-1 text-sm text-muted">
          {t("forms.bulkImport.description")}
        </p>

        <form onSubmit={onSubmit} className="mt-4 flex flex-col gap-3">
          <label className="text-sm font-medium">
            {t("forms.bulkImport.urlLabel")}
            <textarea
              value={text}
              onChange={(e) => setText(e.target.value)}
              placeholder={t("forms.bulkImport.urlPlaceholder")}
              rows={8}
              className={`${input} min-h-[300px] resize-y font-mono`}
            />
          </label>
          <div className="flex items-center justify-between text-xs text-muted">
            <span>
              {t("forms.bulkImport.totalLinesCount", { totalLines })}
              {totalLines > 100 && (
                <span className="font-bold text-flash-coral">{t("forms.bulkImport.maxWarning")}</span>
              )}
              {totalLines > 0 && totalLines < 10 && (
                <span className="font-bold text-flash-coral">{t("forms.bulkImport.minWarning")}</span>
              )}
            </span>
            <span>{t("forms.bulkImport.validCount", { validCount })}</span>
          </div>

          {/* Per-line ✓/✗ preview: feedback before submitting (the server
              still re-validates; this is UX, not the final verdict). */}
          {preview.length > 0 && (
            <ul className="max-h-[160px] overflow-y-auto rounded-xl border-2 border-ink bg-paper-grey/60 p-2 text-xs">
              {preview.map((p) => (
                <li key={p.line} className="flex items-start gap-2 py-0.5 font-mono">
                  <span className={`w-4 shrink-0 font-bold ${p.ok ? "text-ink" : "text-flash-coral"}`}>
                    {p.ok ? "✓" : "✗"}
                  </span>
                  <span className="min-w-0 flex-1 break-all">
                    {p.url}
                    {!p.ok && <span className="ml-1 text-flash-coral">({t(p.reasonKey)})</span>}
                  </span>
                  <span className="shrink-0 text-muted">{t("forms.bulkImport.lineLabel", { line: p.line })}</span>
                </li>
              ))}
            </ul>
          )}

          <label className="text-sm font-medium">
            {t("forms.bulkImport.tagLabel")}
            <input
              value={tagInput}
              onChange={(e) => setTagInput(e.target.value)}
              placeholder={t("forms.bulkImport.tagPlaceholder")}
              maxLength={20}
              className={`${input} mt-1`}
            />
          </label>

          <div className="flex gap-2">
            <SubmitButton
              isLoading={loading}
              loadingLabel={t("forms.bulkImport.loadingLabel")}
              disabled={!canSubmit}
              className={`rounded-full border-2 border-ink ${st.accent} px-4 py-2.5 text-sm font-bold transition-[filter] duration-150 hover:brightness-95 disabled:cursor-not-allowed disabled:opacity-50`}
            >
              {validCount > 0
                ? t("forms.bulkImport.submitLabelWithCount", { validCount })
                : t("forms.bulkImport.submitLabel")}
            </SubmitButton>
            <button
              type="button"
              onClick={onClose}
              className="rounded-full border-2 border-ink bg-white px-4 py-2.5 text-sm font-bold text-ink transition-transform duration-150 hover:-translate-y-0.5"
            >
              {t("common.cancel")}
            </button>
          </div>
        </form>

        {error !== "" && (
          <p className="mt-3 rounded-xl border-2 border-ink bg-paper-grey px-4 py-2.5 text-sm font-medium text-ink">{error}</p>
        )}

        {result !== null && (
          <div className="mt-4 flex flex-col gap-3">
            <p className="text-sm font-bold">
              {result.summary
                ? t("forms.bulkImport.resultSummary", {
                    created: result.summary.created,
                    failed: result.summary.failed,
                  })
                : t("forms.bulkImport.done")}
            </p>
            {created.length > 0 && (
              <ul className="flex flex-col gap-2">
                {created.map((c) => {
                  // The API's short_url is built from the backend Host and still
                  // reads "http://localhost:8081/r/…": it is forced to the
                  // current origin so the href/copy actions in this modal point
                  // at the Next proxy.
                  const shortUrl = sameOriginShortUrl(c.short_url, c.short_code);
                  return (
                    <li
                      key={c.short_code}
                      className="flex items-center gap-2 rounded-xl border-2 border-ink bg-print-white px-3 py-2"
                    >
                      <a
                        href={shortUrl}
                        target="_blank"
                        rel="noreferrer"
                        className="min-w-0 flex-1 truncate font-mono text-xs underline"
                      >
                        {shortUrl}
                      </a>
                      <CopyButton
                        text={shortUrl}
                        label={t("common.copy")}
                        className="shrink-0 rounded-full border-2 border-ink bg-white px-3 py-1 text-xs font-bold"
                      />
                    </li>
                  );
                })}
              </ul>
            )}
            {failed.length > 0 && (
              <ul className="flex flex-col gap-1 rounded-xl border-2 border-flash-coral bg-flash-coral/10 px-3 py-2 text-xs">
                {failed.map((f, i) => {
                  // The row template carries {url} as a token; the URL itself
                  // stays JSX (break-all span): split on the token and
                  // reinsert the node in the middle.
                  const parts = t("forms.bulkImport.failedRow", {
                    line: realLine(f.line),
                    error: f.error,
                  }).split("{url}");
                  const beforeUrl = parts[0];
                  const afterUrl = parts.length > 1 ? parts.slice(1).join("{url}") : "";
                  return (
                    <li key={i} className="font-mono">
                      {beforeUrl}
                      <span className="break-all">{f.url}</span>
                      {afterUrl}
                    </li>
                  );
                })}
              </ul>
            )}
            <button
              type="button"
              onClick={onImported}
              className="rounded-full border-2 border-ink bg-flash-yellow px-4 py-2 text-sm font-bold text-ink transition-transform duration-150 hover:-translate-y-0.5"
            >
              {t("forms.bulkImport.openMyLinks")}
            </button>
          </div>
        )}
        {result !== null && !done && created.length === 0 && failed.length === 0 && (
          <p className="mt-3 text-sm text-muted">{t("forms.bulkImport.emptyResult")}</p>
        )}
      </motion.div>
    </div>
  );
}
