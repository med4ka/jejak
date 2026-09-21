import { Space_Grotesk, Archivo_Black, Work_Sans, JetBrains_Mono, Caveat } from "next/font/google";
import "./globals.css";
import Navbar from "./components/Navbar";
import PageTransition from "./components/PageTransition";

export const metadata = {
  // metadataBase wajib supaya path relatif (mis. /og-default.png) jadi URL
  // absolut di meta tag — scraper WA/Twitter/Discord butuh URL absolut.
  // Set SITE_URL saat deploy production, default localhost untuk dev.
  metadataBase: new URL(process.env.SITE_URL || "http://localhost:3000"),
  title: "Jejak — Link-in-bio untuk Kreator",
  description: "Satu halaman untuk semua link kreator, lengkap dengan click analytics.",
};

// DESIGN.md §2 — Caveat (caption) hanya dipakai di 2 tempat: caption card
// link + header profil (§3). Jangan pakai font-caption di tempat lain.
const display = Space_Grotesk({ subsets: ["latin"], weight: ["500", "700"], variable: "--font-display" });
const blackhead = Archivo_Black({ subsets: ["latin"], weight: "400", variable: "--font-blackhead" });
const body = Work_Sans({ subsets: ["latin"], variable: "--font-body" });
const mono = JetBrains_Mono({ subsets: ["latin"], variable: "--font-mono" });
const caption = Caveat({ subsets: ["latin"], weight: ["600"], variable: "--font-caption" });

export default function RootLayout({ children }) {
  return (
    <html lang="id" className={`${display.variable} ${blackhead.variable} ${body.variable} ${mono.variable} ${caption.variable}`}>
      <body className="min-h-screen bg-print-white font-body text-ink">
        <Navbar />
      <div>{children}</div>
      </body>
    </html>
  );
}
