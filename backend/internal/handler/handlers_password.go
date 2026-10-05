package handler

// Per-link password gate + health fallback pages (migration 17, 2026-10-04).
// Redirect order: password gate (no click log until verified) -> health gate
// (broken destination answers in place) -> logClick -> respondRedirect.
// All pages are inline-style documents like writeGoneHTML: they are served by
// the redirect endpoint and must render with zero external assets.

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"html"
	"net/http"

	"jejak/internal/apierror"
	"jejak/internal/auth"
	"jejak/internal/i18n"
	"jejak/internal/ratelimit"
)

// linkAccessCookie names the 1-hour cookie that remembers a verified
// password per short code. The value is accessDigest(hash), never the bcrypt
// hash itself: a cookie-octet must not contain "/" (bcrypt base64 does).
func linkAccessCookie(shortCode string) string {
	return "jejak_link_access_" + shortCode
}

// accessDigest is what goes into (and is compared against) the cookie: the
// hex SHA-256 of the stored bcrypt hash. Forging a value requires knowing
// the hash, which only the database holds; the digest also keeps the cookie
// ASCII-safe.
func accessDigest(hash string) string {
	sum := sha256.Sum256([]byte(hash))
	return hex.EncodeToString(sum[:])
}

// passwordVerified reports whether the request carries a valid access cookie
// for this link: constant-time compare of the cookie against the digest of
// the CURRENT hash, so a password change invalidates every old cookie.
func passwordVerified(r *http.Request, shortCode, hash string) bool {
	c, err := r.Cookie(linkAccessCookie(shortCode))
	if err != nil || c.Value == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(c.Value), []byte(accessDigest(hash))) == 1
}

// Password form error kinds (the ?e=1 PRG flag and the rate-limit branch):
// keys into the i18n dictionaries, never user-visible prose themselves.
const (
	passwordErrNone        = ""
	passwordErrWrong       = "wrong"
	passwordErrRateLimited = "rate_limited"
)

