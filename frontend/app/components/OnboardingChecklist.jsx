"use client";

import { useEffect, useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import Link from "next/link";
import { Sparkles, Check, X, ArrowRight } from "lucide-react";
import { SPRING } from "../../lib/animations";
import { useTranslation } from "../../lib/I18nProvider";

// =====================================================================
// ONBOARDING CHECKLIST: guidance for new users on the dashboard Summary tab.
// 3 steps + progress bar; shown only while a step is incomplete and the
// checklist has not been dismissed. State detection:
//   1. total_links >= 1  → GET /api/analytics/summary (field total_links)
//   2. bio & display_name != "" → GET /api/profile
//   3. localStorage `jejak_shared=true` → set in DashboardClient.openShare
//      (every ShareModal trigger goes through that single gate).
// Permanent dismissal: localStorage `jejak_onboarding_dismissed=true`.
// The initial render is hidden (open=false) → localStorage reads and fetches
// run inside a useEffect → no flash for accounts that already finished or
// dismissed the checklist, and no hydration mismatch (localStorage is never
// read during SSR).
// =====================================================================

const STEPS = [
  { id: "link", labelKey: "onboarding.steps.link.label", ctaKey: "onboarding.steps.link.cta" },
  { id: "profil", labelKey: "onboarding.steps.profile.label", ctaKey: "onboarding.steps.profile.cta" },
  { id: "share", labelKey: "onboarding.steps.share.label", ctaKey: "onboarding.steps.share.cta" },
];

function CircleCheck({ done }) {
  if (done) {
    return (
      <span
        aria-hidden="true"
        className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-flash-yellow"
      >
        <Check className="h-3.5 w-3.5 text-ink" strokeWidth={3} />
      </span>
    );
  }
  return (
    <span
      aria-hidden="true"
      className="h-5 w-5 shrink-0 rounded-full border-2 border-ink bg-white"
    />
  );
}

export default function OnboardingChecklist({ onGoProfil, onShare }) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const [dismissed, setDismissed] = useState(false);
  // null = still loading; [bool, bool, bool] after resolution.
  const [flags, setFlags] = useState(null);

  useEffect(() => {
    if (localStorage.getItem("jejak_onboarding_dismissed") === "true") {
      setDismissed(true);
      return;
    }
    let alive = true;
    Promise.all([
      fetch("/api/analytics/summary")
        .then((r) => (r.ok ? r.json() : null))
        .catch(() => null),
      fetch("/api/profile")
        .then((r) => (r.ok ? r.json() : null))
        .catch(() => null),
    ])
      .then(([summary, profile]) => {
        if (!alive) {
          return;
        }
        // Both requests failed → the data is unknown, so do not show steps
        // that might be wrong (better absent than misleading).
        if (summary === null && profile === null) {
          return;
        }
        const next = [
          Number(summary?.total_links ?? 0) >= 1,
          Boolean(profile?.bio) && Boolean(profile?.display_name),
          localStorage.getItem("jejak_shared") === "true",
        ];
        setFlags(next);
        if (!next.every(Boolean)) {
          setOpen(true);
        }
      });
    return () => {
      alive = false;
    };
  }, []);

  function dismiss() {
    localStorage.setItem("jejak_onboarding_dismissed", "true");
    setDismissed(true);
    setOpen(false);
  }

  const doneCount = flags ? flags.filter(Boolean).length : 0;
  const allDone = flags !== null && flags.every(Boolean);
  const show = open && !dismissed && !allDone;

  function handleCta(stepId) {
    if (stepId === "profil") {
      onGoProfil?.();
    } else if (stepId === "share") {
      onShare?.();
    }
  }

  return (
    <AnimatePresence>
      {show && (
        <motion.section
          aria-label={t("onboarding.regionLabel")}
          initial={{ opacity: 0, y: 8 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0, y: 8 }}
          transition={SPRING}
          className="rounded-xl border-2 border-ink bg-flash-yellow/10 p-5 shadow-[4px_4px_0_#1C1A12]"
        >
          {/* Header: icon + title + dismiss (X) button on the right. */}
          <div className="flex items-center gap-2">
            <Sparkles className="h-5 w-5 shrink-0 text-ink" aria-hidden="true" />
            <h2 className="min-w-0 flex-1 text-base font-bold text-ink md:text-lg">
              {t("onboarding.title")}
            </h2>
            <button
              type="button"
              onClick={dismiss}
              aria-label={t("onboarding.closeLabel")}
              className="-mr-1 flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-ink/60 transition-colors duration-150 hover:bg-ink/10 hover:text-ink"
            >
              <X className="h-4 w-4" aria-hidden="true" />
            </button>
          </div>

          {/* Progress bar: ink/10 track, flash-yellow fill, width animated
              with SPRING. */}
          <div
            role="progressbar"
            aria-valuemin={0}
            aria-valuemax={3}
            aria-valuenow={doneCount}
            aria-label={t("onboarding.progressLabel")}
            className="mt-3 h-2 overflow-hidden rounded-full bg-ink/10"
          >
            <motion.div
              className="h-full rounded-full bg-flash-yellow"
              initial={{ width: 0 }}
              animate={{ width: `${(doneCount / 3) * 100}%` }}
              transition={SPRING}
            />
          </div>

          <ol className="mt-4 flex flex-col gap-3">
            {STEPS.map((s, i) => {
              const done = Boolean(flags?.[i]);
              return (
                <li key={s.id} className="flex items-center gap-3">
                  <CircleCheck done={done} />
                  <span
                    className={`min-w-0 flex-1 text-sm text-ink ${
                      done ? "line-through opacity-60" : ""
                    }`}
                  >
                    {t(s.labelKey)}
                  </span>
                  {!done &&
                    (s.id === "link" ? (
                      <Link
                        href="/app"
                        className="inline-flex shrink-0 items-center gap-1 rounded-full border-2 border-ink bg-flash-yellow px-3 py-1 text-xs font-bold text-ink transition-transform duration-150 hover:-translate-y-0.5"
                      >
                        {t(s.ctaKey)}
                        <ArrowRight className="h-3 w-3" strokeWidth={3} aria-hidden="true" />
                      </Link>
                    ) : (
                      <button
                        type="button"
                        onClick={() => handleCta(s.id)}
                        className="inline-flex shrink-0 items-center gap-1 rounded-full border-2 border-ink bg-flash-yellow px-3 py-1 text-xs font-bold text-ink transition-transform duration-150 hover:-translate-y-0.5"
                      >
                        {t(s.ctaKey)}
                        <ArrowRight className="h-3 w-3" strokeWidth={3} aria-hidden="true" />
                      </button>
                    ))}
                </li>
              );
            })}
          </ol>
        </motion.section>
      )}
    </AnimatePresence>
  );
}
