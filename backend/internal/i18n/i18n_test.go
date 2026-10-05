package i18n

// Tests: Accept-Language detection, Translate fallback, and parity between
// the embedded backend locales and the frontend dictionaries (the source of
// truth). The parity test reads the frontend JSON via a relative path -
// test-only coupling, never a build dependency (go:embed cannot leave the
// module, which is why the backend mirrors just these keys).

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDetectLocale(t *testing.T) {
	cases := []struct {
		name   string
		header string
		want   string
	}{
		{"empty defaults to Indonesian", "", LocaleID},
		{"missing defaults to Indonesian", "*/*", LocaleID},
		{"plain id", "id", LocaleID},
		{"en wins outright", "en", LocaleEN},
		{"region stripped", "en-US,en;q=0.9", LocaleEN},
		{"q-weight picks German", "de-DE,de;q=0.9,en;q=0.8", LocaleDE},
		{"q-weight picks English", "fr;q=0.9,en;q=0.5", LocaleEN},
		{"unsupported falls back", "fr-FR,fr;q=0.9,ja;q=0.8", LocaleID},
		{"case-insensitive", "EN-us", LocaleEN},
		{"zero-q loses to default", "de;q=0", LocaleID},
		{"wildcard star ignored", "*, en;q=0.7", LocaleEN},
		{"malformed entries skipped", ";;;, de", LocaleDE},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DetectLocale(tc.header); got != tc.want {
				t.Fatalf("DetectLocale(%q) = %q, want %q", tc.header, got, tc.want)
			}
		})
	}
}

func TestTranslateFallback(t *testing.T) {
	if got := Translate(LocaleEN, "link.password.prompt.button"); got != "Open link" {
		t.Fatalf("en button = %q", got)
	}
	if got := Translate(LocaleDE, "link.password.error.wrong"); got != "Falsches Passwort. Versuch es erneut." {
		t.Fatalf("de wrong = %q", got)
	}
	// Unknown locale falls back to Indonesian, never empty.
	if got := Translate("fr", "link.password.prompt.title"); got != "Link ini dilindungi password" {
		t.Fatalf("fr fallback = %q", got)
	}
	// Unknown key returns the key itself (visible, like the frontend).
	if got := Translate(LocaleID, "no.such.key"); got != "no.such.key" {
		t.Fatalf("missing key = %q", got)
	}
}

// backendKeys are the exact dictionary paths the password gate renders.
// Adding a server-rendered string means adding it here, to locales/*.json
// AND to frontend/messages/{id,en,de}.json - the test enforces all three.
var backendKeys = []string{
	"link.password.prompt.pageTitle",
	"link.password.prompt.title",
	"link.password.prompt.subtitle",
	"link.password.prompt.input_label",
	"link.password.prompt.button",
	"link.password.error.wrong",
	"link.password.error.rate_limited",
}

func frontendDoc(t *testing.T, locale string) map[string]any {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	front := filepath.Join(filepath.Dir(file), "..", "..", "..", "frontend", "messages", locale+".json")
	raw, err := os.ReadFile(front)
	if err != nil {
		t.Fatalf("cannot read frontend dictionary %s: %v", front, err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("invalid frontend dictionary %s: %v", front, err)
	}
	return doc
}

func TestParityWithFrontend(t *testing.T) {
	for _, locale := range []string{LocaleID, LocaleEN, LocaleDE} {
		front := frontendDoc(t, locale)
		for _, key := range backendKeys {
			want, ok := lookup(front, key)
			if !ok {
				t.Errorf("frontend %s misses key %q (add it to messages/%s.json)", locale, key, locale)
				continue
			}
			if got := Translate(locale, key); got != want {
				t.Errorf("locale %s key %q: backend %q != frontend %q", locale, key, got, want)
			}
		}
	}
}
