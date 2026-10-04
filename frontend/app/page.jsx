"use client";

// =====================================================================
// LANDING PAGE: JEJAK (full rebuild: mobile-first wireframe, Stage C)
// =====================================================================
// Full rebuild per the agreed wireframe: sections ordered mobile→desktop
// (single column first, scaling to multi-column at sm/lg): 8 sections
// since the honest-copy audit of 2026-09-29 (the testimonial section was
// removed: fictional personas). Content stays STATIC (identical render on
// every pass: no hydration mismatch), BUT the hero uses a FUNCTIONAL
// ShortenForm (posts to /api/shorten through the frontend proxy):
// visitors can shorten a real link and see the result. The demo QR uses
// qrcode.react (client-side: instant, no network).
//
// DESIGN (DESIGN.md): the landing page is always in the "classic" theme:
// neutral and public, deliberately NOT following the signed-in creator's
// theme. Tokens come from the classic preset via themeStyles("classic")
// → st.*.
//
// MOTION: framer-motion (import "framer-motion": this repository uses that
// package name, not motion/react); shared variants live in
// lib/animations.js. LANDING MOTION: 7 primitives (2026-09-29):
//  1. hero headline word by word (y20+blur4 → clean, SPRING_SOFT, 60ms
//     stagger), then sub → form → social proof (explicit delays, ~1.2s)
//  2. mockup card: idle float y ±4px looping over 4s + ±3deg tilt following
//     the pointer (useMotionValue/useTransform with spring reset; NOT a
//     spotlight)
//  3. 2px flash-yellow scroll progress bar (useScroll + scaleX, z-[60])
//  4. "Gratis · Instan · Jujur" social proof, 60ms stagger (present and
//     verified: no numeric counter because the figures are not real data)
//  5. feature cards reveal y30 + SPRING_SOFT, 80ms stagger, once:true
//  6. FAQ panel + chevron already use SPRING (earlier iteration, verified)
//  7. CTA: section reveal + a button pulse fired once (scale 1→1.04→1)
//     800ms after entering the viewport, NOT a loop
// Reduced motion: the global MotionProvider (reducedMotion="user") strips
// transforms; the newer primitives additionally gate on useReducedMotion →
// a 150ms opacity crossfade without y/blur/scale.
// =====================================================================
import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import {
  motion,
  AnimatePresence,
  MotionConfig,
  useAnimationControls,
  useMotionValue,
  useReducedMotion,
  useScroll,
  useSpring,
  useTransform,
} from "framer-motion";
import {
  Camera,
  ChartColumnIncreasing,
  ChevronDown,
  Heart,
  Link2,
  MonitorSmartphone,
  Plus,
  QrCode,
} from "lucide-react";
import { QRCodeCanvas } from "qrcode.react";
import { themeStyles } from "../lib/themes";
import {
  EASE,
  SPRING,
  SPRING_SOFT,
  staggerContainer,
  staggerItem,
  revealContainer,
  revealItem,
  faqChevron,
} from "../lib/animations";
import ThemeBackdrop from "./components/ThemeBackdrop";
import ShortenForm from "./components/ShortenForm";
import CopyButton from "./components/CopyButton";
import { useTranslation } from "../lib/I18nProvider";

// motion.create(Link): the primary CTA button ("Daftar Gratis") gets spring
// gestures (hover -2px, tap scale 0.97 + y 0, Apple detail #4). In
// framer-motion v13 the old motion(Component) pattern is deprecated →
// motion.create.
const MotionLink = motion.create(Link);

// ===== DEMO DATA: static, identical render on every pass =====
const exampleProfile = {
  username: "kreator",
  name: "Kreator Digital",
  links: [
    { short_code: "youtube", original_url: "https://youtube.com/@kreator", click_count: 284 },
    { short_code: "shopee", original_url: "https://shopee.co.id/kreator", click_count: 156 },
    { short_code: "ig", original_url: "https://instagram.com/kreator", click_count: 98 },
    { short_code: "tiktok", original_url: "https://tiktok.com/@kreator", click_count: 210 },
  ],
};

