"use client";

// =====================================================================
// NOTIFICATIONS MODAL: the mobile drawer's "Notifikasi" destination.
// Same feed as the desktop bell (lib/useNotifications: unread badge,
// lazy list, optimistic mark-read) but in a centered modal instead of an
// inline drawer panel, so the drawer stays a short menu and the feed gets
// full-width room. Aggregate rows navigate to /dashboard?filter=broken.
// =====================================================================
import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { AnimatePresence, motion } from "framer-motion";
import { Bell, X } from "lucide-react";
import { formatLocal } from "../../lib/expiry";
import { EASE } from "../../lib/animations";
import { useNotifications } from "../../lib/useNotifications";
import { useTranslation } from "../../lib/I18nProvider";

export default function NotificationsModal({ open, onClose, t }) {
  const { t: tr } = useTranslation();
  const { unread, items, loading, loadList, markRead } = useNotifications();
  const router = useRouter();

  // Load the feed when the modal opens; lock body scroll while it is open.
  useEffect(() => {
    if (!open) return;
    loadList();
    document.body.style.overflow = "hidden";
    function onKey(e) {
      if (e.key === "Escape") onClose();
    }
    document.addEventListener("keydown", onKey);
    return () => {
      document.body.style.overflow = "";
      document.removeEventListener("keydown", onKey);
    };
  }, [open, loadList, onClose]);

  async function onItem(n) {
    await markRead(n);
    if (n.type === "health_aggregate") {
      onClose();
      router.push("/dashboard?filter=broken");
    }
  }

  return (
    <AnimatePresence>
      {open && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.2, ease: EASE }}
          onClick={onClose}
          className="fixed inset-0 z-[70] flex items-center justify-center bg-ink/60 p-4"
        >
          <motion.div
            role="dialog"
            aria-modal="true"
            aria-label={tr("notifications.title")}
            initial={{ opacity: 0, y: 16, scale: 0.98 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: 12, scale: 0.98 }}
            transition={{ duration: 0.2, ease: EASE }}
            onClick={(e) => e.stopPropagation()}
            className={`max-h-[80vh] w-full max-w-sm overflow-hidden rounded-xl border-2 border-ink ${t.panel ?? "bg-white"} shadow-[6px_6px_0px_#1C1A12]`}
          >
            <div className="flex items-center justify-between px-4 py-3">
              <span className={`flex items-center gap-2 text-sm font-bold ${t.text ?? "text-ink"}`}>
                <Bell className="h-4 w-4" strokeWidth={2.5} aria-hidden="true" />
                {tr("notifications.title")}
                {unread > 0 && (
                  <span className="flex h-5 min-w-5 items-center justify-center rounded-full bg-flash-coral px-1 font-mono text-[10px] font-bold leading-none text-white">
                    {unread > 9 ? "9+" : unread}
                  </span>
                )}
              </span>
              <button
                type="button"
                onClick={onClose}
                aria-label={tr("nav.aria.closeMenu")}
                className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-ink/5"
              >
                <X className="h-4 w-4 text-ink" strokeWidth={2.5} aria-hidden="true" />
              </button>
            </div>
            <div className={`border-t ${t.panelRule ?? "border-ink/10"}`} />
            <div className="max-h-[60vh] overflow-y-auto p-2">
              {loading ? (
                <p className={`px-2 py-3 text-sm ${t.textMuted ?? "text-muted"}`}>{tr("notifications.loading")}</p>
              ) : items.length === 0 ? (
                <p className={`px-2 py-3 text-sm ${t.textMuted ?? "text-muted"}`}>{tr("notifications.empty")}</p>
              ) : (
                <ul>
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
            </div>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}
