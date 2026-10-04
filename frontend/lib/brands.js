// Auto-detect brand dari domain URL untuk ikon kartu link di halaman publik
// /u/[username]. Data murni (nama, warna, nama ikon lucide) — komponen yang
// me-render ada di ProfileLinks.jsx.
//
// DESIGN NOTES:
// - KEY = hostname tanpa "www." (exact match diutamakan). Subdomain otomatis
//   jatuh ke root dua-label (mail.google.com → google.com) oleh detectBrand.
//   Domain berformat co.id (shopee.co.id) TIDAK bisa di-derive dari root —
//   wajib terdaftar exact.
// - color = warna brand resmi dipakai sebagai tint bg (20%), border (55%),
//   dan warna ikon. Warna TERANG (luminance > 0.65, mis. Snapchat #FFFC00)
//   membalik pola: bg SOLID + ikon gelap, karena ikon kuning di atas tint
//   kuning pucat tidak kontras (audit kontras lintas 11 tema).
// - icon = key di ICON_MAP (bawah file); kalau brand baru butuh ikon yang
//   belum ada, tambahkan lucide component + key-nya di satu tempat ini.

export const BRANDS = {
  // E-commerce ID
  "shopee.co.id": { name: "Shopee", color: "#EE4D2D", icon: "shopping-bag" },
  "shopee.com": { name: "Shopee", color: "#EE4D2D", icon: "shopping-bag" },
  "shopee.sg": { name: "Shopee", color: "#EE4D2D", icon: "shopping-bag" },
  "shopee.com.my": { name: "Shopee", color: "#EE4D2D", icon: "shopping-bag" },
  "shopee.co.th": { name: "Shopee", color: "#EE4D2D", icon: "shopping-bag" },
  "tokopedia.com": { name: "Tokopedia", color: "#03AC0E", icon: "shopping-bag" },
  "lazada.co.id": { name: "Lazada", color: "#0F146D", icon: "shopping-bag" },
  "bukalapak.com": { name: "Bukalapak", color: "#E31E52", icon: "shopping-bag" },
  "blibli.com": { name: "Blibli", color: "#0095DA", icon: "shopping-bag" },
  "amazon.com": { name: "Amazon", color: "#FF9900", icon: "shopping-cart" },
  "amazon.co.id": { name: "Amazon", color: "#FF9900", icon: "shopping-cart" },
  "etsy.com": { name: "Etsy", color: "#F1641E", icon: "shopping-cart" },

  // Social
  "instagram.com": { name: "Instagram", color: "#E1306C", icon: "instagram" },
  "tiktok.com": { name: "TikTok", color: "#000000", icon: "music-2" },
  "twitter.com": { name: "Twitter", color: "#1DA1F2", icon: "twitter" },
  "x.com": { name: "X", color: "#000000", icon: "twitter" },
  "facebook.com": { name: "Facebook", color: "#1877F2", icon: "facebook" },
  "threads.net": { name: "Threads", color: "#000000", icon: "at-sign" },
  "linkedin.com": { name: "LinkedIn", color: "#0A66C2", icon: "linkedin" },
  "pinterest.com": { name: "Pinterest", color: "#E60023", icon: "image" },
  "reddit.com": { name: "Reddit", color: "#FF4500", icon: "message-circle" },
  "snapchat.com": { name: "Snapchat", color: "#FFFC00", icon: "ghost" },
  "dribbble.com": { name: "Dribbble", color: "#EA4C89", icon: "dribbble" },
  "behance.net": { name: "Behance", color: "#1769FF", icon: "palette" },

  // Video/Music
  "youtube.com": { name: "YouTube", color: "#FF0000", icon: "youtube" },
  "youtu.be": { name: "YouTube", color: "#FF0000", icon: "youtube" },
  "vimeo.com": { name: "Vimeo", color: "#1AB7EA", icon: "video" },
  "spotify.com": { name: "Spotify", color: "#1DB954", icon: "music" },
  "soundcloud.com": { name: "SoundCloud", color: "#FF5500", icon: "music" },
  "twitch.tv": { name: "Twitch", color: "#9146FF", icon: "twitch" },
  "netflix.com": { name: "Netflix", color: "#E50914", icon: "tv" },

  // Messaging
  "wa.me": { name: "WhatsApp", color: "#25D366", icon: "message-circle" },
  "whatsapp.com": { name: "WhatsApp", color: "#25D366", icon: "message-circle" },
  "t.me": { name: "Telegram", color: "#0088CC", icon: "send" },
  "telegram.org": { name: "Telegram", color: "#0088CC", icon: "send" },
  "line.me": { name: "LINE", color: "#00C300", icon: "message-square" },
  "discord.com": { name: "Discord", color: "#5865F2", icon: "message-circle" },
  "discord.gg": { name: "Discord", color: "#5865F2", icon: "message-circle" },
  "slack.com": { name: "Slack", color: "#4A154B", icon: "slack" },
  "zoom.us": { name: "Zoom", color: "#2D8CFF", icon: "video" },

  // Dev
  "github.com": { name: "GitHub", color: "#181717", icon: "github" },
  "gitlab.com": { name: "GitLab", color: "#FC6D26", icon: "gitlab" },
  "stackoverflow.com": { name: "Stack Overflow", color: "#F58025", icon: "code" },
  "dev.to": { name: "DEV Community", color: "#0A0A0A", icon: "code" },
  "figma.com": { name: "Figma", color: "#F24E1E", icon: "figma" },
  "steamcommunity.com": { name: "Steam", color: "#66C0F4", icon: "gamepad-2" },
  "steampowered.com": { name: "Steam", color: "#66C0F4", icon: "gamepad-2" },

  // Blog/Docs
  "medium.com": { name: "Medium", color: "#000000", icon: "book-open" },
  "substack.com": { name: "Substack", color: "#FF6719", icon: "mail" },
  "notion.so": { name: "Notion", color: "#000000", icon: "file-text" },
  "docs.google.com": { name: "Google Docs", color: "#4285F4", icon: "file-text" },
  "drive.google.com": { name: "Google Drive", color: "#4285F4", icon: "hard-drive" },
  "dropbox.com": { name: "Dropbox", color: "#0061FF", icon: "box" },
  "google.com": { name: "Google", color: "#4285F4", icon: "search" },
  "apple.com": { name: "Apple", color: "#000000", icon: "smartphone" },
  "quora.com": { name: "Quora", color: "#B92B27", icon: "circle-help" },
  "canva.com": { name: "Canva", color: "#00C4CC", icon: "palette" },

  // Link-in-bio competitor (biar lucu)
  "linktr.ee": { name: "Linktree", color: "#43E660", icon: "link" },
  "beacons.ai": { name: "Beacons", color: "#4B4EFC", icon: "link" },
  "bio.link": { name: "Bio.link", color: "#000000", icon: "link" },
};

