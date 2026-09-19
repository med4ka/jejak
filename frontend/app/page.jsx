import { redirect } from "next/navigation";

// SEMENTARA (TAHAP B): konten homepage lama sudah pindah utuh ke /app.
// Halaman ini hanya mengalihkan "/" -> "/app" supaya tidak ada tautan yang
// rusak. Landing page yang sebenarnya menyusul di tahap berikutnya.
export default function Home() {
  redirect("/app");
}
