"use client";

import { useEffect, useState } from "react";
import { motion, useMotionValue, useTransform, animate } from "framer-motion";
import {
  Link as LinkIcon,
  MousePointerClick,
  Users,
  TrendingUp,
  TrendingDown,
} from "lucide-react";
import {
  LineChart,
  Line,
  ResponsiveContainer,
  Tooltip,
} from "recharts";
import { SPRING_SOFT } from "../../lib/animations";
import { useTranslation } from "../../lib/I18nProvider";
import { translateError } from "../../lib/errors";
import { ShareButton } from "../components/ShareModal";
import OnboardingChecklist from "../components/OnboardingChecklist";
import ChartTooltip from "../components/ChartTooltip";
import Skeleton from "../components/Skeleton";

// Stat number formatting: values ≥1000 render as "1.2K" (not "1,247":
// dashboard figures are not an invoice table: short yet still readable).
// Below 1000 → round normally.
function formatStat(v) {
  const n = Math.round(v);
  if (Math.abs(n) >= 1000) {
    return `${(n / 1000).toFixed(1)}K`;
  }
  return `${n}`;
}

// The figure counts up from 0 → target on mount: SPRING ~800ms (Apple
// detail #2: a spring, not a duration tween). bounce 0 → damping ratio 1
// (critical), no overshoot: the number never "overshoots" and reverses.
// A value of 0 → no animation (never animate "0 → 0"); set the display
// directly.
function AnimatedNumber({ value, prefix = "" }) {
  const mv = useMotionValue(0);
  const display = useTransform(mv, (v) => `${prefix}${formatStat(v)}`);

  useEffect(() => {
    if (value === 0) {
      mv.set(0);
      return undefined;
    }
    const controls = animate(mv, value, {
      type: "spring",
      duration: 0.8,
      bounce: 0,
    });
    return () => controls.stop();
  }, [value, mv]);

  return <motion.span>{display}</motion.span>;
}

function StatCard({ icon: Icon, value, label, loading, valueClass = "", iconClass = "bg-flash-yellow/20 text-ink", prefix = "" }) {
  return (
    // Stat card: consistent hover lift (Apple detail #4): y -4px with the
    // shadow growing 4px → 6px (the hard 4px 4px 0 base shadow is already
    // present), SPRING_SOFT. The transition lives inside the gesture object
    // so it does not override the entrance/stagger transition of the other
    // cards.
    <motion.div
      whileHover={{
        y: -4,
        boxShadow: "6px 6px 0 #1C1A12",
        transition: SPRING_SOFT,
      }}
      className="rounded-xl border-2 border-ink bg-white p-5 shadow-[4px_4px_0px_#1C1A12]"
    >
      <div className={`mb-3 flex h-10 w-10 items-center justify-center rounded-lg ${iconClass}`}>
        <Icon className="h-5 w-5" aria-hidden="true" />
      </div>
      {loading ? (
        <Skeleton className="mb-1 h-8 w-16" />
      ) : (
        <p className={`mb-1 font-mono text-2xl font-bold md:text-3xl ${valueClass}`}>
          <AnimatedNumber value={value} prefix={prefix} />
        </p>
      )}
      <p className="text-sm text-ink/60">{label}</p>
    </motion.div>
  );
}

