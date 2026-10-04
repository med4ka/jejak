"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { Camera, ExternalLink, Download } from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import { DndContext, closestCenter } from "@dnd-kit/core";
import { SortableContext, verticalListSortingStrategy, arrayMove, useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import QrModal from "../components/QrModal";
import EditLinkModal from "../components/EditLinkModal";
import BulkImportModal from "../components/BulkImportModal";
import ShortenForm from "../components/ShortenForm";
import CopyButton from "../components/CopyButton";
import SubmitButton from "../components/SubmitButton";
import { useToast } from "../components/Toast";
import RingkasanTab from "./RingkasanTab";
import AnalyticsTab from "./AnalyticsTab";
import PengaturanTab from "./PengaturanTab";
import ShareModal, { ShareButton } from "../components/ShareModal";
import { THEMES, themeStyles } from "../../lib/themes";
import ThemeBackdrop from "../components/ThemeBackdrop";
import { EASE, SPRING } from "../../lib/animations";
import { shortPath, absoluteShortUrl } from "../../lib/shortlink";
import { useTranslation } from "../../lib/I18nProvider";

// API_BASE (NEXT_PUBLIC_API_URL / http://localhost:8081) REMOVED: public
// page redirect fix, 2026-09-30. Every short URL is built from the
// same-origin /r/{code} path (lib/shortlink), so no backend host leaks into
// the UI, the clipboard, or QR codes.

// Draggable link row. attributes+listeners are attached to the grip handle
// rather than the whole row so the QR/copy/edit buttons stay normally
// tappable. dragDisabled=true while a tag filter is active (a partial
// reorder would collide with the positions of hidden rows: see onDragEnd).
// The hover lift (Apple detail #4: y -4px + shadow 4px → 6px) uses the
// .link-lift CSS utility (globals.css), NOT framer-motion and NOT Tailwind
// hover:* utilities: dnd-kit owns this row's style.transform through inline
// styles, so if framer-motion also drove transform the drag and hover values
// would collide on the same motion value. Inline styles are applied only
// while dnd is active (transform non-null), keeping rows free of transform
// at rest: the CSS lift can then run and every card starts from an
// identical initial state. .link-lift is wrapped in @media (hover:hover) and
// (pointer:fine): on touch devices :hover sticks after a tap, so the touched
// card would stay permanently lifted and rows would look misaligned (the
// "first card sits higher" bug). CSS cannot spring, hence the short 150ms
// duration; the target values (-translate-y-1 = -4px, 6px shadow) still match
// the specification. Expiry status badge above the slug (link management,
// 2026-09-30): "Expired" is permanent (the redirect already answers 410);
// "Ends in Xh" appears only for schedules within 24 hours (yellow warning):
// beyond 24 hours the detail is available in the edit modal and the row does
// not need a permanent badge. status/expires_at come from the backend
// (scanLinks, UTC): the remaining time is computed locally, which is safe
// because the comparison is absolute.
function expiryBadge(link, st, t) {
  if (link.status === "expired") {
    return (
      <span className="mt-1 inline-block rounded-full border border-flash-coral bg-flash-coral/10 px-2 py-px text-[11px] font-bold text-flash-coral">
        {t("dashboard.links.row.badges.expired")}
      </span>
    );
  }
  if (link.status === "scheduled" && link.expires_at) {
    const ms = new Date(link.expires_at).getTime() - Date.now();
    if (ms > 0 && ms <= 24 * 60 * 60 * 1000) {
      const h = Math.max(1, Math.ceil(ms / (60 * 60 * 1000)));
      return (
        <span className="mt-1 inline-block rounded-full border border-ink bg-flash-yellow px-2 py-px text-[11px] font-bold text-ink">
          {t("dashboard.links.row.badges.endsIn", { hours: h })}
        </span>
      );
    }
  }
  return null;
}

// Health-monitor badge (2026-10-04): only the two failure states are
// surfaced — "Broken" (coral) = destination answered 4xx/5xx, "Timeout"
// (yellow) = unreachable/no response. healthy and unknown render NOTHING:
// a healthy link must not grow a permanent badge (noise policy from the
// spec). health_status comes from scanLinks (GET /api/links).
function healthBadge(link) {
  if (link.health_status === "broken") {
    return (
      <span className="mt-1 mr-1 inline-block rounded-full border border-flash-coral bg-flash-coral/10 px-2 py-px text-[11px] font-bold text-flash-coral">
        Broken
      </span>
    );
  }
  if (link.health_status === "timeout") {
    return (
      <span className="mt-1 mr-1 inline-block rounded-full border border-ink bg-flash-yellow px-2 py-px text-[11px] font-bold text-ink">
        Timeout
      </span>
    );
  }
  return null;
}

function SortableLinkRow({ link, onQr, onEdit, onFeature, onToggleActive, dragDisabled, st }) {
  const { t } = useTranslation();
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: link.short_code,
    disabled: dragDisabled,
  });
  const dndActive = isDragging || transform != null;
  return (
    <div
      ref={setNodeRef}
      style={dndActive ? { transform: CSS.Translate.toString(transform), transition } : undefined}
      className={`flex items-center justify-between gap-3 ${st.radius} ${st.borderW} ${st.border} ${st.card} ${st.shadow} p-5 md:p-6 transition duration-150 link-lift ${isDragging ? "opacity-60" : ""} ${!link.is_active ? "opacity-60" : ""}`}
    >
      <span
        {...attributes}
        {...listeners}
        title={dragDisabled ? t("dashboard.links.row.dragDisabledTitle") : t("dashboard.links.row.dragTitle")}
        className={`shrink-0 select-none font-mono text-sm ${st.textMuted} ${dragDisabled ? "cursor-not-allowed opacity-40" : "cursor-grab touch-none"}`}
      >
        ≡
      </span>
      {/* Featured toggle: a radio control, not a checkbox: only one link per
          account can be featured (the server demotes the others in a single
          transaction). ★ when featured, ☆ otherwise. */}
      <button
        onClick={onFeature}
        title={link.is_featured ? t("dashboard.links.row.unfeatureTitle") : t("dashboard.links.row.featureTitle")}
        aria-pressed={link.is_featured}
        className={`shrink-0 select-none text-lg leading-none transition-colors duration-150 ${link.is_featured ? st.text : st.textMuted}`}
      >
        {link.is_featured ? "★" : "☆"}
      </button>
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium">{link.original_url}</p>
        {/* Badges flow inline: expiry, health (broken/timeout only) and the
            password lock (🔒 when the link requires a password before the
            redirect — link.has_password from scanLinks). */}
        {expiryBadge(link, st, t)}
        {healthBadge(link)}
        {link.has_password && (
          <span
            title="Dilindungi password"
            aria-label="Dilindungi password"
            className="mt-1 mr-1 inline-block select-none rounded-full border border-ink bg-paper-grey px-2 py-px text-[11px] leading-normal"
          >
            🔒
          </span>
        )}
        <p className={`font-mono text-xs ${st.textMuted}`}>
          {t("dashboard.links.row.meta", { shortCode: link.short_code, count: link.click_count })}
        </p>
        {(link.tags || []).length > 0 && (
          <div className="mt-1 flex flex-wrap gap-1">
            {(link.tags || []).map((tag) => (
              <span key={tag} className={`rounded-full border ${st.border} px-2 py-px text-[11px] ${st.textMuted}`}>
                {tag}
              </span>
            ))}
          </div>
        )}
      </div>
      <div className="flex shrink-0 gap-1.5">
        <button
          onClick={onEdit}
          title={t("dashboard.links.row.editTooltip")}
          className={`shrink-0 rounded-full ${st.borderW} ${st.border} ${st.card} px-3 py-0.5 text-xs font-bold transition-colors duration-150 hover:opacity-90`}
        >
          {t("dashboard.links.row.edit")}
        </button>
        <CopyButton text={shortPath(link.short_code)} label={t("common.copy")} />
        <button
          onClick={onQr}
          className={`shrink-0 rounded-full ${st.borderW} ${st.border} ${st.card} px-3 py-0.5 text-xs font-bold transition-colors duration-150 hover:opacity-90`}
        >
          {t("dashboard.links.row.qr")}
        </button>
        <button
          onClick={onToggleActive}
          title={link.is_active ? t("dashboard.links.row.deactivateTooltip") : t("dashboard.links.row.activateTooltip")}
          aria-pressed={!!link.is_active}
          className={`shrink-0 rounded-full ${st.borderW} ${st.border} ${st.card} px-3 py-0.5 text-xs font-bold transition-colors duration-150 hover:opacity-90`}
        >
          {link.is_active ? t("dashboard.links.row.active") : t("dashboard.links.row.inactive")}
        </button>
      </div>
    </div>
  );
}

