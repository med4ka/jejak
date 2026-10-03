"use client";

import { MotionConfig } from "framer-motion";

// Global motion provider: reducedMotion="user" → prefers-reduced-motion:
// reduce makes ALL framer-motion animations across the app collapse to
// opacity-only (transform/translate/scale are skipped, opacity still runs).
// MUST be a client component: context (MotionConfig) cannot be passed from
// a server component to client children, so layout.jsx (server) wraps
// Navbar + PageTransition through this wrapper.
export default function MotionProvider({ children }) {
  return <MotionConfig reducedMotion="user">{children}</MotionConfig>;
}
