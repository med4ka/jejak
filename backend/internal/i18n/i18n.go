// Package i18n renders the handful of server-side user-facing strings
// (the password gate page, migration 17) in the visitor's language.
//
// The frontend dictionaries (frontend/messages/{id,en,de}.json) are the
// source of truth for ALL UI copy. This package embeds only the keys the Go
// server itself renders (locales/*.json, via go:embed): embedding the whole
// frontend dictionaries would couple the Go module to files outside it
// (go:embed forbids ".."), and the API otherwise speaks in stable error
// codes (see internal/apierror) that the frontend translates client-side.
// i18n_test.go asserts the mirrored values stay identical to the frontend
// dictionaries, so drift fails the build instead of shipping silently.
package i18n

import (
	"embed"
	"encoding/json"
	"strconv"
	"strings"
)

//go:embed locales/*.json
var localeFiles embed.FS

// Supported locales, in priority order for documentation purposes only:
// the visitor's Accept-Language ranking decides (DetectLocale); the last
// entry is the fallback when nothing matches.
const (
	LocaleID = "id"
	LocaleEN = "en"
	LocaleDE = "de"
	// DefaultLocale answers requests without (or with unusable)
	// Accept-Language: Indonesian, the product's home locale.
	DefaultLocale = LocaleID
)

// messages holds the parsed per-locale documents (locale -> nested map).
var messages = map[string]map[string]any{}

func init() {
	for _, locale := range []string{LocaleID, LocaleEN, LocaleDE} {
		raw, err := localeFiles.ReadFile("locales/" + locale + ".json")
		if err != nil {
			panic("i18n: cannot read embedded locale " + locale + ": " + err.Error())
		}
		var doc map[string]any
		if err := json.Unmarshal(raw, &doc); err != nil {
			panic("i18n: invalid embedded locale " + locale + ": " + err.Error())
		}
		messages[locale] = doc
	}
}

// DetectLocale picks id/en/de from an Accept-Language header value,
// honouring q-weights ("de-DE,de;q=0.9,en;q=0.8" -> "de"). Only the primary
// subtag is considered ("en-US" -> "en"); unknown tags, "*" and malformed
// entries are skipped. Empty or match-less input returns DefaultLocale.
func DetectLocale(acceptLanguage string) string {
	type candidate struct {
		tag string
		q   float64
	}
	var best string
	bestQ := 0.0
	for _, part := range strings.Split(acceptLanguage, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		tag := part
		q := 1.0
		if i := strings.IndexByte(part, ';'); i >= 0 {
			tag = strings.TrimSpace(part[:i])
			for _, param := range strings.Split(part[i+1:], ";") {
				param = strings.TrimSpace(param)
				if v, ok := strings.CutPrefix(param, "q="); ok {
					if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
						q = f
					}
				}
			}
		}
		if i := strings.IndexByte(tag, '-'); i >= 0 {
			tag = tag[:i]
		}
		tag = strings.ToLower(strings.TrimSpace(tag))
		switch tag {
		case LocaleID, LocaleEN, LocaleDE:
			if q > bestQ {
				bestQ = q
				best = tag
			}
		}
	}
	if best == "" {
		return DefaultLocale
	}
	return best
}

// lookup resolves a dot-path key ("link.password.prompt.title") inside one
// locale document (same semantics as the frontend translate()).
func lookup(doc map[string]any, key string) (string, bool) {
	var cur any = doc
	for _, part := range strings.Split(key, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return "", false
		}
		cur, ok = m[part]
		if !ok {
			return "", false
		}
	}
	s, ok := cur.(string)
	return s, ok
}

// Translate returns the string for key in locale, falling back to the
// default locale and finally to the key itself (never empty: a missing key
// stays visible instead of rendering blank, same rule as the frontend).
func Translate(locale, key string) string {
	if doc, ok := messages[locale]; ok {
		if s, ok := lookup(doc, key); ok {
			return s
		}
	}
	if locale != DefaultLocale {
		if s, ok := lookup(messages[DefaultLocale], key); ok {
			return s
		}
	}
	return key
}
