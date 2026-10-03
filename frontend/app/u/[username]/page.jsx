import { notFound } from "next/navigation";
import ProfileLinks from "../../components/ProfileLinks";
import { themeStyles } from "../../../lib/themes";

const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

// DESIGN.md §5: public profile page: mobile-first single column,
// print-white background. Data is fetched server-side; entrance animations live
// in ProfileLinks (a client component: framer-motion cannot run inside a
// server component). The theme only remaps class names; the preset set is
// CLOSED: the backend coerces unknown keys to "classic", and themeStyles() on
// the frontend also falls back safely, so unexpected colors are never rendered.
function tiltFor(i) {
  return ((i * 37) % 5) - 2;
}

async function fetchProfile(username) {
  // The same fetch serves both generateMetadata and the page; Next.js
  // deduplicates identical fetches within a single request, resulting in
  // only 1 HTTP call.
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
    title: "Profil tidak ditemukan: Jejak",
    description: "Halaman kreator Jejak.",
  };
  try {
    const profile = await fetchProfile(params.username);
    if (!profile) {
      return fallback;
    }
    const bio = (profile.bio || "").slice(0, 160);
    const title = `${profile.display_name} (@${profile.username}): Jejak`;
    const description = bio !== "" ? bio : `Link-in-bio ${profile.username} di Jejak`;
    // og:image / twitter:image are deliberately NOT declared here: the
    // opengraph-image.jsx file (Next file convention, same folder) injects a
    // per-creator 1200×630 image automatically: declaring them below as well
    // would give crawlers 2 <meta> og:image tags (duplicates).
    return {
      title,
      description,
      openGraph: { title, description, type: "profile" },
      twitter: { card: "summary_large_image", title, description },
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
    // Featured links always sort to the TOP (above the regular click_count
    // order); the rest keeps the previous ordering (highest click count first).
    .sort(
      (a, b) =>
        (b.is_featured ? 1 : 0) - (a.is_featured ? 1 : 0) ||
        b.click_count - a.click_count
    )
    .map((l, i) => ({ ...l, tilt: tiltFor(i) }));
  const top = links.length > 0 && links[0].click_count > 0 ? links[0].short_code : null;
  const t = themeStyles(profile.theme);

  return (
    <main className="mx-auto w-full max-w-2xl px-4 pb-16 pt-28">
      {/* Profile header: bright card + handwritten caption (§3, second occurrence). */}
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
        {/* apiBase REMOVED (redirect fix 2026-09-30): links are rendered as the
            relative path /r/{code} so they pass through the Next proxy
            (app/r/[code]/route.js) instead of pointing directly at GO_API_URL.
            GO_API_URL remains in use above ONLY for server-side data fetching. */}
        <ProfileLinks links={links} top={top} theme={profile.theme} />
      </section>
    </main>
  );
}