// writePasswordForm serves the gate page (200) for GET /r/{code} when the
// password has not been verified yet. errKind "" = clean form; "wrong" /
// "rate_limited" = red box. All copy is translated from the visitor's
// Accept-Language (internal/i18n; default Indonesian). The POST target is
// /r/{code}/verify; on success the handler sets the cookie and redirects
// back (PRG), so a browser refresh never resubmits the password.
func writePasswordForm(w http.ResponseWriter, r *http.Request, shortCode, errKind string) {
	locale := i18n.DetectLocale(r.Header.Get("Accept-Language"))
	// Escape even though DetectLocale only returns constants: keeps the
	// taint-sensitive HTML writer (gosec G705) provably safe.
	localeAttr := html.EscapeString(locale)
	tr := func(key string) string { return html.EscapeString(i18n.Translate(locale, key)) }
	errBox := ""
	switch errKind {
	case passwordErrWrong:
		errBox = fmt.Sprintf(`<p class="err">%s</p>`, tr("link.password.error.wrong"))
	case passwordErrRateLimited:
		errBox = fmt.Sprintf(`<p class="err">%s</p>`, tr("link.password.error.rate_limited"))
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	code := html.EscapeString(shortCode)
	fmt.Fprintf(w, `<!doctype html>
<html lang="%s">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex">
<title>%s</title>
<style>
  body { margin:0; min-height:100vh; display:flex; align-items:center; justify-content:center;
         background:#FAFAF7; color:#1C1A12; font-family:ui-monospace,SFMono-Regular,Menlo,monospace; }
  main { width:100%%; max-width:380px; text-align:left; padding:24px; }
  .card { border:2px solid #1C1A12; background:#fff; box-shadow:6px 6px 0px #1C1A12; border-radius:14px; padding:24px; }
  h1 { font-size:1.15rem; margin:0 0 6px; }
  p { margin:0 0 16px; font-size:0.85rem; opacity:0.75; }
  label { display:block; font-size:0.8rem; font-weight:700; margin-bottom:6px; }
  input { width:100%%; box-sizing:border-box; border:2px solid #1C1A12; border-radius:8px; padding:10px 12px;
          font:inherit; font-size:0.95rem; background:#FAFAF7; }
  input:focus { outline:2px solid #FF5C3D; outline-offset:1px; }
  button { width:100%%; margin-top:14px; border:2px solid #1C1A12; border-radius:8px; padding:10px 12px;
           background:#FF5C3D; color:#fff; font:inherit; font-weight:700; font-size:0.95rem;
           box-shadow:3px 3px 0px #1C1A12; cursor:pointer; }
  button:active { transform:translate(2px,2px); box-shadow:1px 1px 0px #1C1A12; }
  .err { border:2px solid #1C1A12; background:#FFE1D9; color:#8C2B18; border-radius:8px; padding:8px 10px;
         font-size:0.8rem; font-weight:700; margin:0 0 14px; }
  .brand { font-size:0.7rem; letter-spacing:0.2em; font-weight:700; opacity:0.6; margin-bottom:14px; }
</style>
</head>
<body>
<main>
  <div class="card">
    <p class="brand">JEJAK</p>
    <h1>%s</h1>
    <p>%s</p>
    %s
    <form method="post" action="/r/%s/verify" autocomplete="off">
      <label for="password">%s</label>
      <input id="password" name="password" type="password" required minlength="4" autofocus>
      <button type="submit">%s</button>
    </form>
  </div>
</main>
</body>
</html>
`, localeAttr,
		tr("link.password.prompt.pageTitle"),
		tr("link.password.prompt.title"),
		tr("link.password.prompt.subtitle"),
		errBox, code,
		tr("link.password.prompt.input_label"),
		tr("link.password.prompt.button"))
}

// passwordErrorFromQuery maps the ?e=1 PRG flag (set by a failed verify
// redirect) to the form error kind; any other query = clean form.
func passwordErrorFromQuery(r *http.Request) string {
	if r.URL.Query().Get("e") == "1" {
		return passwordErrWrong
	}
	return passwordErrNone
}

// HandlePasswordVerify handles POST /r/{code}/verify (form-encoded, browser
// form - not JSON). Rate limit: 10 attempts/minute per IP+code (failed
// attempts Record, success Reset so the owner is never punished for typos).
// Wrong password: PRG redirect to /r/{code}?e=1 WITHOUT setting a cookie and
// WITHOUT logging a click (the visitor still has not reached the target).
// Correct: HttpOnly cookie (SHA-256 digest, 1 hour, Path=/r/) + PRG back.
func (h *Handler) HandlePasswordVerify(shortCode string, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		apierror.WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}
	rlKey := ratelimit.Key(
		ratelimit.ClientIP(r.RemoteAddr, r.Header.Get("X-Forwarded-For")), shortCode)
	if h.VerifyLimiter != nil && !h.VerifyLimiter.Allow(rlKey) {
		w.WriteHeader(http.StatusTooManyRequests)
		writePasswordForm(w, r, shortCode, passwordErrRateLimited)
		return
	}

	// ParseForm caps the body at 10MB by default; the field we need is one
	// small password. Any parse failure behaves like a wrong password.
	_ = r.ParseForm()
	password := r.PostFormValue("password")

	link, err := h.Store.GetLink(shortCode)
	if err != nil {
		apierror.WriteError(w, http.StatusNotFound, "LINK_NOT_FOUND", "URL not found")
		return
	}
	// The gate was removed between form render and submit: no cookie needed,
	// go straight to the link (redirect path applies its own rules).
	if link.PasswordHash == "" {
		h.VerifyLimiterReset(rlKey)
		http.Redirect(w, r, "/r/"+shortCode, http.StatusSeeOther)
		return
	}
	if password == "" || !auth.CheckPassword(link.PasswordHash, password) {
		if h.VerifyLimiter != nil {
			h.VerifyLimiter.Record(rlKey)
		}
		http.Redirect(w, r, "/r/"+shortCode+"?e=1", http.StatusSeeOther)
		return
	}
	h.VerifyLimiterReset(rlKey)
	// #nosec G124 -- Secure comes from COOKIE_SECURE/TLS (see the field below)
	http.SetCookie(w, &http.Cookie{
		Name:     linkAccessCookie(shortCode),
		Value:    accessDigest(link.PasswordHash),
		Path:     "/r/",
		MaxAge:   3600, // 1 hour (spec); the digest changes with the hash, so
		HttpOnly: true, // a password change locks old cookies out
		SameSite: http.SameSiteLaxMode,
		// Secure from COOKIE_SECURE (auth.SecureCookies, same gate as the
		// session cookie) OR plain TLS: on http://localhost dev the cookie
		// must NOT be Secure or the browser would drop it silently.
		Secure: auth.SecureCookies || r.TLS != nil, // #nosec G124 -- gated by COOKIE_SECURE/TLS (mirrors auth.SetCookie)
	})
	http.Redirect(w, r, "/r/"+shortCode, http.StatusSeeOther)
}

