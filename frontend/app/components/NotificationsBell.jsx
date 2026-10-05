"use client";

// =====================================================================
// NOTIFICATIONS BELL (navbar desktop): unread badge + dropdown feed fed by
// GET /api/notifications (in-app only: the backend has no email/SMS
// infrastructure, see PROGRESS). An item is marked read via
// PUT /api/notifications/{id}/read and drops out of the unread count.
// Feed state lives in lib/useNotifications (shared with the mobile
// NotificationsModal). Copy comes from messages/*.json via useTranslation
// (tr); the `t` prop carries THEME TOKENS (t.navCircle, t.panel, ...).
// =====================================================================
import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { Bell } from "lucide-react";
import { AnimatePresence, motion } from "framer-motion";
import { formatLocal } from "../../lib/expiry";
import { EASE } from "../../lib/animations";
import { useNotifications } from "../../lib/useNotifications";
import { useTranslation } from "../../lib/I18nProvider";

export default function NotificationsBell({ t }) {
  const { t: tr } = useTranslation();
  const { unread, items, loading, loadList, markRead, refreshUnread } = useNotifications();
  const [open, setOpen] = useState(false);
  const wrapRef = useRef(null);
  const router = useRouter();

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

  async function toggle() {
    const next = !open;
    setOpen(next);
    if (next) {
      loadList();
    } else {
      // Re-sync the badge after reading (or missing) items.
      refreshUnread();
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

  const panel = `absolute right-0 top-full z-50 mt-2 w-[min(88vw,340px)] origin-top-right ${t.radiusLarge ?? ""} ${t.panel ?? ""} border-2 border-ink`;

  return (
    <div ref={wrapRef} className="relative">
      <button
        type="button"
        onClick={toggle}
        aria-expanded={open}
        aria-haspopup="true"
        aria-label={unread > 0 ? tr("notifications.ariaWithCount", { count: unread }) : tr("notifications.title")}
        className={`relative flex h-10 w-10 items-center justify-center rounded-full ${t.navCircle ?? ""}`}
      >
        <Bell className="h-5 w-5" strokeWidth={2.5} aria-hidden="true" />
        {unread > 0 && (
          <span className="-right-1 -top-1 absolute flex h-5 min-w-5 items-center justify-center rounded-full border-2 border-ink bg-flash-coral px-1 font-mono text-[10px] font-bold leading-none text-print-white">
            {unread > 9 ? "9+" : unread}
          </span>
        )}
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
                {tr("notifications.title")}
              </span>
              <button
                type="button"
                onClick={loadList}
                className={`text-xs font-bold ${t.textMuted ?? "text-muted"} hover:opacity-70`}
              >
                {tr("notifications.reload")}
              </button>
            </div>
            <div className={`my-1 border-t ${t.panelRule ?? ""}`} />
            {loading ? (
              <p className={`px-2 py-3 text-sm ${t.textMuted ?? "text-muted"}`}>{tr("notifications.loading")}</p>
            ) : items.length === 0 ? (
              <p className={`px-2 py-3 text-sm ${t.textMuted ?? "text-muted"}`}>{tr("notifications.empty")}</p>
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
                          aria-label={tr("notifications.unreadAria")}
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
