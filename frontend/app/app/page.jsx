"use client";

import Link from "next/link";
import { themeStyles } from "../../lib/themes";
import ThemeBackdrop from "../components/ThemeBackdrop";
import ShortenForm from "../components/ShortenForm";

// /app page: content IDENTICAL to the former homepage (moved verbatim in the
// stage B refactor). The "/" homepage now only redirects here.
//
// THEME SCOPE (decision 2026-09-29): this page is ALWAYS classic (Instant
// Print). It once followed the creator theme (fetching GET /api/profile and
// setting body[data-profile-theme], with a stale effect as well because of the
// empty deps []), but the refinement pass narrowed the scope: the theme applies
// ONLY on /u/[username] plus the navbar while standing there. The shortener is
// presented on the instant print stage for EVERY user: guest and logged-in
// alike: so the "utility" never changes appearance.
// The form itself was extracted to components/ShortenForm.jsx.
export default function AppHome() {
  const st = themeStyles("classic");

  return (
    <main className={`mx-auto w-full max-w-6xl px-4 pb-16 pt-20 sm:px-6 lg:px-8 ${st.text}`}>

      <ThemeBackdrop />
      <p className={`inline-block ${st.radiusFull} ${st.borderW} ${st.border} ${st.chip} px-3 py-1 font-mono text-xs font-bold uppercase tracking-[0.15em] ${st.textMuted}`}>
        Link-in-bio untuk kreator
      </p>
      <h1 className={`mt-3 ${st.headingFont} text-4xl font-bold leading-tight ${st.text}`}>
        Satu halaman
        <br />
        untuk semua link kamu.
      </h1>
      <p className={`mt-2 text-sm ${st.textMuted}`}>
        Persingkat link di bawah: kalau kamu login, link otomatis masuk ke profil publikmu.
      </p>

      <div className="mt-6">
        <ShortenForm st={st} />
      </div>

      <p className="mt-6">
        <Link href="/links" className={`inline-block ${st.radiusFull} ${st.borderW} ${st.border} ${st.accent} px-5 py-2 text-sm font-bold`}>
          Lihat analytics →
        </Link>
      </p>
    </main>
  );
}
