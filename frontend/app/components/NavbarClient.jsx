"use client";

// =====================================================================
// NAVBAR CLIENT: interactive half of the auth-aware navbar.
// Navbar.jsx (Server Component) reads the `jejak_session` cookie and passes
// `isLoggedIn` to this component. Responsibilities: mobile hamburger toggle,
// avatar dropdown, AuthModal (login/register), claiming of anonymous links,
// and logout. Chrome colors follow the page theme via body[data-profile-theme]
// (observed with a MutationObserver): the attribute is set only by
// ProfileLinks on /u/[username], so the navbar on every other page stays on
// the classic preset (Instant Print).
// All chrome colors come from the theme tokens (nav/navHover/cta/ghost/
// panel/drawer/navCircle/logoDot in lib/themes.js): never hard-coded per
// page.
// =====================================================================
import { useCallback, useEffect, useRef, useState } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { Bell, Check, ChartColumnIncreasing, ChevronDown, Languages, LayoutDashboard, Link2, LogOut, Menu, MonitorSmartphone, Plus, QrCode, X } from "lucide-react";
import { AnimatePresence, motion } from "framer-motion";
import AuthModal from "./AuthModal";
import LanguageSwitcher, { LANGUAGE_ARIA, LANGUAGE_OPTIONS } from "./LanguageSwitcher";
import NotificationsBell from "./NotificationsBell";
import NotificationsModal from "./NotificationsModal";
import { themeStyles } from "../../lib/themes";
import { DEFAULT_TRANSITION, EASE, SPRING } from "../../lib/animations";
import { useTranslation } from "../../lib/I18nProvider";
import { setLocale } from "../../lib/i18n";
import { useNotifications } from "../../lib/useNotifications";

