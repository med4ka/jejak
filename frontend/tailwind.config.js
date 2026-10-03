/** @type {import('tailwindcss').Config} */
module.exports = {
  // Content MUST include ./lib: classes like bg-[#121212] (darkroom) live as
  // string literals INSIDE lib/themes.js - if lib is not scanned, Tailwind
  // JIT never generates those classes and "darkroom" renders white-on-white
  // (the picker label bug).
  content: ["./app/**/*.{js,jsx}", "./lib/**/*.{js,jsx}"],
  theme: {
    extend: {
      // DESIGN.md §1 - Instant Print tokens. Legacy tokens (stage-black,
      // spot-white, laminate, laminate-lime, holo-pink) have been removed.
      colors: {
        "print-white": "#FAFAF7",
        "paper-grey": "#EDEBE4",
        ink: "#1C1A12",
        muted: "#8A8578",
        "flash-yellow": "#FFD23F",
        "flash-coral": "#FF5C3D",
        "flash-orange": "#FF6B1A",
      },
      // DESIGN.md §2 - Space Grotesk (display), Work Sans (body),
      // JetBrains Mono (numerals), Caveat (caption, used VERY sparingly).
      fontFamily: {
        display: ["var(--font-display)", "sans-serif"],
        blackhead: ["var(--font-blackhead)", "sans-serif"],
        body: ["var(--font-body)", "sans-serif"],
        mono: ["var(--font-mono)", "monospace"],
        caption: ["var(--font-caption)", "cursive"],
      },
    },
  },
  plugins: [],
};
