"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { Plus } from "lucide-react";
import { AnimatePresence, motion } from "framer-motion";
import AuthModal from "./AuthModal";
import { themeStyles } from "../../lib/themes";

// DESIGN.md §4 (revisi Instant Print) — capsule floating bar: fixed, top 20px,
// pill penuh, max-width 640-720px. Background print-white SOLID + border ink
// 2px tegas (pengganti shadow — TIDAK PAKAI shadow sama sekali). Saat scroll:
// border 2px → 2.5px (via outline trick di bawah) + padding -20%, transisi 200ms.
// Tombol Daftar di sini versi OUTLINE (1 CTA solid kuning per view hanya untuk
// aksi utama view tersebut — lihat tiap halaman).
export default function Navbar() {
  const [scrolled, setScrolled] = useState(false);
  const [username, setUsername] = useState(null);
  const [modal, setModal] = useState(null);
  const [claimMsg, setClaimMsg] = useState(null);
  const claimTimer = useRef(null);
  const router = useRouter();
  const pathname = usePathname();
  // Profil (display_name + avatar_url) hanya dibutuhkan saat login untuk
  // avatar dropdown — bukan data kritis, diam saja jika gagal.
  const [profile, setProfile] = useState(null);
  const [menuOpen, setMenuOpen] = useState(false);
  const accountRef = useRef(null);

  useEffect(() => {
    setUsername(localStorage.getItem("jejak_username"));
    function onScroll() {
      setScrolled(window.scrollY > 24);
    }
    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => {
      window.removeEventListener("scroll", onScroll);
      clearTimeout(claimTimer.current);
    };
  }, []);

  // Setelah login/register sukses: klaim link anonim yang dibuat di browser
  // ini (localStorage). Key DIHAPUS baik sukses maupun gagal supaya tidak
  // dicoba berulang-ulang; toast hanya kalau ada yang benar-benar ke-klaim.
  async function handleAuthSuccess(u) {
    setUsername(u);
    setModal(null);
    let pending = [];
    try {
      const raw = localStorage.getItem("jejak_unclaimed_links");
      const parsed = raw ? JSON.parse(raw) : [];
      if (Array.isArray(parsed)) {
        pending = parsed.filter((c) => typeof c === "string" && c !== "");
      }
    } catch {
      // entry korup diabaikan — tetap dibersihkan di bawah
    }
    localStorage.removeItem("jejak_unclaimed_links");
    if (pending.length === 0) {
      router.push("/dashboard");
      return;
    }
    try {
      const res = await fetch("/api/links/claim", {
        body: JSON.stringify({ short_codes: pending }),
      });
      const text = await res.text();
      let data = {};
      try {
        data = JSON.parse(text);
      } catch {
        return;
      }
      const n = res.ok ? Number(data.claimed) || 0 : 0;
      if (n > 0) {
        setClaimMsg(`${n} link berhasil ditambahkan ke akunmu`);
        clearTimeout(claimTimer.current);
        claimTimer.current = setTimeout(() => setClaimMsg(null), 2500);
      }
    } catch {
      // klaim gagal diam-diam: key sudah dibersihkan, tidak dicoba ulang
    }
    router.push("/dashboard");
  }

  async function loadProfile() {
    try {
      const res = await fetch("/api/profile", { cache: "no-store" });
      const text = await res.text();
      let d = {};
      try { d = JSON.parse(text); } catch { return; }
      if (res.ok) {
        setProfile({ name: d.display_name || "", avatar_url: d.avatar_url || "", theme: d.theme || "classic" });
      }
    } catch {
      // navbar tidak bergantung pada profil — diam saja saat gagal
    }
  }

  // Muat profil saat login + refresh tiap navigasi (supaya avatar terbaru
  // muncul setelah edit profil di dashboard).
  useEffect(() => {
    if (username === null) { setProfile(null); return; }
    loadProfile();
  }, [username, pathname]);

  // DashboardClient me-broadcast jejak:profile-updated setelah PUT /api/profile
  // sukses (theme/avatar/display_name berubah TANPA navigasi). Kalau tidak,
  // profile.theme di sini nyangkut di nilai lama → navbar tidak ikut tema baru
  // (mis. glass) sampai pindah halaman. Listener ini memperbaiki stall-nya.
  useEffect(() => {
    if (username === null) return;
    function onProfileUpdated() {
      loadProfile();
    }
    window.addEventListener("jejak:profile-updated", onProfileUpdated);
    return () => window.removeEventListener("jejak:profile-updated", onProfileUpdated);
  }, [username]);

  // Tutup dropdown jika klik di luar atau tekan Escape.
  useEffect(() => {
    if (!menuOpen) return;
    function onDoc(e) {
      if (accountRef.current && !accountRef.current.contains(e.target)) {
        setMenuOpen(false);
      }
    }
    function onKey(e) {
      if (e.key === "Escape") setMenuOpen(false);
    }
    document.addEventListener("mousedown", onDoc);
    document.addEventListener("touchstart", onDoc, { passive: true });
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDoc);
      document.removeEventListener("touchstart", onDoc);
      document.removeEventListener("keydown", onKey);
    };
  }, [menuOpen]);

  async function logout() {
    setMenuOpen(false);
    try {
      await fetch("/api/logout", { method: "POST" });
    } finally {
      localStorage.removeItem("jejak_username");
      setUsername(null);
      router.push("/");
    }
  }

  // Fitur "Navbar ikut tema halaman": semula tema hanya dibaca saat berdiri di
  // /dashboard (onDashboard), dipakai profile.theme sendiri. Akibatnya di halaman
  // publik /u yang bertema Darkroom, navbar tetap terang (border-ink bg-print-white)
  // — bug. Sekarang tema dibaca dari body[data-profile-theme] yang dipasang oleh
  // ProfileLinks (/u) dan DashboardClient (/dashboard): satu sumber kebenaran yang
  // sama, tanpa gerbang pathname. classic/coral → pill terang (identik dengan
  // default); darkroom → pill gelap; glass → frosted.
  const [pageTheme, setPageTheme] = useState("classic");
  useEffect(() => {
    const read = () => setPageTheme(document.body.dataset.profileTheme || "classic");
    read();
    const mo = new MutationObserver(read);
    mo.observe(document.body, { attributes: true, attributeFilter: ["data-profile-theme"] });
    return () => mo.disconnect();
  }, []);
  const t = themeStyles(pageTheme);
  const asLight = pageTheme === "classic" || pageTheme === "coral";
  // Task 1+2 — pill navbar FROSTED PERMANEN di semua tema & semua state scroll
  // (ala Apple): bg putih transparan + backdrop blur tebal + border putih tipis,
  // TANPA bentuk solid bg-print-white yang nyatu dengan body di puncak halaman
  // (sebelumnya classic/coral pill = bg-print-white ≈ bg body → "menghilang" di
  // posisi paling atas; baru kelihatan setelah discroll saat outline-ink muncul;
  // glass satu-satunya yang frosted). Sekarang 1 gaya frosted konsisten.
  // Pill navbar per-tema: BUKAN 1-gaya-frosted-untuk-semua (sebelumnya malah
  // bikin classic/coral/glass tampil glassmorphism semua, padahal maunya hanya
  // tema Glass yang frosted). Kini:
  //   - classic/coral : pill SOLID putih (bg-print-white) + border ink 2px +
  //                     shadow lembut → tetap kelihatan di posisi paling atas
  //                     (fix Task 1 tanpa harus pakai glassmorphism).
  //   - darkroom      : frosted GELAP agar kontras dengan body #121212.
  //   - glass         : satunya-satunya glassmorphism (frosted putih + blur).
  const pillCls =
    pageTheme === "glass"
      ? "rounded-full bg-white/[38%] backdrop-blur-2xl border-[1px] border-white/35 shadow-[0_8px_30px_rgba(28,26,18,0.16)]"
      : pageTheme === "darkroom"
        ? "rounded-full bg-ink/60 backdrop-blur-xl border-[1px] border-print-white/20"
        : "rounded-full bg-print-white border-2 border-ink shadow-[0_6px_24px_rgba(28,26,18,0.18)]";
  const logoCls = t.text;
  const capsuleInactive = t.textMuted;
  const capsuleHover = "";
  // Chrome interior navbar (tombol aksi, avatar, dropdown) ikut tema juga —
  // kalau tidak, di atas pill Darkroom yang gelap tetap ada tombol terang.
  const actionPill = asLight
    ? "rounded-full border-2 border-ink bg-print-white text-ink hover:bg-paper-grey/60"
    : `${t.radiusFull} ${t.borderW} ${t.border} ${t.card} hover:opacity-80`;
  const avatarFrame = asLight
    ? "border-[1.5px] border-ink bg-print-white text-ink"
    : `${t.borderW} ${t.border} ${t.card}`;
  const menuCls = asLight
    ? "rounded-xl border-2 border-ink bg-print-white p-2"
    : `${t.radiusLarge} ${t.borderW} ${t.border} ${t.card} p-2`;
  const menuName = asLight ? "text-ink" : t.text;
  const menuMuted = asLight ? "text-muted" : t.textMuted;
  const menuDivider = asLight ? "border-ink/40" : "border-print-white/25";
  const menuItem = asLight ? "text-ink hover:bg-paper-grey/60" : `${t.text} hover:bg-white/[8%]`;
  const ghostBtn = asLight ? "text-ink/70 hover:text-ink" : t.textMuted;

  return (
    <>
      <header
        className={`fixed left-1/2 top-5 z-50 w-[min(92vw,680px)] -translate-x-1/2 ${pillCls} transition-all duration-200 ${
          scrolled ? "px-4 py-2" : "px-5 py-3"
        }`}
      >
        <nav className="flex items-center justify-between gap-3">
          <Link href="/" className={`flex items-center gap-2 font-display text-sm font-bold tracking-widest ${logoCls} transition-opacity duration-150 hover:opacity-80`}>
            <span className={`inline-block h-2.5 w-2.5 rounded-[2px] ${pageTheme === "coral" ? "bg-flash-orange" : "bg-flash-yellow"} outline outline-1 outline-ink`} aria-hidden="true" />
            JEJAK
          </Link>

          <div className="flex items-center gap-4 text-sm">
            {username === null ? (
              <Link
                href="/app"
                className={`${actionPill} px-4 py-1 font-medium transition-colors duration-150`}
              >
                + Link Baru
              </Link>
            ) : (
              <div className="flex items-center">
                {/* Kapsul indikator geser (sama teknik tab dashboard):
                    motion.span layoutId=navbar-active di belakang item aktif,
                    transisi spring redam 300/30 — konsisten dengan dashboard. */}
                <Link
                  href="/app"
                  className={`relative flex items-center gap-1 rounded-full px-4 py-1 text-sm transition-colors duration-200 ${
                    pathname === "/app" ? "font-bold text-ink" : capsuleInactive
                  }${capsuleHover}`}
                >
                  {pathname === "/app" && (
                    <motion.span
                      layoutId="navbar-active"
                      transition={{ type: "spring", stiffness: 300, damping: 30 }}
                      className={`absolute inset-0 rounded-full ${pageTheme === "coral" ? "bg-flash-orange" : "bg-flash-yellow"}`}
                    />
                  )}
                  <span className="relative flex items-center gap-1">
                    <Plus className="h-4 w-4" aria-hidden="true" />
                    <span className="text-sm">Link Baru</span>
                  </span>
                </Link>
                <Link
                  href="/dashboard"
                  className={`relative rounded-full px-4 py-1 text-sm transition-colors duration-200 ${
                    pathname === "/dashboard" ? "font-bold text-ink" : capsuleInactive
                  }${capsuleHover}`}
                >
                  {pathname === "/dashboard" && (
                    <motion.span
                      layoutId="navbar-active"
                      transition={{ type: "spring", stiffness: 300, damping: 30 }}
                      className={`absolute inset-0 rounded-full ${pageTheme === "coral" ? "bg-flash-orange" : "bg-flash-yellow"}`}
                    />
                  )}
                  <span className="relative">Dashboard</span>
                </Link>
              </div>
            )}
            {username === null ? (
              <>
                <button onClick={() => setModal("login")} className={`${ghostBtn} transition-colors duration-200`}>
                  Masuk
                </button>
                <button
                  onClick={() => setModal("register")}
                  className={`${actionPill} px-4 py-1 font-medium transition-colors duration-150`}
                >
                  Daftar
                </button>
              </>
            ) : (
            <>
              {/* Avatar dropdown — Dashboard sudah di dalam container kapsul di atas. */}
              <div className="relative" ref={accountRef}>
                <button
                  type="button"
                  onClick={() => setMenuOpen((v) => !v)}
                  aria-expanded={menuOpen}
                  aria-haspopup="menu"
                  aria-label="Menu akun"
                  title={`@${username}`}
                  className={`flex h-8 w-8 items-center justify-center overflow-hidden rounded-full ${avatarFrame} text-sm font-bold transition-opacity duration-150 hover:opacity-80 focus:outline-none focus-visible:ring-2 focus-visible:ring-flash-coral`}
                >
                  {profile && profile.avatar_url !== "" ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img src={profile.avatar_url} alt="" className="h-full w-full object-cover" />
                  ) : (
                    <span>{((profile?.name || username) || "").slice(0, 1).toUpperCase()}</span>
                  )}
                </button>
                <AnimatePresence>
                  {menuOpen && (
                    <motion.div
                      role="menu"
                      initial={{ opacity: 0, y: -4, scale: 0.96 }}
                      animate={{ opacity: 1, y: 0, scale: 1 }}
                      exit={{ opacity: 0, y: -4, scale: 0.96 }}
                      transition={{ duration: 0.15, ease: "easeOut" }}
                      className={`absolute right-0 top-full z-50 mt-2 w-56 origin-top-right ${menuCls}`}
                    >
                      <div className="px-3 py-1.5">
                        <p className={`truncate text-sm font-bold ${menuName}`}>{profile?.name || "Pengguna"}</p>
                        <p className={`truncate font-mono text-xs ${menuMuted}`}>@{username}</p>
                      </div>
                      <div className={`my-1 border-t ${menuDivider}`} />
                      <button
                        type="button"
                        role="menuitem"
                        onClick={logout}
                        className={`flex w-full items-center gap-2 rounded-lg px-3 py-1.5 text-sm font-medium transition-colors duration-150 ${menuItem}`}
                      >
                        Keluar
                      </button>
                    </motion.div>
                  )}
                </AnimatePresence>
              </div>
            </>
            )}
          </div>
        </nav>
      </header>

      <AnimatePresence>
        {modal !== null && (
          <AuthModal
            mode={modal}
            onClose={() => setModal(null)}
            onSuccess={handleAuthSuccess}
            st={t}
          />
        )}
      </AnimatePresence>

      <AnimatePresence>
        {claimMsg !== null && (
          <div className="pointer-events-none fixed inset-x-0 bottom-6 z-[70] flex justify-center">
            <motion.div
              initial={{ opacity: 0, y: 12 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: 8 }}
              transition={{ duration: 0.2, ease: "easeOut" }}
              className="rounded-full border-2 border-ink bg-ink px-4 py-2 text-sm font-bold text-print-white"
            >
              {claimMsg}
            </motion.div>
          </div>
        )}
      </AnimatePresence>
    </>
  );
}
