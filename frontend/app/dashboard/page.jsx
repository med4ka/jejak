import { Suspense } from "react";
import DashboardClient from "./DashboardClient";

// Server wrapper so /dashboard can declare its own clear title (client
// components may not export metadata in the App Router).
export const metadata = {
  title: "Dashboard",
  description: "Edit profil kreator dan lihat link + analytics.",
};

export default function DashboardPage() {
  return (
    <div className="mx-auto w-full max-w-2xl px-4 pb-16 pt-24">
      {/* Suspense: DashboardClient reads ?filter=broken via useSearchParams
          (bell aggregate); a Client Component with useSearchParams inside a
          prerendered page must sit behind a boundary or the build fails. */}
      <Suspense fallback={null}>
        <DashboardClient />
      </Suspense>
    </div>
  );
}
