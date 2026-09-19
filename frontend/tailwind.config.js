/** @type {import('tailwindcss').Config} */
module.exports = {
  // Content HARUS mencakup ./lib: kelas seperti bg-[#121212] (darkroom)
  // hidup sebagai string literals DI lib/themes.js — kalau lib tidak
  // di-scan, Tailwind JIT tidak pernah meng-generate kelas itu dan
  // "darkroom" tampil putih-di-atas-putih (bug picker label).
  content: ["./app/**/*.{js,jsx}", "./lib/**/*.{js,jsx}"],
  theme: {
    extend: {
      // DESIGN.md §1 — Instant Print tokens. Token lama (stage-black,
      // spot-white, laminate, laminate-lime, holo-pink) sudah dihapus.
      colors: {
        "print-white": "#FAFAF7",
        "paper-grey": "#EDEBE4",
        ink: "#1C1A12",
        muted: "#8A8578",
        "flash-yellow": "#FFD23F",
        "flash-coral": "#FF5C3D",
        "flash-orange": "#FF6B1A",
      },
      // DESIGN.md §2 — Space Grotesk (display), Work Sans (body),
      // JetBrains Mono (angka), Caveat (caption, dipakai SANGAT terbatas).
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