// VerifyLimiterReset forgives the bucket after a success (nil-safe; mirrors
// ratelimit.Limiter.Reset).
func (h *Handler) VerifyLimiterReset(key string) {
	if h.VerifyLimiter != nil {
		h.VerifyLimiter.Reset(key)
	}
}

// brokenOnly maps a health_status to what the redirect payload stores:
// "broken" or "" - healthy/timeout/unknown all redirect normally, and the
// small payload keeps the cache lean.
func brokenOnly(status string) string {
	if status == "broken" {
		return "broken"
	}
	return ""
}

// serveHealthGate answers in place for a broken destination: 1s fallback
// interstitial when the owner configured one, the Jejak-styled 503 page
// otherwise. Called BEFORE logClick: the visitor never reached a
// destination, so the hit is not a click (same rule as the password gate).
func serveHealthGate(w http.ResponseWriter, fallbackURL string) {
	if fallbackURL != "" {
		writeHealthFallbackHTML(w, fallbackURL)
		return
	}
	writeHealthProblemHTML(w)
}

// writeHealthFallbackHTML serves the 1s interstitial when the destination is
// broken but a fallback_url exists: banner text (so the visitor understands
// the detour), automatic meta refresh to the fallback, plus a manual link
// for browsers that ignore meta refresh. Status 200: the page itself works
// (only the original destination is down).
func writeHealthFallbackHTML(w http.ResponseWriter, fallbackURL string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fb := html.EscapeString(fallbackURL)
	fmt.Fprintf(w, `<!doctype html>
<html lang="id">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex">
<meta http-equiv="refresh" content="1; url=%s">
<title>Dialihkan</title>
<style>
  body { margin:0; min-height:100vh; display:flex; align-items:center; justify-content:center;
         background:#FAFAF7; color:#1C1A12; font-family:ui-monospace,SFMono-Regular,Menlo,monospace; }
  main { width:100%%; max-width:420px; text-align:center; padding:24px; }
  .banner { border:2px solid #1C1A12; background:#FFF3C4; box-shadow:6px 6px 0px #1C1A12;
            border-radius:14px; padding:20px; font-size:0.9rem; font-weight:700; }
  p { margin:12px 0 0; font-size:0.8rem; opacity:0.75; }
  a { color:#C2381B; font-weight:700; }
</style>
</head>
<body>
<main>
  <div class="banner">Link ini dialihkan karena tujuan asli tidak tersedia.</div>
  <p>Membuka halaman pengganti dalam 1 detik. Jika tidak, klik <a href="%s">di sini</a>.</p>
</main>
</body>
</html>
`, fb, fb)
}

// writeHealthProblemHTML answers 503 Service Unavailable for a broken
// destination WITHOUT a fallback: a document (not a plain-text error) in the
// same neo-brutal style as writeGoneHTML, with noindex so search engines do
// not rank the error page.
func writeHealthProblemHTML(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Retry-After", "21600") // worker re-checks every 6h
	w.WriteHeader(http.StatusServiceUnavailable)
	fmt.Fprintf(w, `<!doctype html>
<html lang="id">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex">
<title>Link sedang bermasalah</title>
<style>
  body { margin:0; min-height:100vh; display:flex; align-items:center; justify-content:center;
         background:#FAFAF7; color:#1C1A12; font-family:ui-monospace,SFMono-Regular,Menlo,monospace; }
  main { text-align:center; padding:24px; }
  h1 { font-size:2rem; margin:0 0 8px; }
  h1 span { color:#FF5C3D; }
  p { margin:0; font-size:0.95rem; opacity:0.75; }
</style>
</head>
<body>
<main>
  <h1><span>503</span> Link sedang bermasalah</h1>
  <p>Tujuan link ini tidak dapat dijangkau. Silakan coba lagi nanti.</p>
</main>
</body>
</html>
`)
}
