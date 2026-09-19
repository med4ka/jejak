package shortener

import (
	"testing"
)

func TestGenerateShortCode(t *testing.T) {
	tests := []struct {
		length int
		want   string
	}{
		{1, "a"}, // minimal length
		{6, "abcdef"}, // typical length
		{10, "abcdefghij"}, // longer code
	}

	for _, tt := range tests {
		got, err := GenerateShortCode(tt.length)
		if err != nil {
			t.Errorf("GenerateShortCode(%d) returned error: %v", tt.length, err)
			continue
		}
		if len(got) != tt.length {
			t.Errorf("GenerateShortCode(%d) = %q, want length %d", tt.length, got, tt.length)
		}
		// Verify all characters come from the allowed alphabet
		letters := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		for _, c := range got {
			found := false
			for _, l := range letters {
				if rune(c) == l {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("GenerateShortCode(%d) = %q contains disallowed character %c", tt.length, got, c)
			}
		}
	}
}

func TestValidSlug(t *testing.T) {
	tests := []struct {
		slug  string
		want  bool
		limit string // "valid", "too_short", "too_long", "bad_chars"
	}{
		{"a", false, "too_short"}, // 1 char < 3 minimum
		{"ab", false, "too_short"}, // 2 chars < 3 minimum
		{"abc", true, "valid"}, // exactly 3 chars
		{"AbC123", true, "valid"}, // mixed case + digits
		{"abc_xyz", true, "valid"}, // with underscore
		{"abc-def", true, "valid"}, // with hyphen
		{"abcdefghijklmnopqrstuvwxyz0123", true, "valid"}, // exactly 30 chars (26+4)
		{"abcdefghijklmnopqrstuvwxyz01234", false, "too_long"}, // 31 chars > 30
		{"abc!", false, "bad_chars"}, // invalid character
		{"ab c", false, "bad_chars"}, // space
	}

	for _, tt := range tests {
		got := ValidSlug(tt.slug)
		if got != tt.want {
			t.Errorf("ValidSlug(%q) = %v, want %v", tt.slug, got, tt.want)
		}
	}
}

func TestIsReserved(t *testing.T) {
	tests := []struct {
		slug   string
		want   bool // true = reserved (should be rejected), false = OK
	}{
		{"api", true},
		{"u", true},
		{"r", true},
		{"login", true},
		{"register", true},
		{"dashboard", true},
		{"links", true},
		{"admin", true},
		{"static", true},
		{"favicon.ico", true},
		{"Api", true}, // case-insensitive
		{"API", true},
		{"Login", true},
		// These should NOT be reserved
		{"apiku", false}, // "apiku" != "api" exact match
		{"urlink", false}, // not reserved
		{"myprofile", false}, // not reserved
		{"ufoo", false}, // starts with u but not exact "u"
		{" r", false}, // space padding
	}

	for _, tt := range tests {
		got := IsReserved(tt.slug)
		if got != tt.want {
			t.Errorf("IsReserved(%q) = %v, want %v", tt.slug, got, tt.want)
		}
	}
}