export default function NavbarClient({ isLoggedIn }) {
  const { t: tr, locale } = useTranslation();
  const [username, setUsername] = useState(null);
  const [modal, setModal] = useState(null);
  const [claimMsg, setClaimMsg] = useState(null);
  const claimTimer = useRef(null);
  const router = useRouter();
  const pathname = usePathname();
  // The profile (display_name + avatar_url) is only needed while logged in,
  // for the avatar dropdown: not critical data, so failures are ignored
  // silently.
  const [profile, setProfile] = useState(null);
  const [menuOpen, setMenuOpen] = useState(false); // dropdown avatar (desktop)
  const [drawerOpen, setDrawerOpen] = useState(false); // hamburger (mobile)
  const [fiturOpen, setFiturOpen] = useState(false); // dropdown Fitur (desktop)
  const [fiturAccOpen, setFiturAccOpen] = useState(false); // accordion Fitur (drawer)
  const [notifOpen, setNotifOpen] = useState(false); // notifications modal (from drawer)
  // Drawer badge count: its own feed instance (the desktop bell and the
  // modal each hold theirs); all read the same endpoint.
  const { unread } = useNotifications();
  const closeNotif = useCallback(() => setNotifOpen(false), []);
  const accountRef = useRef(null);
  const fiturRef = useRef(null);

  // Four "Fitur" entries: a dropdown trigger on desktop, an accordion inside
  // the mobile drawer. All of them target /#fitur (the landing page has a
  // single features section; the entries differ in content, not destination).
  // The anchor is always prefixed with "/" so it resolves from any page
  // (/dashboard, /app, /u/...).
  const FEATURE_ITEMS = [
    { label: tr("nav.features.customSlug.label"), desc: tr("nav.features.customSlug.desc"), icon: Link2 },
    { label: tr("nav.features.smartLink.label"), desc: tr("nav.features.smartLink.desc"), icon: MonitorSmartphone },
    { label: tr("nav.features.qrCode.label"), desc: tr("nav.features.qrCode.desc"), icon: QrCode },
    { label: tr("nav.features.analytics.label"), desc: tr("nav.features.analytics.desc"), icon: ChartColumnIncreasing },
  ];

  // "Fitur" is not listed here: it already became the dropdown trigger
  // (FEATURE_ITEMS).
  const PUBLIC_LINKS = [
    { label: tr("nav.menu.demo"), href: "/#demo" },
    { label: tr("nav.menu.faq"), href: "/#faq" },
  ];

  // Minimal authenticated menu: "Link Saya" and "Analytics" were REMOVED from
  // the navbar (2026-09-22): all three pointed at the same /dashboard route and
  // offered no distinct function, which was confusing. Intended follow-up: move
  // those features into the dashboard as tabs/sections.
  const AUTH_LINKS = [
    { label: tr("nav.menu.dashboard"), href: "/dashboard" },
  ];

  useEffect(() => {
    setUsername(localStorage.getItem("jejak_username"));
    return () => clearTimeout(claimTimer.current);
  }, []);

  // Theme-aware navbar: read body[data-profile-theme], which is set by
  // ProfileLinks (/u): the only writer of that attribute (dashboard and /app
  // never set it; landing and /app default to "classic", i.e. Instant Print).
  const [pageTheme, setPageTheme] = useState("classic");
  useEffect(() => {
    const read = () => setPageTheme(document.body.dataset.profileTheme || "classic");
    read();
    const mo = new MutationObserver(read);
    mo.observe(document.body, { attributes: true, attributeFilter: ["data-profile-theme"] });
    return () => mo.disconnect();
  }, []);

  // Load the profile when logged in, and refresh it on every navigation so
  // the latest avatar appears after the profile is edited in the dashboard.
  useEffect(() => {
    if (!isLoggedIn) {
      setProfile(null);
      return;
    }
    loadProfile();
  }, [isLoggedIn, pathname]);

  // DashboardClient broadcasts jejak:profile-updated after a successful
  // PUT /api/profile (theme/avatar/display_name change with no navigation):
  // this listener refreshes the navbar profile without leaving the page.
  useEffect(() => {
    if (!isLoggedIn) return;
    function onProfileUpdated() {
      loadProfile();
    }
    window.addEventListener("jejak:profile-updated", onProfileUpdated);
    return () => window.removeEventListener("jejak:profile-updated", onProfileUpdated);
  }, [isLoggedIn]);

  // Dismiss the avatar dropdown on outside click or Escape; the listeners are
  // attached only while the menu is open.
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

  // Desktop "Fitur" dropdown: outside click closes it, Escape closes it. It is
  // currently the only nav dropdown: if another one is added, opening it must
  // close this one so that only a single dropdown is ever open.
  useEffect(() => {
    if (!fiturOpen) return;
    function onDoc(e) {
      if (fiturRef.current && !fiturRef.current.contains(e.target)) {
        setFiturOpen(false);
      }
    }
    function onKey(e) {
      if (e.key === "Escape") setFiturOpen(false);
    }
    document.addEventListener("mousedown", onDoc);
    document.addEventListener("touchstart", onDoc, { passive: true });
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDoc);
      document.removeEventListener("touchstart", onDoc);
      document.removeEventListener("keydown", onKey);
    };
  }, [fiturOpen]);

  // Mobile drawer: body scroll is locked while it is open, and Escape closes
  // it so background content cannot be scrolled from underneath the overlay.
  useEffect(() => {
    if (!drawerOpen) return;
    document.body.style.overflow = "hidden";
    function onKey(e) {
      if (e.key === "Escape") setDrawerOpen(false);
    }
    document.addEventListener("keydown", onKey);
    return () => {
      document.body.style.overflow = "";
      document.removeEventListener("keydown", onKey);
    };
  }, [drawerOpen]);

  // The "Fitur" accordion resets together with the drawer so it does not
  // reopen in an expanded state on the next visit.
  useEffect(() => {
    if (!drawerOpen) {
      setFiturAccOpen(false);
    }
  }, [drawerOpen]);

  // Any route change closes both menus and the notifications modal: keyed on
  // pathname so links outside the drawer (desktop nav, CTA) cannot leave a
  // stale open state behind.
  useEffect(() => {
    setDrawerOpen(false);
    setMenuOpen(false);
    setFiturOpen(false);
    setNotifOpen(false);
  }, [pathname]);

  async function loadProfile() {
    try {
      const res = await fetch("/api/profile", { cache: "no-store" });
      const text = await res.text();
      let d = {};
      try { d = JSON.parse(text); } catch { return; }
      if (res.ok) {
        setProfile({
          name: d.display_name || "",
          avatar_url: d.avatar_url || "",
          username: d.username || "",
          theme: d.theme || "classic",
        });
      }
    } catch {
      // the navbar does not depend on the profile: stay silent on failure
    }
  }

  // After a successful login/registration: claim the anonymous links created
  // in this browser (localStorage). The key is removed whether the claim
  // succeeds or fails so it is never retried, and the toast appears only when
  // at least one link was actually claimed. Ends with router.push("/dashboard")
  // + router.refresh(): /api/login has already set the session cookie, and
  // refresh forces the server Navbar to re-render with the latest login state.
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
      // a corrupted entry is skipped: the key is still cleared below
    }
    localStorage.removeItem("jejak_unclaimed_links");
    if (pending.length > 0) {
      try {
        const res = await fetch("/api/links/claim", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ short_codes: pending }),
        });
        const text = await res.text();
        let data = {};
        try {
          data = JSON.parse(text);
        } catch {
          data = {};
        }
        const n = res.ok ? Number(data.claimed) || 0 : 0;
        if (n > 0) {
          setClaimMsg(tr("nav.claimMessage", { count: n }));
          clearTimeout(claimTimer.current);
          claimTimer.current = setTimeout(() => setClaimMsg(null), 2500);
        }
      } catch {
        // a failed claim stays silent: the key is already cleared, so nothing
        // is retried
      }
    }
    router.push("/dashboard");
    router.refresh();
  }

  async function logout() {
    setMenuOpen(false);
    setDrawerOpen(false);
    try {
      await fetch("/api/logout", { method: "POST" });
    } finally {
      localStorage.removeItem("jejak_username");
      setUsername(null);
      router.push("/");
      // Force the server Navbar to re-render with the cleared cookie: pushing
      // to the same "/" route does not always refresh the RSC payload.
      router.refresh();
    }
  }

  const t = themeStyles(pageTheme);

  // The chrome bar and every navbar color come from the theme tokens in
  // lib/themes.js (constraint: never hardcode bg-print-white/80 for all
  // themes). Classic matches the Instant Print specification exactly (the
  // navbar on /, /dashboard and /app is unchanged); the other presets carry
  // their own stage colors.
  const barCls = t.nav;
  const navLinkCls = `font-medium ${t.text} underline-offset-4 transition-colors duration-150 ${t.navHover}`;
  const ghostBtnCls = t.ghost;
  const primaryBtnCls = t.cta;
  const primaryBase = "inline-flex items-center gap-1 rounded-full px-4 py-1.5 text-sm font-bold";
  const ghostBase = "inline-flex items-center rounded-full px-4 py-1.5 text-sm font-medium";

  const drawerDividerCls = `my-2 border-t ${t.panelRule ?? "border-ink/10"}`;
  const drawerDividerWideCls = `my-6 border-t ${t.panelRule ?? "border-ink/10"}`;
  const drawerPanelCls = t.drawer;

  // "Fitur" dropdown items: exact specification styling (bg-white panel,
  // border-2 ink, hard shadow, 12px panel radius / 8px item radius, hover
  // flash-yellow/20). A solid white panel sits above the themed bar: it is an
  // overlay on top of the navbar background, so the adaptive navbar itself
  // stays untouched.
  const fiturItemCls =
    "flex w-full items-start gap-3 rounded-lg px-2.5 py-2.5 transition-colors duration-150 hover:bg-flash-yellow/20";

  function FeatureIcon({ icon: Icon, className = "" }) {
    return (
      <span
        className={`mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border-2 border-ink bg-flash-yellow ${className}`}
        aria-hidden="true"
      >
        <Icon className="h-4 w-4 text-ink" strokeWidth={2.5} />
      </span>
    );
  }

  // Drawer menu row: icon (20px) + label with room for a right-side badge
  // (notification count) or an accordion chevron. Renders a Link when href
  // is set, a button otherwise. Text/hover follow the page theme (the drawer
  // panel adapts via t.drawer); the danger row stays flash-coral on every
  // theme. No emojis: lucide icons only.
  function DrawerRow({ icon: Icon, label, badge, href, onClick, danger, expanded, ariaLabel }) {
    const cls = `flex w-full items-center gap-3 rounded-lg px-4 py-3 text-sm font-medium transition-colors duration-150 ${
      danger ? "text-flash-coral hover:bg-flash-coral/10" : `${t.text ?? "text-ink"} ${t.panelHover ?? "hover:bg-ink/5"}`
    }`;
    const content = (
      <>
        {Icon ? <Icon className="h-5 w-5 shrink-0" strokeWidth={2} aria-hidden="true" /> : null}
        <span className="min-w-0 flex-1 truncate text-left">{label}</span>
        {typeof badge === "number" && badge > 0 ? (
          <span className="flex h-5 min-w-5 items-center justify-center rounded-full bg-flash-coral px-1 font-mono text-xs font-bold leading-none text-white">
            {badge > 9 ? "9+" : badge}
          </span>
        ) : null}
        {expanded === null || expanded === undefined ? null : (
          <ChevronDown
            className={`h-4 w-4 shrink-0 transition-transform duration-200 ${expanded ? "rotate-180" : ""}`}
            strokeWidth={2.5}
            aria-hidden="true"
          />
        )}
      </>
    );
    if (href) {
      return (
        <Link href={href} onClick={onClick} aria-label={ariaLabel} className={cls}>
          {content}
        </Link>
      );
    }
    return (
      <button type="button" onClick={onClick} aria-label={ariaLabel} className={cls}>
        {content}
      </button>
    );
  }

  return (
    <>
      <header className={`fixed inset-x-0 top-0 z-50 h-16 ${barCls}`}>
        <div className="mx-auto flex h-full max-w-6xl items-center justify-between gap-3 px-4 sm:px-6 lg:px-8">
          {/* Left: logo + desktop nav links */}
          <div className="flex items-center gap-6">
            <Link
              href="/"
              className={`flex shrink-0 items-center gap-2 font-display text-sm font-bold tracking-widest ${t.text} transition-opacity duration-150 hover:opacity-80`}
            >
              <span
                className={`inline-block h-2.5 w-2.5 rounded-[2px] ${t.logoDot}`}
                aria-hidden="true"
              />
              {tr("nav.brand")}
            </Link>

            <nav className="hidden items-center gap-6 text-sm lg:flex" aria-label={tr("nav.aria.mainMenu")}>
              {isLoggedIn ? (
                AUTH_LINKS.map((item) => (
                  <Link key={item.label} href={item.href} className={navLinkCls}>
                    {item.label}
                  </Link>
                ))
              ) : (
                <>
                  {/* "Fitur": dropdown trigger, not a plain link */}
                  <div className="relative" ref={fiturRef}>
                    <button
                      type="button"
                      onClick={() => setFiturOpen((v) => !v)}
                      aria-expanded={fiturOpen}
                      aria-haspopup="true"
                      aria-controls="fitur-dropdown"
                      className={`${navLinkCls} flex items-center gap-1`}
                    >
                      {tr("nav.menu.features")}
                      <ChevronDown
                        className={`h-3.5 w-3.5 transition-transform duration-200 ${
                          fiturOpen ? "rotate-180" : ""
                        }`}
                        strokeWidth={2.5}
                        aria-hidden="true"
                      />
                    </button>
                    <AnimatePresence>
                      {fiturOpen && (
                        <motion.div
                          id="fitur-dropdown"
                          initial={{ opacity: 0, y: -8 }}
                          animate={{ opacity: 1, y: 0 }}
                          exit={{ opacity: 0, y: -8 }}
                          transition={{ duration: 0.18, ease: EASE }}
                          className="absolute left-0 top-full z-50 mt-2 w-72 rounded-xl border-2 border-ink bg-white p-2 shadow-[4px_4px_0px_#1C1A12]"
                        >
                          {FEATURE_ITEMS.map((f) => (
                            <Link
                              key={f.label}
                              href="/#fitur"
                              onClick={() => setFiturOpen(false)}
                              className={fiturItemCls}
                            >
                              <FeatureIcon icon={f.icon} />
                              <span className="min-w-0">
                                <span className="block text-sm font-medium text-ink">{f.label}</span>
                                <span className="block truncate text-xs text-muted">{f.desc}</span>
                              </span>
                            </Link>
                          ))}
                        </motion.div>
                      )}
                    </AnimatePresence>
                  </div>
                  {PUBLIC_LINKS.map((item) => (
                    <Link key={item.label} href={item.href} className={navLinkCls}>
                      {item.label}
                    </Link>
                  ))}
                </>
              )}
            </nav>
          </div>

          {/* Right: desktop CTA + mobile hamburger */}
          <div className="flex items-center gap-3">
            <div className="hidden items-center gap-3 lg:flex">
              {isLoggedIn ? (
                <>
                  <motion.div whileHover={{ y: -2 }} transition={{ duration: 0.15, ease: EASE }}>
                    <Link href="/app" className={`${primaryBase} ${primaryBtnCls}`}>
                      <Plus className="h-4 w-4" strokeWidth={2.5} aria-hidden="true" />
                      {tr("nav.cta.newLink")}
                    </Link>
                  </motion.div>

                  {/* Health-monitor notifications (logged in only): unread
                      badge + dropdown feed; placed before the language
                      switcher to keep CTAs right-most. */}
                  <NotificationsBell t={t} />

                  <LanguageSwitcher />

                  {/* Avatar dropdown */}
                  <div className="relative" ref={accountRef}>
                    <button
                      type="button"
                      onClick={() => setMenuOpen((v) => !v)}
                      aria-expanded={menuOpen}
                      aria-haspopup="menu"
                      aria-label={tr("nav.aria.accountMenu")}
                      title={username ? tr("nav.account.title", { username }) : tr("nav.account.titleFallback")}
                      className={`flex items-center gap-1 focus:outline-hidden focus-visible:ring-2 focus-visible:ring-flash-coral`}
                    >
                      <span
                        className={`flex h-10 w-10 items-center justify-center overflow-hidden rounded-full text-sm font-bold ${t.navCircle}`}
                      >
                        {profile && profile.avatar_url !== "" ? (
                          // eslint-disable-next-line @next/next/no-img-element
                          <img src={profile.avatar_url} alt="" className="h-full w-full object-cover" />
                        ) : (
                          <span>{((profile?.name || username) || "").slice(0, 1).toUpperCase()}</span>
                        )}
                      </span>
                      <ChevronDown className={`h-4 w-4 ${t.textMuted}`} aria-hidden="true" />
                    </button>
                    <AnimatePresence>
                      {menuOpen && (
                        <motion.div
                          role="menu"
                          initial={{ opacity: 0, y: -8 }}
                          animate={{ opacity: 1, y: 0 }}
                          exit={{ opacity: 0, y: -8 }}
                          transition={{ duration: 0.18, ease: EASE }}
                          className={`absolute right-0 top-full z-50 mt-2 w-56 origin-top-right ${t.radiusLarge} ${t.panel} p-2`}
                        >
                          <div className="px-3 py-1.5">
                            <p className={`truncate text-sm font-bold ${t.text}`}>
                              {profile?.name || tr("nav.account.nameFallback")}
                            </p>
                            {(username || profile?.username) && (
                              <p className={`truncate font-mono text-xs ${t.textMuted}`}>
                                @{username || profile?.username}
                              </p>
                            )}
                          </div>
                          <div className={`my-1 border-t ${t.panelRule}`} />
                          <Link
                            href="/dashboard"
                            role="menuitem"
                            onClick={() => setMenuOpen(false)}
                            className={`flex w-full items-center gap-2 rounded-lg px-3 py-1.5 text-sm font-medium transition-colors duration-150 ${t.text} ${t.panelHover}`}
                          >
                            {tr("nav.account.profile")}
                          </Link>
                          <button
                            type="button"
                            role="menuitem"
                            onClick={logout}
                            className={`flex w-full items-center gap-2 rounded-lg px-3 py-1.5 text-sm font-medium transition-colors duration-150 ${t.text} ${t.panelHover}`}
                          >
                            {tr("nav.account.logout")}
                          </button>
                        </motion.div>
                      )}
                    </AnimatePresence>
                  </div>
                </>
              ) : (
                <>
                  <LanguageSwitcher />
                  <button
                    type="button"
                    onClick={() => setModal("login")}
                    className={`${ghostBase} ${ghostBtnCls} transition-colors duration-150`}
                  >
                    {tr("nav.cta.login")}
                  </button>
                  <motion.div whileHover={{ y: -2 }} transition={{ duration: 0.15, ease: EASE }}>
                    <button
                      type="button"
                      onClick={() => setModal("register")}
                      className={`${primaryBase} ${primaryBtnCls} transition-shadow duration-150`}
                    >
                      {tr("nav.cta.register")}
                    </button>
                  </motion.div>
                </>
              )}
            </div>

            {/* Hamburger: mobile only (< lg) */}
            <button
              type="button"
              onClick={() => setDrawerOpen(true)}
              aria-label={tr("nav.aria.openMenu")}
              aria-expanded={drawerOpen}
              className={`flex h-10 w-10 items-center justify-center rounded-full lg:hidden ${ghostBtnCls}`}
            >
              <Menu className="h-5 w-5" strokeWidth={2.5} aria-hidden="true" />
            </button>
          </div>
        </div>
      </header>

      {/* ===== MOBILE DRAWER: slides in from the right ===== */}
      <AnimatePresence>
        {drawerOpen && (
          <>
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ duration: 0.2, ease: EASE }}
              onClick={() => setDrawerOpen(false)}
              className="fixed inset-0 z-[55] bg-ink/60 lg:hidden"
              aria-hidden="true"
            />
            <motion.aside
              role="dialog"
              aria-modal="true"
              aria-label={tr("nav.aria.drawer")}
              initial={{ x: "100%" }}
              animate={{ x: 0 }}
              exit={{ x: "100%" }}
              transition={{ duration: 0.3, ease: EASE }}
              className={`fixed right-0 top-0 z-[60] flex h-full w-[min(85vw,340px)] flex-col overflow-y-auto lg:hidden ${drawerPanelCls}`}
            >
              <div className={`flex items-center justify-between border-b px-4 py-3 ${t.panelRule}`}>
                <span className={`font-display text-sm font-bold tracking-widest ${t.text}`}>{tr("nav.brand")}</span>
                <button
                  type="button"
                  onClick={() => setDrawerOpen(false)}
                  aria-label={tr("nav.aria.closeMenu")}
                  className={`flex h-10 w-10 items-center justify-center rounded-full ${t.navCircle}`}
                >
                  <X className="h-5 w-5" strokeWidth={2.5} aria-hidden="true" />
                </button>
              </div>

              <nav className="flex flex-col gap-1 p-4" aria-label={tr("nav.aria.mobileMenu")}>
                {isLoggedIn ? (
                  <>
                    {/* Primary CTA first with clear air below it (mb-4): it
                        must never cover the rows underneath. */}
                    <Link
                      href="/app"
                      onClick={() => setDrawerOpen(false)}
                      className={`${primaryBase} ${primaryBtnCls} mb-4 w-full justify-center py-3`}
                    >
                      <Plus className="h-4 w-4" strokeWidth={2.5} aria-hidden="true" />
                      {tr("nav.cta.newLink")}
                    </Link>
                    <div className={drawerDividerCls} />
                    <DrawerRow
                      icon={LayoutDashboard}
                      label={tr("nav.menu.dashboard")}
                      href="/dashboard"
                      onClick={() => setDrawerOpen(false)}
                    />
                    {/* Notifications live in a separate modal (not an inline
                        panel): the drawer stays a short menu. Profile editing
                        lives in the dashboard's Profil tab: no separate row. */}
                    <DrawerRow
                      icon={Bell}
                      label={tr("notifications.title")}
                      badge={unread}
                      onClick={() => {
                        setDrawerOpen(false);
                        setNotifOpen(true);
                      }}
                    />
                    <div className={drawerDividerWideCls} />
                    <DrawerLanguage locale={locale} />
                    <div className={drawerDividerWideCls} />
                    <DrawerRow icon={LogOut} label={tr("nav.account.logout")} onClick={logout} danger />
                  </>
                ) : (
                  <>
                    {/* "Fitur": accordion: tap to expand the four entries */}
                    <button
                      type="button"
                      onClick={() => setFiturAccOpen((v) => !v)}
                      aria-expanded={fiturAccOpen}
                      aria-controls="fitur-accordion"
                      className={`flex w-full items-center gap-3 rounded-lg px-4 py-3 text-sm font-medium transition-colors duration-150 ${t.text ?? "text-ink"} ${t.panelHover ?? "hover:bg-ink/5"}`}
                    >
                      <span className="min-w-0 flex-1 truncate text-left">{tr("nav.menu.features")}</span>
                      <ChevronDown
                        className={`h-4 w-4 shrink-0 transition-transform duration-200 ${
                          fiturAccOpen ? "rotate-180" : ""
                        }`}
                        strokeWidth={2.5}
                        aria-hidden="true"
                      />
                    </button>
                    <AnimatePresence initial={false}>
                      {fiturAccOpen && (
                        <motion.div
                          id="fitur-accordion"
                          initial={{ height: 0, opacity: 0 }}
                          animate={{ height: "auto", opacity: 1 }}
                          exit={{ height: 0, opacity: 0 }}
                          transition={DEFAULT_TRANSITION}
                          className="overflow-hidden"
                        >
                          <div className="flex flex-col gap-1 py-1 pl-3">
                            {FEATURE_ITEMS.map((f) => (
                              <Link
                                key={f.label}
                                href="/#fitur"
                                onClick={() => {
                                  setFiturAccOpen(false);
                                  setDrawerOpen(false);
                                }}
                                className={fiturItemCls}
                              >
                                <FeatureIcon icon={f.icon} />
                                <span className="min-w-0">
                                  <span className="block text-sm font-medium text-ink">{f.label}</span>
                                  <span className="block truncate text-xs text-muted">{f.desc}</span>
                                </span>
                              </Link>
                            ))}
                          </div>
                        </motion.div>
                      )}
                    </AnimatePresence>
                    {PUBLIC_LINKS.map((item) => (
                      <DrawerRow
                        key={item.label}
                        label={item.label}
                        href={item.href}
                        onClick={() => setDrawerOpen(false)}
                      />
                    ))}
                    <div className={drawerDividerWideCls} />
                    <DrawerLanguage locale={locale} />
                    <div className={drawerDividerWideCls} />
                    <div className="flex flex-col gap-3">
                      <button
                        type="button"
                        onClick={() => {
                          setDrawerOpen(false);
                          setModal("login");
                        }}
                        className={`${ghostBase} ${ghostBtnCls} w-full justify-center py-3`}
                      >
                        {tr("nav.cta.login")}
                      </button>
                      <button
                        type="button"
                        onClick={() => {
                          setDrawerOpen(false);
                          setModal("register");
                        }}
                        className={`${primaryBase} ${primaryBtnCls} w-full justify-center py-3`}
                      >
                        {tr("nav.cta.register")}
                      </button>
                    </div>
                  </>
                )}
              </nav>
            </motion.aside>
          </>
        )}
      </AnimatePresence>

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

      <NotificationsModal open={notifOpen} onClose={closeNotif} t={t} />
      <AnimatePresence>
        {claimMsg !== null && (
          <div className="pointer-events-none fixed inset-x-0 bottom-6 z-[70] flex justify-center">
            <motion.div
              initial={{ opacity: 0, y: 12 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: 8 }}
              transition={{ duration: 0.2, ease: EASE }}
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

// Pill language switcher for the drawer (module scope: a nested definition
// would recreate the component type on every NavbarClient render, unmounting
// it and wiping the dropdown's open state whenever the parent re-renders,
// e.g. when the unread-count fetch resolves). Classic pill + dropdown card.
// The menu is IN-FLOW (relative, pushes content down) rather than absolute:
// an absolute menu would need ~140px of permanently reserved space to clear
// the rows below, leaving a crater when closed; in-flow cannot overlap by
// construction and stays inside the 340px drawer at 320px viewports.
// Options show native names; the active one is highlighted flash-yellow.
// setLocale() reloads the page, so no state sync is needed.
function DrawerLanguage({ locale }) {
  const active = LANGUAGE_OPTIONS.find((o) => o.locale === locale) || LANGUAGE_OPTIONS[0];
  const [open, setOpen] = useState(false);
  const rootRef = useRef(null);
  useEffect(() => {
    if (!open) return undefined;
    function onDoc(e) {
      if (rootRef.current && !rootRef.current.contains(e.target)) setOpen(false);
    }
    function onKey(e) {
      if (e.key === "Escape") setOpen(false);
    }
    document.addEventListener("mousedown", onDoc);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDoc);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);
  return (
    <div ref={rootRef} className="relative z-20">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-label={LANGUAGE_ARIA[locale] || LANGUAGE_ARIA.id}
        className="inline-flex items-center gap-1.5 rounded-full border-2 border-ink bg-white px-3 py-1.5 text-sm font-bold text-ink transition-colors duration-150 hover:bg-ink/5"
      >
        <Languages className="h-3.5 w-3.5" strokeWidth={2.5} aria-hidden="true" />
        {active.short}
        <ChevronDown
          className={`h-3.5 w-3.5 transition-transform duration-200 ${open ? "rotate-180" : ""}`}
          strokeWidth={2.5}
          aria-hidden="true"
        />
      </button>
      <AnimatePresence>
        {open && (
          <motion.div
            role="menu"
            initial={{ opacity: 0, scale: 0.96 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0, scale: 0.96 }}
              transition={SPRING}
              className="mt-2 w-full origin-top rounded-[12px] border-2 border-ink bg-white p-2 shadow-[4px_4px_0_#1C1A12]"
            >
            {LANGUAGE_OPTIONS.map((o) => {
              const isActive = o.locale === active.locale;
              return (
                <button
                  key={o.locale}
                  type="button"
                  role="menuitem"
                  onClick={() => {
                    setOpen(false);
                    setLocale(o.locale);
                  }}
                  aria-current={isActive || undefined}
                  className={`flex w-full items-center gap-2 rounded-[8px] px-3 py-2 text-left text-sm text-ink transition-colors duration-150 hover:bg-ink/5 ${
                    isActive ? "bg-flash-yellow font-bold" : ""
                  }`}
                >
                  <span className="w-7 shrink-0 font-mono text-xs font-bold">{o.short}</span>
                  <span className="min-w-0 flex-1 truncate">{o.label}</span>
                  {isActive ? <Check className="h-4 w-4 shrink-0" strokeWidth={2.5} aria-hidden="true" /> : null}
                </button>
              );
            })}
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
