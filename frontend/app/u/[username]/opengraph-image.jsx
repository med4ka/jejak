import { ImageResponse } from "next/og";
import { googleFont } from "../../../lib/ogFonts";

// =====================================================================
// DYNAMIC OG IMAGE per creator (Next file convention -> automatically used
// as og:image + twitter:image, WITHOUT duplicating it in generateMetadata).
// Spec: 1200x630, flash-yellow bg, 120px avatar, display name in Space
// Grotesk 700 60px, @username, bio in Work Sans 24px max 2 lines,
// top 3 links as outline pills with ink shadow, Jejak logo bottom-right.
// Google fonts are "all-or-nothing": if both load -> use them; if any fails
// -> fall back to next/og's default ("sans serif") so no fontFamily is
// unloaded in satori (which would fail the render).
// =====================================================================

const GO_API_URL = process.env.GO_API_URL || "http://localhost:8081";

const INK = "#1C1A12";
const YELLOW = "#FFD23F";

export const size = { width: 1200, height: 630 };
export const contentType = "image/png";
export const alt = "Pratinjau link-in-bio Jejak";

async function fetchProfile(username) {
  try {
    const res = await fetch(`${GO_API_URL}/api/u/${username}`, { cache: "no-store" });
    if (!res.ok) {
      return null;
    }
    return await res.json();
  } catch {
    return null;
  }
}

async function fetchAvatar(url) {
  if (!url) {
    return null;
  }
  try {
    const absolute = url.startsWith("/") ? `${GO_API_URL}${url}` : url;
    const res = await fetch(absolute, { cache: "force-cache" });
    if (!res.ok) {
      return null;
    }
    return await res.arrayBuffer();
  } catch {
    return null;
  }
}

function linkLabel(l) {
  return l.original_url || `/${l.short_code}`;
}

function Card({ profile, avatar, titleFont, bodyFont }) {
  const name = profile ? profile.display_name : "Jejak";
  const username = profile ? profile.username : "jejak";
  const bio = profile
    ? (profile.bio || "").slice(0, 160)
    : "Link-in-bio untuk kreator Indonesia";
  // Same order as the page: featured first, then by highest click count.
  const topLinks = [...(profile?.links || [])]
    .sort(
      (a, b) =>
        (b.is_featured ? 1 : 0) - (a.is_featured ? 1 : 0) ||
        b.click_count - a.click_count
    )
    .slice(0, 3);

  return (
    <div
      style={{
        width: "100%",
        height: "100%",
        backgroundColor: YELLOW,
        display: "flex",
        flexDirection: "column",
        justifyContent: "space-between",
        padding: 72,
        fontFamily: bodyFont,
      }}
    >
      <div style={{ display: "flex", flexDirection: "column", gap: 24 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 28 }}>
          {avatar ? (
            <img
              src={avatar}
              width={120}
              height={120}
              style={{
                borderRadius: 60,
                border: `4px solid ${INK}`,
                objectFit: "cover",
              }}
            />
          ) : (
            <div
              style={{
                width: 120,
                height: 120,
                borderRadius: 60,
                border: `4px solid ${INK}`,
                backgroundColor: "#FAFAF7",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                fontSize: 56,
                fontWeight: 700,
                color: INK,
                fontFamily: titleFont,
              }}
            >
              {name.slice(0, 1).toUpperCase()}
            </div>
          )}
          <div style={{ display: "flex", flexDirection: "column", gap: 4, minWidth: 0 }}>
            <div
              style={{
                fontSize: 60,
                fontWeight: 700,
                color: INK,
                fontFamily: titleFont,
                whiteSpace: "nowrap",
                overflow: "hidden",
                textOverflow: "ellipsis",
                maxWidth: 820,
              }}
            >
              {name.slice(0, 24)}
            </div>
            {/* Gabung jadi 1 expression: literal "@"+{username} = 2 child
                node → satori minta display:flex di parent (padded text). */}
            <div style={{ fontSize: 32, color: INK, opacity: 0.7 }}>{`@${username}`}</div>
          </div>
        </div>
        {bio !== "" && (
          <div style={{ fontSize: 24, color: INK, opacity: 0.85, maxWidth: 960 }}>{bio}</div>
        )}
        <div
          style={{
            display: "flex",
            flexDirection: "column",
            gap: 14,
            marginTop: 8,
          }}
        >
          {topLinks.map((l) => (
            <div
              key={l.short_code}
              style={{
                alignSelf: "flex-start",
                maxWidth: 880,
                backgroundColor: "#FFFFFF",
                border: `3px solid ${INK}`,
                borderRadius: 999,
                padding: "14px 28px",
                fontSize: 24,
                fontWeight: 700,
                color: INK,
                boxShadow: `4px 4px 0 ${INK}`,
                whiteSpace: "nowrap",
                overflow: "hidden",
                textOverflow: "ellipsis",
              }}
            >
              {linkLabel(l)}
            </div>
          ))}
        </div>
      </div>

      {/* Logo kanan-bawah: wordmark + dot putih ber-outline (mirror dari
          dot kuning di navbar; di bg kuning, putih yang terlihat). */}
      <div
        style={{
          display: "flex",
          justifyContent: "flex-end",
          alignItems: "center",
          gap: 10,
        }}
      >
        <div
          style={{
            fontSize: 44,
            fontWeight: 700,
            color: INK,
            fontFamily: titleFont,
            letterSpacing: -1,
          }}
        >
          Jejak
        </div>
        <div
          style={{
            width: 18,
            height: 18,
            borderRadius: 5,
            backgroundColor: "#FFFFFF",
            border: `3px solid ${INK}`,
          }}
        />
      </div>
    </div>
  );
}

export default async function OpengraphImage({ params }) {
  const [profile, sg, ws] = await Promise.all([
    fetchProfile(params.username),
    googleFont("Space Grotesk", 700),
    googleFont("Work Sans", 400),
  ]);
  const avatar = await fetchAvatar(profile?.avatar_url);
  const both = Boolean(sg && ws);
  return new ImageResponse(
    <Card
      profile={profile}
      avatar={avatar}
      titleFont={both ? "Space Grotesk" : "sans serif"}
      bodyFont={both ? "Work Sans" : "sans serif"}
    />,
    { width: size.width, height: size.height, fonts: both ? [sg, ws] : undefined }
  );
}
