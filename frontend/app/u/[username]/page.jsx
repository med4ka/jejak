import { notFound } from "next/navigation";
import ProfileLinks from "../../components/ProfileLinks";
import { themeStyles } from "../../../lib/themes";

const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

// DESIGN.md §5 — halaman profil publik: mobile-first 1 kolom, background
// print-white. Data diambil server-side; animasi entrance di ProfileLinks
// (client component — framer-motion tidak bisa di server component).
// Tema (classic|darkroom|coral) hanya mengubah pemetaan class; PRESET TERTUTUP
// — key tak dikenal di-coerce backend ke "classic", dan themeStyles() di
// frontend juga fallback aman sehingga tidak pernah render warna tak wajar.
function tiltFor(i) {
  return ((i * 37) % 5) - 2;
}

async function fetchProfile(username) {
  // Fetch yang sama dipakai generateMetadata + halaman; Next.js
  // mendedup fetch identik dalam 1 request jadi cuma 1 HTTP call.
  const res = await fetch(`${GO_API_URL}/api/u/${username}`, { cache: "no-store" });
  if (res.status === 404) {
    return null;
  }
  if (!res.ok) {
    throw new Error("Gagal memuat profil kreator");
  }
  return res.json();
}

export async function generateMetadata({ params }) {
  const fallback = {
    title: "Profil tidak ditemukan — Jejak",
    description: "Halaman kreator Jejak.",
  };
  try {
    const profile = await fetchProfile(params.username);
    if (!profile) {
      return fallback;
    }
    const bio = (profile.bio || "").slice(0, 160);
    const title = `${profile.display_name} (@${profile.username}) — Jejak`;
    const description = bio !== "" ? bio : `Link-in-bio ${profile.username} di Jejak`;
    const images = [profile.avatar_url || "/og-default.png"];
    return {
      title,
      description,
      openGraph: { title, description, images, type: "profile" },
      twitter: { card: "summary_large_image", title, description, images },
    };
  } catch {
    return fallback;
  }
}

export default async function CreatorPage({ params }) {
  const profile = await fetchProfile(params.username);
  if (!profile) {
    notFound();
  }

  const links = [...(profile.links || [])]
    // Link unggulan selalu PALING ATAS (di atas urutan click_count biasa),
    // sisanya tetap urutan lama (click tertinggi dulu).
    .sort(
      (a, b) =>
        (b.is_featured ? 1 : 0) - (a.is_featured ? 1 : 0) ||
        b.click_count - a.click_count
    )
    .map((l, i) => ({ ...l, tilt: tiltFor(i) }));
  const top = links.length > 0 && links[0].click_count > 0 ? links[0].short_code : null;
  const t = themeStyles(profile.theme);

  return (
    <main>
      {/* Header profil: kartu terang + caption tulisan tangan (§3, lokasi ke-2). */}
      <section className={`${t.radiusLarge} ${t.borderW} p-6 ${t.border} ${t.card}`}>
        <div className="flex items-center gap-4">
          {profile.avatar_url ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={profile.avatar_url}
              alt={`Avatar ${profile.display_name}`}
              className={`${t.radiusFull} ${t.borderW} h-20 w-20 shrink-0 object-cover ${t.border}`}
            />
          ) : (
            <div
              aria-hidden="true"
              className={`${t.radiusFull} ${t.borderW} flex h-20 w-20 shrink-0 items-center justify-center font-display text-2xl font-bold ${t.border} ${t.avatar}`}
            >
              {profile.display_name.slice(0, 1).toUpperCase()}
            </div>
          )}
          <div className="min-w-0">
            <h1 className={`${t.headingFont} ${t.heading}`}>{profile.display_name}</h1>
            <p className={`font-caption text-xl leading-tight ${t.caption}`}>@{profile.username}</p>
          </div>
        </div>
        {profile.bio !== "" && <p className="mt-3 text-sm">{profile.bio}</p>}
        {Array.isArray(profile.socials) && profile.socials.length > 0 && (
          <div className="mt-3 flex flex-wrap gap-2">
            {profile.socials.map((s, i) => (
              <a
                key={`${s.platform}-${i}`}
                href={s.url}
                target="_blank"
                rel="noreferrer"
                className={`${t.radiusFull} ${t.borderW} px-3 py-0.5 text-xs font-bold uppercase tracking-wide ${t.border} ${t.card}`}
              >
                {s.platform}
              </a>
            ))}
          </div>
        )}
      </section>

      <section className="mt-4 flex flex-col gap-4">
        <ProfileLinks links={links} top={top} apiBase={GO_API_URL} theme={profile.theme} />
      </section>
    </main>
  );
}
