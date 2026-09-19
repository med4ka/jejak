// Tema preset untuk halaman publik /u/[username]. Backend hanya menyimpan key
// string dalam set tertutup {classic, darkroom, coral, glass}; mapping
// class ada di sini satu-satunya sumber kebenaran.
//
// Field per preset:
// - label          — nama tampil (dropdown/picker).
// - page           — warna dasar "halaman" (body rule tambahan di globals.css
//                    untuk background kompleks: body[data-profile-theme=...]).
// - border         — warna border (dipakai header/card/avatar/pill).
// - borderW        — lebar border normal (default border-2; glass 1px).
// - borderStrong   — lebar border link UNGGULAN (glass = sama 1px).
// - radius / radiusLarge / radiusFull — sudut card, header, elemen pill/avatar.
// - heading        — class heading nama profil (size + weight).
// - headingFont    — font family untuk heading (Space Grotesk utk semua
//                    preset; dipakai DashboardClient h1/h2 saat dashboard ikut
//                    tema). Disimpan sebagai field supaya tetap konsisten
//                    dengan preset yang nanti punya font khas.
// - card           — background+teks card header & link (glass = blur translucent).
// - caption        — teks caption kecil.
// - avatar         — fallback inisial (dan bar avatar di preview picker).
// - accent         — badge "★ Unggulan".
// - badge          — badge "TRENDING".
// - text / textMuted — warna teks primer/sekunder (dipakai dashboard saat
//                    dashboard ikut tema kreator; glass: putih SOLID, bukan
//                    opacity rendah — kontras wajib di atas blob manapun).
// - placeholder    — class placeholder untuk input/textarea (glass: putih/50,
//                    dark themes: putih/50, light themes: muted).
// - featuredClass  — kelas tambahan EKSTRA untuk card UNGGULAN (kosong utk
//                    semua preset aktif; dipertahankan mekanismenya).
// - chip           — background SOLID untuk card preview di PICKER dashboard.
//                    Picker butuh swatch yang TETAP terlihat walau dashboard
//                    terang (saat tema classic/coral aktif): kalau pakai
//                    st.card langsung, glass (bg transparan + teks putih)
//                    jadi putih-di-atas-putih dan "menghilang" dari picker
//                    (lihat haul bug: picker Glass hanya muncul saat tema
//                    gelap aktif). chip dipakai HANYA di picker, bukan di
//                    halaman publik. glass memakai panggung #1C1C1E agar
//                    frosted-nya nyata.
//
// Semua warna tetap pakai token DESIGN.md §1, dengan pengecualian terencana:
// darkroom #121212; glass = panggung abu netral #1C1C1E (ala macOS/iOS Control
// Center) + blob putih/abu netral di ThemeBackdrop (efek cahaya lembut, BUKAN
// pelangi — lihat ThemeBackdrop.jsx).
export const THEMES = ["classic", "darkroom", "coral", "glass"];

export const THEME_STYLES = {
  classic: {
    label: "Classic",
    page: "bg-print-white text-ink",
    border: "border-ink",
    borderW: "border-2",
    borderStrong: "border-4",
    radius: "rounded-xl",
    radiusLarge: "rounded-2xl",
    radiusFull: "rounded-full",
    heading: "text-2xl font-bold",
    headingFont: "font-display",
    card: "bg-print-white text-ink",
    caption: "text-muted",
    avatar: "bg-flash-yellow text-ink",
    accent: "bg-flash-yellow text-ink",
    badge: "bg-flash-coral text-print-white",
    text: "text-ink",
    textMuted: "text-muted",
    placeholder: "placeholder:text-muted",
    featuredClass: "",
    chip: "bg-print-white border-ink text-ink",
  },
  darkroom: {
    label: "Darkroom",
    page: "bg-[#121212] text-print-white",
    border: "border-print-white",
    borderW: "border-2",
    borderStrong: "border-4",
    radius: "rounded-xl",
    radiusLarge: "rounded-2xl",
    radiusFull: "rounded-full",
    heading: "text-2xl font-bold",
    headingFont: "font-display",
    card: "bg-[#121212] text-print-white",
    caption: "text-print-white/70",
    avatar: "bg-flash-coral text-print-white",
    accent: "bg-flash-coral text-print-white",
    badge: "bg-flash-coral text-print-white",
    text: "text-print-white",
    textMuted: "text-print-white/70",
    placeholder: "placeholder:text-print-white/50",
    featuredClass: "",
    chip: "bg-[#121212] border-print-white text-print-white",
  },
  coral: {
    label: "Coral",
    page: "bg-print-white text-ink",
    border: "border-ink",
    borderW: "border-2",
    borderStrong: "border-4",
    radius: "rounded-xl",
    radiusLarge: "rounded-2xl",
    radiusFull: "rounded-full",
    heading: "text-2xl font-bold",
    headingFont: "font-display",
    card: "bg-print-white text-ink",
    caption: "text-muted",
    avatar: "bg-flash-orange text-ink",
    accent: "bg-flash-orange text-ink",
    badge: "bg-flash-orange text-ink",
    text: "text-ink",
    textMuted: "text-muted",
    placeholder: "placeholder:text-muted",
    featuredClass: "",
    chip: "bg-print-white border-ink text-ink",
  },
  glass: {
    label: "Glass",
    page: "text-ink",
    border: "border-ink/10",
    borderW: "border-[1px]",
    borderStrong: "border-[1px]",
    radius: "rounded-xl",
    radiusLarge: "rounded-2xl",
    radiusFull: "rounded-full",
    heading: "text-2xl font-bold",
    headingFont: "font-display",
    // Glass = mode TERANG penuh (macOS Control Center light): backdrop
    // radial-gradient LEMBUT putih→abu terang (#F5F5F5→#E8E8E8, dirender
    // ThemeBackdrop — BUKAN foto noise), card frosted PUTIH transparan +
    // blur, teks INK gelap. Grain halus (3-5%) dipasang GLOBAL lewat
    // body[data-profile-theme]::after di globals.css → berlaku juga ke
    // classic/darkroom/coral (satu sumber kebenaran tekstur).
    card: "bg-white/60 backdrop-blur-2xl text-ink",
    caption: "text-ink/70",
    avatar: "bg-white/55 backdrop-blur-2xl text-ink",
    accent: "bg-white/55 backdrop-blur-2xl text-ink",
    badge: "bg-white/55 backdrop-blur-2xl text-ink",
    text: "text-ink",
    textMuted: "text-ink/70",
    placeholder: "placeholder:text-ink/50",
    featuredClass: "",
    // Chip picker: SOLID PUTIH + border ink supaya swatch selalu kelihatan
    // di dashboard mana pun (frosted transparan justru jadi putih-atas-putih).
    chip: "bg-print-white border-ink text-ink",
  },
};

export function themeStyles(key) {
  return THEME_STYLES[key] || THEME_STYLES.classic;
}