"use client";

import { useEffect, useState } from "react";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  ResponsiveContainer,
} from "recharts";
import { themeStyles } from "../../lib/themes";

const MONTHS = ["Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"];

// "2026-09-08" -> "8 Sep" (singkat, cukup untuk sumbu X yang padat).
function shortDate(iso) {
  const parts = iso.split("-");
  if (parts.length !== 3) {
    return iso;
  }
  return `${parseInt(parts[2], 10)} ${MONTHS[parseInt(parts[1], 10) - 1] || ""}`;
}

// Warna graf per tema (Task 3 — dulu hardcode untuk panggung CLASSIC saja:
// AXIS_TICK fill #8A8578 + grid #1C1A12 + card bg-paper-grey; akibatnya
// pas dashboard ikut tema glass/darkroom, card & teks sumbu tidak nyambung
// dengan tema berikutnya):
//   - classic/coral (panggung terang): teks sumbu INK lembut #6E6A5E, grid
//     INK tipis. Cocok di atas card print-white / paper-grey terang.
//   - darkroom (panggung gelap):      teks sumbu PRINT-WHITE lembut #B5B2A6,
//     grid PRINT-WHITE tipis — kontras tetap kebaca di atas card #121212.
//   - glass (kini LIGHT penuh):       teks sumbu INK lembut #6E6A5E, grid
//     INK tipis — terbaca di card frosted PUTIH + blur.
// Line chart juga ikut tema: coral = aksen oranye brand, glass = abu-abu
// (siluet di atas kaca), classic & darkroom tetap kuning flash brand.
const CHART_COLORS = {
  classic: { axisTick: "#6E6A5E", grid: "#1C1A12", gridOpacity: 0.12, line: "#FFD23F" },
  darkroom: { axisTick: "#B5B2A6", grid: "#F5F5F0", gridOpacity: 0.16, line: "#FF6B1A" },
  coral: { axisTick: "#6E6A5E", grid: "#1C1A12", gridOpacity: 0.12, line: "#FF6B1A" },
  glass: { axisTick: "#6E6A5E", grid: "#1C1A12", gridOpacity: 0.14, line: "#9AA0A8" },
};

export default function ClicksChart({ theme = "classic", st }) {
  const [data, setData] = useState(null);
  const [error, setError] = useState("");
  const styles = st || themeStyles(theme);
  const cc = CHART_COLORS[theme] || CHART_COLORS.classic;
  const axisTick = { fontFamily: "var(--font-mono)", fontSize: 11, fill: cc.axisTick };

  useEffect(() => {
    fetch("/api/analytics/clicks-by-day")
      .then(async (res) => {
        const text = await res.text();
        let parsed = [];
        try {
          parsed = JSON.parse(text);
        } catch {
          throw new Error(text || "Gagal memuat grafik");
        }
        if (!res.ok) {
          throw new Error(parsed.error || "Gagal memuat grafik");
        }
        setData(Array.isArray(parsed) ? parsed : []);
      })
      .catch((err) => setError(err.message));
  }, []);

  // Card grafik ikut tema (Task 3): radius, lebar border, warna border,
  // bg+teks pakai st.* dari tema aktif — BUKAN hardcode bg-paper-grey/ink.
  const cardClass = `${styles.radiusLarge} ${styles.borderW} ${styles.border} ${styles.card} ${styles.text} p-5`;

  return (
    <section className={cardClass}>
      <h2 className={`font-display text-lg font-bold ${styles.text}`}>Klik 30 hari terakhir</h2>

      {error !== "" && <p className={`mt-2 text-sm ${styles.textMuted}`}>{error}</p>}

      {error === "" && data === null && (
        <p className={`mt-2 text-sm ${styles.textMuted}`}>Memuat grafik...</p>
      )}

      {error === "" && data !== null && data.every((d) => d.count === 0) && (
        <p className={`mt-2 text-sm ${styles.textMuted}`}>
          Belum ada klik 30 hari terakhir — bagikan link-mu supaya grafiknya hidup.
        </p>
      )}

      {error === "" && data !== null && data.some((d) => d.count > 0) && (
        <div className="mt-3 h-[220px] w-full">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={data} margin={{ top: 4, right: 8, bottom: 0, left: -12 }}>
              <CartesianGrid stroke={cc.grid} strokeOpacity={cc.gridOpacity} vertical={false} />
              <XAxis
                dataKey="date"
                tickFormatter={shortDate}
                tick={{ fontFamily: "var(--font-mono)", fontSize: 11, fill: cc.axisTick }}
                minTickGap={32}
                tickLine={false}
                axisLine={{ stroke: cc.grid, strokeOpacity: cc.gridOpacity }}
              />
              <YAxis
                allowDecimals={false}
                tick={{ fontFamily: "var(--font-mono)", fontSize: 11, fill: cc.axisTick }}
                tickLine={false}
                axisLine={false}
                width={36}
              />
              <Line
                type="monotone"
                dataKey="count"
                stroke={cc.line}
                strokeWidth={2.5}
                dot={false}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>
      )}
    </section>
  );
}
