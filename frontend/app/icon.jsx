import { ImageResponse } from "next/og";
import { googleFont } from "../lib/ogFonts";

// =====================================================================
// FAVICON (Next file convention: automatically linked as <link rel="icon">).
// Design: a 32×32 rounded yellow square (flash-yellow #FFD23F) with a bold
// ink "J" and a thin ink outline so it stays visible on dark tabs: a
// miniature version of the logo. Space Grotesk 700 is fetched from Google
// (cached in ogFonts.js); if the fetch fails next/og falls back to its
// default font ("sans serif") and the image still renders.
// Note: the vendored next/dist/compiled/@vercel/og has a Windows asset
// loading bug (join(import.meta.url,...) → ERR_INVALID_URL): patched
// automatically by scripts/fix-og-windows.js (postinstall). See
// opengraph-image.jsx.
// =====================================================================

export const size = { width: 32, height: 32 };
export const contentType = "image/png";
export const alt = "Jejak";

const INK = "#1C1A12";
const YELLOW = "#FFD23F";

export default async function Icon() {
  const sg = await googleFont("Space Grotesk", 700);
  return new ImageResponse(
    (
      <div
        style={{
          width: "100%",
          height: "100%",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          backgroundColor: YELLOW,
          border: `1px solid ${INK}`,
          borderRadius: 6,
        }}
      >
        <span
          style={{
            fontSize: 21,
            fontWeight: 700,
            color: INK,
            fontFamily: sg ? "Space Grotesk" : "sans serif",
          }}
        >
          J
        </span>
      </div>
    ),
    { width: size.width, height: size.height, fonts: sg ? [sg] : undefined }
  );
}