// Mini chart: same data as ClicksChart, different visual: a plain line,
// 120px tall, without detailed axes/grid. The full chart remains in the
// Analytics tab.
function MiniTrend({ data, loading }) {
  const { t } = useTranslation();
  if (loading) {
    return <Skeleton className="h-[120px] w-full" />;
  }
  if (!data || data.length === 0 || data.every((d) => d.count === 0)) {
    return (
      <p className="text-sm text-ink/60">
        {t("dashboard.emptyStates.noClicks30d")}
      </p>
    );
  }
  return (
    <div className="h-[120px] w-full">
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={data} margin={{ top: 8, right: 4, bottom: 4, left: 4 }}>
          {/* Same as the Analytics chart: Instant Print tooltip plus a
              vertical line on hover (cursor). The mini chart has no axes:
              dates are still read from the tooltip. Cursor-occlusion fix: the
              card is pinned above the chart, outside the mouse area (position
              y), offset 16, allowEscapeViewBox y:true (the narrow area must
              not clip the card), zIndex 100 + pointerEvents none. */}
          <Tooltip
            content={<ChartTooltip />}
            cursor={{ stroke: "#1C1A12", strokeOpacity: 0.35, strokeWidth: 1.5 }}
            isAnimationActive={false}
            position={{ y: -30 }}
            offset={16}
            allowEscapeViewBox={{ x: false, y: true }}
            wrapperStyle={{ zIndex: 100, pointerEvents: "none" }}
          />
          <Line
            type="monotone"
            dataKey="count"
            stroke="#FFD23F"
            strokeWidth={2.5}
            dot={false}
            activeDot={{ r: 4, strokeWidth: 2, stroke: "#1C1A12", fill: "#FFD23F" }}
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}

export default function RingkasanTab({ displayName, links, profileReady, onShare, onGoProfil }) {
  const { t } = useTranslation();
  const [days, setDays] = useState(null);
  const [summary, setSummary] = useState(null);

  useEffect(() => {
    fetch("/api/analytics/clicks-by-day")
      .then(async (res) => {
        const text = await res.text();
        let parsed = [];
        try {
          parsed = JSON.parse(text);
        } catch {
          throw new Error(text || t("errors.analytics.trendLoad"));
        }
        if (!res.ok) {
          throw new Error(translateError(parsed, t, text || t("errors.analytics.trendLoad")));
        }
        setDays(Array.isArray(parsed) ? parsed : []);
      })
      .catch(() => setDays([]));

    fetch("/api/analytics/summary")
      .then(async (res) => {
        const text = await res.text();
        let parsed;
        try {
          parsed = JSON.parse(text);
        } catch {
          throw new Error(text || t("errors.analytics.summaryLoad"));
        }
        if (!res.ok) {
          throw new Error(translateError(parsed, t, text || t("errors.analytics.summaryLoad")));
        }
        setSummary(parsed);
      })
      .catch(() => setSummary(null));
  }, []);

  const loading = !profileReady || summary === null;

  // The 4 stat cards come from the summary endpoint (total active links,
  // total clicks, 30-day uniques, growth of the last 30 days vs the
  // previous 30): they fall back to 0 while loading or after an error.
  const totalActive = summary?.total_links ?? 0;
  const totalClicks = summary?.total_clicks ?? 0;
  const unique30 = summary?.unique_clicks_30d ?? 0;
  const growth = summary?.growth_pct ?? 0;

  const top5 = [...links]
    .sort((a, b) => (Number(b.click_count) || 0) - (Number(a.click_count) || 0))
    .slice(0, 5);

  const growthPrefix = growth > 0 ? "+" : growth < 0 ? "" : "+";
  const growthClass =
    growth > 0 ? "text-[#22C55E]" : growth < 0 ? "text-flash-coral" : "text-ink";

  return (
    <>
      <header className="mb-6 flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <h2
            className="text-xl font-bold md:text-2xl"
            style={{ fontFamily: "var(--font-display), sans-serif" }}
          >
            {displayName
              ? t("dashboard.ringkasan.greeting", { displayName })
              : t("dashboard.ringkasan.greetingFallback")}
          </h2>
          <p className="mt-1 text-sm text-ink/60">
            {t("dashboard.ringkasan.subtitle")}
          </p>
        </div>
        <ShareButton onClick={onShare} />
      </header>

      {/* New-user onboarding: above the stat cards; it disappears
          automatically once every step is complete or it is dismissed
          (localStorage). The step 2 CTA jumps to the Profil tab, step 3 opens
          the ShareModal (props from DashboardClient). */}
      <OnboardingChecklist onGoProfil={onGoProfil} onShare={onShare} />

      <div className="grid grid-cols-1 gap-3 md:grid-cols-2 md:gap-4 lg:grid-cols-4">
        <StatCard
          icon={LinkIcon}
          value={totalActive}
          label={t("dashboard.ringkasan.stats.totalLinks")}
          loading={loading}
        />
        <StatCard
          icon={MousePointerClick}
          value={totalClicks}
          label={t("dashboard.ringkasan.stats.totalClicks")}
          loading={loading}
        />
        <StatCard
          icon={Users}
          value={unique30}
          label={t("dashboard.ringkasan.stats.uniqueClicks30d")}
          loading={loading}
        />
        <StatCard
          icon={growth >= 0 ? TrendingUp : TrendingDown}
          value={growth}
          label={t("dashboard.ringkasan.stats.growthPct")}
          loading={loading}
          prefix={growthPrefix}
          valueClass={growthClass}
          iconClass={
            growth > 0
              ? "bg-[#22C55E]/20 text-[#22C55E]"
              : growth < 0
                ? "bg-flash-coral/20 text-flash-coral"
                : "bg-flash-yellow/20 text-ink"
          }
        />
      </div>

      <section className="rounded-xl border-2 border-ink bg-white p-5 shadow-[4px_4px_0px_#1C1A12]">
        <h3 className="mb-4 text-lg font-bold" style={{ fontFamily: "var(--font-display), sans-serif" }}>
          {t("dashboard.ringkasan.trendTitle")}
        </h3>
        <MiniTrend data={days} loading={days === null} />
      </section>

      <section>
        <h3 className="mb-4 text-lg font-bold" style={{ fontFamily: "var(--font-display), sans-serif" }}>
          {t("dashboard.ringkasan.topLinksTitle")}
        </h3>
        {loading ? (
          <div className="flex flex-col gap-3" aria-hidden="true">
            {[0, 1, 2, 3, 4].map((i) => (
              <Skeleton key={i} className="h-14 w-full" />
            ))}
          </div>
        ) : top5.length === 0 ? (
          <p className="text-sm text-ink/60">{t("dashboard.emptyStates.noTopLinks")}</p>
        ) : (
          <ol className="flex flex-col gap-3">
            {top5.map((l, i) => (
              <li
                key={l.short_code}
                className="flex items-center gap-3 rounded-xl border-2 border-ink bg-white p-4 shadow-[4px_4px_0px_#1C1A12]"
              >
                <span className="font-mono text-sm font-bold text-ink/40" aria-hidden="true">
                  {String(i + 1).padStart(2, "0")}
                </span>
                <div className="min-w-0 flex-1">
                  <p className="truncate font-mono text-sm font-bold">/{l.short_code}</p>
                  <p className="truncate text-xs text-ink/60">{l.original_url}</p>
                </div>
                <span className="shrink-0 font-mono text-sm font-bold">
                  {t("dashboard.ringkasan.topLinks.clickCount", {
                    count: Number(l.click_count) || 0,
                  })}
                </span>
              </li>
            ))}
          </ol>
        )}
      </section>
    </>
  );
}
