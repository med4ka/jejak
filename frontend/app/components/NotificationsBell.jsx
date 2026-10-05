"use client";

// =====================================================================
// NOTIFICATIONS BELL (navbar, health monitor): unread badge + dropdown
// feed fed by GET /api/notifications (in-app only: the backend has no
// email/SMS infrastructure, see PROGRESS). An item is marked read via
// PUT /api/notifications/{id}/read and drops out of the unread count.
// Two placements share this component:
//   - variant "icon"   : round bell button, absolute dropdown (desktop nav)
//   - variant "drawer" : full-width row button, in-flow panel (mobile)
// Copy is hard-coded Indonesian (i18n dictionaries off-limits for the
// health-monitor task, same policy as EditLinkModal/ShortenForm).
// =====================================================================
import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { Bell } from "lucide-react";
import { AnimatePresence, motion } from "framer-motion";
import { formatLocal } from "../../lib/expiry";
import { EASE } from "../../lib/animations";

export default function NotificationsBell({ t, variant = "icon", triggerClassName = "" }) {
  const [unread, setUnread] = useState(0);
  const [open, setOpen] = useState(false);
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(false);
  const wrapRef = useRef(null);
  const router = useRouter();

  // Unread count on mount (cheap single-row query; failures stay silent:
  // the bell is decoration, never a blocker for navigation).
  useEffect(() => {
    let alive = true;
    (async () => {
      try {
        const res = await fetch("/api/notifications");
        if (!res.ok || !alive) return;
        const data = await res.json();
        if (alive && typeof data.unread === "number") setUnread(data.unread);
      } catch {
        // network/offline: keep the previous count
      }
    })();
    return () => {
      alive = false;
    };
  }, []);

  // Outside click / Escape closes the dropdown, attached only while open.
  useEffect(() => {
    if (!open) return;
    function onDoc(e) {
      if (wrapRef.current && !wrapRef.current.contains(e.target)) setOpen(false);
    }
    function onKey(e) {
      if (e.key === "Escape") setOpen(false);
    }
    document.addEventListener("mousedown", onDoc);
    document.addEventListener("touchstart", onDoc, { passive: true });
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDoc);
      document.removeEventListener("touchstart", onDoc);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  async function loadList() {
    setLoading(true);
    try {
      const res = await fetch("/api/notifications");
      if (!res.ok) return;
      const data = await res.json();
      if (Array.isArray(data.notifications)) setItems(data.notifications);
      if (typeof data.unread === "number") setUnread(data.unread);
    } catch {
      // keep the previous list
    } finally {
      setLoading(false);
    }
  }

  async function toggle() {
    const next = !open;
    setOpen(next);
    if (next) {
      loadList();
    } else {
      // Re-sync the badge after reading (or missing) items.
      try {
        const res = await fetch("/api/notifications");
        if (res.ok) {
          const data = await res.json();
          if (typeof data.unread === "number") setUnread(data.unread);
        }
      } catch {
        // ignore
      }
    }
  }

  async function markRead(n) {
    if (n.read) return;
    // Optimistic: flip locally first so the row de-bolds instantly; roll the
    // badge back on failure.
    setItems((prev) => prev.map((x) => (x.id === n.id ? { ...x, read: true } : x)));
    setUnread((u) => Math.max(0, u - 1));
    try {
      const res = await fetch(`/api/notifications/${n.id}/read`, { method: "PUT" });
      if (!res.ok) {
        setItems((prev) => prev.map((x) => (x.id === n.id ? { ...x, read: false } : x)));
        setUnread((u) => u + 1);
      }
    } catch {
      setItems((prev) => prev.map((x) => (x.id === n.id ? { ...x, read: false } : x)));
      setUnread((u) => u + 1);
    }
  }

  async function onItem(n) {
    await markRead(n);
    // The batch's health_aggregate row ("Dan N link lainnya...") is the
    // entry point into the broken-only dashboard view (health.go cap: a
    // user gets at most 5 individual + 1 aggregate rows per batch).
    if (n.type === "health_aggregate") {
      setOpen(false);
      router.push("/dashboard?filter=broken");
    }
  }

  const isDrawer = variant === "drawer";
  const trigger =
    variant === "icon"
      ? `relative flex h-10 w-10 items-center justify-center rounded-full ${t.navCircle ?? ""}`
      : `${triggerClassName} justify-between`;

  const panel = isDrawer
    ? `mt-1 w-full overflow-hidden rounded-xl border-2 border-ink ${t.panel ?? ""}`
    : `absolute right-0 top-full z-50 mt-2 w-[min(88vw,340px)] origin-top-right ${t.radiusLarge ?? ""} ${t.panel ?? ""} border-2 border-ink`;

  return (
    <div ref={wrapRef} className={isDrawer ? "relative" : "relative"}>
      <button
        type="button"
        onClick={toggle}
        aria-expanded={open}
        aria-haspopup="true"
        aria-label={`Notifikasi${unread > 0 ? ` (${unread} belum dibaca)` : ""}`}
        className={trigger}
      >
        <Bell className="h-5 w-5" strokeWidth={2.5} aria-hidden="true" />
        {unread > 0 && (
          <span
            className={`absolute flex items-center justify-center rounded-full border-2 border-ink bg-flash-coral px-1 font-mono text-[10px] font-bold leading-none text-print-white ${
              isDrawer ? "right-8 top-1/2 -translate-y-1/2" : "-right-1 -top-1 h-5 min-w-5"
            }`}
          >
            {unread > 9 ? "9+" : unread}
          </span>
        )}
        {isDrawer && <span>Notifikasi{unread > 0 ? ` (${unread})` : ""}</span>}
      </button>

      <AnimatePresence>
        {open && (
          <motion.div
            initial={{ opacity: 0, y: -6 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -6 }}
            transition={{ duration: 0.15, ease: EASE }}
            className={`${panel} p-2`}
          >
            <div className="flex items-center justify-between px-2 py-1">
              <span className={`text-xs font-bold uppercase tracking-widest ${t.textMuted ?? "text-muted"}`}>
                Notifikasi
              </span>
              <button
                type="button"
                onClick={loadList}
                className={`text-xs font-bold ${t.textMuted ?? "text-muted"} hover:opacity-70`}
              >
                Muat ulang
              </button>
            </div>
            <div className={`my-1 border-t ${t.panelRule ?? ""}`} />
            {loading ? (
              <p className={`px-2 py-3 text-sm ${t.textMuted ?? "text-muted"}`}>Memuat...</p>
            ) : items.length === 0 ? (
              <p className={`px-2 py-3 text-sm ${t.textMuted ?? "text-muted"}`}>Belum ada notifikasi.</p>
            ) : (
              <ul className="max-h-72 overflow-y-auto">
                {items.map((n) => (
                  <li key={n.id}>
                    <button
                      type="button"
                      onClick={() => onItem(n)}
                      className={`flex w-full items-start gap-2 rounded-lg px-2 py-2 text-left text-sm transition-colors duration-150 ${t.panelHover ?? "hover:bg-black/5"} ${
                        n.read ? t.textMuted ?? "text-muted" : `${t.text ?? "text-ink"} font-semibold`
                      }`}
                    >
                      {!n.read && (
                        <span
                          className="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-flash-coral"
                          aria-label="Belum dibaca"
                        />
                      )}
                      <span className="min-w-0">
                        <span className="block break-words">{n.message}</span>
                        <span className={`mt-0.5 block font-mono text-[10px] ${t.textMuted ?? "text-muted"}`}>
                          {n.short_code ? `/${n.short_code} · ${formatLocal(n.created_at)}` : formatLocal(n.created_at)}
                        </span>
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
