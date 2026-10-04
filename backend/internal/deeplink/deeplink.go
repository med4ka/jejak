// Package deeplink maps Indonesian e-commerce URLs to mobile app deep links.
//
// The redirect handler uses this on mobile user agents: a short link whose
// destination is a Shopee/Tokopedia/... product first tries to open the
// merchant APP (custom URL scheme) and falls back to the web page when the
// app is not installed. Desktop requests skip the interstitial entirely and
// go straight to the web URL.
//
// Deviation from the original spec sketch: host matching is NOT raw
// strings.Contains. "shopee.co.id.evil.com" contains "shopee.co.id", so a
// substring check would let an attacker-controlled host masquerade as a
// platform (the interstitial would then hand the browser an app scheme built
// from their URL). matchHost demands an exact registrable-domain match or a
// real subdomain, mirroring handler.matchHost in the referrer classifier.
package deeplink

import (
	"net/url"
	"regexp"
	"strings"
)

// matchHost reports whether host is domain itself or a subdomain of it.
// host must already be lower-cased and stripped of a leading "www.".
func matchHost(host, domain string) bool {
	return host == domain || strings.HasSuffix(host, "."+domain)
}

// legacyProductID matches the "-i.{shopid}.{itemid}" slug format that both
// Shopee and Tokopedia use for product URLs (e.g. "kaos-putih-i.12345.67890").
// The two groups are kept as raw digits only: they are re-embedded in an app
// scheme, so anything else (quotes, slashes, dots beyond the pattern) must
// never come from the URL.
var legacyProductID = regexp.MustCompile(`-i\.(\d+)\.(\d+)`)

// productPath extracts "{a}/{b}" from a ".../product/{a}/{b}" path (the
// modern product URL of Shopee and Tokopedia). Non-digit segments fail: a
// product ID is always numeric, and the result feeds an app scheme.
func productPath(path string) (string, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 3 || parts[0] != "product" || !isDigits(parts[1]) || !isDigits(parts[2]) {
		return "", false
	}
	return parts[1] + "/" + parts[2], true
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// DetectEcommerce returns the e-commerce platform and app scheme for a given
// URL. Returns "", "" if the URL is not a supported e-commerce URL or cannot
// be parsed. An empty scheme with a non-empty platform means "platform
// recognised, web-only" (the redirect then behaves exactly like a normal
// link; only the dashboard badge is shown).
func DetectEcommerce(rawURL string) (platform string, scheme string) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || u.Hostname() == "" {
		return "", ""
	}
	// Only real web URLs: a scheme-relative "//host/path" parses with a host
	// but no scheme, and "javascript:"/"data:" carry a host-free path. The
	// shortener validates this on CREATE; repeating the check here keeps the
	// interstitial from ever building an app scheme out of a legacy or
	// externally written cache value.
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", ""
	}
	// Hostname() already drops any port; lower-case so the comparisons are
	// case-insensitive (URL hosts are case-insensitive per RFC 3986).
	host := strings.ToLower(strings.TrimPrefix(u.Hostname(), "www."))

	switch {
	case matchHost(host, "shopee.co.id") || matchHost(host, "shopee.sg") ||
		matchHost(host, "shopee.my") || matchHost(host, "shopee.th") ||
		matchHost(host, "shopee.vn") || matchHost(host, "shopee.com.tw"):
		return "shopee", shopeeScheme(u)
	case matchHost(host, "tokopedia.com"):
		return "tokopedia", tokopediaScheme(u)
	case matchHost(host, "tiktok.com"):
		// Web-only ON PURPOSE: m.tiktok.com already hands the request to the
		// TikTok app through universal links when it is installed. Routing
		// through "snssdk1233://" first would open the app HOME instead of
		// this specific video/product page: a downgrade.
		return "tiktok", ""
	case matchHost(host, "lazada.co.id") || matchHost(host, "lazada.com"):
		return "lazada", "lazada://"
	case matchHost(host, "blibli.com"):
		return "blibli", "blibli://"
	case matchHost(host, "bukalapak.com"):
		return "bukalapak", "bukalapak://"
	}
	return "", ""
}

// shopeeScheme builds a product deep link when the URL structure is known,
// otherwise the app root. Shopee's documented scheme is shopeeid://; product
// pages open via shopeeid://product/{shopid}/{itemid}. Unknown paths (search,
// category, promo) still open the app: landing on the app home beats landing
// nowhere, and the 1.5s fallback covers the not-installed case.
func shopeeScheme(u *url.URL) string {
	if ids, ok := productIDs(u.Path); ok {
		return "shopeeid://product/" + ids
	}
	return "shopeeid://"
}

// tokopediaScheme mirrors shopeeScheme with Tokopedia's scheme. Same two URL
// shapes: /product/{shopid}/{itemid} (modern) and the "-i.{a}.{b}" slug
// (legacy). Unrecognised paths open the app root.
func tokopediaScheme(u *url.URL) string {
	if ids, ok := productIDs(u.Path); ok {
		return "tokopedia://product/" + ids
	}
	return "tokopedia://"
}

// productIDs returns "shopid/itemid" for both supported URL shapes.
func productIDs(path string) (string, bool) {
	if ids, ok := productPath(path); ok {
		return ids, true
	}
	if m := legacyProductID.FindStringSubmatch(path); m != nil {
		return m[1] + "/" + m[2], true
	}
	return "", false
}
