"use client";

import { useEffect, useState } from "react";
import { motion } from "framer-motion";

// Dialog auth (login/register) — dibuka dari navbar, bukan halaman terpisah.
// Desktop: terpusat max-width 420px. Mobile: bottom-sheet (slide dari bawah).
// Submit via fetch ke endpoint yang ada (JSON), tanpa reload: sukses → tutup
// + navbar update. Animasi §6: fade + scale halus, ease-out 200ms, tanpa bounce.
export default function AuthModal({ mode, onClose, onSuccess, st }) {
  const isRegister = mode === "register";
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [bio, setBio] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    function onKey(e) {
      if (e.key === "Escape") {
        onClose();
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  async function onSubmit(e) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      const payload = isRegister
        ? { username, display_name: displayName, bio, password }
        : { username, password };
      const res = await fetch(`/api/${mode}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      const text = await res.text();
      let data = {};
      try {
        data = JSON.parse(text);
      } catch {
        throw new Error(text || "Gagal");
      }
      if (!res.ok) {
        throw new Error(data.error || text || "Gagal");
      }
      localStorage.setItem("jejak_username", data.username);
      onSuccess(data.username);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }

  const input =
    `w-full rounded-xl ${st.borderW} ${st.border} ${st.card} px-3 py-2 text-sm transition-colors duration-150 focus:outline-none ${st.placeholder}`;

  return (
    <div className="pointer-events-none fixed inset-0 z-[60] flex items-end justify-center sm:items-start sm:px-4 sm:pt-[18vh]">
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        transition={{ duration: 0.2, ease: "easeOut" }}
        onClick={onClose}
        className="pointer-events-auto absolute inset-0 bg-ink/60"
        aria-hidden="true"
      />
      <motion.div
        role="dialog"
        aria-modal="true"
        aria-label={isRegister ? "Daftar" : "Masuk"}
        initial={{ opacity: 0, y: 24, scale: 0.98 }}
        animate={{ opacity: 1, y: 0, scale: 1 }}
        exit={{ opacity: 0, y: 12, scale: 0.98 }}
        transition={{ duration: 0.2, ease: "easeOut" }}
        className={`pointer-events-auto relative w-full rounded-t-2xl ${st.borderW} ${st.border} ${st.card} p-6 sm:max-w-[420px] sm:rounded-2xl`}
      >
        <h2 className="font-display text-xl font-bold">
          {isRegister ? "Daftar jadi Kreator" : "Masuk"}
        </h2>
          <p className={`mt-1 text-sm ${st.textMuted}`}>
          {isRegister ? "Satu akun, satu halaman publik untuk semua link kamu." : "Sesi login tersimpan sebagai cookie HttpOnly."}
        </p>

        <form onSubmit={onSubmit} className="mt-4 flex flex-col gap-3">
          <input value={username} onChange={(e) => setUsername(e.target.value)} required placeholder="username" autoFocus className={input} />
          {isRegister && (
            <>
              <input value={displayName} onChange={(e) => setDisplayName(e.target.value)} required placeholder="Nama tampil" className={input} />
              <input value={bio} onChange={(e) => setBio(e.target.value)} placeholder="Bio singkat (opsional)" className={input} />
            </>
          )}
          <input value={password} onChange={(e) => setPassword(e.target.value)} required minLength={isRegister ? 8 : undefined} type="password" placeholder={isRegister ? "Password (min 8 karakter)" : "password"} className={input} />
          <motion.button
            type="submit"
            disabled={loading}
            whileTap={{ scale: 0.97 }}
            transition={{ duration: 0.2, ease: "easeOut" }}
            className={`rounded-full border-2 border-ink ${st.accent} px-4 py-2.5 text-sm font-bold transition-[filter] duration-150 hover:brightness-95 disabled:opacity-50`}
          >
            {loading ? "..." : isRegister ? "Buat Akun" : "Masuk"}
          </motion.button>
        </form>

        {error !== "" && (
          <p className={`mt-3 rounded-xl ${st.borderW} ${st.border} ${st.card} px-4 py-2.5 text-sm font-medium`}>{error}</p>
        )}
      </motion.div>
    </div>
  );
}
