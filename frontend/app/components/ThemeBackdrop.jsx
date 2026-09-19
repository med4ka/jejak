"use client";

// Tema GLASS jadi versi LIGHT penuh (ala macOS Control Center mode terang):
// panggung = radial-gradient lembut abu-putih TERANG (#F5F5F5 → #E8E8E8),
// BUKAN foto noise/static kasar (Task 1 — backdrop foto asing dihapus).
// Card frosted PUTIH transparan + teks INK dipatok di themes.js.
//
// Grain halus: SVG feTurbulence noise, opacity 3-5% (di sini 4%), pointer-
// events-none, di-render di SEMUA tema (classic/darkroom/coral/glass) supaya
// semua background dapat tekstur lembut konsisten (Task 2) — bukan cuma
// tema foto. Grain ini tipis banget; background utama tetap dibaca.
//
// Render layered:
// 1. <div grain> — fixed inset-0 -z-10 pointer-events-none, background-image
//    SVG noise halus opacity 4%. Selalu ada untuk semua tema.
// 2. (glass saja) <div gradient> — radial-gradient terang fixed inset-0
//    -z-10 di bawah grain. Tema lain punya bg solid sendiri dari themes;
//    cukup grain-nya.
export default function ThemeBackdrop({ theme }) {
  const grain = {
    backgroundImage:
      "url(\"data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='160' height='160'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='2' stitchTiles='stitch'/%3E%3CfeColorMatrix type='saturate' values='0'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)' opacity='1'/%3E%3C/svg%3E\")",
  };

  return (
    <>
      {/* Grain halus untuk SEMUA tema — Task 2. */}
      <div
        aria-hidden="true"
        className="pointer-events-none fixed inset-0 -z-10"
        style={{ ...grain, opacity: 0.04 }}
      />
      {theme === "glass" && (
        <div
          aria-hidden="true"
          className="pointer-events-none fixed inset-0 -z-10"
          style={{
            background:
              "radial-gradient(120% 90% at 50% 0%, #F5F5F5 0%, #ECECEC 45%, #E0E0E0 100%)",
          }}
        />
      )}
    </>
  );
}
