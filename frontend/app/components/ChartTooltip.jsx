"use client";

// Custom Recharts tooltip for the /dashboard charts (Summary + Analytics).
// Instant Print styling per specification: bg-white card, border-2 ink, hard
// 4px shadow, 8px radius (rounded-lg), p-2.
// Fonts: the date label uses Work Sans (inherited from body/font-body), the
// click NUMBER uses JetBrains Mono (font-mono): do not swap them.
//
// Date format "17 Sep 2026" (no comma) derived from the ISO "2026-09-17"
// sent by the /api/analytics/clicks-by-day endpoint. The month abbreviations
// come from the caller (dashboard.chart.months.* of the active locale).

import { useTranslation } from "../../lib/I18nProvider";

export function formatChartDate(iso, months) {
  const parts = String(iso ?? "").split("-");
  if (parts.length !== 3) return String(iso ?? "");
  const day = parseInt(parts[2], 10);
  const month = months[parseInt(parts[1], 10) - 1];
  if (!month || Number.isNaN(day)) return String(iso);
  return `${day} ${month} ${parts[0]}`;
}

export default function ChartTooltip({ active, payload, label }) {
  const { t } = useTranslation();
  if (!active || !payload || payload.length === 0) {
    return null;
  }
  // Translated month abbreviations, Jan … Dec (same order as the dictionary
  // dashboard.chart.months).
  const months = [
    t("dashboard.chart.months.jan"),
    t("dashboard.chart.months.feb"),
    t("dashboard.chart.months.mar"),
    t("dashboard.chart.months.apr"),
    t("dashboard.chart.months.may"),
    t("dashboard.chart.months.jun"),
    t("dashboard.chart.months.jul"),
    t("dashboard.chart.months.aug"),
    t("dashboard.chart.months.sep"),
    t("dashboard.chart.months.oct"),
    t("dashboard.chart.months.nov"),
    t("dashboard.chart.months.dec"),
  ];
  // `label` from Recharts is the active X-axis value. The Summary MiniTrend
  // has no <XAxis>, so label can be undefined; fall back to the `date` field
  // of the active datum so the date still renders.
  const date = label ?? payload[0]?.payload?.date;
  return (
    <div className="rounded-lg border-2 border-ink bg-white p-2 shadow-[4px_4px_0px_#1C1A12]">
      <p className="text-xs font-medium text-ink/70">{formatChartDate(date, months)}</p>
      <p className="font-mono text-sm font-bold text-ink">
        {t("dashboard.chart.tooltipClickCount", { count: payload[0].value })}
      </p>
    </div>
  );
}
