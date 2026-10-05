"use client";

import { useCallback, useEffect, useState } from "react";

// useNotifications: shared feed logic for the navbar bell (desktop
// dropdown) and the notifications modal (opened from the mobile drawer).
// One hook instance = one independent copy of the feed state; both read the
// same GET /api/notifications endpoint.
//
// - unread: badge count, fetched on mount (cheap single-row query).
// - items/loading/loadList: the dropdown/modal feed, loaded lazily on open.
// - markRead(n): optimistic single-row read (badge rolls back on failure).
// Failures stay silent throughout: the bell is decoration, never a blocker
// for navigation.
export function useNotifications() {
  const [unread, setUnread] = useState(0);
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(false);

  const refreshUnread = useCallback(async () => {
    try {
      const res = await fetch("/api/notifications");
      if (!res.ok) return;
      const data = await res.json();
      if (typeof data.unread === "number") setUnread(data.unread);
    } catch {
      // network/offline: keep the previous count
    }
  }, []);

  useEffect(() => {
    refreshUnread();
  }, [refreshUnread]);

  const loadList = useCallback(async () => {
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
  }, []);

  const markRead = useCallback(async (n) => {
    if (!n || n.read) return;
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
  }, []);

  return { unread, items, loading, loadList, markRead, refreshUnread };
}
