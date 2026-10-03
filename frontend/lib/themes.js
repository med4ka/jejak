// Preset themes for the public profile page /u/[username]. The backend stores
// only a string key from the closed set {classic, darkroom, coral, glass,
// risoPrint, peach, lavender, matcha, sakura, ocean, sunset}; the class
// mapping in this file is the single source of truth.
//
// SCOPE (decision 2026-09-29): themes apply ONLY to /u/[username] plus the
// navbar while that page is being shown (read through
// body[data-profile-theme], which is set only by ProfileLinks). Dashboard,
// /app, and the landing page MUST render Instant Print (classic): do not
// attach the theme attribute there.
//
// RATIONALE: the chrome (navbar) must follow the public page theme without
// knowing which page is active: NavbarClient reads body[data-profile-theme]
// via a MutationObserver and then selects tokens from this file. Therefore
// ALL chrome colors: bar, link hover, buttons, dropdown panel, drawer:
// must be TOKEN FIELDS here rather than hardcoded ternaries in NavbarClient:
// one source of truth, and adding a theme amounts to adding one preset
// object.
//
// Fields per preset:
// - label         : display name (dropdown/picker).
// - page          : base "page" color (extra body rule in globals.css for
//                    complex backgrounds: body[data-profile-theme=...]).
// - border        : border color (used by header/card/avatar/pill).
// - borderW       : normal border width (default border-2; glass 1px).
// - borderStrong  : border width for FEATURED links (glass: same 1px).
// - radius / radiusLarge / radiusFull: corner radii for card, header, and
//                    pill/avatar elements.
// - heading       : profile-name heading class (size + weight).
// - headingFont   : font family for headings (Space Grotesk for every
//                    preset).
// - card          : header/link card background + text (glass: translucent
//                    blur).
// - caption       : small caption text.
// - avatar        : fallback initials (and avatar bar in the picker
//                    preview).
// - accent        : "★ Unggulan" (featured) badge.
// - badge         : "TRENDING" badge.
// - text / textMuted: primary/secondary text colors.
// - placeholder   : input/textarea placeholder class (glass: iOS-style
//                    #8E8E93, dark themes: white/50, light themes: muted).
// - featuredClass : EXTRA additional class for the FEATURED card (empty for
//                    every active preset; the mechanism is retained).
// - chip          : SOLID background for preview cards in the dashboard
//                    PICKER. The picker needs a swatch that stays visible
//                    even on a light dashboard: glass uses solid white with
//                    an ink border (a frosted translucent surface would be
//                    white-on-white and "disappear").
//                    chip is used ONLY in the picker, never on the public
//                    page.
// - shadow        : shadow for link cards on the public page. Instant Print
//                    = hard offset (a print stuck to the stage); glass = the
//                    Liquid Glass recipe: inset top-edge highlight + long
//                    drop shadow (see the comment on the glass preset).
//
// ---- CHROME TOKENS (navbar): consumed by NavbarClient, never hardcoded ----
// - nav           : navbar bar class (bg + border-b) following the theme
//                    stage.
// - navHover      : nav link hover (classic: flash-yellow; coral: dark
//                    #C43A20 so it passes WCAG AA at 4.7:1: #FF5C3D on
//                    cream is only 2.7:1; darkroom: orange accent).
// - cta           : primary navbar button ("Link Baru"/"Daftar Gratis").
// - ghost         : secondary button ("Masuk") + hamburger button.
// - logoDot       : small square in the JEJAK logo.
// - panel         : account dropdown panel (border + bg + blur for glass).
// - panelHover    : hover state for dropdown/drawer items.
// - panelRule     : separator line (border-t/b) inside the panel.
// - drawer        : mobile drawer panel (left border only).
// - navCircle     : circular chrome elements: account avatar + drawer close
//                    button.
//
// Palette per theme (per the "Refine 4 themes" spec):
// - classic : UNCHANGED: full Instant Print (#FAFAF7/ink/yellow).
// - darkroom: deep #121212 stage, #F5F5F5 text, ORANGE accent #FF6B35
//              (border + shadow follow the accent: "darkroom photo
//              developing").
// - coral   : warm cream stage #F5F1E8, CORAL accent #FF5C3D (ink text for
//              small badges), orange badge, ink shadow.
// - glass   : Apple Liquid Glass (WWDC25): paper = linear-gradient(135deg,
//              #F0F0F0,#FFF,#F5F5F7) in globals.css; every surface material
//              = backdrop blur(20px) saturate(180%), white/70 background (for
//              text contrast), 1px white/28 rim-light border with /55 on top,
//              box-shadow inset 1px highlight + 0 12px 40px/35% drop.
// - risoPrint: Risograph (preset 5, 2026-09-29): cream paper #F4F0E6 + SVG
//              grain 0.03 (globals.css body rule + .riso-paper on cards),
//              ink blue #1F3A5F (not Instant Print black), accents
//              #FF5C39/#2B7A78, hard shadow tinted with riso ink, 8px radius.
//
// ---- 4 "playful/cute" themes (presets 6-9, 2026-09-30) ----
// Palettes exactly per the spec; character differs through
// RADIUS/SHADOW/BORDER:
// - peach    : warm & summery: paper #FFF1E6, brown ink #2B1F1A (not
//               black), accents #FF9B7B + #FFC8A8, 2px ink border, 4px 4px 0
//               ink hard shadow, 12px radius.
// - lavender : soft & dreamy: paper #F5F0FF, purple ink #2A2140, accents
//               #A88BEB + #C9B6F0, 2px ink border, SOFT shadow
//               6px 6px 0 rgba(42,33,64,.15) (not hard), 16px radius.
// - matcha   : zen & earthy: paper #EFF5EC, green ink #1F2E1A, accents
//               #7BA05B + #A8C68A, thinner 1.5px border, subtle 3px 3px 0
//               ink shadow, 10px radius → recycled-paper impression.
// - sakura   : kawaii & gentle: paper #FFF5F8, pink ink #2B1F26, accents
//               #F5A4B8 + #FAD0DA, TINTED pink shadow 4px 4px 0
//               rgba(245,164,184,.5), 16px radius, accent bar on the RIGHT
//               of the card.
//
// ---- 2 new themes "Ocean + Sunset" (presets 10-11, 2026-10-01) ----
// Palettes exactly per the spec; character differs through RADIUS/SHADOW:
// - ocean   : deep, calm, marine: paper #E8F2F7, nautical blue ink #0F2A3D,
//              accents #4A90B8 + #7BB3D1, 2px ink border, 5px 5px 0 ink hard
//              shadow, 12px radius, LEFT accent bar 4px.
// - sunset  : warm, dramatic, golden hour: paper #FFF3E0, brown ink
//              #3D1F0F, accents #F5A623 + #E8634A, 2px ink border, 4px 4px 0
//              ink hard shadow, 14px radius, LEFT accent bar 4px.
// WCAG audit (see the PROGRESS entry): ink@paper 13.02/13.66, muted 0.68
// 4.97/5.11, navHover 5.24 (#2A6A8A) / 4.57 (#A35F00: the specified
// #C77800 is only 3.13:1 → darkened; audit rule: text is darkened, bg/border
// stay at the specification). Text on the ocean accent uses the darker ink
// #0A1F2E (4.78:1; ink #0F2A3D on #4A90B8 is only 4.21:1 for the 11px
// badge).
//
// OPTIONAL TOKENS: used by risoPrint + the 6 newer themes (peach/lavender/
// matcha/sakura/ocean/sunset); the older presets (classic/darkroom/coral/
// glass) LACK them → legacy behavior is identical; see the conditions in
// ProfileLinks/DashboardClient:
// - swatch       : array of 3 hex values for the picker preview (ink/
//                   accent/paper strips).
// - slug         : slug row /code class (riso: mono, not Caveat).
// - url          : URL text class on the card (muted ink/65: opacity AA;
//                   riso /[0.72]).
// - countColor   : click-count number class (riso: dark orange #C43A20, AA
//                   text).
// - barPrimary/barSecondary: card accent bar (alternates by card index;
//                   newer themes use the SAME color for both → uniform bar,
//                   sakura uses border-r / right).
export const THEMES = [
  "classic",
  "darkroom",
  "coral",
  "glass",
  "risoPrint",
  "peach",
  "lavender",
  "matcha",
  "sakura",
  "ocean",
  "sunset",
];

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
    shadow: "shadow-[4px_4px_0px_#1C1A12]",
    // Chrome: 1:1 with the Instant Print specification (legacy navbar).
    nav: "bg-print-white/80 backdrop-blur-xl border-b-2 border-ink",
    navHover: "hover:text-flash-yellow hover:underline",
    cta: "bg-flash-yellow border-2 border-ink text-ink shadow-[4px_4px_0px_#1C1A12]",
    ghost: "border-2 border-ink bg-transparent text-ink hover:bg-paper-grey/60",
    logoDot: "bg-flash-yellow outline outline-1 outline-ink",
    panel: "border-2 border-ink bg-print-white",
    panelHover: "hover:bg-paper-grey/60",
    panelRule: "border-ink/40",
    drawer: "bg-print-white border-l-2 border-ink text-ink",
    navCircle: "border-2 border-ink bg-print-white text-ink",
  },
  darkroom: {
    label: "Darkroom",
    page: "bg-[#121212] text-[#F5F5F5]",
    border: "border-[#FF6B35]",
    borderW: "border-2",
    borderStrong: "border-4",
    radius: "rounded-xl",
    radiusLarge: "rounded-2xl",
    radiusFull: "rounded-full",
    heading: "text-2xl font-bold",
    headingFont: "font-display",
    card: "bg-[#121212] text-[#F5F5F5]",
    caption: "text-[#F5F5F5]/70",
    // ORANGE accent #FF6B35 (refine specification: card border + shadow also
    // follow the accent: "orange print in a dark room"). INK text on the
    // accent: white-on-orange ~2.9:1 (fails AA for the 11px badge); ink ≈
    // 6:1 ✓.
    avatar: "bg-[#FF6B35] text-ink",
    accent: "bg-[#FF6B35] text-ink",
    // Status badge: coral (DESIGN.md §5) + ink text: white-on-coral
    // ~3.1:1 fails AA; ink ≈ 5.7:1 ✓.
    badge: "bg-flash-coral text-ink",
    text: "text-[#F5F5F5]",
    textMuted: "text-[#F5F5F5]/70",
    placeholder: "placeholder:text-[#F5F5F5]/50",
    featuredClass: "",
    chip: "bg-[#121212] border-[#FF6B35] text-[#F5F5F5]",
    shadow: "shadow-[4px_4px_0px_#FF6B35]",
    // Chrome: solid dark bar + orange border-b, orange link hover.
    nav: "bg-[#121212] border-b-2 border-[#FF6B35]",
    navHover: "hover:text-[#FF6B35] hover:underline",
    cta: "bg-flash-yellow border-2 border-[#FF6B35] text-ink shadow-[4px_4px_0px_#FF6B35]",
    ghost: "border-2 border-[#FF6B35] bg-transparent text-[#F5F5F5] hover:bg-[#FF6B35]/15",
    logoDot: "bg-[#FF6B35]",
    panel: "border-2 border-[#FF6B35] bg-[#121212]",
    panelHover: "hover:bg-white/[8%]",
    panelRule: "border-[#FF6B35]/40",
    drawer: "bg-[#121212] border-l-2 border-[#FF6B35] text-[#F5F5F5]",
    navCircle: "border-2 border-[#FF6B35] bg-[#121212] text-[#F5F5F5]",
  },
  coral: {
    label: "Coral",
    // Warm cream stage #F5F1E8 (identical body rule in globals.css).
    page: "bg-[#F5F1E8] text-ink",
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
    // CORAL accent #FF5C3D: ink text for small-badge contrast (white ≈
    // 3.1:1 ✗).
    avatar: "bg-flash-coral text-ink",
    accent: "bg-flash-coral text-ink",
    // Orange status badge: deliberately a different hue from the accent
    // (as in classic).
    badge: "bg-flash-orange text-ink",
    text: "text-ink",
    textMuted: "text-muted",
    placeholder: "placeholder:text-muted",
    featuredClass: "",
    chip: "bg-print-white border-ink text-ink",
    shadow: "shadow-[4px_4px_0px_#1C1A12]",
    // Chrome: cream bar, DARK coral link hover #C43A20 (4.7:1 on cream;
    // pure #FF5C3D is only 2.7:1 → fails AA, so it is reserved for
    // BG/border).
    nav: "bg-[#F5F1E8]/85 backdrop-blur-xl border-b-2 border-ink",
    navHover: "hover:text-[#C43A20] hover:underline",
    cta: "bg-flash-yellow border-2 border-ink text-ink shadow-[4px_4px_0px_#1C1A12]",
    ghost: "border-2 border-ink bg-transparent text-ink hover:bg-flash-coral/10",
    logoDot: "bg-flash-orange outline outline-1 outline-ink",
    panel: "border-2 border-ink bg-print-white",
    panelHover: "hover:bg-paper-grey/60",
    panelRule: "border-ink/40",
    drawer: "bg-print-white border-l-2 border-ink text-ink",
    navCircle: "border-2 border-ink bg-print-white text-ink",
  },
  glass: {
    label: "Glass",
    // Paper = linear-gradient(135deg,#F0F0F0 0%,#FFFFFF 50%,#F5F5F7 100%),
    // fixed: rendered by globals.css body[data-profile-theme="glass"]
    // (no longer a radial-gradient in ThemeBackdrop).
    page: "text-ink",
    // Liquid Glass RIM LIGHT: mirrors the CSS recipe:
    //   border: 1px solid rgba(255,255,255,0.28);
    //   border-top-color: rgba(255,255,255,0.55);
    // (borderW = border-[1px] = 1px sides; the brighter top edge is the
    // highlight sheen typical of glass material, side/bottom edges dimmer).
    border: "border-white/[0.28] border-t-white/[0.55]",
    borderW: "border-[1px]",
    borderStrong: "border-[1px]",
    radius: "rounded-2xl",
    radiusLarge: "rounded-3xl",
    radiusFull: "rounded-full",
    heading: "text-2xl font-bold",
    headingFont: "font-display",
    // Frosted card: white/70 background (RAISED from /50: ink text on the
    // white gradient gains contrast and no longer washes out) + Liquid Glass
    // material backdrop blur(20px) saturate(180%): saturate keeps the
    // colors behind the glass "alive" (color sheen) instead of dull.
    card: "bg-white/70 backdrop-blur-[20px] backdrop-saturate-[180%] text-ink",
    caption: "text-ink/70",
    avatar: "bg-white/70 backdrop-blur-[20px] backdrop-saturate-[180%] text-ink",
    // Specification "accent primary" = rgba(255,255,255,0.7).
    accent: "bg-white/70 backdrop-blur-[20px] backdrop-saturate-[180%] text-ink",
    // TRENDING badge: solid INK + white text (15:1 ✓): legible over both
    // frosted cards and the light gradient.
    badge: "bg-ink text-print-white",
    text: "text-ink",
    textMuted: "text-ink/70",
    // Specification "accent secondary" = #8E8E93 (iOS grey) for the
    // placeholder.
    placeholder: "placeholder:text-[#8E8E93]",
    featuredClass: "",
    chip: "bg-print-white border-ink text-ink",
    // Liquid Glass depth: CSS recipe:
    //   box-shadow: inset 0 1px 0 rgba(255,255,255,0.35),
    //               0 12px 40px rgba(0,0,0,0.35);
    // The inset highlight gives the inner edge a "sheen"; the long drop
    // shadow lifts the card off the gradient stage (previously only 0.08:
    // nearly invisible, so the glass felt flat).
    shadow: "shadow-[inset_0_1px_0_rgba(255,255,255,0.35),0_12px_40px_rgba(0,0,0,0.35)]",
    // Chrome: the SAME glass material (blur 20px + saturate 180%) on every
    // navbar surface; the rim light is applied to the edges of
    // cta/ghost/panel/drawer/navCircle (previously uniform white/80 → now
    // 28% + 55% on top).
    nav: "bg-white/60 backdrop-blur-[20px] backdrop-saturate-[180%] border-b border-black/[0.06]",
    navHover: "hover:text-ink/60 hover:underline",
    cta: "bg-white/70 backdrop-blur-[20px] backdrop-saturate-[180%] border border-white/[0.28] border-t-white/[0.55] text-ink shadow-[0_4px_16px_rgba(0,0,0,0.08)]",
    ghost: "border border-white/[0.28] border-t-white/[0.55] bg-white/70 backdrop-blur-[20px] backdrop-saturate-[180%] text-ink hover:bg-white/90 shadow-[0_4px_16px_rgba(0,0,0,0.06)]",
    logoDot: "bg-ink",
    panel: "border border-white/[0.28] border-t-white/[0.55] bg-white/70 backdrop-blur-[20px] backdrop-saturate-[180%]",
    panelHover: "hover:bg-black/[4%]",
    panelRule: "border-ink/15",
    drawer: "bg-white/70 backdrop-blur-[20px] backdrop-saturate-[180%] border-l border-white/[0.28] border-t-white/[0.55] text-ink",
    navCircle: "border border-white/[0.28] border-t-white/[0.55] bg-white/70 text-ink",
  },
  risoPrint: {
    label: "Riso Print",
    // Riso cream paper #F4F0E6: the REAL background lives in globals.css
    // (body[data-profile-theme="risoPrint"] = cream + SVG noise 0.03, same
    // pattern as glass). The page token still carries the stage color for
    // consistency.
    page: "bg-[#F4F0E6] text-[#1F3A5F]",
    border: "border-[#1F3A5F]",
    borderW: "border-2",
    borderStrong: "border-4",
    // 8px = "slightly sharper than 12px" (riso spec: printed impression).
    radius: "rounded-lg",
    radiusLarge: "rounded-xl",
    radiusFull: "rounded-full",
    heading: "text-2xl font-bold",
    headingFont: "font-display",
    // Card = cream paper + its own grain (.riso-paper in globals.css):
    // ThemeBackdrop (global grain) sits BEHIND the opaque card, so the card
    // needs its own texture.
    card: "riso-paper text-[#1F3A5F]",
    // @username uses the caption font (Caveat) in other themes; riso is
    // overridden to JetBrains Mono in globals.css (spec: NOT a handwritten
    // font).
    caption: "text-[#1F3A5F]/[0.72]",
    // Avatar/accent: orange-ink background + ink-blue glyphs (24px bold
    // glyphs = large text → 3.7:1 passes WCAG large-text 3:1; every other
    // combination on #FF5C39 fails AA).
    avatar: "bg-[#FF5C39] text-[#1F3A5F]",
    accent: "bg-[#FF5C39] text-[#1F3A5F]",
    // 11px badge = small text → needs ≥4.5:1: ink yellow (accent-tertiary,
    // "used sparingly") + ink blue = 7.9:1. (Green/orange with riso ink
    // fail AA at this size.)
    badge: "bg-flash-yellow text-[#1F3A5F]",
    text: "text-[#1F3A5F]",
    // Secondary text: opacity 0.72 (WCAG fix 2026-09-30): 4.65:1 on cream
    // = AA; previous value 0.65 = 3.85:1. The /[0.72] form is an arbitrary
    // value: 72 lies outside Tailwind's default opacity scale (multiples of
    // 5), so plain "/72" is not generated (verified via CLI). Applies to
    // every riso muted variant (textMuted, placeholder, caption, url).
    textMuted: "text-[#1F3A5F]/[0.72]",
    placeholder: "placeholder:text-[#1F3A5F]/[0.72]",
    featuredClass: "",
    chip: "bg-[#F4F0E6] border-[#1F3A5F] text-[#1F3A5F]",
    // Instant Print-style hard shadow but tinted with riso ink (not
    // #1C1A12).
    shadow: "shadow-[4px_4px_0px_#1F3A5F]",
    // Navbar: solid cream bar (printed, not frosted) + 2px ink border-b.
    nav: "bg-[#F4F0E6] border-b-2 border-[#1F3A5F]",
    // Link hover = DARK orange ink #C43A20 (one shade step down from
    // #FF5C39): 4.64:1 on cream passes AA (WCAG fix 2026-09-30; #FF5C39 =
    // 2.70:1: the bright orange REMAINS for bg/border/bar, never for
    // text).
    navHover: "hover:text-[#C43A20]",
    // Primary CTA: ink yellow for ink-blue text (7.9:1): with #FF5C39 as
    // bg, no palette combination passes AA for normal text.
    cta: "bg-flash-yellow border-2 border-[#1F3A5F] text-[#1F3A5F] shadow-[4px_4px_0px_#1F3A5F]",
    ghost: "border-2 border-[#1F3A5F] bg-transparent text-[#1F3A5F] hover:bg-paper-grey/60",
    logoDot: "bg-[#FF5C39] outline outline-1 outline-[#1F3A5F]",
    panel: "border-2 border-[#1F3A5F] bg-[#F4F0E6]",
    panelHover: "hover:bg-paper-grey/60",
    panelRule: "border-[#1F3A5F]/40",
    drawer: "bg-[#F4F0E6] border-l-2 border-[#1F3A5F] text-[#1F3A5F]",
    navCircle: "border-2 border-[#1F3A5F] bg-[#F4F0E6] text-[#1F3A5F]",
    // ---- Optional tokens (risoPrint only: see the file header) ----
    // Picker preview: 3 strips of ink / primary accent / secondary accent.
    swatch: ["#1F3A5F", "#FF5C39", "#2B7A78"],
    // Slug: full JetBrains Mono (not Caveat) in full dark ink.
    slug: "font-mono text-sm font-bold text-[#1F3A5F]",
    url: "text-[#1F3A5F]/[0.72]",
    // Click count = dark orange #C43A20 (WCAG fix: 4.64:1 AA on cream;
    // previous value #FF5C39 = 2.70:1. Bright orange remains for
    // bg/border/bar).
    countColor: "text-[#C43A20]",
    // Left accent bar 4px, alternating orange/green per card (index
    // parity).
    barPrimary: "border-l-4 border-l-[#FF5C39]",
    barSecondary: "border-l-4 border-l-[#2B7A78]",
  },
  // ---- Presets 6-9 (2026-09-30): playful/cute. Every token & class EXISTS
  // in these presets so the picker/chrome/card URL inherit the theme colors.
  // WCAG audit (see the PROGRESS entry): primary text ≥12.9:1, muted
  // ≥4.6:1, nav hover ≥4.9:1, ink-on-accent badge/CTA ≥4.8:1. ----
  peach: {
    label: "Peach",
    page: "bg-[#FFF1E6] text-[#2B1F1A]",
    border: "border-[#2B1F1A]",
    borderW: "border-2",
    borderStrong: "border-4",
    radius: "rounded-xl",
    radiusLarge: "rounded-2xl",
    radiusFull: "rounded-full",
    heading: "text-2xl font-bold",
    headingFont: "font-display",
    // Card = peach paper (same as the stage): distinguished by a 2px ink
    // border + hard shadow, identical pattern to darkroom/risoPrint.
    card: "bg-[#FFF1E6] text-[#2B1F1A]",
    caption: "text-[#2B1F1A]/65",
    // Brown ink on the peach accent = 7.78:1 (11px badge passes AA).
    avatar: "bg-[#FF9B7B] text-[#2B1F1A]",
    accent: "bg-[#FF9B7B] text-[#2B1F1A]",
    // TRENDING uses light peach + ink = 10.7:1 (different hue from the ★
    // accent).
    badge: "bg-[#FFC8A8] text-[#2B1F1A]",
    text: "text-[#2B1F1A]",
    // muted 0.65 exactly per specification = 4.80:1 AA for normal text.
    textMuted: "text-[#2B1F1A]/65",
    placeholder: "placeholder:text-[#2B1F1A]/65",
    featuredClass: "",
    chip: "bg-[#FFF1E6] border-[#2B1F1A] text-[#2B1F1A]",
    // Specification: hard shadow 4px 4px 0 ink.
    shadow: "shadow-[4px_4px_0px_#2B1F1A]",
    nav: "bg-[#FFF1E6]/85 backdrop-blur-xl border-b-2 border-[#2B1F1A]",
    // Hover = DARK peach #A8481F (5.25:1 on paper: #FF9B7B is only
    // ~1.9:1; bright accents are for bg/border/decoration only, same pattern
    // as coral).
    navHover: "hover:text-[#A8481F] hover:underline",
    cta: "bg-[#FF9B7B] border-2 border-[#2B1F1A] text-[#2B1F1A] shadow-[4px_4px_0px_#2B1F1A]",
    ghost: "border-2 border-[#2B1F1A] bg-transparent text-[#2B1F1A] hover:bg-[#FF9B7B]/20",
    logoDot: "bg-[#FF9B7B] outline outline-1 outline-[#2B1F1A]",
    panel: "border-2 border-[#2B1F1A] bg-[#FFF1E6]",
    panelHover: "hover:bg-[#FF9B7B]/15",
    panelRule: "border-[#2B1F1A]/40",
    drawer: "bg-[#FFF1E6] border-l-2 border-[#2B1F1A] text-[#2B1F1A]",
    navCircle: "border-2 border-[#2B1F1A] bg-[#FFF1E6] text-[#2B1F1A]",
    // Picker: 3 strips of ink / accent / paper.
    swatch: ["#2B1F1A", "#FF9B7B", "#FFF1E6"],
    url: "text-[#2B1F1A]/65",
    // Left accent bar 4px solid accent (both parity slots use the same
    // color).
    barPrimary: "border-l-4 border-l-[#FF9B7B]",
    barSecondary: "border-l-4 border-l-[#FF9B7B]",
  },
  lavender: {
    label: "Lavender",
    page: "bg-[#F5F0FF] text-[#2A2140]",
    border: "border-[#2A2140]",
    borderW: "border-2",
    borderStrong: "border-4",
    // 16px = rounder than the other themes (rounded-2xl).
    radius: "rounded-2xl",
    radiusLarge: "rounded-3xl",
    radiusFull: "rounded-full",
    heading: "text-2xl font-bold",
    headingFont: "font-display",
    card: "bg-[#F5F0FF] text-[#2A2140]",
    caption: "text-[#2A2140]/65",
    // Dark purple on lavender = 5.44:1.
    avatar: "bg-[#A88BEB] text-[#2A2140]",
    accent: "bg-[#A88BEB] text-[#2A2140]",
    badge: "bg-[#C9B6F0] text-[#2A2140]",
    text: "text-[#2A2140]",
    // muted 0.65 = 4.64:1 AA.
    textMuted: "text-[#2A2140]/65",
    placeholder: "placeholder:text-[#2A2140]/65",
    featuredClass: "",
    chip: "bg-[#F5F0FF] border-[#2A2140] text-[#2A2140]",
    // Specification: SOFT shadow (not hard): ink tint at 15%, airier.
    shadow: "shadow-[6px_6px_0px_rgba(42,33,64,0.15)]",
    nav: "bg-[#F5F0FF]/85 backdrop-blur-xl border-b-2 border-[#2A2140]",
    navHover: "hover:text-[#5B3FA8] hover:underline",
    cta: "bg-[#A88BEB] border-2 border-[#2A2140] text-[#2A2140] shadow-[6px_6px_0px_rgba(42,33,64,0.15)]",
    ghost: "border-2 border-[#2A2140] bg-transparent text-[#2A2140] hover:bg-[#A88BEB]/20",
    logoDot: "bg-[#A88BEB] outline outline-1 outline-[#2A2140]",
    panel: "border-2 border-[#2A2140] bg-[#F5F0FF]",
    panelHover: "hover:bg-[#A88BEB]/20",
    panelRule: "border-[#2A2140]/40",
    drawer: "bg-[#F5F0FF] border-l-2 border-[#2A2140] text-[#2A2140]",
    navCircle: "border-2 border-[#2A2140] bg-[#F5F0FF] text-[#2A2140]",
    swatch: ["#2A2140", "#A88BEB", "#F5F0FF"],
    url: "text-[#2A2140]/65",
    barPrimary: "border-l-4 border-l-[#A88BEB]",
    barSecondary: "border-l-4 border-l-[#A88BEB]",
  },
  matcha: {
    label: "Matcha",
    page: "bg-[#EFF5EC] text-[#1F2E1A]",
    // 1.5px border (thinner than other themes: spec: handmade-paper
    // feel).
    border: "border-[#1F2E1A]",
    borderW: "border-[1.5px]",
    borderStrong: "border-[3px]",
    radius: "rounded-[10px]",
    radiusLarge: "rounded-xl",
    radiusFull: "rounded-full",
    heading: "text-2xl font-bold",
    headingFont: "font-display",
    card: "bg-[#EFF5EC] text-[#1F2E1A]",
    // muted 0.65 for matcha = 4.44:1 (FAILS AA) → 0.72 = 5.48:1 (small
    // deviation from the spec; spec rule: darken the shade for text).
    caption: "text-[#1F2E1A]/[0.72]",
    // Matcha green + ink = 4.80:1 (passes AA for the 11px badge).
    avatar: "bg-[#7BA05B] text-[#1F2E1A]",
    accent: "bg-[#7BA05B] text-[#1F2E1A]",
    badge: "bg-[#A8C68A] text-[#1F2E1A]",
    text: "text-[#1F2E1A]",
    textMuted: "text-[#1F2E1A]/[0.72]",
    placeholder: "placeholder:text-[#1F2E1A]/[0.72]",
    featuredClass: "",
    chip: "bg-[#EFF5EC] border-[#1F2E1A] text-[#1F2E1A]",
    // Specification: 3px 3px 0 ink shadow (softer).
    shadow: "shadow-[3px_3px_0px_#1F2E1A]",
    nav: "bg-[#EFF5EC]/85 backdrop-blur-xl border-b-[1.5px] border-[#1F2E1A]",
    navHover: "hover:text-[#43602C] hover:underline",
    cta: "bg-[#7BA05B] border-[1.5px] border-[#1F2E1A] text-[#1F2E1A] shadow-[3px_3px_0px_#1F2E1A]",
    ghost: "border-[1.5px] border-[#1F2E1A] bg-transparent text-[#1F2E1A] hover:bg-[#7BA05B]/20",
    logoDot: "bg-[#7BA05B] outline outline-1 outline-[#1F2E1A]",
    panel: "border-[1.5px] border-[#1F2E1A] bg-[#EFF5EC]",
    panelHover: "hover:bg-[#7BA05B]/20",
    panelRule: "border-[#1F2E1A]/40",
    drawer: "bg-[#EFF5EC] border-l-[1.5px] border-[#1F2E1A] text-[#1F2E1A]",
    navCircle: "border-[1.5px] border-[#1F2E1A] bg-[#EFF5EC] text-[#1F2E1A]",
    swatch: ["#1F2E1A", "#7BA05B", "#EFF5EC"],
    url: "text-[#1F2E1A]/[0.72]",
    // Left accent bar 3px (matcha spec: thinner than the 4px used by the
    // other themes).
    barPrimary: "border-l-[3px] border-l-[#7BA05B]",
    barSecondary: "border-l-[3px] border-l-[#7BA05B]",
  },
  sakura: {
    label: "Sakura",
    page: "bg-[#FFF5F8] text-[#2B1F26]",
    border: "border-[#2B1F26]",
    borderW: "border-2",
    borderStrong: "border-4",
    radius: "rounded-2xl",
    radiusLarge: "rounded-3xl",
    radiusFull: "rounded-full",
    heading: "text-2xl font-bold",
    headingFont: "font-display",
    card: "bg-[#FFF5F8] text-[#2B1F26]",
    caption: "text-[#2B1F26]/65",
    // Sakura pink + dark pink ink = 8.20:1.
    avatar: "bg-[#F5A4B8] text-[#2B1F26]",
    accent: "bg-[#F5A4B8] text-[#2B1F26]",
    badge: "bg-[#FAD0DA] text-[#2B1F26]",
    text: "text-[#2B1F26]",
    // muted 0.65 = 4.85:1 AA.
    textMuted: "text-[#2B1F26]/65",
    placeholder: "placeholder:text-[#2B1F26]/65",
    featuredClass: "",
    chip: "bg-[#FFF5F8] border-[#2B1F26] text-[#2B1F26]",
    // Specification: TINTED pink shadow (not ink): soft impression.
    shadow: "shadow-[4px_4px_0px_rgba(245,164,184,0.5)]",
    nav: "bg-[#FFF5F8]/85 backdrop-blur-xl border-b-2 border-[#2B1F26]",
    // Hover = DARK pink #A44561 (5.46:1; #F5A4B8 only ~1.7:1 on paper).
    navHover: "hover:text-[#A44561] hover:underline",
    cta: "bg-[#F5A4B8] border-2 border-[#2B1F26] text-[#2B1F26] shadow-[4px_4px_0px_rgba(245,164,184,0.5)]",
    ghost: "border-2 border-[#2B1F26] bg-transparent text-[#2B1F26] hover:bg-[#F5A4B8]/30",
    logoDot: "bg-[#F5A4B8] outline outline-1 outline-[#2B1F26]",
    panel: "border-2 border-[#2B1F26] bg-[#FFF5F8]",
    panelHover: "hover:bg-[#F5A4B8]/25",
    panelRule: "border-[#2B1F26]/40",
    drawer: "bg-[#FFF5F8] border-l-2 border-[#2B1F26] text-[#2B1F26]",
    navCircle: "border-2 border-[#2B1F26] bg-[#FFF5F8] text-[#2B1F26]",
    swatch: ["#2B1F26", "#F5A4B8", "#FFF5F8"],
    url: "text-[#2B1F26]/65",
    // Accent bar on the RIGHT of the card (unlike the other 8 themes, which
    // place it on the left).
    barPrimary: "border-r-4 border-r-[#F5A4B8]",
    barSecondary: "border-r-4 border-r-[#F5A4B8]",
  },
  // ---- Presets 10-11 (2026-10-01): Ocean + Sunset. ----
  ocean: {
    label: "Ocean",
    page: "bg-[#E8F2F7] text-[#0F2A3D]",
    border: "border-[#0F2A3D]",
    borderW: "border-2",
    borderStrong: "border-4",
    // 12px (specification) = rounded-xl.
    radius: "rounded-xl",
    radiusLarge: "rounded-2xl",
    radiusFull: "rounded-full",
    heading: "text-2xl font-bold",
    headingFont: "font-display",
    // Card = light blue paper (same as the stage): distinguished by a 2px
    // ink border + hard shadow, identical pattern to
    // peach/lavender/matcha/sakura.
    card: "bg-[#E8F2F7] text-[#0F2A3D]",
    // muted 0.68 exactly per specification = 4.97:1 AA.
    caption: "text-[#0F2A3D]/[0.68]",
    // Ink #0F2A3D on accent #4A90B8 = 4.21:1 (the 11px badge needs 4.5) →
    // text-on-accent (avatar, ★, CTA) uses the DARKER ink #0A1F2E =
    // 4.78:1. Audit rule: darken the shade for TEXT, the accent bg stays at
    // the specified #4A90B8.
    avatar: "bg-[#4A90B8] text-[#0A1F2E]",
    accent: "bg-[#4A90B8] text-[#0A1F2E]",
    // TRENDING uses light accentSecondary blue + ink = 6.49:1.
    badge: "bg-[#7BB3D1] text-[#0F2A3D]",
    text: "text-[#0F2A3D]",
    textMuted: "text-[#0F2A3D]/[0.68]",
    placeholder: "placeholder:text-[#0F2A3D]/[0.68]",
    featuredClass: "",
    chip: "bg-[#E8F2F7] border-[#0F2A3D] text-[#0F2A3D]",
    // Specification: hard shadow 5px 5px 0 ink.
    shadow: "shadow-[5px_5px_0px_#0F2A3D]",
    nav: "bg-[#E8F2F7]/85 backdrop-blur-xl border-b-2 border-[#0F2A3D]",
    // Hover = dark ocean blue #2A6A8A (5.24:1 on paper: the specification
    // was already dark; pure #4A90B8 is only 3.09:1 → bg/border only).
    navHover: "hover:text-[#2A6A8A] hover:underline",
    cta: "bg-[#4A90B8] border-2 border-[#0F2A3D] text-[#0A1F2E] shadow-[5px_5px_0px_#0F2A3D]",
    ghost: "border-2 border-[#0F2A3D] bg-transparent text-[#0F2A3D] hover:bg-[#4A90B8]/20",
    logoDot: "bg-[#4A90B8] outline outline-1 outline-[#0F2A3D]",
    panel: "border-2 border-[#0F2A3D] bg-[#E8F2F7]",
    panelHover: "hover:bg-[#4A90B8]/15",
    panelRule: "border-[#0F2A3D]/40",
    drawer: "bg-[#E8F2F7] border-l-2 border-[#0F2A3D] text-[#0F2A3D]",
    navCircle: "border-2 border-[#0F2A3D] bg-[#E8F2F7] text-[#0F2A3D]",
    swatch: ["#0F2A3D", "#4A90B8", "#E8F2F7"],
    url: "text-[#0F2A3D]/[0.68]",
    // Left accent bar 4px solid accent (both parity slots use the same
    // color).
    barPrimary: "border-l-4 border-l-[#4A90B8]",
    barSecondary: "border-l-4 border-l-[#4A90B8]",
  },
  sunset: {
    label: "Sunset",
    page: "bg-[#FFF3E0] text-[#3D1F0F]",
    border: "border-[#3D1F0F]",
    borderW: "border-2",
    borderStrong: "border-4",
    // 14px (specification): outside the Tailwind scale → arbitrary value.
    radius: "rounded-[14px]",
    radiusLarge: "rounded-2xl",
    radiusFull: "rounded-full",
    heading: "text-2xl font-bold",
    headingFont: "font-display",
    card: "bg-[#FFF3E0] text-[#3D1F0F]",
    // muted 0.68 exactly per specification = 5.11:1 AA.
    caption: "text-[#3D1F0F]/[0.68]",
    // Brown ink on the sunset accent = 7.39:1 (passes at every text size).
    avatar: "bg-[#F5A623] text-[#3D1F0F]",
    accent: "bg-[#F5A623] text-[#3D1F0F]",
    // TRENDING uses red-orange + ink = 4.51:1 (passes AA, barely).
    badge: "bg-[#E8634A] text-[#3D1F0F]",
    text: "text-[#3D1F0F]",
    textMuted: "text-[#3D1F0F]/[0.68]",
    placeholder: "placeholder:text-[#3D1F0F]/[0.68]",
    featuredClass: "",
    chip: "bg-[#FFF3E0] border-[#3D1F0F] text-[#3D1F0F]",
    // Specification: hard shadow 4px 4px 0 ink.
    shadow: "shadow-[4px_4px_0px_#3D1F0F]",
    nav: "bg-[#FFF3E0]/85 backdrop-blur-xl border-b-2 border-[#3D1F0F]",
    // Spec #C77800 = 3.13:1 on paper (FAILS AA for text) → replaced by
    // #A35F00 = 4.57:1 (darker shade for text; pure #F5A623 is only 1.85:1
    // → bright accents only for bg/border/decoration).
    navHover: "hover:text-[#A35F00] hover:underline",
    cta: "bg-[#F5A623] border-2 border-[#3D1F0F] text-[#3D1F0F] shadow-[4px_4px_0px_#3D1F0F]",
    ghost: "border-2 border-[#3D1F0F] bg-transparent text-[#3D1F0F] hover:bg-[#F5A623]/25",
    logoDot: "bg-[#F5A623] outline outline-1 outline-[#3D1F0F]",
    panel: "border-2 border-[#3D1F0F] bg-[#FFF3E0]",
    panelHover: "hover:bg-[#F5A623]/20",
    panelRule: "border-[#3D1F0F]/40",
    drawer: "bg-[#FFF3E0] border-l-2 border-[#3D1F0F] text-[#3D1F0F]",
    navCircle: "border-2 border-[#3D1F0F] bg-[#FFF3E0] text-[#3D1F0F]",
    swatch: ["#3D1F0F", "#F5A623", "#FFF3E0"],
    url: "text-[#3D1F0F]/[0.68]",
    barPrimary: "border-l-4 border-l-[#F5A623]",
    barSecondary: "border-l-4 border-l-[#F5A623]",
  },
};

export function themeStyles(key) {
  return THEME_STYLES[key] || THEME_STYLES.classic;
}
