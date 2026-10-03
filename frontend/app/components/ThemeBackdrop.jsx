"use client";

// Subtle grain for ALL themes: SVG feTurbulence noise, opacity 3-5% (4% here),
// pointer-events-none, fixed -z-10 beneath the content: a consistent soft
// texture that never covers the main background.
//
// Redesign note 2026-09-29: the GLASS-specific radial-gradient backdrop was
// REMOVED: glass paper is now a linear-gradient (Apple-style spec) rendered
// directly in globals.css through body[data-profile-theme="glass"], the single
// source of truth for paper shared with the solid themes. This component is
// therefore purely texture and no longer takes a theme prop.
export default function ThemeBackdrop() {
  const grain = {
    backgroundImage:
      "url(\"data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='160' height='160'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='2' stitchTiles='stitch'/%3E%3CfeColorMatrix type='saturate' values='0'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)' opacity='1'/%3E%3C/svg%3E\")",
  };

  return (
    <div
      aria-hidden="true"
      className="pointer-events-none fixed inset-0 -z-10"
      style={{ ...grain, opacity: 0.04 }}
    />
  );
}
