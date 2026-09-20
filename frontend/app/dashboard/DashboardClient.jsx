"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { Camera } from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import { DndContext, closestCenter } from "@dnd-kit/core";
import { SortableContext, verticalListSortingStrategy, arrayMove, useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import QrModal from "../components/QrModal";
import EditLinkModal from "../components/EditLinkModal";
import ClicksChart from "../components/ClicksChart";
import ShortenForm from "../components/ShortenForm";
import CopyButton from "../components/CopyButton";
import { THEMES, themeStyles } from "../../lib/themes";
import ThemeBackdrop from "../components/ThemeBackdrop";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081";

// Baris link yang bisa di-drag. attributes+listeners ditempel di grip
// (bukan seluruh baris) supaya tombol QR/salin/edit tetap bisa di-tap normal.
// dragDisabled=true saat filter tag aktif (reorder parsial akan menabrak
// position baris yang tersembunyi — lihat onDragEnd).
function SortableLinkRow({ link, onQr, onEdit, onFeature, onToggleActive, dragDisabled, st }) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: link.short_code,
    disabled: dragDisabled,
  });
  return (
    <div
      ref={setNodeRef}
      style={{ transform: CSS.Translate.toString(transform), transition }}
      className={`flex items-center justify-between gap-3 ${st.radius} ${st.borderW} ${st.border} ${st.card} px-4 py-2.5 transition-colors duration-150 hover:opacity-90 ${isDragging ? "opacity-60" : ""} ${!link.is_active ? "opacity-60" : ""}`}
    >
      <span
        {...attributes}
        {...listeners}
        title={dragDisabled ? "Reset filter untuk mengurutkan" : "Tarik untuk mengurutkan"}
        className={`shrink-0 select-none font-mono text-sm ${st.textMuted} ${dragDisabled ? "cursor-not-allowed opacity-40" : "cursor-grab touch-none"}`}
      >
        ≡
      </span>
      {/* Toggle unggulan — radio, bukan checkbox: hanya 1 link per akun bisa
          aktif (server menurunkan yang lain dalam 1 transaksi). ★ saat
          featured, ☆ saat tidak. */}
      <button
        onClick={onFeature}
        title={link.is_featured ? "Hapus unggulan" : "Jadikan unggulan"}
        aria-pressed={link.is_featured}
        className={`shrink-0 select-none text-lg leading-none transition-colors duration-150 ${link.is_featured ? st.text : st.textMuted}`}
      >
        {link.is_featured ? "★" : "☆"}
      </button>
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium">{link.original_url}</p>
        <p className={`font-mono text-xs ${st.textMuted}`}>
          /{link.short_code} · {link.click_count} klik
        </p>
        {(link.tags || []).length > 0 && (
          <div className="mt-1 flex flex-wrap gap-1">
            {(link.tags || []).map((t) => (
              <span key={t} className={`rounded-full border ${st.border} px-2 py-px text-[11px] ${st.textMuted}`}>
                {t}
              </span>
            ))}
          </div>
        )}
      </div>
      <div className="flex shrink-0 gap-1.5">
        <button
          onClick={onEdit}
          title="Atur URL per-device & tag"
          className={`shrink-0 rounded-full ${st.borderW} ${st.border} ${st.card} px-3 py-0.5 text-xs font-bold transition-colors duration-150 hover:opacity-90`}
        >
          Edit
        </button>
        <CopyButton text={`${API_BASE}/r/${link.short_code}`} label="Salin" />
        <button
          onClick={onQr}
          className={`shrink-0 rounded-full ${st.borderW} ${st.border} ${st.card} px-3 py-0.5 text-xs font-bold transition-colors duration-150 hover:opacity-90`}
        >
          QR
        </button>
        <button
          onClick={onToggleActive}
          title={link.is_active ? "Nonaktifkan link (redirect jadi 410, tetap tampil di dashboard)" : "Aktifkan kembali (redirect jalan lagi)"}
          aria-pressed={!!link.is_active}
          className={`shrink-0 rounded-full ${st.borderW} ${st.border} ${st.card} px-3 py-0.5 text-xs font-bold transition-colors duration-150 hover:opacity-90`}
        >
          {link.is_active ? "Aktif" : "Nonaktif"}
        </button>
      </div>
    </div>
  );
}

