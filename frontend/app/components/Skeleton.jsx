"use client";

// Loading placeholder with a moving shimmer (Apple-style detail #6).
// ink/8 base, a soft highlight band travelling left → right, 1.5s loop
// (@keyframes shimmer lives in app/globals.css: a CSS animation, not framer).
// 8px rounding by default; height/width are supplied through className (h-6
// for text, h-8 for numbers, h-14 for rows, etc.).
// aria-hidden: purely decorative: assistive technology gets the loading
// status from the surrounding context, not from this empty placeholder.
export default function Skeleton({ className = "" }) {
  return <div aria-hidden="true" className={`skeleton ${className}`} />;
}
