"use client";

import { useEffect, useState } from "react";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
} from "recharts";
import { themeStyles } from "../../lib/themes";
import { useTranslation } from "../../lib/I18nProvider";
import { translateError } from "../../lib/errors";
import ChartTooltip from "./ChartTooltip";

// "2026-09-08" → "8 Sep" (short form: sufficient for a dense X axis).
// `months` are the translated month abbreviations (dashboard.chart.months.*)
// of the active locale.
function shortDate(iso, months) {
  const parts = iso.split("-");
  if (parts.length !== 3) {
    return iso;
  }
  return `${parseInt(parts[2], 10)} ${months[parseInt(parts[1], 10) - 1] || ""}`;
}

// Per-theme chart colors (previously hard-coded for the CLASSIC stage only:
// AXIS_TICK fill #8A8578 + grid #1C1A12 + bg-paper-grey card; as a result,
// when the dashboard followed a glass or darkroom theme, the card and the axis
// labels no longer matched the surrounding theme):
//   - classic/coral (light stage): soft INK axis labels #6E6A5E, thin INK
//     grid. Reads well on print-white / paper-grey cards.
//   - darkroom (dark stage):       soft PRINT-WHITE axis labels #B5B2A6, thin
//     PRINT-WHITE grid: contrast stays legible on a #121212 card.
//   - glass (now fully LIGHT):     soft INK axis labels #6E6A5E, thin INK
//     grid: legible on the frosted WHITE blurred card.
// The line color follows the theme as well: coral = brand orange accent, glass
// = grey (a silhouette over glass), classic & darkroom keep the brand flash
// yellow.
const CHART_COLORS = {
  classic: { axisTick: "#6E6A5E", grid: "#1C1A12", gridOpacity: 0.12, line: "#FFD23F" },
  darkroom: { axisTick: "#B5B2A6", grid: "#F5F5F0", gridOpacity: 0.16, line: "#FF6B1A" },
  coral: { axisTick: "#6E6A5E", grid: "#1C1A12", gridOpacity: 0.12, line: "#FF6B1A" },
  glass: { axisTick: "#6E6A5E", grid: "#1C1A12", gridOpacity: 0.14, line: "#9AA0A8" },
};

export default function ClicksChart({ theme = "classic", st, range = "30d" }) {
  const { t } = useTranslation();
  const [data, setData] = useState(null);
  const [error, setError] = useState("");
  const styles = st || themeStyles(theme);
  const cc = CHART_COLORS[theme] || CHART_COLORS.classic;
  const axisTick = { fontFamily: "var(--font-mono)", fontSize: 11, fill: cc.axisTick };
  // Translated month abbreviations for the X axis (same order as MONTHS in
  // the dictionary: Jan … Dec).
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
  // "7d"/"30d"/"90d" → 7/30/90 (so the heading and the empty-state copy are
  // honest about the window). The default 30 = the legacy endpoint called
  // without a parameter.
  const days = parseInt(range, 10) || 30;

  // Analytics-tab chart (depth analytics): a ranged timeseries endpoint, so
  // the heading and copy follow the active range. The LEGACY
  // /api/analytics/clicks-by-day endpoint is deliberately NOT used here: it
  // remains the source of the Summary MiniTrend (an unchanged 30-day window).
  useEffect(() => {
    let alive = true;
    setData(null); // range changed -> do not show figures from the old range
    setError("");
    fetch(`/api/analytics/timeseries?range=${encodeURIComponent(range)}`)
      .then(async (res) => {
        const text = await res.text();
        let parsed;
        try {
          parsed = JSON.parse(text);
        } catch {
          throw new Error(text || t("errors.analytics.chartLoad"));
        }
        if (!res.ok) {
          throw new Error(translateError(parsed, t, text || t("errors.analytics.chartLoad")));
        }
        if (alive) {
          setData(Array.isArray(parsed.days) ? parsed.days : []);
        }
      })
      .catch((err) => {
        if (alive) setError(err.message);
      });
    return () => {
      alive = false;
    };
  }, [range]);

  // Chart card follows the theme: radius, border width, border color,
  // background and text come from st.* of the active theme: never hard-coded
  // bg-paper-grey/ink.
  const cardClass = `${styles.radiusLarge} ${styles.borderW} ${styles.border} ${styles.card} ${styles.text} p-5 md:p-6`;

  return (
    <section className={cardClass}>
      <h2 className={`font-display text-lg font-bold mb-4 ${styles.text}`}>
        {t("dashboard.chart.heading", { days })}
      </h2>

      {error !== "" &&         <p className={`text-sm ${styles.textMuted}`}>{error}</p>}

      {error === "" && data === null && (
        <p className={`text-sm ${styles.textMuted}`}>{t("dashboard.chart.loading")}</p>
      )}

      {error === "" && data !== null && data.every((d) => d.count === 0) && (
        <p className={`text-sm ${styles.textMuted}`}>
          {t("dashboard.chart.empty", { days })}
        </p>
      )}

      {error === "" && data !== null && data.some((d) => d.count > 0) && (
        <div className="h-[220px] w-full">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={data} margin={{ top: 4, right: 8, bottom: 0, left: -12 }}>
              <CartesianGrid stroke={cc.grid} strokeOpacity={cc.gridOpacity} vertical={false} />
              {/* Hover tooltip: Instant Print card (see ChartTooltip.jsx).
                  cursor = vertical line tracking the active point (a vertical
                  curve in LineChart, in place of a separate ReferenceLine).
                  Fix for the card being covered by the cursor (Apple-style
                  detail #1):
                  - position y:-70 pins the card 70px above the chart (outside
                    the cursor area: the pointer never overlaps it); Recharts
                    treats position.y as an ABSOLUTE translate (not relative to
                    the point), so every hovered point yields the same result.
                  - offset 16 keeps a 16px horizontal gap from the cursor (x
                    still follows the cursor and flips automatically when it
                    leaves the right edge).
                  - allowEscapeViewBox y:true permits leaving the viewBox
                    vertically (a short plot area no longer forces the card to
                    move in / get cut off).
                  - wrapperStyle zIndex 100 + pointerEvents none keeps the card
                    above other elements without blocking clicks or hover. */}
              <Tooltip
                content={<ChartTooltip />}
                cursor={{ stroke: "#1C1A12", strokeOpacity: 0.35, strokeWidth: 1.5 }}
                isAnimationActive={false}
                position={{ y: -70 }}
                offset={16}
                allowEscapeViewBox={{ x: false, y: true }}
                wrapperStyle={{ zIndex: 100, pointerEvents: "none" }}
              />
              <XAxis
                dataKey="date"
                tickFormatter={(d) => shortDate(d, months)}
                tick={{ fontFamily: "var(--font-mono)", fontSize: 11, fill: cc.axisTick }}
                minTickGap={32}
                tickLine={false}
                axisLine={{ stroke: cc.grid, strokeOpacity: cc.gridOpacity }}
              />
              <YAxis
                allowDecimals={false}
                tick={{ fontFamily: "var(--font-mono)", fontSize: 11, fill: cc.axisTick }}
                tickLine={false}
                axisLine={false}
                width={36}
              />
              <Line
                type="monotone"
                dataKey="count"
                stroke={cc.line}
                strokeWidth={2.5}
                dot={false}
                activeDot={{ r: 4, strokeWidth: 2, stroke: "#1C1A12", fill: cc.line }}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>
      )}
    </section>
  );
}
