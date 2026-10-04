"use client";

import { useState } from "react";
import { LogOut, Trash2, ShieldAlert } from "lucide-react";
import SubmitButton from "../components/SubmitButton";
import ConfirmModal from "../components/ConfirmModal";
import { useToast } from "../components/Toast";
import { useTranslation } from "../../lib/I18nProvider";

// Compact JSON fetch helper for the four /api/account/* endpoints: throws an
// Error carrying the server's message (Go answers plain text through
// http.Error, so the raw text is used when the body is not JSON). Every
// action in this tab goes through it.
async function callAccount(url, method, body, fallbackMessage) {
  const res = await fetch(url, {
    method,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  const text = await res.text();
  let data = {};
  try {
    data = JSON.parse(text);
  } catch {
    /* plain-text error from Go: fall back to the raw text below */
  }
  if (!res.ok) {
    throw new Error(data.error || text || fallbackMessage);
  }
  return data;
}

// Account settings tab (2026-09-30): section A change email, B change
// password, C Danger Zone (log out everywhere + two-step account deletion).
// Every sensitive action (email/password/delete) verifies the password on
// the server; success is announced through the global toast rather than an
// inline banner: the forms are cleared afterwards so passwords do not
// linger in state after use.
export default function PengaturanTab({ st }) {
  const toast = useToast();
  const { t } = useTranslation();

  // Section A: change email.
  const [email, setEmail] = useState("");
  const [emailPassword, setEmailPassword] = useState("");
  const [emailBusy, setEmailBusy] = useState(false);

  // Section B: change password.
  const [currentPw, setCurrentPw] = useState("");
  const [newPw, setNewPw] = useState("");
  const [confirmPw, setConfirmPw] = useState("");
  const [pwBusy, setPwBusy] = useState(false);

  // Section C: Danger Zone.
  const [logoutAllOpen, setLogoutAllOpen] = useState(false);
  const [logoutBusy, setLogoutBusy] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deletePw, setDeletePw] = useState("");
  const [deleteBusy, setDeleteBusy] = useState(false);

  const inputCls = `w-full rounded-xl border-2 border-ink bg-white px-3 py-2 text-sm focus:border-flash-yellow focus:ring-2 focus:ring-flash-yellow/30 focus:outline-hidden transition-all duration-150`;
  const primaryBtn = `rounded-full border-2 border-ink ${st.accent} px-4 py-2.5 text-sm font-bold transition-[filter] duration-150 hover:brightness-95`;

  async function onSubmitEmail(e) {
    e.preventDefault();
    if (!email.includes("@") || !email.includes(".")) {
      toast.error(t("toast.emailInvalid"));
      return;
    }
    if (emailPassword === "") {
      toast.error(t("toast.emailCurrentPasswordRequired"));
      return;
    }
    setEmailBusy(true);
    try {
      await callAccount("/api/account/email", "PUT", { email, password: emailPassword }, t("errors.account.requestFailed"));
      toast.success(t("toast.emailUpdated"));
      setEmail("");
      setEmailPassword("");
    } catch (err) {
      toast.error(err.message);
    } finally {
      setEmailBusy(false);
    }
  }

  async function onSubmitPassword(e) {
    e.preventDefault();
    if (currentPw === "") {
      toast.error(t("toast.passwordCurrentRequired"));
      return;
    }
    if (newPw.length < 8) {
      toast.error(t("toast.newPasswordTooShort"));
      return;
    }
    if (newPw !== confirmPw) {
      toast.error(t("toast.passwordMismatch"));
      return;
    }
    setPwBusy(true);
    try {
      await callAccount(
        "/api/account/password",
        "PUT",
        {
          current_password: currentPw,
          new_password: newPw,
        },
        t("errors.account.requestFailed")
      );
      toast.success(t("toast.passwordUpdated"));
      setCurrentPw("");
      setNewPw("");
      setConfirmPw("");
    } catch (err) {
      toast.error(err.message);
    } finally {
      setPwBusy(false);
    }
  }

  async function doLogoutAll() {
    setLogoutBusy(true);
    try {
      await callAccount("/api/account/logout-all", "POST", {}, t("errors.account.requestFailed"));
      toast.success(t("toast.allDevicesLoggedOut"));
      // The cookie has been cleared server-side and every session revoked →
      // go straight to the landing page (a hard navigation so all dashboard
      // state is discarded as well).
      window.location.assign("/");
    } catch (err) {
      toast.error(err.message);
      setLogoutBusy(false);
    }
  }

  async function doDeleteAccount() {
    setDeleteBusy(true);
    try {
      await callAccount("/api/account", "DELETE", { confirmation: "HAPUS", password: deletePw }, t("errors.account.requestFailed"));
      toast.success(t("toast.accountDeleted"));
      window.location.assign("/");
    } catch (err) {
      toast.error(err.message);
      setDeleteBusy(false);
      setDeletePw("");
    }
  }

  return (
    <>
      {/* Section A: Change Email */}
      <section className={`rounded-xl border-2 border-ink bg-white p-5 text-ink md:p-6`}>
        <h2 className="font-display text-lg font-bold">{t("dashboard.settings.account.email.title")}</h2>
        <p className="mt-1 text-sm text-ink/70">{t("dashboard.settings.account.email.description")}</p>
        <form onSubmit={onSubmitEmail} className="mt-4 flex flex-col gap-4">
          <label className="text-sm font-medium" htmlFor="acc-email">
            {t("dashboard.settings.account.email.emailLabel")}
            <input
              id="acc-email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
              maxLength={255}
              autoComplete="email"
              placeholder={t("dashboard.settings.account.email.emailPlaceholder")}
              className={`${inputCls} mt-1`}
            />
          </label>
          <label className="text-sm font-medium" htmlFor="acc-email-pw">
            {t("dashboard.settings.account.email.passwordLabel")}
            <input
              id="acc-email-pw"
              type="password"
              value={emailPassword}
              onChange={(e) => setEmailPassword(e.target.value)}
              required
              autoComplete="current-password"
              className={`${inputCls} mt-1`}
            />
          </label>
          <SubmitButton isLoading={emailBusy} loadingLabel={t("dashboard.actions.saving")} className={`w-fit ${primaryBtn}`}>
            {t("dashboard.settings.account.email.submit")}
          </SubmitButton>
        </form>
      </section>

      {/* Section B: Change Password */}
      <section className={`rounded-xl border-2 border-ink bg-white p-5 text-ink md:p-6`}>
        <h2 className="font-display text-lg font-bold">{t("dashboard.settings.account.password.title")}</h2>
        <p className="mt-1 text-sm text-ink/70">{t("dashboard.settings.account.password.description")}</p>
        <form onSubmit={onSubmitPassword} className="mt-4 flex flex-col gap-4">
          <label className="text-sm font-medium" htmlFor="acc-current-pw">
            {t("dashboard.settings.account.password.currentLabel")}
            <input
              id="acc-current-pw"
              type="password"
              value={currentPw}
              onChange={(e) => setCurrentPw(e.target.value)}
              required
              autoComplete="current-password"
              className={`${inputCls} mt-1`}
            />
          </label>
          <label className="text-sm font-medium" htmlFor="acc-new-pw">
            {t("dashboard.settings.account.password.newLabel")}
            <input
              id="acc-new-pw"
              type="password"
              value={newPw}
              onChange={(e) => setNewPw(e.target.value)}
              required
              minLength={8}
              autoComplete="new-password"
              className={`${inputCls} mt-1`}
            />
          </label>
          <label className="text-sm font-medium" htmlFor="acc-confirm-pw">
            {t("dashboard.settings.account.password.confirmLabel")}
            <input
              id="acc-confirm-pw"
              type="password"
              value={confirmPw}
              onChange={(e) => setConfirmPw(e.target.value)}
              required
              minLength={8}
              autoComplete="new-password"
              className={`${inputCls} mt-1`}
            />
          </label>
          <SubmitButton isLoading={pwBusy} loadingLabel={t("dashboard.actions.saving")} className={`w-fit ${primaryBtn}`}>
            {t("dashboard.settings.account.password.submit")}
          </SubmitButton>
        </form>
      </section>

      {/* Section C: Danger Zone (a dedicated card: destructive actions do
          not use theme tokens: always coral/ink so they stay consistent
          across every dashboard theme). */}
      <section className="rounded-xl border-2 border-flash-coral bg-white p-5 text-ink md:p-6">
        <div className="flex items-center gap-2">
          <ShieldAlert size={18} aria-hidden="true" className="shrink-0 text-flash-coral" />
          <h2 className="font-display text-lg font-bold">{t("dashboard.settings.account.danger.title")}</h2>
        </div>
        <div className="mt-4 flex flex-col gap-4">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="min-w-0">
              <p className="text-sm font-bold">{t("dashboard.settings.account.danger.logoutTitle")}</p>
              <p className="text-sm text-ink/70">{t("dashboard.settings.account.danger.logoutDescription")}</p>
            </div>
            <button
              type="button"
              onClick={() => setLogoutAllOpen(true)}
              className="inline-flex shrink-0 items-center gap-1.5 rounded-full border-2 border-ink bg-white px-4 py-2 text-sm font-bold text-ink transition-transform duration-150 hover:-translate-y-0.5"
            >
              <LogOut size={14} aria-hidden="true" />
              {t("dashboard.settings.account.danger.logoutButton")}
            </button>
          </div>
          <hr className="border-ink/15" />
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="min-w-0">
              <p className="text-sm font-bold">{t("dashboard.settings.account.danger.deleteTitle")}</p>
              <p className="text-sm text-ink/70">{t("dashboard.settings.account.danger.deleteDescription")}</p>
            </div>
            <button
              type="button"
              onClick={() => setDeleteOpen(true)}
              className="inline-flex shrink-0 items-center gap-1.5 rounded-full border-2 border-ink bg-flash-coral px-4 py-2 text-sm font-bold text-white transition-transform duration-150 hover:-translate-y-0.5"
            >
              <Trash2 size={14} aria-hidden="true" />
              {t("dashboard.settings.account.danger.deleteButton")}
            </button>
          </div>
        </div>
      </section>

      {/* Modal 1: log out of all devices (single confirmation). */}
      <ConfirmModal
        open={logoutAllOpen}
        onClose={() => !logoutBusy && setLogoutAllOpen(false)}
        title={t("dashboard.settings.account.confirmLogout.title")}
        description={t("dashboard.settings.account.confirmLogout.description")}
        confirmLabel={t("dashboard.settings.account.confirmLogout.confirmLabel")}
        busy={logoutBusy}
        onConfirm={doLogoutAll}
      />

      {/* Modal 2: delete the account in two steps: type "HAPUS" + password. */}
      <ConfirmModal
        open={deleteOpen}
        onClose={() => !deleteBusy && setDeleteOpen(false)}
        title={t("dashboard.settings.account.confirmDelete.title")}
        description={t("dashboard.settings.account.confirmDelete.description")}
        confirmLabel={t("dashboard.settings.account.confirmDelete.confirmLabel")}
        confirmVariant="danger"
        requireTyping={t("dashboard.settings.account.confirmDelete.typeToken")}
        busy={deleteBusy}
        onConfirm={doDeleteAccount}
      >
        <label className="mt-4 block text-sm font-medium" htmlFor="acc-delete-pw">
          {t("dashboard.settings.account.confirmDelete.passwordLabel")}
          <input
            id="acc-delete-pw"
            type="password"
            value={deletePw}
            onChange={(e) => setDeletePw(e.target.value)}
            required
            autoComplete="current-password"
            className={`${inputCls} mt-1`}
          />
        </label>
      </ConfirmModal>
    </>
  );
}