// Creator dashboard (login required): profile edit form
// (display_name, bio, avatar_url, socials): the same fetch pattern as
// AuthModal: JSON request/response, no reload, prefilled from GET
// /api/profile.
export default function DashboardClient() {
  const { t } = useTranslation();
  const [username, setUsername] = useState(null);
  const [checked, setChecked] = useState(false);
  const [displayName, setDisplayName] = useState("");
  const [bio, setBio] = useState("");
  const [avatarUrl, setAvatarUrl] = useState("");
  // Local avatar upload: the file selected by the visitor plus its preview
  // object URL. The file is not sent immediately: preview first, multipart
  // upload on submit.
  const [avatarFile, setAvatarFile] = useState(null);
  const [avatarPreview, setAvatarPreview] = useState("");
  // "Change photo" overlay above the avatar (aria-label "Ganti foto"): shown
  // on hover (desktop) / touch (mobile), driven by state so it stays in sync
  // with the framer-motion AnimatePresence.
  const [avatarHover, setAvatarHover] = useState(false);
  // The hidden file input is triggered through a ref: the avatar is a
  // button, not a label.
  const avatarInputRef = useRef(null);
  const [socials, setSocials] = useState([]);
  const [links, setLinks] = useState([]);
  const [qrCode, setQrCode] = useState(null);
  const [editing, setEditing] = useState(null);
  // ShareModal for the public page: opened from two triggers (Ringkasan and
  // Profil).
  const [shareOpen, setShareOpen] = useState(false);
  const [orderMsg, setOrderMsg] = useState("");
  // Default tab is "Ringkasan" (deliberately not in the URL: a refresh always
  // returns here).
  const [tab, setTab] = useState("ringkasan");
  // Analytics tab range (deviation E, 2026-09-30): the state is held here
  // rather than in AnalyticsTab so changing the range does not reset the
  // other cards, and it can be tracked in the URL via history.replaceState:
  // refreshes and shared links carry the same ?range=. The initial value is
  // validated on mount (see useEffect); values outside 7d/30d/90d are
  // ignored (the backend rejects them with 400).
  const [range, setRange] = useState("30d");
  // Bulk import modal (link management): opened from the "Link saya"
  // toolbar.
  const [bulkOpen, setBulkOpen] = useState(false);
  // Cross-component toast (introduced with Account settings): used for QR
  // ZIP download results and the 200-link limit without moving focus.
  const toast = useToast();
  // true after GET /api/profile succeeds: distinguishes "still loading" from
  // "genuinely empty" for the Ringkasan stats skeletons.
  const [profileReady, setProfileReady] = useState(false);
  const [tagFilter, setTagFilter] = useState("");
  const [notice, setNotice] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  // API keys tab: the key list plus the freshly generated key (shown only
  // once).
  const [apiKeys, setApiKeys] = useState([]);
  const [generatedKey, setGeneratedKey] = useState("");
  const [keyLabel, setKeyLabel] = useState("");
  const [keyLoading, setKeyLoading] = useState(false);
  const [keyMsg, setKeyMsg] = useState("");
  const [keyErr, setKeyErr] = useState("");
  const [apiDocsOpen, setApiDocsOpen] = useState(false);
  const [theme, setTheme] = useState("classic");
  // THEME SCOPE (decision 2026-09-29): the dashboard does NOT follow the
  // creator's theme: body[data-profile-theme] is NOT applied here (only
  // ProfileLinks on /u/[username] may set it), so the dashboard always renders
  // in the Instant Print style. `theme` is retained ONLY for (1) the Profile
  // preset picker and (2) the PUT /api/profile payload. Styling keeps using
  // the classic preset.
  const st = themeStyles("classic");
  const inputThemed = `w-full ${st.radius} ${st.borderW} ${st.border} ${st.card} px-3 py-2 text-sm ${st.placeholder} focus:border-flash-yellow focus:ring-2 focus:ring-flash-yellow/30 focus:outline-hidden transition-all duration-150`;
  const welcomeStep1 = t("dashboard.links.welcome.step1").split("{action}");
  const welcomeStep2 = t("dashboard.links.welcome.step2").split("{action}");

  // Single entry point for opening ShareModal (two triggers: Ringkasan and
  // Profil): also marks onboarding step 3 (localStorage jejak_shared) on the
  // first share.
  function openShare() {
    localStorage.setItem("jejak_shared", "true");
    setShareOpen(true);
  }

  // Change the analytics range: state + URL (deviation E). replaceState, NOT
  // router.push: the URL changes without a Next navigation, so other tabs
  // and their data are not remounted and the back button is not filled with
  // an entry for every pill click.
  function changeRange(next) {
    setRange(next);
    try {
      const url = new URL(window.location.href);
      url.searchParams.set("range", next);
      window.history.replaceState(null, "", url);
    } catch {
      // The URL could not be read (sandboxed or old browser): the range still
      // works through state: the URL exists only for refresh/share purposes.
    }
  }

  useEffect(() => {
    // Range from the URL (deviation E): ?range=7d|30d|90d is applied on
    // mount; other values are ignored so a malformed URL can never trigger a
    // backend 400.
    const r = new URLSearchParams(window.location.search).get("range");
    if (r === "7d" || r === "30d" || r === "90d") {
      setRange(r);
    }

    // Dashboard authentication is the jejak_session COOKIE, NOT
    // localStorage. localStorage.jejak_username can disappear (cleared,
    // different port, different browser) while the session is still valid:
    // that was the source of the false "log in first" message shown even
    // though the navbar (cookieStore) and /api/profile both reported an
    // authenticated session. Single source of truth for BOTH the navbar and
    // the dashboard: GET /api/profile (401 = not logged in, 200 = logged in).
    loadProfile().then((authed) => {
      // API keys are loaded only when the session is valid: that endpoint
      // also answers 401 for anonymous visitors, so calling it earlier is
      // pointless (wasted work and noisy 401s in the logs).
      if (!authed) return;
      fetch("/api/keys", { cache: "no-store" })
        .then(async (res) => { const raw = await res.text(); try { const d = JSON.parse(raw); if (res.ok && Array.isArray(d)) setApiKeys(d); } catch {} })
        .catch(() => {});
    });
  }, []);

  // Load profile + links from GET /api/profile. Used on first open AND after
  // editing a link (so the edited row immediately shows the latest
  // device_rules/tags without a page reload). Since the false "log in first"
  // fix, this function is ALSO the page's auth gate: the session is read from
  // the cookie (credentials "include"), not localStorage. Returns
  // Promise<boolean>: true = valid session (the mount effect uses it to
  // decide whether to load the API keys as well).
  function loadProfile() {
    return fetch("/api/profile", { cache: "no-store", credentials: "include" })
      .then(async (res) => {
        // 401 = no cookie / expired session → show the "Masuk dulu"
        // ("log in first") message. checked is set HERE rather than at the
        // start of mount so the first frame is always the "Memuat..."
        // skeleton: no false "Masuk dulu" flash while the response is still
        // in flight.
        if (res.status === 401) {
          setUsername(null);
          setProfileReady(false);
          setChecked(true);
          return false;
        }
        const text = await res.text();
        let data = {};
        try {
          data = JSON.parse(text);
        } catch {
          throw new Error(text || t("errors.dashboard.loadProfile"));
        }
        if (!res.ok) {
          throw new Error(data.error || t("errors.dashboard.loadProfile"));
        }
        // The username comes from the profile response: the same source the
        // navbar displays (both read /api/profile over the cookie session).
        setUsername(data.username || null);
        setDisplayName(data.display_name || "");
        setBio(data.bio || "");
        setAvatarUrl(data.avatar_url || "");
        setSocials(Array.isArray(data.socials) ? data.socials : []);
        setLinks(Array.isArray(data.links) ? data.links : []);
        setTheme(THEMES.includes(data.theme) ? data.theme : "classic");
        setProfileReady(true);
        setError("");
        setChecked(true);
        return true;
      })
      .catch((err) => {
        // A failed server call does NOT imply a missing session (network
        // outage / 500). The page still renders and the error message is
        // shown in the !username branch rather than an accusation that the
        // visitor is logged out.
        setError(err.message);
        setChecked(true);
        return false;
      });
  }

  function updateSocial(i, field, value) {
    setSocials((prev) => prev.map((s, idx) => (idx === i ? { ...s, [field]: value } : s)));
  }

  // Object URLs from createObjectURL must be revoked when the preview
  // changes or the component unmounts, otherwise memory leaks (the browser
  // keeps the file pinned).
  useEffect(() => () => {
    if (avatarPreview) URL.revokeObjectURL(avatarPreview);
  }, [avatarPreview]);

  // Pick a file through the interactive avatar → show a local preview first.
  // e.target.value is reset so selecting the same file again still fires
  // onChange (a browser file input does not fire the event a second time
  // unless its value is reset to "").
  function handleAvatarFile(e) {
    const f = e.target.files && e.target.files[0];
    e.target.value = "";
    if (!f) return;
    setAvatarFile(f);
    setAvatarPreview(URL.createObjectURL(f));
  }

  function removeSocial(i) {
    setSocials((prev) => prev.filter((_, idx) => idx !== i));
  }

  // API keys load when their tab is opened (lazy, not at profile mount).
  useEffect(() => {
    if (tab === "api-keys") {
      loadKeys();
    }
  }, [tab]);

  function loadKeys() {
    fetch("/api/keys", { cache: "no-store" })
      .then(async (res) => {
        const text = await res.text();
        let data = [];
        try {
          data = JSON.parse(text);
        } catch {
          throw new Error(text || t("errors.dashboard.loadKey"));
        }
        if (!res.ok) {
          throw new Error(data.error || t("errors.dashboard.loadKey"));
        }
        setApiKeys(Array.isArray(data) ? data : []);
      })
      .catch((err) => setKeyErr(err.message));
  }

  // Generate a new key: the plaintext appears only ONCE, in this response.
  async function handleGenerateKey(e) {
    e.preventDefault();
    setKeyErr("");
    setKeyMsg("");
    setGeneratedKey("");
    setKeyLoading(true);
    try {
      const res = await fetch("/api/keys", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ label: keyLabel }),
      });
      const text = await res.text();
      let data = {};
      try {
        data = JSON.parse(text);
      } catch {
        throw new Error(text || t("errors.dashboard.generateKey"));
      }
      if (!res.ok) {
        throw new Error(data.error || t("errors.dashboard.generateKey"));
      }
      setGeneratedKey(data.key);
      setKeyLabel("");
      setKeyMsg(t("dashboard.settings.apiKeys.createdNotice"));
      loadKeys();
    } catch (err) {
      setKeyErr(err.message);
    } finally {
      setKeyLoading(false);
    }
  }

  async function handleDeleteKey(id) {
    if (!window.confirm(t("dashboard.settings.apiKeys.deleteConfirm"))) {
      return;
    }
    setKeyErr("");
    setKeyMsg("");
    try {
      const res = await fetch(`/api/keys/${id}`, { method: "DELETE" });
      const text = await res.text();
      let data = {};
      try {
        data = JSON.parse(text);
      } catch {
        throw new Error(text || t("errors.dashboard.deleteKey"));
      }
      if (!res.ok) {
        throw new Error(data.error || t("errors.dashboard.deleteKey"));
      }
      setKeyMsg(t("dashboard.settings.apiKeys.deletedNotice"));
      loadKeys();
    } catch (err) {
      setKeyErr(err.message);
    }
  }

  function fmtDate(ts) {
    if (!ts) {
      return ": ";
    }
    const d = new Date(ts);
    if (Number.isNaN(d.getTime())) {
      return ": ";
    }
    return d.toLocaleString("id-ID", { dateStyle: "medium", timeStyle: "short" });
  }

  // A new order is applied optimistically in the UI first, then persisted via
  // PUT /api/links/reorder (the server writes in one transaction). If the
  // save fails → the UI reverts to the previous order so it never misleads.
  // Filtering is client-side (the full list is already loaded: no extra
  // endpoint is warranted at this scale). Unique tags are derived from the
  // data, not from static configuration.
  const allTags = [...new Set(links.flatMap((l) => l.tags || []))].sort();
  const visibleLinks = tagFilter === "" ? links : links.filter((l) => (l.tags || []).includes(tagFilter));

  async function onDragEnd(event) {
    const { active, over } = event;
    if (tagFilter !== "" || !over || active.id === over.id) {
      return;
    }
    const prev = links;
    const oldIndex = prev.findIndex((l) => l.short_code === active.id);
    const newIndex = prev.findIndex((l) => l.short_code === over.id);
    const next = arrayMove(prev, oldIndex, newIndex);
    setLinks(next);
    setOrderMsg(t("dashboard.links.order.savingOrder"));
    try {
      const res = await fetch("/api/links/reorder", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ order: next.map((l) => l.short_code) }),
      });
      if (!res.ok) {
        throw new Error(t("errors.dashboard.reorderFailed"));
      }
      setOrderMsg(t("dashboard.links.order.orderSaved"));
    } catch {
      setLinks(prev);
      setOrderMsg(t("dashboard.links.order.orderFailed"));
    }
  }

  // Featured toggle via PUT /api/links/{short_code} {is_featured: bool}:
  // mirrors onDragEnd: update optimistically for responsiveness; if the
  // persist fails, state is reloaded from the server rather than guessed
  // back with a local revert.
  async function toggleFeature(link) {
    const featured = !link.is_featured;
    setLinks((prev) =>
      prev.map((x) =>
        x.short_code === link.short_code
          ? { ...x, is_featured: featured }
          : featured
            ? { ...x, is_featured: false }
            : x
      )
    );
    setOrderMsg(featured ? t("dashboard.links.order.savingFeature") : t("dashboard.links.order.removingFeature"));
    try {
      const res = await fetch(`/api/links/${link.short_code}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ is_featured: featured }),
      });
      if (!res.ok) {
        throw new Error(t("errors.dashboard.changeReverted"));
      }
      setOrderMsg(featured ? t("dashboard.links.order.featureSaved") : t("dashboard.links.order.featureRemoved"));
    } catch {
      loadProfile();
      setOrderMsg(t("dashboard.links.order.failed"));
    }
  }

  // Active/inactive toggle via PUT /api/links/{short_code} {is_active: bool}.
  // Like toggleFeature: update optimistically first, reload server state on
  // failure. A disabled link still appears in the dashboard (the owner must
  // be able to re-enable it) but its public URL stops redirecting (410): see
  // HandleRedirect.
  async function toggleActive(link) {
    const active = !link.is_active;
    setLinks((prev) =>
      prev.map((x) =>
        x.short_code === link.short_code ? { ...x, is_active: active } : x
      )
    );
    setOrderMsg(t("dashboard.links.order.savingStatus"));
    try {
      const res = await fetch(`/api/links/${link.short_code}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ is_active: active }),
      });
      if (!res.ok) {
        throw new Error(t("errors.dashboard.changeReverted"));
      }
      setOrderMsg(active ? t("dashboard.links.order.statusActive") : t("dashboard.links.order.statusInactive"));
    } catch {
      loadProfile();
      setOrderMsg(t("dashboard.links.order.failed"));
    }
  }

  // Download every QR code as a single ZIP (link management): fetch → blob →
  // save, NOT window.location: so a 400 (0 matches / >200 / 401) can be
  // reported through a toast instead of navigating the browser to a JSON
  // error page.
  async function downloadAllQr() {
    const target = tagFilter ? visibleLinks : links;
    if (target.length === 0) {
      toast.error(tagFilter ? t("toast.noLinksWithTag") : t("toast.noLinksToDownload"));
      return;
    }
    if (target.length > 200) {
      toast.error(t("toast.tooManyLinksForQr", { count: target.length }));
      return;
    }
    try {
      const q = tagFilter ? `?tag=${encodeURIComponent(tagFilter)}` : "";
      const res = await fetch(`/api/links/qr-bulk${q}`);
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || t("errors.dashboard.qrZipFailed"));
      }
      const blob = await res.blob();
      const cd = res.headers.get("Content-Disposition") || "";
      const m = cd.match(/filename="([^"]+)"/);
      const a = document.createElement("a");
      a.href = URL.createObjectURL(blob);
      a.download = m ? m[1] : "jejak-qr.zip";
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(a.href);
      toast.success(t("toast.qrDownloaded", { count: target.length }));
    } catch (err) {
      toast.error(err.message);
    }
  }

  function openBulk() {
    setBulkOpen(true);
  }

  // After an import completes: refresh the list (latest badges and click
  // counts) and close the modal: the "Buka Link Saya" button in
  // BulkImportModal calls this.
  function onBulkImported() {
    loadProfile();
    setTab("links");
    setBulkOpen(false);
  }

  async function onSubmit(e) {
    e.preventDefault();
    setError("");
    setNotice("");
    setLoading(true);
    try {
      let finalAvatar = avatarUrl;
      // If a local file was selected, upload it first (multipart) to
      // POST /api/profile/avatar; the response contains the stored path that
      // becomes the final avatar_url. A manually entered URL is overwritten
      // by this file: the file wins because it is the active selection.
      if (avatarFile) {
        const fd = new FormData();
        fd.append("avatar", avatarFile);
        const up = await fetch("/api/profile/avatar", { method: "POST", body: fd });
        const upText = await up.text();
        if (!up.ok) {
          // The two user-actionable failures get localized copy: 413 is the
          // server's 3 MB body cap (MaxBytesReader) and 400 with the
          // magic-byte message is an unsupported format. Other statuses fall
          // back to the upstream text so real server errors stay visible.
          let msg = upText || t("errors.dashboard.uploadPhotoFailed");
          if (up.status === 413) {
            msg = t("errors.dashboard.avatarTooLarge");
          } else if (up.status === 400 && upText.indexOf("Only JPG") !== -1) {
            msg = t("errors.dashboard.avatarBadFormat");
          }
          throw new Error(msg);
        }
        let upData = {};
        try {
          upData = JSON.parse(upText);
        } catch {
          throw new Error(t("errors.dashboard.uploadPhotoFailed"));
        }
        // Without a path in the response the following PUT would silently
        // send avatar_url: undefined (the field drops out of the JSON) and
        // CLEAR the stored avatar — fail loudly instead.
        if (!upData.avatar_url) {
          throw new Error(t("errors.dashboard.uploadPhotoFailed"));
        }
        finalAvatar = upData.avatar_url;
        setAvatarUrl(finalAvatar);
        if (avatarPreview) URL.revokeObjectURL(avatarPreview);
        setAvatarPreview("");
        setAvatarFile(null);
      }
      const res = await fetch("/api/profile", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ display_name: displayName, bio, avatar_url: finalAvatar, socials, theme }),
      });
      const text = await res.text();
      let data = {};
      try {
        data = JSON.parse(text);
      } catch {
        throw new Error(text || t("errors.dashboard.saveProfile"));
      }
      if (!res.ok) {
        throw new Error(data.error || text || t("errors.dashboard.saveProfile"));
      }
      setNotice(t("dashboard.settings.profile.savedNotice"));
      // The navbar holds its OWN profile state (avatar/name) that is only
      // refreshed on navigation (pathname/username). After a save on the
      // dashboard the pathname does not change, so the navbar would keep
      // showing stale data. An event is broadcast so the navbar re-fetches
      // the profile without navigation (see the NavbarClient listener).
      window.dispatchEvent(new Event("jejak:profile-updated"));
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }

  // Loading frame: a skeleton, NOT "Masuk dulu". The logged-in decision may
  // only be made after /api/profile answers (401 → the next branch).
  if (!checked) {
    return (
      <main className={st.text}>
        <h1 className={`${st.headingFont} text-2xl font-bold`}>{t("dashboard.heading")}</h1>
        <div className="mt-6 space-y-4" aria-busy="true" aria-label={t("dashboard.loading.dashboardLabel")}>
          <div className={`${st.radiusLarge} ${st.borderW} ${st.border} ${st.card} h-24 w-full opacity-60`} />
          <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
            <div className={`${st.radiusLarge} ${st.borderW} ${st.border} ${st.card} h-24 opacity-60`} />
            <div className={`${st.radiusLarge} ${st.borderW} ${st.border} ${st.card} h-24 opacity-60`} />
            <div className={`${st.radiusLarge} ${st.borderW} ${st.border} ${st.card} h-24 opacity-60`} />
          </div>
          <p className="text-sm text-muted">{t("dashboard.loading.general")}</p>
        </div>
      </main>
    );
  }

  // No username in state = /api/profile answered 401 (missing cookie /
  // expired session). If the failure has another cause (network, 500), that
  // message is displayed instead of claiming the visitor never logged in.
  if (!username) {
    return (
      <main>
<h1 className="text-center font-display text-2xl font-bold">{t("dashboard.heading")}</h1>
        <p className="mt-2 text-sm text-muted">
          {error !== "" ? error : t("dashboard.notLoggedIn")}
        </p>
      </main>
    );
  }

  return (
    <main className={st.text}>
      <ThemeBackdrop />
      <div className="space-y-6 md:space-y-8">
        <h1 className={`${st.headingFont} text-2xl font-bold`}>{t("dashboard.heading")}</h1>

      {/* Tab bar pills stay on a single row: horizontal scrolling on mobile
          instead of wrapping, so the bar never grows in height. */}
      <div className="mb-6 md:mb-8 flex overflow-x-auto rounded-full border-2 border-ink bg-white p-1">
        {[
          { id: "ringkasan", label: t("dashboard.tabs.ringkasan") },
          { id: "links", label: t("dashboard.tabs.linkSaya") },
          { id: "analytics", label: t("dashboard.tabs.analytics") },
          { id: "profil", label: t("dashboard.tabs.profile") },
          { id: "api-keys", label: t("dashboard.tabs.apiKeys") },
          { id: "pengaturan", label: t("dashboard.tabs.settings") },
        ].map((tabItem) => (
          <button
            key={tabItem.id}
            onClick={() => setTab(tabItem.id)}
            aria-current={tab === tabItem.id ? "page" : undefined}
            className={`relative shrink-0 whitespace-nowrap rounded-full px-4 py-2 text-sm transition-colors duration-150 ease-out ${
              tab === tabItem.id ? "bg-flash-yellow font-bold text-ink" : "text-ink/60 hover:text-ink"
            }`}
          >
            {tabItem.label}
          </button>
        ))}
      </div>

      <AnimatePresence mode="wait">
        <motion.div
          key={tab}
          initial={{ opacity: 0, y: 4 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.2, ease: EASE }}
          className="space-y-4"
        >
      {tab === "ringkasan" && (
        <RingkasanTab
          displayName={displayName}
          links={links}
          profileReady={profileReady}
          onShare={openShare}
          onGoProfil={() => setTab("profil")}
        />
      )}
      {tab === "analytics" && (
        <AnalyticsTab st={st} range={range} onRange={changeRange} />
      )}
      {tab === "profil" && (
      <>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className={`text-sm ${st.text}`}>
          {t("dashboard.settings.profile.publicPageLabel")}{" "}
          <Link href={`/u/${username}`} className="font-medium text-flash-coral underline transition-opacity duration-150 hover:opacity-70">
            /u/{username}
          </Link>
        </p>
        <div className="flex flex-wrap items-center gap-2">
          <a
            href={`/u/${username}`}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex shrink-0 items-center gap-1.5 rounded-full border-2 border-ink bg-white px-3 py-1.5 text-xs font-bold text-ink transition-transform duration-150 hover:-translate-y-0.5 focus:outline-hidden focus-visible:ring-2 focus-visible:ring-flash-yellow"
          >
            <ExternalLink size={14} aria-hidden="true" />
            {t("dashboard.settings.profile.viewProfile")}
          </a>
          <ShareButton onClick={openShare} />
        </div>
      </div>

      <form onSubmit={onSubmit} className="flex flex-col gap-4">
        <label className="text-sm font-medium">
          {t("dashboard.settings.profile.displayNameLabel")}
          <input value={displayName} onChange={(e) => setDisplayName(e.target.value)} required maxLength={100} className={`${inputThemed} mt-1`} />
        </label>
        <label className="text-sm font-medium">
          {t("dashboard.settings.profile.bioLabel")}
          <textarea value={bio} onChange={(e) => setBio(e.target.value)} maxLength={500} rows={3} className={`${inputThemed} mt-1`} />
        </label>
        <div>
          <label htmlFor="avatar" className="block text-sm font-medium">
            {t("dashboard.settings.profile.avatarLabel")}
          </label>
          {/* The avatar is one interactive element: photo/initial as the base,
              with a dark "Ganti foto" overlay on hover (desktop) / touch
              (mobile); clicking anywhere on it triggers the hidden file
              input (id="avatar" matches the label above the circle).
              Empty state = 96x96 circle with a dashed ink border so the
              placeholder never competes with the label. Files are validated
              server-side (magic bytes, jpg/png/webp, 2MB max); the client
              preview is for UX only, security always lives on the server. */}
          <div className="mt-2 flex items-center gap-3">
            <button
              type="button"
              onClick={() => avatarInputRef.current?.click()}
              onMouseEnter={() => setAvatarHover(true)}
              onMouseLeave={() => setAvatarHover(false)}
              onFocus={() => setAvatarHover(true)}
              onBlur={() => setAvatarHover(false)}
              onTouchStart={() => setAvatarHover(true)}
              aria-label={t("dashboard.settings.profile.avatarChange")}
              title={t("dashboard.settings.profile.avatarChange")}
              className={`relative flex h-24 w-24 shrink-0 items-center justify-center overflow-hidden rounded-full focus:outline-hidden focus-visible:ring-2 focus-visible:ring-flash-coral ${
                avatarPreview !== "" || avatarUrl.trim() !== ""
                  ? `${st.borderW} ${st.border} ${st.card}`
                  : "border-2 border-dashed border-ink"
              }`}
            >
              {avatarPreview !== "" || avatarUrl.trim() !== "" ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img
                  src={avatarPreview || avatarUrl}
                  alt={t("dashboard.settings.profile.avatarAlt")}
                  className="h-full w-full object-cover"
                />
              ) : (
                <span className={`font-display text-2xl font-bold ${st.text}`}>
                  {(displayName.trim() !== "" ? displayName : username).slice(0, 1).toUpperCase()}
                </span>
              )}
              <AnimatePresence>
                {(avatarHover || avatarFile) && (
                  <motion.span
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    exit={{ opacity: 0 }}
                    transition={SPRING}
                    className="absolute inset-0 flex flex-col items-center justify-center gap-0.5 bg-ink/60 text-print-white"
                  >
                    <Camera className="h-5 w-5" aria-hidden="true" />
                    <span className="text-[10px] font-bold">{t("dashboard.settings.profile.avatarChange")}</span>
                  </motion.span>
                )}
              </AnimatePresence>
            </button>
            <input ref={avatarInputRef} id="avatar" type="file" accept="image/jpeg,image/png,image/webp" onChange={handleAvatarFile} className="hidden" />
            {avatarFile && <p className="text-xs text-muted">{t("dashboard.settings.profile.avatarFileHint")}</p>}
          </div>
          {/* The manual URL option is deliberately kept as a separate
              alternative. */}
          <label className={`mt-2 block text-xs font-medium ${st.textMuted}`}>
            {t("dashboard.settings.profile.avatarUrlLabel")}
            <input value={avatarUrl} onChange={(e) => setAvatarUrl(e.target.value)} placeholder={t("dashboard.settings.profile.avatarUrlPlaceholder")} className={`${inputThemed} mt-1`} />
          </label>
        </div>

        <div className="mt-2">
          <p className="text-sm font-medium">{t("dashboard.settings.profile.socialsLabel")}</p>
          <div className="mt-2 flex flex-col gap-2">
            {socials.map((s, i) => (
              <div key={i} className="flex gap-2">
                <input value={s.platform} onChange={(e) => updateSocial(i, "platform", e.target.value)} placeholder={t("dashboard.settings.profile.socialPlatformPlaceholder")} maxLength={30} className={`${inputThemed}`} />
                <input value={s.url} onChange={(e) => updateSocial(i, "url", e.target.value)} placeholder={t("dashboard.settings.profile.socialUrlPlaceholder")} className={`${inputThemed}`} />
                <button type="button" onClick={() => removeSocial(i)} aria-label={t("dashboard.settings.profile.removeSocialAria")} className={`w-9 shrink-0 rounded-full ${st.borderW} ${st.border} ${st.card} px-0 text-sm font-bold`}>
                  ×
                </button>
              </div>
            ))}
          </div>
          {socials.length < 10 && (
            <button
              type="button"
              onClick={() => setSocials((prev) => [...prev, { platform: "", url: "" }])}
              className={`mt-2 rounded-full ${st.borderW} ${st.border} ${st.card} px-4 py-1.5 text-sm font-medium`}
            >
              {t("dashboard.settings.profile.addSocial")}
            </button>
          )}
        </div>

        <div className="mt-2">
          <p className="text-sm font-medium">{t("dashboard.settings.profile.themeLabel")}</p>
          <div className="mt-2 flex flex-wrap gap-2">
            {THEMES.map((key) => {
              const st = themeStyles(key);
              const active = theme === key;
              // The GLASS portion of st.card is transparent (bg-white/[8%])
              // because it targets the dark public page. Used directly as the
              // picker BUTTON background, the Glass chip would be
              // white-on-white on light dashboard themes (classic/coral) and
              // disappear from the picker. st.chip = a SOLID per-preset
              // swatch that always contrasts.
              return (
                <button
                  key={key}
                  type="button"
                  onClick={() => setTheme(key)}
                  aria-pressed={active}
                  aria-label={t("dashboard.settings.profile.themeOptionAria", { themeName: st.label })}
                  className={`w-28 ${st.radius} ${st.borderW} p-2 text-left transition-colors duration-150 ${st.chip} ${
                    active ? "ring-2 ring-flash-yellow ring-offset-1" : "opacity-75 hover:opacity-100"
                  }`}
                >
                  <div className={`rounded-md border ${st.border} p-1.5`}>
                    {st.swatch ? (
                      // RisoPrint (optional token): 3 ink strips: ink /
                      // primary accent / secondary accent. Colors are applied
                      // through inline style: a dynamic bg-[${hex}] class
                      // cannot be scanned by Tailwind (purge).
                      <div className="flex gap-1">
                        {st.swatch.map((c) => (
                          <div key={c} className="h-4 flex-1 rounded-xs" style={{ backgroundColor: c }} />
                        ))}
                      </div>
                    ) : (
                      <>
                        <div className={`h-2 w-10 rounded-xs ${st.avatar}`} />
                        <div className="mt-1 h-1 w-full rounded-xs bg-current opacity-50" />
                        <div className="mt-0.5 h-1 w-12 rounded-xs bg-current opacity-30" />
                      </>
                    )}
                  </div>
                  <p className="mt-1.5 text-xs font-bold">{st.label}</p>
                </button>
              );
            })}
          </div>
        </div>

        <SubmitButton
          isLoading={loading}
          loadingLabel={t("dashboard.actions.saving")}
          className={`mt-2 rounded-full border-2 border-ink ${st.accent} px-4 py-2.5 text-sm font-bold transition-[filter] duration-150 hover:brightness-95`}
        >
          {t("dashboard.settings.profile.saveButton")}
        </SubmitButton>
      </form>

      {notice !== "" && (
        <p className={`${st.radius} ${st.borderW} ${st.border} ${st.card} px-4 py-2.5 text-sm font-medium`}>{notice}</p>
      )}
      {error !== "" && (
        <p className={`${st.radius} ${st.borderW} ${st.border} ${st.card} px-4 py-2.5 text-sm font-medium`}>{error}</p>
      )}
      </>
      )}
      {tab === "api-keys" && (
      <>
      <button
        type="button"
        onClick={() => setApiDocsOpen(!apiDocsOpen)}
        className={`flex items-center gap-1 text-sm font-medium ${st.textMuted} transition-colors duration-150`}
      >
        {t("dashboard.settings.apiKeys.docsToggle")}
        <motion.span
          animate={{ rotate: apiDocsOpen ? 180 : 0 }}
          transition={SPRING}
          className="inline-block"
        >
          ↓
        </motion.span>
      </button>
      <AnimatePresence initial={false}>
        {apiDocsOpen && (
          <motion.div
            key="api-docs"
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={SPRING}
            className="overflow-hidden"
          >
            <p className={`mt-2 text-sm ${st.textMuted}`}>{t("dashboard.settings.apiKeys.docsExampleIntro")}</p>
            <pre className={`mt-2 overflow-x-auto ${st.radius} ${st.borderW} ${st.border} ${st.card} px-4 py-3 font-mono text-sm leading-relaxed`}>
              <code>{`curl -X POST https://jejak.app/api/v1/shorten \\
  -H "Authorization: Bearer jjk_xxxxx" \\
  -H "Content-Type: application/json" \\
  -d '{"url": "https://example.com"}'`}</code>
            </pre>
          </motion.div>
        )}
      </AnimatePresence>

      <form onSubmit={handleGenerateKey} className="flex flex-col gap-2">
        <label htmlFor="key-label" className="text-sm font-medium">
          {t("dashboard.settings.apiKeys.labelLabel")}
          <input
            id="key-label"
            value={keyLabel}
            onChange={(e) => setKeyLabel(e.target.value)}
            maxLength={100}
            placeholder={t("dashboard.settings.apiKeys.labelPlaceholder")}
            className={`${inputThemed} mt-1`}
          />
        </label>
        <SubmitButton
          isLoading={keyLoading}
          loadingLabel={t("dashboard.actions.creating")}
          className={`rounded-full border-2 border-ink ${st.accent} px-4 py-2.5 text-sm font-bold transition-[filter] duration-150 hover:brightness-95`}
        >
          {t("dashboard.settings.apiKeys.createButton")}
        </SubmitButton>
      </form>

      {generatedKey !== "" && (
        <div className={`${st.radius} ${st.borderW} ${st.border} ${st.card} px-4 py-3`}>
          <p className="text-sm font-bold">{t("dashboard.settings.apiKeys.generatedWarning")}</p>
          <div className="mt-2 flex flex-wrap items-center gap-2">
            <code className="min-w-0 break-all font-mono text-sm">{generatedKey}</code>
            <CopyButton text={generatedKey} label={t("common.copy")} />
          </div>
        </div>
      )}

      {keyMsg !== "" && (
        <p className={`${st.radius} ${st.borderW} ${st.border} ${st.card} px-4 py-2.5 text-sm font-medium`}>{keyMsg}</p>
      )}
      {keyErr !== "" && (
        <p className={`${st.radius} ${st.borderW} ${st.border} ${st.card} px-4 py-2.5 text-sm font-medium`}>{keyErr}</p>
      )}

      <section>
        <h2 className={`${st.headingFont} text-lg font-bold mb-4`}>{t("dashboard.settings.apiKeys.activeKeysTitle")}</h2>
        {apiKeys.length === 0 ? (
          <p className={`text-sm text-muted`}>{t("dashboard.emptyStates.noKeys")}</p>
        ) : (
          <div className="flex flex-col gap-4">
            {apiKeys.map((k) => (
              <div key={k.id} className={`flex items-center justify-between gap-3 ${st.radius} ${st.borderW} ${st.border} ${st.card} p-5 md:p-6`}>
                <div className="min-w-0">
                  <p className="truncate text-sm font-medium">{k.label !== "" ? k.label : t("dashboard.settings.apiKeys.noLabel")}</p>
                  <p className={`font-mono text-xs ${st.textMuted}`}>
                    {t("dashboard.settings.apiKeys.keyMeta", { createdAt: fmtDate(k.created_at), lastUsedAt: fmtDate(k.last_used_at) })}
                  </p>
                </div>
                <button
                  onClick={() => handleDeleteKey(k.id)}
                  className={`shrink-0 rounded-full ${st.borderW} ${st.border} ${st.card} px-3 py-0.5 text-xs font-bold transition-colors duration-150 hover:opacity-90`}
                >
                  {t("dashboard.settings.apiKeys.deleteButton")}
                </button>
              </div>
            ))}
          </div>
        )}
      </section>
      </>
      )}
      {tab === "pengaturan" && (
        <PengaturanTab st={st} />
      )}
      {tab === "links" && (
      <>
      {/* Stage D: the shorten form sits inline at the top of the tab so an
          authenticated visitor can create a new link WITHOUT leaving the
          dashboard. `st` is passed so the form follows the active dashboard
          theme tokens; onSuccess calls loadProfile() (the existing
          link-loading function) so the new link appears in the list below
          WITHOUT a full page reload. */}
      <ShortenForm st={st} onSuccess={() => loadProfile()} />
      {links.length === 0 && bio === "" && apiKeys.length === 0 && (
          <div className={`${st.radius} ${st.borderW} ${st.border} ${st.card} p-5 md:p-6`}>
          <p className={`${st.headingFont} text-lg font-bold mb-4`}>{t("dashboard.links.welcome.heading")}</p>
          <ol className="flex flex-col gap-2 text-sm">
            <li>
              {welcomeStep1[0]}
              <button
                type="button"
                onClick={() => window.scrollTo({ top: 0, behavior: "smooth" })}
                className="font-medium text-flash-coral underline transition-colors duration-150 hover:text-ink"
              >
                {t("dashboard.links.welcome.step1Action")}
              </button>
              {welcomeStep1[1]}
            </li>
            <li>
              {welcomeStep2[0]}
              <button
                type="button"
                onClick={() => setTab("profil")}
                className="font-medium text-flash-coral underline transition-colors duration-150 hover:text-ink"
              >
                {t("dashboard.links.welcome.step2Action")}
              </button>
              {welcomeStep2[1]}
            </li>
          </ol>
        </div>
      )}
      <section>
        <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
          <h2 className={`${st.headingFont} text-lg font-bold`}>{t("dashboard.links.title")}</h2>
          {/* Bulk actions (link management): mass import + download all QR
              codes. Import runs in a modal; QR download goes through
              fetch→blob in downloadAllQr so failures surface as toasts. */}
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={openBulk}
              className="inline-flex shrink-0 items-center gap-1.5 rounded-full border-2 border-ink bg-white px-3 py-1.5 text-xs font-bold text-ink transition-transform duration-150 hover:-translate-y-0.5 focus:outline-hidden focus-visible:ring-2 focus-visible:ring-flash-yellow"
            >
              {t("dashboard.links.toolbar.bulkImport")}
            </button>
            <button
              type="button"
              onClick={downloadAllQr}
              title={t("dashboard.links.toolbar.qrDownloadTitle")}
              className="inline-flex shrink-0 items-center gap-1.5 rounded-full border-2 border-ink bg-white px-3 py-1.5 text-xs font-bold text-ink transition-transform duration-150 hover:-translate-y-0.5 focus:outline-hidden focus-visible:ring-2 focus-visible:ring-flash-yellow"
            >
              <Download size={14} aria-hidden="true" />
              {t("dashboard.links.toolbar.qrDownload")}
            </button>
          </div>
        </div>
        {allTags.length > 0 && (
          <div className="mb-4 flex items-center gap-2 text-sm">
            <label htmlFor="tag-filter" className={st.textMuted}>{t("dashboard.links.filter.label")}</label>
            <select
              id="tag-filter"
              value={tagFilter}
              onChange={(e) => setTagFilter(e.target.value)}
              className={`rounded-full ${st.borderW} ${st.border} ${st.card} px-3 py-1 text-sm focus:border-flash-yellow focus:ring-2 focus:ring-flash-yellow/30 focus:outline-hidden transition-all duration-150`}
            >
              <option value="">{t("dashboard.links.filter.allOption")}</option>
              {allTags.map((tag) => (
                <option key={tag} value={tag}>{tag}</option>
              ))}
            </select>
          </div>
        )}
        {links.length === 0 ? (
          <p className={`text-sm ${st.textMuted}`}>{t("dashboard.emptyStates.noLinks")}</p>
        ) : visibleLinks.length === 0 ? (
          <p className={`text-sm ${st.textMuted}`}>{t("dashboard.emptyStates.noTagMatch")}</p>
        ) : (
          <DndContext collisionDetection={closestCenter} onDragEnd={onDragEnd}>
            <SortableContext items={visibleLinks.map((l) => l.short_code)} strategy={verticalListSortingStrategy}>
              <div className="flex flex-col gap-4">
                {visibleLinks.map((l) => (
                  <SortableLinkRow
                    key={l.short_code}
                    link={l}
                    onQr={() => setQrCode(l.short_code)}
                    onEdit={() => setEditing(l)}
                    onFeature={() => toggleFeature(l)}
                    onToggleActive={() => toggleActive(l)}
                    dragDisabled={tagFilter !== ""}
                    st={st}
                  />
                ))}
              </div>
            </SortableContext>
          </DndContext>
        )}
        {orderMsg !== "" && (
          <p className={`text-sm ${st.textMuted}`}>{orderMsg}</p>
        )}
        </section>
      </>
      )}
        </motion.div>
      </AnimatePresence>
      </div>

      <AnimatePresence>
        {qrCode !== null && (
          <QrModal
            shortCode={qrCode}
            // The QR needs an absolute URL (not a path): use the BROWSER
            // origin, not the backend: a scanned code must open the page's
            // origin (through the Next proxy), never localhost:8081.
            shortUrl={absoluteShortUrl(qrCode)}
            onClose={() => setQrCode(null)}
            st={st}
          />
        )}
        {editing !== null && (
          <EditLinkModal
            key={editing.short_code}
            link={editing}
            onClose={() => setEditing(null)}
            onSaved={loadProfile}
            st={st}
          />
        )}
        {bulkOpen && (
          <BulkImportModal st={st} onClose={() => setBulkOpen(false)} onImported={onBulkImported} />
        )}
        {shareOpen && (
          <ShareModal username={username} onClose={() => setShareOpen(false)} />
        )}
      </AnimatePresence>
    </main>
  );
}
