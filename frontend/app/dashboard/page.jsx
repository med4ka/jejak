import DashboardClient from "./DashboardClient";

// Wrapper server supaya /dashboard punya title sendiri yang jelas
// (komponen client tidak boleh export metadata di App Router).
export const metadata = {
  title: "Dashboard — Jejak",
  description: "Edit profil kreator dan lihat link + analytics.",
};

export default function DashboardPage() {
  return (
    <div className="mx-auto w-full max-w-2xl px-4 pb-16 pt-24">
      <DashboardClient />
    </div>
  );
}
