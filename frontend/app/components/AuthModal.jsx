"use client";

import { useEffect, useState } from "react";
import { motion } from "framer-motion";
import SubmitButton from "./SubmitButton";
import { useTranslation } from "../../lib/I18nProvider";

// Auth dialog (login/register): opened from the navbar rather than as a
// separate page. Desktop: centered, 420px max-width. Mobile: bottom sheet
// (slides up from the bottom). Submits via fetch to the existing endpoints
// (JSON), without a reload: success → close + navbar update. Animation §6:
// gentle fade + scale, ease-out 200ms, no bounce.
export default function AuthModal({ mode, onClose, onSuccess, st }) {
  const { t } = useTranslation();
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
        throw new Error(text || t("errors.auth.requestFailed"));
      }
      if (!res.ok) {
        throw new Error(data.error || text || t("errors.auth.requestFailed"));
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
    `w-full rounded-xl ${st.borderW} ${st.border} ${st.card} px-3 py-2 text-sm transition-all duration-150 focus:border-flash-yellow focus:ring-2 focus:ring-flash-yellow/30 focus:outline-hidden ${st.placeholder}`;

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
        aria-label={isRegister ? t("auth.register.ariaLabel") : t("auth.login.ariaLabel")}
        initial={{ opacity: 0, y: 24, scale: 0.98 }}
        animate={{ opacity: 1, y: 0, scale: 1 }}
        exit={{ opacity: 0, y: 12, scale: 0.98 }}
        transition={{ duration: 0.2, ease: "easeOut" }}
        className={`pointer-events-auto relative w-full rounded-t-2xl ${st.borderW} ${st.border} ${st.card} p-6 sm:max-w-[420px] sm:rounded-2xl`}
      >
        <h2 className="font-display text-xl font-bold">
          {isRegister ? t("auth.register.title") : t("auth.login.title")}
        </h2>
          <p className={`mt-1 text-sm ${st.textMuted}`}>
          {isRegister ? t("auth.register.subtitle") : t("auth.login.subtitle")}
        </p>

        <form onSubmit={onSubmit} className="mt-4 flex flex-col gap-3">
          <input value={username} onChange={(e) => setUsername(e.target.value)} required placeholder={t("auth.usernamePlaceholder")} autoFocus className={input} />
          {isRegister && (
            <>
              <input value={displayName} onChange={(e) => setDisplayName(e.target.value)} required placeholder={t("auth.register.displayNamePlaceholder")} className={input} />
              <input value={bio} onChange={(e) => setBio(e.target.value)} placeholder={t("auth.register.bioPlaceholder")} className={input} />
            </>
          )}
          <input value={password} onChange={(e) => setPassword(e.target.value)} required minLength={isRegister ? 8 : undefined} type="password" placeholder={isRegister ? t("auth.register.passwordPlaceholder") : t("auth.login.passwordPlaceholder")} className={input} />
          <SubmitButton
            isLoading={loading}
            loadingLabel={isRegister ? t("auth.register.submitLoading") : t("auth.login.submitLoading")}
            className={`rounded-full border-2 border-ink ${st.accent} px-4 py-2.5 text-sm font-bold transition-[filter] duration-150 hover:brightness-95`}
          >
            {isRegister ? t("auth.register.submit") : t("auth.login.submit")}
          </SubmitButton>
        </form>

        {error !== "" && (
          <p className={`mt-3 rounded-xl ${st.borderW} ${st.border} ${st.card} px-4 py-2.5 text-sm font-medium`}>{error}</p>
        )}
      </motion.div>
    </div>
  );
}
