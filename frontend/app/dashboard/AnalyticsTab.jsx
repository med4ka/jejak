"use client";

import { useEffect, useState } from "react";
import { Download } from "lucide-react";
import { useTranslation } from "../../lib/I18nProvider";
import ClicksChart from "../components/ClicksChart";

// The offered ranges are exactly the ranges the BACKEND allows (parseRange
// rejects anything else with a 400). Offering a button the server cannot
// execute only produces errors: hence this list is deliberately duplicated
// here rather than derived from a single configurable source.

// Horizontal breakdown card: label + count + a bar showing its share of the
// range total. The bar uses the theme border/card tokens (st.*) so it
// follows the dashboard's Instant Print style: no hardcoded colors outside
// the brand accents.
function BreakdownCard({ title, items, total, labels, st }) {
  const { t } = useTranslation();
  return (
    <section className={`${st.radiusLarge} ${st.borderW} ${st.border} ${st.card} ${st.text} p-5 md:p-6`}>
      <h2 className={`${st.headingFont} mb-4 text-lg font-bold`}>{title}</h2>

      {!items ? (
        <p className={`text-sm ${st.textMuted}`}>{t("dashboard.loading.general")}</p>
      ) : items.length === 0 ? (
        <p className={`text-sm ${st.textMuted}`}>{t("dashboard.emptyStates.noClicksInRange")}</p>
      ) : (
        <ul className="flex flex-col gap-3">
          {items.map((it) => {
            const pct = total > 0 ? Math.round((it.count / total) * 100) : 0;
            return (
              <li key={it.key}>
                <div className="flex items-baseline justify-between gap-3">
                  <span className="truncate font-mono text-sm font-bold">
                    {labels[it.key] || it.key}
                  </span>
                  <span className="shrink-0 font-mono text-sm">
                    {it.count}
                    <span className={`ml-1 ${st.textMuted}`}>{pct}%</span>
                  </span>
                </div>
                <div
                  className={`mt-1.5 h-3 w-full overflow-hidden ${st.borderW} ${st.border} rounded-full`}
                  aria-hidden="true"
                >
                  <div
                    className="h-full rounded-full bg-flash-yellow transition-[width] duration-300"
                    style={{ width: `${pct}%` }}
                  />
                </div>
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}

// Analytics tab (depth analytics, 2026-09-30): range pills + time-series
// chart + 2 breakdown cards (device, referrer) + CSV export. The range is
// HELD by DashboardClient (deviation E: ?range= in the URL through
// history.replaceState) so a refresh does not throw the visitor back to
// 30 days and an "open in new tab" link carries the context.
export default function AnalyticsTab({ st, range, onRange }) {
  const { t } = useTranslation();
  const RANGES = [
    { id: "7d", label: t("dashboard.analytics.ranges.last7Days") },
    { id: "30d", label: t("dashboard.analytics.ranges.last30Days") },
    { id: "90d", label: t("dashboard.analytics.ranges.last90Days") },
  ];

  // Translated labels for the backend buckets. Unknown keys are still
  // displayed as-is (forward-compatible: a new column in the database does
  // not produce a mysteriously empty card).
  const DEVICE_LABELS = {
    desktop: t("dashboard.analytics.devices.desktop"),
    mobile: t("dashboard.analytics.devices.mobile"),
    tablet: t("dashboard.analytics.devices.tablet"),
    bot: t("dashboard.analytics.devices.bot"),
    unknown: t("dashboard.analytics.devices.unknown"),
  };

  const REFERRER_LABELS = {
    direct: t("dashboard.analytics.referrers.direct"),
    search: t("dashboard.analytics.referrers.search"),
    social: t("dashboard.analytics.referrers.social"),
    chat: t("dashboard.analytics.referrers.chat"),
    other: t("dashboard.analytics.referrers.other"),
  };

  const EXPORTS = [
    { mode: "daily", label: t("dashboard.analytics.exports.daily") },
    { mode: "links", label: t("dashboard.analytics.exports.perLink") },
    { mode: "clicks", label: t("dashboard.analytics.exports.rawClicks") },
  ];

  const [device, setDevice] = useState(null);
  const [referrer, setReferrer] = useState(null);
  const [dataPer, setDataPer] = useState("");
  const [error, setError] = useState("");

  // Both breakdowns are fetched together: they share the same range, so
  // their loading state matches (no left card left empty while the right one
  // is already full because a single request was slow).
  useEffect(() => {
    let alive = true;
    setDevice(null);
    setReferrer(null);
    setError("");

    const load = (kind, setter) =>
      fetch(`/api/analytics/breakdown?kind=${kind}&range=${encodeURIComponent(range)}`)
        .then(async (res) => {
          const text = await res.text();
          let parsed;
          try {
            parsed = JSON.parse(text);
          } catch {
            throw new Error(text || t("errors.analytics.breakdownLoad"));
          }
          if (!res.ok) {
            throw new Error(parsed.error || t("errors.analytics.breakdownLoad"));
          }
          if (alive) {
            setter(Array.isArray(parsed.items) ? parsed.items : []);
            setDataPer(parsed.data_per || "");
          }
          return parsed;
        });

    Promise.all([load("device", setDevice), load("referrer", setReferrer)]).catch((err) => {
      if (alive) setError(err.message);
    });

    return () => {
      alive = false;
    };
  }, [range]);

  const deviceTotal = (device || []).reduce((sum, it) => sum + (Number(it.count) || 0), 0);
  const referrerTotal = (referrer || []).reduce((sum, it) => sum + (Number(it.count) || 0), 0);

  return (
    <>
      {/* Range control: a pill row reusing the dashboard tab-bar pattern so
          the control reads as native to the rest of the dashboard. */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap gap-2">
          {RANGES.map((r) => (
            <button
              key={r.id}
              type="button"
              onClick={() => onRange(r.id)}
              aria-pressed={range === r.id}
              className={`rounded-full border-2 border-ink px-4 py-2 text-sm transition-colors duration-150 ${
                range === r.id
                  ? "bg-flash-yellow font-bold text-ink"
                  : "bg-white text-ink/60 hover:text-ink"
              }`}
            >
              {r.label}
            </button>
          ))}
        </div>
        {dataPer !== "" && (
          <p className={`text-xs ${st.textMuted}`}>
            {t("dashboard.analytics.sourceOfData", { dataPer })}
          </p>
        )}
      </div>

      {error !== "" && <p className={`text-sm ${st.textMuted}`}>{error}</p>}

      <ClicksChart st={st} range={range} />

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <BreakdownCard
          title={t("dashboard.analytics.deviceTitle")}
          items={device}
          total={deviceTotal}
          labels={DEVICE_LABELS}
          st={st}
        />
        <BreakdownCard
          title={t("dashboard.analytics.referrerTitle")}
          items={referrer}
          total={referrerTotal}
          labels={REFERRER_LABELS}
          st={st}
        />
      </div>

      <section className={`${st.radiusLarge} ${st.borderW} ${st.border} ${st.card} ${st.text} p-5 md:p-6`}>
        <h2 className={`${st.headingFont} mb-1 text-lg font-bold`}>{t("dashboard.analytics.exportTitle")}</h2>
        <p className={`mb-4 text-sm ${st.textMuted}`}>
          {t("dashboard.analytics.exportDescription", {
            rangeLabel: RANGES.find((r) => r.id === range)?.label || range,
          })}
        </p>
        <div className="flex flex-wrap gap-2">
          {EXPORTS.map((m) => (
            <a
              key={m.mode}
              href={`/api/analytics/export?mode=${m.mode}&range=${encodeURIComponent(range)}`}
              download
              className={`inline-flex shrink-0 items-center gap-1.5 rounded-full border-2 border-ink bg-white px-4 py-2 text-sm font-bold text-ink transition-transform duration-150 hover:-translate-y-0.5 focus:outline-hidden focus-visible:ring-2 focus-visible:ring-flash-yellow`}
            >
              <Download size={14} aria-hidden="true" />
              {m.label}
            </a>
          ))}
        </div>
      </section>
    </>
  );
}