// Fallback netral untuk domain tak dikenal (juga dipakai URL rusak).
const FALLBACK = { name: "Link", color: "#1C1A12", icon: "link", matched: null };

// detectBrand: host → brand entry. Urutan: exact → root dua-label → fallback.
export function detectBrand(url) {
  try {
    const parsed = new URL(url);
    const host = parsed.hostname.replace(/^www\./, "");

    // Exact match
    if (BRANDS[host]) return { ...BRANDS[host], matched: host };

    // Subdomain match (mail.google.com → google.com)
    const parts = host.split(".");
    if (parts.length > 2) {
      const root = parts.slice(-2).join(".");
      if (BRANDS[root]) return { ...BRANDS[root], matched: root };
    }

    return { ...FALLBACK };
  } catch {
    return { ...FALLBACK };
  }
}

// Relative luminance (WCAG) utk keputusan rendering:
// - > 0.65 (sangat terang, mis. Snapchat #FFFC00): tint 20% + ikon berwarna
//   tidak kontras → bg SOLID + ikon gelap.
// - < 0.10 (sangat gelap, mis. X/Medium #000000): tint 20% + ikon hitam
//   hilang di tema gelap (darkroom) → bg SOLID + ikon putih + border tipis
//   putih supaya box tetap terpisah dari kartu gelap.
export function luminance(hex) {
  const m = /^#([0-9a-f]{6})$/i.exec(hex);
  if (!m) return 0;
  const n = parseInt(m[1], 16);
  const chan = (v) => {
    const s = v / 255;
    return s <= 0.03928 ? s / 12.92 : Math.pow((s + 0.055) / 1.055, 2.4);
  };
  return (
    0.2126 * chan((n >> 16) & 255) +
    0.7152 * chan((n >> 8) & 255) +
    0.0722 * chan(n & 255)
  );
}

export function isBrightColor(hex) {
  return luminance(hex) > 0.65;
}

export function isDarkColor(hex) {
  return luminance(hex) < 0.1;
}

// "#EE4D2D" + alpha 0..1 → "rgba(238,77,45,0.2)" utk inline style.
export function rgba(hex, alpha) {
  const m = /^#([0-9a-f]{6})$/i.exec(hex);
  if (!m) return `rgba(0,0,0,${alpha})`;
  const n = parseInt(m[1], 16);
  return `rgba(${(n >> 16) & 255},${(n >> 8) & 255},${n & 255},${alpha})`;
}

// ── ICON MAP ──
// Nama key sama dengan nilai `icon` di BRANDS. Semua component sudah diverifikasi
// ada di lucide-react ^0.469.0.
import {
  ShoppingBag,
  ShoppingCart,
  Instagram,
  Music2,
  Twitter,
  Facebook,
  Linkedin,
  Image,
  MessageCircle,
  Ghost,
  Youtube,
  Video,
  Music,
  Twitch,
  Send,
  MessageSquare,
  Github,
  Gitlab,
  Code,
  BookOpen,
  Mail,
  FileText,
  HardDrive,
  Link,
  Tv,
  Smartphone,
  Search,
  Figma,
  Dribbble,
  Palette,
  Gamepad2,
  CircleHelp,
  Slack,
  Box,
  AtSign,
} from "lucide-react";

export const ICON_MAP = {
  "shopping-bag": ShoppingBag,
  "shopping-cart": ShoppingCart,
  instagram: Instagram,
  "music-2": Music2,
  twitter: Twitter,
  facebook: Facebook,
  linkedin: Linkedin,
  image: Image,
  "message-circle": MessageCircle,
  ghost: Ghost,
  youtube: Youtube,
  video: Video,
  music: Music,
  twitch: Twitch,
  send: Send,
  "message-square": MessageSquare,
  github: Github,
  gitlab: Gitlab,
  code: Code,
  "book-open": BookOpen,
  mail: Mail,
  "file-text": FileText,
  "hard-drive": HardDrive,
  link: Link,
  tv: Tv,
  smartphone: Smartphone,
  search: Search,
  figma: Figma,
  dribbble: Dribbble,
  palette: Palette,
  "gamepad-2": Gamepad2,
  "circle-help": CircleHelp,
  slack: Slack,
  box: Box,
  "at-sign": AtSign,
};