// Halaman dashboard kreator (login wajib): form edit profil
// (display_name, bio, avatar_url, socials) — pola fetch sama seperti AuthModal:
// JSON request/response, tanpa reload, prefill dari GET /api/profile.
export default function DashboardClient() {
  const [username, setUsername] = useState(null);
  const [checked, setChecked] = useState(false);
  const [displayName, setDisplayName] = useState("");
  const [bio, setBio] = useState("");
  const [avatarUrl, setAvatarUrl] = useState("");
  // Avatar upload lokal: file yang dipilih user + preview object URL-nya.
  // File tidak langsung dikirim — preview dulu, baru multipart saat submit.
  const [avatarFile, setAvatarFile] = useState(null);
  const [avatarPreview, setAvatarPreview] = useState("");
  // Overlay "Ganti foto" di atas avatar: tampil saat hover (desktop) / sentuh
  // (mobile), dikontrol state supaya sinkron dengan framer-motion AnimatePresence.
  const [avatarHover, setAvatarHover] = useState(false);
  // Input file tersembunyi dipicu via ref — avatar adalah tombol, bukan label.
  const avatarInputRef = useRef(null);
  const [socials, setSocials] = useState([]);
  const [links, setLinks] = useState([]);
  const [qrCode, setQrCode] = useState(null);
  const [editing, setEditing] = useState(null);
  const [orderMsg, setOrderMsg] = useState("");
  const [tab, setTab] = useState("links");
  const [tagFilter, setTagFilter] = useState("");
  const [notice, setNotice] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  // API Keys (tab ke-3): list + key yang baru digenerate (hanya tampil sekali).
  const [apiKeys, setApiKeys] = useState([]);
  const [generatedKey, setGeneratedKey] = useState("");
  const [keyLabel, setKeyLabel] = useState("");
  const [keyLoading, setKeyLoading] = useState(false);
  const [keyMsg, setKeyMsg] = useState("");
  const [keyErr, setKeyErr] = useState("");
  const [apiDocsOpen, setApiDocsOpen] = useState(false);
  const [theme, setTheme] = useState("classic");
  // Fitur "Dashboard ikut tema kreator": body di-set ke tema aktif (sama
  // teknik ProfileLinks di /u) — body[data-profile-theme=...] di globals.css
  // men-swap background (darkroom/glass); classic/coral tanpa rule →
  // body tetap print-white. Cleanup saat unmount.
  useEffect(() => {
    document.body.dataset.profileTheme = theme;
    return () => {
      delete document.body.dataset.profileTheme;
    };
  }, [theme]);

  const st = themeStyles(theme);
  const inputThemed = `w-full ${st.radius} ${st.borderW} ${st.border} ${st.card} px-3 py-2 text-sm ${st.placeholder} focus:outline-none`;

  useEffect(() => {
    const u = localStorage.getItem("jejak_username");
    setUsername(u);
    setChecked(true);
    if (!u) {
      return;
    }
    loadProfile();
    fetch("/api/keys", { cache: "no-store" })
      .then(async (res) => { const t = await res.text(); try { const d = JSON.parse(t); if (res.ok && Array.isArray(d)) setApiKeys(d); } catch {} })
      .catch(() => {});
  }, []);

  // Muat profil + links dari GET /api/profile. Dipakai saat pertama buka DAN
  // setelah edit link (supaya baris yang diedit langsung tampil dengan
  // device_rules/tags terbaru tanpa reload halaman).
  function loadProfile() {
    fetch("/api/profile")
      .then(async (res) => {
        const text = await res.text();
        let data = {};
        try {
          data = JSON.parse(text);
        } catch {
          throw new Error(text || "Gagal memuat profil");
        }
        if (!res.ok) {
          throw new Error(data.error || "Gagal memuat profil");
        }
        setDisplayName(data.display_name || "");
        setBio(data.bio || "");
        setAvatarUrl(data.avatar_url || "");
        setSocials(Array.isArray(data.socials) ? data.socials : []);
        setLinks(Array.isArray(data.links) ? data.links : []);
        setTheme(THEMES.includes(data.theme) ? data.theme : "classic");
      })
      .catch((err) => setError(err.message));
  }

  function updateSocial(i, field, value) {
    setSocials((prev) => prev.map((s, idx) => (idx === i ? { ...s, [field]: value } : s)));
  }

  // Objek URL dari createObjectURL harus di-revoke pas preview ganti atau
  // komponen unmount, kalau tidak bocor memory (browser pin file-nya).
  useEffect(() => () => {
    if (avatarPreview) URL.revokeObjectURL(avatarPreview);
  }, [avatarPreview]);

  // Pilih file dari avatar interaktif -> tampilkan preview lokal dulu.
  // e.target.value direset supaya memilih file yang sama lagi tetap memicu
  // onChange (file input browser tidak menembak event keduanya kalau value
  // tidak di-reset ke "").
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

  // API keys dimuat saat tab-nya dibuka (lazy, bukan saat mount profil).
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
          throw new Error(text || "Gagal memuat key");
        }
        if (!res.ok) {
          throw new Error(data.error || "Gagal memuat key");
        }
        setApiKeys(Array.isArray(data) ? data : []);
      })
      .catch((err) => setKeyErr(err.message));
  }

  // Generate key baru: plaintext hanya tampil SEKALI di response ini.
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
        throw new Error(text || "Gagal generate key");
      }
      if (!res.ok) {
        throw new Error(data.error || "Gagal generate key");
      }
      setGeneratedKey(data.key);
      setKeyLabel("");
      setKeyMsg("Key berhasil dibuat. Salin sekarang — tidak akan ditampilkan lagi.");
      loadKeys();
    } catch (err) {
      setKeyErr(err.message);
    } finally {
      setKeyLoading(false);
    }
  }

  async function handleDeleteKey(id) {
    if (!window.confirm("Hapus API key ini? Integrasi yang memakainya akan berhenti bekerja.")) {
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
        throw new Error(text || "Gagal hapus key");
      }
      if (!res.ok) {
        throw new Error(data.error || "Gagal hapus key");
      }
      setKeyMsg("Key dihapus.");
      loadKeys();
    } catch (err) {
      setKeyErr(err.message);
    }
  }

  function fmtDate(ts) {
    if (!ts) {
      return "—";
    }
    const d = new Date(ts);
    if (Number.isNaN(d.getTime())) {
      return "—";
    }
    return d.toLocaleString("id-ID", { dateStyle: "medium", timeStyle: "short" });
  }

  // Urutan baru langsung dioptimistic-update di UI, lalu disimpan via
  // PUT /api/links/reorder (server tulis 1 transaction). Gagal simpan →
  // UI dikembalikan ke urutan semula supaya tidak menipu.
  // Filter client-side (list sudah dimuat penuh — tidak perlu endpoint baru
  // untuk skala ini). Tag unik diturunkan dari data, bukan config statis.
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
    setOrderMsg("Menyimpan urutan...");
    try {
      const res = await fetch("/api/links/reorder", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ order: next.map((l) => l.short_code) }),
      });
      if (!res.ok) {
        throw new Error("Gagal menyimpan urutan");
      }
      setOrderMsg("Urutan tersimpan.");
    } catch {
      setLinks(prev);
      setOrderMsg("Gagal menyimpan — urutan dikembalikan.");
    }
  }

  // Toggle unggulan via PUT /api/links/{short_code} {is_featured: bool} —
  // mirror onDragEnd: optimistic dulu biar responsif, kalau persist gagal
  // state di-reload dari server (bukan tebak-tebakan revert lokal).
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
    setOrderMsg(featured ? "Menyimpan unggulan..." : "Menghapus unggulan...");
    try {
      const res = await fetch(`/api/links/${link.short_code}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ is_featured: featured }),
      });
      if (!res.ok) {
        throw new Error("Gagal menyimpan unggulan");
      }
      setOrderMsg(featured ? "Unggulan disimpan." : "Unggulan dihapus.");
    } catch {
      loadProfile();
      setOrderMsg("Gagal — perubahan dibatalkan.");
    }
  }

  // Toggle aktif/nonaktif via PUT /api/links/{short_code} {is_active: bool}.
  // Mirip toggleFeature: optimistic dulu, reload state server kalau gagal.
  // Link nonaktif tetap tampil di dashboard (pemilik harus bisa menyalakan
  // lagi) tapi URL publiknya berhenti redirect (410) — lihat HandleRedirect.
  async function toggleActive(link) {
    const active = !link.is_active;
    setLinks((prev) =>
      prev.map((x) =>
        x.short_code === link.short_code ? { ...x, is_active: active } : x
      )
    );
    setOrderMsg("Menyimpan status...");
    try {
      const res = await fetch(`/api/links/${link.short_code}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ is_active: active }),
      });
      if (!res.ok) {
        throw new Error("Gagal mengubah status");
      }
      setOrderMsg(active ? "Link aktif kembali." : "Link dinonaktifkan.");
    } catch {
      loadProfile();
      setOrderMsg("Gagal — perubahan dibatalkan.");
    }
  }

  async function onSubmit(e) {
    e.preventDefault();
    setError("");
    setNotice("");
    setLoading(true);
    try {
      let finalAvatar = avatarUrl;
      // Jika ada file lokal yang dipilih, upload dulu (multipart) ke
      // POST /api/profile/avatar; response-nya berisi path localStorage
      // yang menjadi avatar_url final. URL manual di-overwrite oleh file
      // ini — file menang, karena itu pilihan aktif user.
      if (avatarFile) {
        const fd = new FormData();
        fd.append("avatar", avatarFile);
        const up = await fetch("/api/profile/avatar", { method: "POST", body: fd });
        const upText = await up.text();
        let upData = {};
        try {
          upData = JSON.parse(upText);
        } catch {
          throw new Error(upText || "Gagal upload foto");
        }
        if (!up.ok) {
          throw new Error(upData.error || upText || "Gagal upload foto");
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
        throw new Error(text || "Gagal menyimpan");
      }
      if (!res.ok) {
        throw new Error(data.error || text || "Gagal menyimpan");
      }
      setNotice("Profil tersimpan.");
      // Navbar memakai state profile-nya SENDIRI yang hanya di-refresh saat
      // navigasi (pathname/username). Setelah save di dashboard, pathname tidak
      // berubah → navbar akan pakai theme LAMA (stall). Broadcast event supaya
      // navbar re-fetch profil tanpa navigasi (lihat Navbar.jsx listener).
      window.dispatchEvent(new Event("jejak:profile-updated"));
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }

  if (!checked) {
    return (
      <main>
        <p className="text-sm text-muted">Memuat...</p>
      </main>
    );
  }

  if (!username) {
    return (
      <main>
<h1 className="text-center font-display text-2xl font-bold">Dashboard</h1>
        <p className="mt-2 text-sm text-muted">
          Masuk dulu lewat tombol Masuk di navbar untuk edit profil.
        </p>
      </main>
    );
  }

  return (
    <main className={st.text}>
      <ThemeBackdrop theme={theme} />
      <h1 className={`${st.headingFont} text-2xl font-bold`}>Dashboard</h1>

      <div className={`relative mt-4 grid grid-cols-3 ${st.radiusFull} ${st.borderW} ${st.border} ${st.card} p-1`}>
        {[
          { id: "links", label: "Link Saya" },
          { id: "profil", label: "Profil" },
          { id: "api-keys", label: "API Keys" },
        ].map((t) => (
          <button
            key={t.id}
            onClick={() => setTab(t.id)}
            className={`relative rounded-full px-4 py-2 text-sm font-bold ${
              tab === t.id ? "text-ink" : st.textMuted
            }`}
          >
            {tab === t.id && (
                <motion.span
                  layoutId="dashboard-tab"
                  transition={{ type: "spring", stiffness: 300, damping: 30 }}
                  className={`absolute inset-0 rounded-full ${st.accent}`}
                />
            )}
            <span className="relative">{t.label}</span>
          </button>
        ))}
      </div>

      <motion.div
        key={tab}
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.2, ease: "easeOut" }}
      >
      {tab === "profil" && (
      <>
      <p className={`mt-4 text-sm ${st.text}`}>
        Halaman publikmu:{" "}
        <Link href={`/u/${username}`} className="font-medium text-flash-coral underline transition-opacity duration-150 hover:opacity-70">
          /u/{username}
        </Link>
      </p>

      <form onSubmit={onSubmit} className="mt-4 flex flex-col gap-3">
        <label className="text-sm font-medium">
          Nama tampil
          <input value={displayName} onChange={(e) => setDisplayName(e.target.value)} required maxLength={100} className={`${inputThemed} mt-1`} />
        </label>
        <label className="text-sm font-medium">
          Bio (maks 500)
          <textarea value={bio} onChange={(e) => setBio(e.target.value)} maxLength={500} rows={3} className={`${inputThemed} mt-1`} />
        </label>
        <div>
          <p className="text-sm font-medium">Avatar</p>
          {/* Avatar = 1 elemen interaktif: foto/inisial sebagai dasar, overlay
              gelap "Ganti foto" muncul role hover (desktop) / sentuh (mobile).
              Klik di mana pun pada elemen memicu input file tersembunyi.
              File divalidasi server (magic bytes, jpg/png/webp, maks 2MB);
              preview client hanya untuk UX, keamanan selalu di server. */}
          <div className="mt-2 flex items-center gap-3">
            <button
              type="button"
              onClick={() => avatarInputRef.current?.click()}
              onMouseEnter={() => setAvatarHover(true)}
              onMouseLeave={() => setAvatarHover(false)}
              onFocus={() => setAvatarHover(true)}
              onBlur={() => setAvatarHover(false)}
              onTouchStart={() => setAvatarHover(true)}
              aria-label="Ganti foto"
              title="Ganti foto"
              className={`relative flex h-20 w-20 shrink-0 items-center justify-center overflow-hidden rounded-full ${st.borderW} ${st.border} ${st.card} focus:outline-none focus-visible:ring-2 focus-visible:ring-flash-coral`}
            >
              {avatarPreview !== "" || avatarUrl.trim() !== "" ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img
                  src={avatarPreview || avatarUrl}
                  alt="Avatar"
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
                    transition={{ duration: 0.15, ease: "easeOut" }}
                    className="absolute inset-0 flex flex-col items-center justify-center gap-0.5 bg-ink/60 text-print-white"
                  >
                    <Camera className="h-5 w-5" aria-hidden="true" />
                    <span className="text-[10px] font-bold">Ganti foto</span>
                  </motion.span>
                )}
              </AnimatePresence>
            </button>
            <input ref={avatarInputRef} type="file" accept="image/jpeg,image/png,image/webp" onChange={handleAvatarFile} className="hidden" />
            {avatarFile && <p className="text-xs text-muted">Foto baru — akan diunggah saat Simpan Profil.</p>}
          </div>
          {/* Opsi URL manual tetap dipertahankan sebagai alternatif terpisah. */}
          <label className={`mt-2 block text-xs font-medium ${st.textMuted}`}>
            ... atau URL avatar langsung (kosongkan = tanpa avatar)
            <input value={avatarUrl} onChange={(e) => setAvatarUrl(e.target.value)} placeholder="https://..." className={`${inputThemed} mt-1`} />
          </label>
        </div>

        <div className="mt-2">
          <p className="text-sm font-medium">Sosial (maks 10)</p>
          <div className="mt-2 flex flex-col gap-2">
            {socials.map((s, i) => (
              <div key={i} className="flex gap-2">
                <input value={s.platform} onChange={(e) => updateSocial(i, "platform", e.target.value)} placeholder="Platform (mis. IG)" maxLength={30} className={`${inputThemed}`} />
                <input value={s.url} onChange={(e) => updateSocial(i, "url", e.target.value)} placeholder="https://..." className={`${inputThemed}`} />
                <button type="button" onClick={() => removeSocial(i)} aria-label="Hapus" className={`w-9 shrink-0 rounded-full ${st.borderW} ${st.border} ${st.card} px-0 text-sm font-bold`}>
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
              + Tambah sosial
            </button>
          )}
        </div>

        <div className="mt-2">
          <p className="text-sm font-medium">Tema halaman publik</p>
          <div className="mt-2 flex flex-wrap gap-2">
            {THEMES.map((key) => {
              const st = themeStyles(key);
              const active = theme === key;
              // BAGIAN st.card GLASS bersifat transparan (bg-white/[8%]) karena
              // dia untuk halaman publik gelap. Kalau dipakai langsung sebagai
              // background TOMBOL picker, saat dashboard bertema terang (classic/
              // coral) tombol Glass jadi putih-di-atas-putih dan "hilang" dari
              // picker. st.chip = swatch SOLID per-preset, selalu kontras.
              return (
                <button
                  key={key}
                  type="button"
                  onClick={() => setTheme(key)}
                  aria-pressed={active}
                  aria-label={`Tema ${st.label}`}
                  className={`w-28 ${st.radius} ${st.borderW} p-2 text-left transition-colors duration-150 ${st.chip} ${
                    active ? "ring-2 ring-flash-yellow ring-offset-1" : "opacity-75 hover:opacity-100"
                  }`}
                >
                  <div className={`rounded-md border ${st.border} p-1.5`}>
                    <div className={`h-2 w-10 rounded-sm ${st.avatar}`} />
                    <div className="mt-1 h-1 w-full rounded-sm bg-current opacity-50" />
                    <div className="mt-0.5 h-1 w-12 rounded-sm bg-current opacity-30" />
                  </div>
                  <p className="mt-1.5 text-xs font-bold">{st.label}</p>
                </button>
              );
            })}
          </div>
        </div>

        <motion.button
          type="submit"
          disabled={loading}
          whileTap={{ scale: 0.97 }}
          transition={{ duration: 0.2, ease: "easeOut" }}
          className={`mt-2 rounded-full border-2 border-ink ${st.accent} px-4 py-2.5 text-sm font-bold transition-[filter] duration-150 hover:brightness-95 disabled:opacity-50`}
        >
          {loading ? "..." : "Simpan Profil"}
        </motion.button>
      </form>

      {notice !== "" && (
        <p className={`mt-3 ${st.radius} ${st.borderW} ${st.border} ${st.card} px-4 py-2.5 text-sm font-medium`}>{notice}</p>
      )}
      {error !== "" && (
        <p className={`mt-3 ${st.radius} ${st.borderW} ${st.border} ${st.card} px-4 py-2.5 text-sm font-medium`}>{error}</p>
      )}
      </>
      )}
      {tab === "api-keys" && (
      <>
      <button
        type="button"
        onClick={() => setApiDocsOpen(!apiDocsOpen)}
        className={`mt-4 flex items-center gap-1 text-sm font-medium ${st.textMuted} transition-colors duration-150`}
      >
        Cara pakai API
        <motion.span
          animate={{ rotate: apiDocsOpen ? 180 : 0 }}
          transition={{ duration: 0.2 }}
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
            transition={{ duration: 0.25, ease: "easeInOut" }}
            className="overflow-hidden"
          >
            <p className={`mt-2 text-sm ${st.textMuted}`}>Contoh pemanggilan:</p>
            <pre className={`mt-2 overflow-x-auto ${st.radius} ${st.borderW} ${st.border} ${st.card} px-4 py-3 font-mono text-sm leading-relaxed`}>
              <code>{`curl -X POST https://jejak.app/api/v1/shorten \\
  -H "Authorization: Bearer jjk_xxxxx" \\
  -H "Content-Type: application/json" \\
  -d '{"url": "https://example.com"}'`}</code>
            </pre>
          </motion.div>
        )}
      </AnimatePresence>

      <form onSubmit={handleGenerateKey} className="mt-4 flex flex-col gap-2">
        <label htmlFor="key-label" className="text-sm font-medium">
          Label (opsional, untuk dikenali)
          <input
            id="key-label"
            value={keyLabel}
            onChange={(e) => setKeyLabel(e.target.value)}
            maxLength={100}
            placeholder="mis. CI script"
            className={`${inputThemed} mt-1`}
          />
        </label>
        <motion.button
          type="submit"
          disabled={keyLoading}
          whileTap={{ scale: 0.97 }}
          transition={{ duration: 0.2, ease: "easeOut" }}
          className={`rounded-full border-2 border-ink ${st.accent} px-4 py-2.5 text-sm font-bold transition-[filter] duration-150 hover:brightness-95 disabled:opacity-50`}
        >
          {keyLoading ? "..." : "Buat Key Baru"}
        </motion.button>
      </form>

      {generatedKey !== "" && (
        <div className={`mt-3 ${st.radius} ${st.borderW} ${st.border} ${st.card} px-4 py-3`}>
          <p className="text-sm font-bold">Simpan sekarang — tidak akan ditampilkan lagi!</p>
          <div className="mt-2 flex flex-wrap items-center gap-2">
            <code className="min-w-0 break-all font-mono text-sm">{generatedKey}</code>
            <CopyButton text={generatedKey} label="Salin" />
          </div>
        </div>
      )}

      {keyMsg !== "" && (
        <p className={`mt-3 ${st.radius} ${st.borderW} ${st.border} ${st.card} px-4 py-2.5 text-sm font-medium`}>{keyMsg}</p>
      )}
      {keyErr !== "" && (
        <p className={`mt-3 ${st.radius} ${st.borderW} ${st.border} ${st.card} px-4 py-2.5 text-sm font-medium`}>{keyErr}</p>
      )}

      <section className="mt-5">
        <h2 className={`${st.headingFont} text-lg font-bold`}>Key aktif</h2>
        {apiKeys.length === 0 ? (
          <p className="mt-2 text-sm text-muted">Belum ada key — buat satu lewat formulir di atas.</p>
        ) : (
          <div className="mt-2 flex flex-col gap-2">
            {apiKeys.map((k) => (
              <div key={k.id} className={`flex items-center justify-between gap-3 ${st.radius} ${st.borderW} ${st.border} ${st.card} px-4 py-2.5`}>
                <div className="min-w-0">
                  <p className="truncate text-sm font-medium">{k.label !== "" ? k.label : "(tanpa label)"}</p>
                  <p className={`font-mono text-xs ${st.textMuted}`}>
                    dibuat {fmtDate(k.created_at)} · terakhir dipakai {fmtDate(k.last_used_at)}
                  </p>
                </div>
                <button
                  onClick={() => handleDeleteKey(k.id)}
                  className={`shrink-0 rounded-full ${st.borderW} ${st.border} ${st.card} px-3 py-0.5 text-xs font-bold transition-colors duration-150 hover:opacity-90`}
                >
                  Hapus
                </button>
              </div>
            ))}
          </div>
        )}
      </section>
      </>
      )}
      {tab === "links" && (
      <>
      {/* TAHAP D: form shorten inline di puncak tab — user login bisa bikin
          link baru TANPA keluar dashboard. `st` diteruskan biar form ikut
          token tema dashboard yang aktif; onSuccess memanggil loadProfile()
          (fungsi load link yang sudah ada) supaya link baru langsung muncul
          di list bawahnya TANPA reload halaman penuh. */}
      <ShortenForm st={st} onSuccess={() => loadProfile()} />
      <ClicksChart theme={theme} st={st} />
      {links.length === 0 && bio === "" && apiKeys.length === 0 && (
        <div className={`mt-4 ${st.radius} ${st.borderW} ${st.border} ${st.card} px-4 py-4`}>
          <p className={`${st.headingFont} text-lg font-bold`}>Selamat datang! Mulai dari sini →</p>
          <ol className="mt-3 flex flex-col gap-2 text-sm">
            <li>
              1.{" "}
              <button
                type="button"
                onClick={() => window.scrollTo({ top: 0, behavior: "smooth" })}
                className="font-medium text-flash-coral underline transition-colors duration-150 hover:text-ink"
              >
                ↑ Buat short-link pertamamu di form atas
              </button>
              .
            </li>
            <li>
              2.{" "}
              <button
                type="button"
                onClick={() => setTab("profil")}
                className="font-medium text-flash-coral underline transition-colors duration-150 hover:text-ink"
              >
                Lengkapi profil
              </button>{" "}
              biar halaman publikmu siap dibagikan.
            </li>
          </ol>
        </div>
      )}
      <section className="mt-4">
        <div className="flex items-center justify-between gap-3">
          <h2 className={`${st.headingFont} text-lg font-bold`}>Link saya</h2>
        </div>
        {allTags.length > 0 && (
          <div className="mt-2 flex items-center gap-2 text-sm">
            <label htmlFor="tag-filter" className={st.textMuted}>Filter tag:</label>
            <select
              id="tag-filter"
              value={tagFilter}
              onChange={(e) => setTagFilter(e.target.value)}
              className={`rounded-full ${st.borderW} ${st.border} ${st.card} px-3 py-1 text-sm focus:outline-none`}
            >
              <option value="">Semua</option>
              {allTags.map((t) => (
                <option key={t} value={t}>{t}</option>
              ))}
            </select>
          </div>
        )}
        {links.length === 0 ? (
          <p className={`mt-2 text-sm ${st.textMuted}`}>Belum ada link. Buat dari halaman utama dulu.</p>
        ) : visibleLinks.length === 0 ? (
          <p className={`mt-2 text-sm ${st.textMuted}`}>Tidak ada link dengan tag ini.</p>
        ) : (
          <DndContext collisionDetection={closestCenter} onDragEnd={onDragEnd}>
            <SortableContext items={visibleLinks.map((l) => l.short_code)} strategy={verticalListSortingStrategy}>
              <div className="mt-3 flex flex-col gap-2">
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
          <p className={`mt-2 text-sm ${st.textMuted}`}>{orderMsg}</p>
        )}
      </section>
      </>
      )}
      </motion.div>

      <AnimatePresence>
        {qrCode !== null && (
          <QrModal
            shortCode={qrCode}
            shortUrl={`${API_BASE}/r/${qrCode}`}
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
      </AnimatePresence>
    </main>
  );
}