export default function Home() {
  const st = themeStyles("classic");
  const [openFaq, setOpenFaq] = useState(null);
  const { t } = useTranslation();

  // ===== SECTION COPY: kept inside the component (module scope has no
  // t()); every entry below is the exact key path from messages/*.json. =====
  const platformPills = [
    { name: t("landing.socialProof.platforms.youtube"), cls: "bg-flash-coral" },
    { name: t("landing.socialProof.platforms.shopee"), cls: "bg-flash-orange" },
    { name: t("landing.socialProof.platforms.instagram"), cls: "bg-flash-yellow" },
    { name: t("landing.socialProof.platforms.tiktok"), cls: "bg-ink text-print-white" },
    { name: t("landing.socialProof.platforms.tokopedia"), cls: "bg-paper-grey" },
    { name: t("landing.socialProof.platforms.spotify"), cls: "bg-flash-coral" },
  ];

  // Values without quantitative claims (honest-copy audit, 2026-09-29): the
  // former figures ("500+ link", "1.2M+ klik", "300+ kreator") had no backend
  // source: there is no public aggregate endpoint, so they must not be
  // displayed. Each "figure" here is replaced by a product promise that can be
  // verified independently (FAQ + app page): free at this stage, one-step
  // shorten, real clicks.
  const stats = [
    { value: t("landing.socialProof.stats.free.value"), label: t("landing.socialProof.stats.free.label") },
    { value: t("landing.socialProof.stats.instant.value"), label: t("landing.socialProof.stats.instant.label") },
    { value: t("landing.socialProof.stats.honest.value"), label: t("landing.socialProof.stats.honest.label") },
  ];

  const features = [
    {
      icon: Link2,
      title: t("landing.features.items.customSlug.title"),
      desc: t("landing.features.items.customSlug.desc"),
    },
    {
      icon: MonitorSmartphone,
      title: t("landing.features.items.smartLink.title"),
      desc: t("landing.features.items.smartLink.desc"),
    },
    {
      icon: QrCode,
      title: t("landing.features.items.qrCode.title"),
      desc: t("landing.features.items.qrCode.desc"),
    },
    {
      icon: ChartColumnIncreasing,
      title: t("landing.features.items.analytics.title"),
      desc: t("landing.features.items.analytics.desc"),
    },
  ];

  // Testimonial section REMOVED (honest-copy audit, 2026-09-29): the quotes
  // plus their attributed names/roles were fictional personas without a
  // source: they must not be presented as real customers. Restore this
  // section ONLY once genuine, permissioned testimonials exist (see
  // PROGRESS.md: remaining TODO).

  const faqs = [
    {
      q: t("landing.faq.items.comparison.q"),
      a: t("landing.faq.items.comparison.a"),
    },
    {
      q: t("landing.faq.items.free.q"),
      a: t("landing.faq.items.free.a"),
    },
    {
      q: t("landing.faq.items.customSlug.q"),
      a: t("landing.faq.items.customSlug.a"),
    },
    {
      q: t("landing.faq.items.analytics.q"),
      a: t("landing.faq.items.analytics.a"),
    },
  ];

  // ===== Reduced-motion gate =====
  // The global MotionProvider already strips transforms (y/rotate/scale);
  // the animations remaining here are forced into a ≤150ms opacity crossfade
  // without y/blur: including the blur filter that reducedMotion="user"
  // does NOT remove (blur is not a transform property).
  const reduce = useReducedMotion();
  const fade150 = {
    hidden: { opacity: 0 },
    show: { opacity: 1, transition: { duration: 0.15 } },
  };

  // ===== Primitive #1: hero sequencing schedule =====
  // badge 0 → headline words start at 50ms (60ms stagger × 5 words ≈ 0.3s +
  // SPRING_SOFT settle) → sub 0.55s → form 0.8s → social proof 1.0s
  // → ~1.2s total from mount.
  const HERO = { badge: 0, words: 0.05, sub: 0.55, form: 0.8, proof: 1.0 };
  const word = reduce
    ? fade150
    : {
        hidden: { opacity: 0, y: 20, filter: "blur(4px)" },
        show: { opacity: 1, y: 0, filter: "blur(0px)", transition: SPRING_SOFT },
      };
  const heroWords = {
    hidden: {},
    show: { transition: { delayChildren: HERO.words, staggerChildren: 0.06 } },
  };
  // The headline keeps its per-word motion.span split (stagger above); the
  // words are taken from the dictionary: line 1 / line 2 split on "\n",
  // then each line split on " " (id/en/de: 2 words + 3 words).
  const [headlineLine1, headlineLine2] = t("landing.hero.headline").split("\n");
  const [hw1, hw2] = headlineLine1.split(" ");
  const [hw3, hw4, hw5] = headlineLine2.split(" ");
  const stage = (delay) =>
    reduce
      ? {
          hidden: { opacity: 0 },
          show: { opacity: 1, transition: { duration: 0.15, delay } },
        }
      : {
          hidden: { opacity: 0, y: 8 },
          show: { opacity: 1, y: 0, transition: { ...SPRING_SOFT, delay } },
        };
  // Feature cards under reduced motion: drop y30, keep a 150ms crossfade.
  const cardItem = reduce ? fade150 : revealItem;

  // ===== Primitive #3: scroll progress (2px bar above the navbar) =====
  const { scrollYProgress } = useScroll();

  // ===== Primitive #2: mockup card tilt =====
  // Pointer position (-0.5..0.5) → ±3deg rotation driven by a spring (a
  // smooth reset to 0 once the pointer leaves). Transform-only, mounted only
  // when !reduce.
  const tiltX = useMotionValue(0);
  const tiltY = useMotionValue(0);
  const springCfg = {
    stiffness: SPRING.stiffness,
    damping: SPRING.damping,
    mass: SPRING.mass,
  };
  const rotY = useSpring(useTransform(tiltX, [-0.5, 0.5], [-3, 3]), springCfg);
  const rotX = useSpring(useTransform(tiltY, [-0.5, 0.5], [3, -3]), springCfg);
  const handleTiltMove = (e) => {
    const r = e.currentTarget.getBoundingClientRect();
    tiltX.set((e.clientX - r.left) / r.width - 0.5);
    tiltY.set((e.clientY - r.top) / r.height - 0.5);
  };
  const handleTiltLeave = () => {
    tiltX.set(0);
    tiltY.set(0);
  };

  // ===== Primitive #7: one-shot CTA button pulse =====
  const ctaPulse = useAnimationControls();
  const pulsedRef = useRef(false);
  const pulseTimer = useRef(null);
  useEffect(() => () => window.clearTimeout(pulseTimer.current), []);
  const handleCtaEnter = () => {
    if (pulsedRef.current || reduce) return;
    pulsedRef.current = true;
    // keyframes and springs cannot be combined in framer (keyframes run as
    // tweens) → two consecutive springs: up to 1.04, then back to 1, still
    // fired only once.
    pulseTimer.current = window.setTimeout(() => {
      ctaPulse
        .start({ scale: 1.04, transition: SPRING })
        .then(() => ctaPulse.start({ scale: 1, transition: SPRING }));
    }, 800);
  };

  // Footer copyright carries the {heart} inline node: split the translated
  // string on the placeholder token and reinsert <Heart/> between the parts.
  const [copyrightBeforeHeart, copyrightAfterHeart] = t("footer.copyright", {
    year: new Date().getFullYear(),
  }).split("{heart}");

  return (
    <MotionConfig reducedMotion="user">
    <main className={`min-h-screen ${st.page}`}>
      <ThemeBackdrop />

      {/* ===== PRIMITIVE #3: scroll progress bar =====
          A 2px flash-yellow line at the very top of the viewport, stacked
          above the fixed navbar (z-50). Width = scrollProgress via scaleX
          with a left origin: transform only, filling 0% → 100% from the top
          to the bottom of the scroll. Scroll-linked rather than an autonomous
          animation, so it keeps running under reduced motion. */}
      <motion.div
        aria-hidden="true"
        style={{ scaleX: scrollYProgress, transformOrigin: "left" }}
        className="fixed left-0 top-0 z-[60] h-[2px] w-full bg-flash-yellow"
      />

      {/* ===== HERO: single column on mobile (text→form→social proof→
          mockup), two columns on desktop (form left, profile mockup right)
          ===== */}
      {/* z-10: the hero content sits above ThemeBackdrop (-z-10) and below
          the fixed navbar (z-50). The padding-top compensates for the height
          of the navbar pill fixed at top-5: pt-24 on mobile,
          pt-28/lg:pt-32 on desktop so the badge is never covered. */}
      <section className="relative z-10 mx-auto max-w-6xl px-4 pb-16 pt-24 md:pt-28 lg:pt-32 sm:px-6 lg:px-8">
        <div className="grid grid-cols-1 items-center gap-12 lg:grid-cols-2 lg:gap-16">
          {/* left column: badge → headline (word by word) → subheadline →
              FUNCTIONAL ShortenForm → mini social proof.
              PRIMITIVE #1: stages use explicit delays rather than a parent
              stagger so the subheadline follows only AFTER the headline has
              finished. */}
          <motion.div
            variants={{ hidden: {}, show: {} }}
            initial="hidden"
            animate="show"
            className="text-center lg:text-left"
          >
            <motion.span
              variants={stage(HERO.badge)}
              className={`inline-block ${st.chip} ${st.radiusFull} ${st.borderW} ${st.border} px-3 py-1 font-mono text-xs font-bold uppercase tracking-[0.15em]`}
            >
              {t("landing.hero.badge")}
            </motion.span>

            {/* Headline per word: y20 + blur(4px) → clean, SPRING_SOFT, 60ms
                stagger. The copy is NOT modified: it is only split into one
                span per word (inline-block; the spaces between spans are the
                original spaces). */}
            <motion.h1
              variants={heroWords}
              className={`mt-6 ${st.headingFont} text-4xl font-bold leading-tight sm:text-5xl`}
            >
              <motion.span variants={word} className="inline-block">{hw1}</motion.span>{" "}
              <motion.span variants={word} className="inline-block">{hw2}</motion.span>
              <br />
              <motion.span variants={word} className="inline-block">{hw3}</motion.span>{" "}
              <motion.span variants={word} className="inline-block">{hw4}</motion.span>{" "}
              <motion.span variants={word} className="inline-block">{hw5}</motion.span>
            </motion.h1>

            <motion.p
              variants={stage(HERO.sub)}
              className={`mt-4 max-w-xl text-base ${st.textMuted}`}
            >
              {t("landing.hero.subheadline")}
            </motion.p>

            {/* The hero form is live, not decorative: ShortenForm posts to
                /api/shorten. */}
            <motion.div variants={stage(HERO.form)} className="mt-8">
              <ShortenForm st={st} />
            </motion.div>

            {/* mini social proof: HONEST: no fabricated figures or avatars.
                The ShortenForm above does work without an account (anonymous
                links are claimed after sign-up), so this claim can be
                verified directly by visitors. */}
            <motion.div
              variants={stage(HERO.proof)}
              className="mt-6 flex items-center justify-center gap-2 lg:justify-start"
            >
              <span
                className={`flex h-6 w-6 shrink-0 items-center justify-center ${st.accent} ${st.radiusFull} ${st.borderW} ${st.border}`}
                aria-hidden="true"
              >
                <Link2 className="h-3.5 w-3.5" strokeWidth={2.5} />
              </span>
              <p className={`text-xs ${st.textMuted}`}>
                {t("landing.hero.proof")}
              </p>
            </motion.div>
          </motion.div>

          {/* right column: public creator profile mockup: HONEST MOCKUP: an
              ordinary product preview card (not a fake browser bar or phone
              frame), labelled "contoh" so the data inside (clicks
              284/210/...) is not read as real metrics. */}
          <motion.div
            initial={{ opacity: 0, y: 12 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.25, ease: EASE, delay: 0.16 }}
            className="w-full"
          >
            {/* PRIMITIVE #2: float + tilt on THIS CARD ONLY (not the page):
                idle float y ±4px looping 4s ease-in-out; ±3deg tilt following
                the pointer through motion values (rotate kept separate from y
                so the two do not fight over the transform; no spotlight or
                cursor gradient). Under reduced motion the handlers are not
                attached and MotionProvider strips the float. */}
            <motion.div
              animate={{ y: [0, -4, 0, 4, 0] }}
              transition={{ duration: 4, ease: "easeInOut", repeat: Infinity }}
              style={{ rotateX: rotX, rotateY: rotY, transformPerspective: 600 }}
              onMouseMove={reduce ? undefined : handleTiltMove}
              onMouseLeave={reduce ? undefined : handleTiltLeave}
              className={`w-full ${st.card} ${st.borderW} ${st.border} ${st.radiusLarge} border-2 p-6`}
            >
            <div className="flex items-center gap-4">
              <div
                className={`relative flex h-14 w-14 items-center justify-center ${st.avatar} ${st.radiusFull} ${st.borderW} ${st.border} border-2 text-xl font-bold`}
              >
                <Camera
                  className={`absolute -bottom-1 -right-1 h-4 w-4 ${st.radiusFull} bg-print-white p-0.5`}
                  strokeWidth={2.5}
                />
                K
              </div>
              <div>
                <p className={`text-xs ${st.textMuted}`}>@{exampleProfile.username}</p>
                <p className={`${st.headingFont} text-lg font-bold`}>{exampleProfile.name}</p>
              </div>
              <span
                className={`ml-auto shrink-0 ${st.radiusFull} border-2 border-ink bg-paper-grey px-2 py-0.5 font-mono text-[10px] font-bold uppercase tracking-widest text-ink`}
              >
                {t("landing.mockup.badge")}
              </span>
            </div>

            <div className="mt-5 flex flex-col gap-2.5">
              {exampleProfile.links.map((l) => (
                <div
                  key={l.short_code}
                  className={`flex items-center justify-between gap-3 border-2 ${st.borderW} ${st.border} ${st.radius} ${st.card} px-3.5 py-2.5`}
                >
                  <div>
                    <p className="font-mono text-xs font-bold">r/{l.short_code}</p>
                    <p className={`truncate text-[11px] ${st.textMuted}`}>{l.original_url}</p>
                  </div>
                  <span className="font-mono text-xs font-bold">{l.click_count}</span>
                </div>
              ))}
            </div>

            <p className={`mt-3 flex items-center justify-center gap-2 text-center font-mono text-[10px] uppercase tracking-[0.2em] ${st.textMuted}`}>
              {/* The profile page URL uses the real /u/{username} format (not
                  @username, which has no route). */}
              jejak.app/u/{exampleProfile.username}
              <CopyButton
                text={`https://jejak.app/u/${exampleProfile.username}`}
                label={t("landing.mockup.copyButton")}
                className="relative shrink-0 rounded-full border-2 border-ink bg-print-white px-2 py-0.5 text-[10px] font-bold normal-case tracking-normal text-ink transition-colors duration-150 hover:bg-paper-grey/60 after:absolute after:-inset-x-4 after:-inset-y-4 after:content-['']"
              />
            </p>
            </motion.div>
          </motion.div>
        </div>
      </section>

      {/* ===== SOCIAL PROOF: platform row + 3 product values ===== */}
      <section className={`border-t-2 ${st.border}`}>
        <div className="mx-auto max-w-6xl px-4 py-12 sm:px-6 lg:px-8">
          {/* Honest caption: the pills below are PLATFORMS a link can point
              to, not customers or testimonial givers: hence no "Trusted
              by ..." caption, which would be an unsourced claim. */}
          <p className={`text-center text-xs ${st.textMuted}`}>
            {t("landing.socialProof.caption")}
          </p>

          {/* STATIC pill row (motion audit, 2026-09-29): the previous
              infinite marquee was purely decorative: dropping it loses no
              information, and the row wraps onto the next line on narrow
              screens. */}
          <div className="mt-6 flex flex-wrap items-center justify-center gap-3">
            {platformPills.map((b) => (
              <span
                key={b.name}
                className={`flex h-9 shrink-0 items-center px-4 font-mono text-sm font-bold uppercase tracking-wide ${b.cls} ${st.borderW} ${st.border} border-2`}
              >
                {b.name}
              </span>
            ))}
          </div>

          {/* 3 product values (not fabricated statistics): staggered;
              stacked in a single column on mobile so 4-6 letter words at
              text-3xl do not clog a 3-column grid at 320px. */}
          <motion.div
            variants={staggerContainer}
            initial="hidden"
            whileInView="show"
            viewport={{ once: true, amount: 0.3 }}
            className="mt-10 grid grid-cols-1 gap-6 text-center sm:grid-cols-3"
          >
            {stats.map((s) => (
              <motion.div key={s.label} variants={staggerItem}>
                <p className={`${st.headingFont} text-3xl font-bold sm:text-4xl`}>{s.value}</p>
                <p className={`mt-1 text-xs ${st.textMuted}`}>{s.label}</p>
              </motion.div>
            ))}
          </motion.div>
        </div>
      </section>

      {/* ===== KEY FEATURES: 4-card grid ===== */}
      {/* scroll-mt compensates for the fixed navbar so the #fitur anchor does
          not land hidden beneath the navbar pill. */}
      <section id="fitur" className={`scroll-mt-24 md:scroll-mt-28 border-t-2 ${st.border}`}>
        <div className="mx-auto max-w-6xl px-4 py-14 sm:px-6 lg:px-8">
          <h2 className={`${st.headingFont} text-2xl font-bold ${st.heading}`}>
            {t("landing.features.heading")}
          </h2>
          {/* PRIMITIVE #5: staggered scroll reveal: y30 → 0, SPRING_SOFT,
              80ms stagger, once:true (no re-animation when scrolling back and
              forth). */}
          <motion.div
            variants={revealContainer}
            initial="hidden"
            whileInView="show"
            viewport={{ once: true, amount: 0.3 }}
            className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4"
          >
            {features.map((f) => (
              <motion.div
                key={f.title}
                variants={cardItem}
                whileHover={{
                  y: -4,
                  boxShadow: "6px 6px 0 #1C1A12",
                  transition: SPRING_SOFT,
                }}
                className={`${st.card} ${st.shadow} ${st.borderW} ${st.border} ${st.radiusLarge} border-2 p-5`}
              >
                <div className={`flex h-10 w-10 items-center justify-center ${st.accent} ${st.radius}`}>
                  <f.icon className="h-5 w-5" strokeWidth={2.5} />
                </div>
                <h3 className={`mt-3 ${st.headingFont} text-base font-bold`}>{f.title}</h3>
                <p className={`mt-1 text-sm ${st.textMuted}`}>{f.desc}</p>
              </motion.div>
            ))}
          </motion.div>
        </div>
      </section>

      {/* ===== INTERACTIVE DEMO: slug → preview + realtime QR (no network)
          ===== */}
      {/* id="demo": the navbar anchor target. The slug input inside carries a
          different id ("demo-slug"): the two must not be conflated. */}
      <section id="demo" className={`scroll-mt-24 md:scroll-mt-28 border-t-2 ${st.border}`}>
        <div className="mx-auto max-w-3xl px-4 py-14 sm:px-6 lg:px-8">
          <h2 className={`${st.headingFont} text-2xl font-bold ${st.heading}`}>
            {t("landing.demo.heading")}
          </h2>
          <DemoInteractive st={st} />
        </div>
      </section>

      {/* ===== FAQ: 4-item accordion ===== */}
      <section id="faq" className={`scroll-mt-24 md:scroll-mt-28 border-t-2 ${st.border}`}>
        <div className="mx-auto max-w-3xl px-4 py-14 sm:px-6 lg:px-8">
          <h2 className={`${st.headingFont} text-2xl font-bold ${st.heading}`}>
            {t("landing.faq.heading")}
          </h2>
          <div className={`mt-6 ${st.card} ${st.radiusLarge} ${st.borderW} ${st.border} border-2`}>
            {faqs.map((f) => (
              <div key={f.q} className={`border-b-2 ${st.border} last:border-b-0`}>
                <button
                  type="button"
                  onClick={() => setOpenFaq(openFaq === f.q ? null : f.q)}
                  aria-expanded={openFaq === f.q}
                  aria-controls={`faq-panel-${f.q}`}
                  className="flex w-full items-center justify-between gap-4 px-5 py-4 text-left"
                >
                  <span className={`${st.headingFont} text-sm font-bold`}>{f.q}</span>
                  <motion.span
                    variants={faqChevron}
                    initial="hidden"
                    animate={openFaq === f.q ? "show" : "hidden"}
                    className={`shrink-0 ${st.textMuted}`}
                  >
                    <ChevronDown className="h-4 w-4" strokeWidth={2.5} />
                  </motion.span>
                </button>
                <AnimatePresence initial={false}>
                  {openFaq === f.q && (
                    <motion.div
                      key="panel"
                      id={`faq-panel-${f.q}`}
                      initial={{ height: 0, opacity: 0 }}
                      animate={{ height: "auto", opacity: 1 }}
                      exit={{ height: 0, opacity: 0 }}
                      transition={SPRING}
                      className="overflow-hidden"
                    >
                      <p className={`px-5 pb-5 text-sm leading-relaxed ${st.textMuted}`}>{f.a}</p>
                    </motion.div>
                  )}
                </AnimatePresence>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* ===== ACCENT BLOCK: solid flash-yellow, full width ===== */}
      {/* The former quantitative claim (the "500+ link" monthly figure) was
          removed: it had no source. The block now states a product promise
          the FAQ can substantiate ("free at this stage, no card"). */}
      <section className={`border-t-2 ${st.border}`}>
        <div className={`${st.accent} ${st.border}`}>
          <div className="mx-auto max-w-3xl px-6 py-20 text-center">
            <p className={`${st.headingFont} font-bold leading-tight`}>
              <span className="block text-4xl sm:text-5xl">{t("landing.accent.title")}</span>
              <span className="mt-1 block text-xl sm:text-2xl">
                {t("landing.accent.subtitle")}
              </span>
            </p>
            <p className={`mt-4 inline-block border-t-2 px-2 pt-2 font-mono text-xs font-bold uppercase tracking-[0.2em]`}>
              {t("landing.accent.eyebrow")}
            </p>
          </div>
        </div>
      </section>

      {/* ===== CLOSING CTA: PRIMITIVE #7 =====
          The section fades + y8 → 0 when it enters the viewport (once); the
          button inside pulses ONCE (scale 1 → 1.04 → 1, two SPRING springs)
          800ms after the section enters: NOT a loop. Under reduced motion
          the pulse is skipped and the section becomes a 150ms crossfade. */}
      <motion.section
        className={`border-t-2 ${st.border}`}
        initial={{ opacity: 0, y: 8 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true, amount: 0.3 }}
        transition={reduce ? { duration: 0.15 } : SPRING_SOFT}
        onViewportEnter={handleCtaEnter}
      >
        <div className="mx-auto max-w-3xl px-6 py-16 text-center">
          <h2 className={`${st.headingFont} text-2xl font-bold ${st.heading}`}>
            {t("landing.cta.heading")}
          </h2>
          <p className={`mt-3 max-w-md mx-auto text-base leading-relaxed ${st.textMuted}`}>
            {t("landing.cta.subtext")}
          </p>
          {/* Primary button: SPRING gestures (hover -2px, tap 0.97 + y 0):
              Apple detail #4. animate={ctaPulse} = primitive #7 (one-shot
              pulse); motion.create(Link) is declared at module scope. */}
          <MotionLink
            href="/app"
            animate={ctaPulse}
            whileHover={{ y: -2, transition: SPRING }}
            whileTap={{ scale: 0.97, y: 0, transition: SPRING }}
            className={`mt-8 inline-flex items-center gap-2 ${st.accent} ${st.radiusFull} ${st.borderW} ${st.border} px-7 py-3 text-sm font-bold`}
          >
            {t("landing.cta.button")}
            <Plus className="h-4 w-4" strokeWidth={2.5} />
          </MotionLink>
        </div>
      </motion.section>

      {/* ===== FOOTER: brand block + 3 link columns ===== */}
      <footer className={`border-t-2 ${st.border}`}>
        <div className="mx-auto max-w-6xl px-4 py-12 sm:px-6 lg:px-8">
          <div className="grid grid-cols-2 gap-8 md:grid-cols-4">
            <div className="col-span-2 md:col-span-1">
              <p className={`${st.headingFont} text-lg font-bold`}>
                {t("footer.brand")}<span className={`${st.accent} px-1`}>.</span>
              </p>
              <p className={`mt-2 text-sm ${st.textMuted}`}>
                {t("footer.tagline")}
              </p>
            </div>
            <FooterCol
              title={t("footer.columnTitles.product")}
              links={[
                { label: t("footer.links.shorten"), href: "/app" },
                { label: t("footer.links.linkInBio"), href: "/app" },
                { label: t("footer.links.qrCode"), href: "/#demo" },
              ]}
              st={st}
            />
            <FooterCol
              title={t("footer.columnTitles.company")}
              links={[{ label: t("footer.links.contact"), href: "/contact" }]}
              st={st}
            />
            <FooterCol
              title={t("footer.columnTitles.legal")}
              links={[
                { label: t("footer.links.privacy"), href: "/privacy" },
                { label: t("footer.links.terms"), href: "/terms" },
              ]}
              st={st}
            />
          </div>
          <p className={`mt-10 border-t-2 ${st.border} pt-5 text-center font-mono text-xs ${st.textMuted}`}>
            {copyrightBeforeHeart}
            <Heart
              className="inline h-3.5 w-3.5 -translate-y-px fill-flash-coral text-flash-coral"
              size={14}
              aria-hidden="true"
            />
            {copyrightAfterHeart}
          </p>
        </div>
      </footer>
    </main>
    </MotionConfig>
  );
}

// =====================================================================
// Interactive demo: type a slug → link preview + realtime QR (no network).
// No fetch: the QR code is generated client-side from the slug value.
// =====================================================================
function DemoInteractive({ st }) {
  const { t } = useTranslation();
  const [slug, setSlug] = useState("nama-kamu");
  // The real server short URL: jejak.app/r/{code} (the /r/ route is the
  // redirect API; a slug + domain without /r/ is the /u/ profile page: not
  // a short link).
  const display = `jejak.app/r/${slug.trim() === "" ? "nama-kamu" : slug.trim()}`;

  return (
    <div className={`mt-6 grid grid-cols-1 gap-6 md:grid-cols-2 ${st.card} ${st.borderW} ${st.border} ${st.radiusLarge} border-2 p-6`}>
      <div>
        <label
          htmlFor="demo-slug"
          className={`text-xs ${st.headingFont} font-bold uppercase tracking-[0.15em]`}
        >
          {t("landing.demo.slugLabel")}
        </label>
        <div className={`mt-2 flex items-center ${st.borderW} ${st.border} ${st.radius} overflow-hidden border-2`}>
          <span className={`px-3 py-2.5 font-mono text-xs font-bold ${st.accent}`}>jejak.app/r/</span>
          <input
            id="demo-slug"
            type="text"
            value={slug}
            maxLength={30}
            onChange={(e) => setSlug(e.target.value)}
            placeholder={t("landing.demo.slugPlaceholder")}
            className={`w-full min-w-0 px-3 py-2.5 font-mono text-sm font-bold ${st.card} focus:border-flash-yellow focus:ring-2 focus:ring-flash-yellow/30 focus:outline-hidden transition-all duration-150`}
          />
        </div>
        <p className={`mt-3 font-mono text-sm font-bold ${st.text}`}>{display}</p>
        <p className={`mt-1 text-xs ${st.textMuted}`}>
          {t("landing.demo.previewNote")}
        </p>
      </div>

      <div className={`flex items-center justify-center ${st.borderW} ${st.border} ${st.radius} border-2 bg-print-white py-6`}>
        <QRCodeCanvas value={`https://${display}`} size={140} level="M" />
      </div>
    </div>
  );
}

// =====================================================================
// Footer column: heading + link list. Used for the three link columns at
// the bottom of the footer.
// =====================================================================
function FooterCol({ title, links, st }) {
  return (
    <div>
      <h4 className={`${st.headingFont} text-sm font-bold`}>{title}</h4>
      <ul className="mt-3 flex flex-col gap-2">
        {links.map((l) => (
          <li key={l.label}>
            <Link href={l.href} className={`text-sm ${st.textMuted} hover:underline`}>
              {l.label}
            </Link>
          </li>
        ))}
      </ul>
    </div>
  );
}
