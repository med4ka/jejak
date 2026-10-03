"use client";

// =====================================================================
// SHARED ANIMATION VARIANTS: framer-motion (fadeUp, staggerContainer,
// faq chevron/panel). Single source of truth for the landing page and other
// pages, following DESIGN.md: ease-out cubic-bezier(0.22,1,0.36,1), short
// durations 150-250ms, 60ms stagger: NO bounce, NO soft shadows, to keep
// the "instant print" feel.
//
// Consumed via <motion.* variants={...}> so each call site can re-wire
// (initial/animate/whileInView) without rewriting the same variants.
// =====================================================================

// EASE: 4-number array, NOT a CSS string. Motion (framer-motion / motion)
// accepts only 3 easing forms: a built-in name ("easeOut"), a 4-number array
// [x1, y1, x2, y2], or a function. A "cubic-bezier(...)" string →
// "Invalid easing type" error at runtime (see troubleshooting
// https://motion.dev/troubleshooting/invalid-easing-type).
export const EASE = [0.22, 1, 0.36, 1];

// =====================================================================
// SPRING: UI STATE (hover, tap, modal, dropdown) uses a natural spring,
// NOT a duration tween (Apple detail). EASE remains for scroll-reveal /
// entrance (a more "linear-print" feel there: see the EASE comment).
// =====================================================================
// SPRING: responsive, settles ~200ms: for buttons (hover/tap) & modals.
// damping 30 vs critical 2*sqrt(300*0.8)≈31 → ratio 0.97 ≈ critical with
// NO overshoot/oscillation (discipline: no bounce in UI state).
export const SPRING = { type: "spring", stiffness: 300, damping: 30, mass: 0.8 };
// SPRING_SOFT: softer, for card hover (rise -4px + shadow): feels
// "floating" without being harsh, still without meaningful overshoot.
export const SPRING_SOFT = { type: "spring", stiffness: 180, damping: 24 };

// Plain fade-up: for single elements (badge, headline, CTA, feature card,
// testimonial card, FAQ item). Small offset (12px) so it eases in rather
// than slides.
export const fadeUp = {
  hidden: { opacity: 0, y: 12 },
  show: {
    opacity: 1,
    y: 0,
    transition: { duration: 0.22, ease: EASE },
  },
};

// Wrapper container for staggering its children (hero text block, feature
// grid, statistic figures, testimonials). delayChildren 0 + stagger 0.06
// (60ms).
export const staggerContainer = {
  hidden: {},
  show: {
    transition: { delayChildren: 0, staggerChildren: 0.06 },
  },
};

// Item that "follows the parent": attached to each child of
// staggerContainer.
export const staggerItem = {
  hidden: { opacity: 0, y: 12 },
  show: {
    opacity: 1,
    y: 0,
    transition: { duration: 0.22, ease: EASE },
  },
};

// Scroll-reveal for feature cards (landing motion primitive #5): larger
// offset (30px) + SPRING_SOFT, 80ms stagger: used together with
// whileInView { once: true } so it does not re-animate when scrolling back
// and forth.
export const revealContainer = {
  hidden: {},
  show: { transition: { staggerChildren: 0.08 } },
};
export const revealItem = {
  hidden: { opacity: 0, y: 30 },
  show: { opacity: 1, y: 0, transition: SPRING_SOFT },
};

// FAQ chevron variant: rotates 180 degrees when opened, returns to 0 when
// closed. SPRING transition (dropdown/UI state → spring, Apple detail 1):
// not a 0.2 duration: the rotation then feels like it "opens by itself",
// without overshoot.
export const faqChevron = {
  hidden: { rotate: 0 },
  show: { rotate: 180, transition: SPRING },
};

// Default transition for cases that need one without variants (e.g. a
// useEffect on mount):
export const DEFAULT_TRANSITION = { duration: 0.22, ease: EASE };
