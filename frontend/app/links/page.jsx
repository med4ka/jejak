"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";

// Halaman /links di-DEPRECATE: versi lama menampilkan SEMUA short-link dari
// SEMUA kreator ke publik tanpa filter kepemilikan (kebocoran privasi kecil).
// Dashboard Profil sudah cukup untuk lihat link+analytics milik sendiri.
// Route ini dipertahankan hanya sebagai pintu redirect supaya bookmark lama
// tidak 404: login -> /dashboard, belum login -> /.
export default function LinksRedirect() {
  const router = useRouter();

  useEffect(() => {
    const username = localStorage.getItem("jejak_username");
    router.replace(username ? "/dashboard" : "/");
  }, [router]);

  return (
    <main>
      <p className="text-sm text-muted">Mengalihkan...</p>
    </main>
  );
}
