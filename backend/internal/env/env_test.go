package env

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGet covers the fallback contract: unset/empty → default, set → value
// (an explicitly empty var still falls back, matching Get's docs).
func TestGet(t *testing.T) {
	t.Setenv("JEJAK_T_GET", "value")
	if got := Get("JEJAK_T_GET", "def"); got != "value" {
		t.Fatalf("Get set = %q, want value", got)
	}
	if got := Get("JEJAK_T_UNSET_XYZ", "def"); got != "def" {
		t.Fatalf("Get unset = %q, want def", got)
	}
	t.Setenv("JEJAK_T_EMPTY", "")
	if got := Get("JEJAK_T_EMPTY", "def"); got != "def" {
		t.Fatalf("Get empty = %q, want def (empty falls back)", got)
	}
}

// TestStripQuotes: surrounding single/double quotes are removed once; inner
// quotes and unquoted values pass through.
func TestStripQuotes(t *testing.T) {
	cases := map[string]string{
		`"hello"`: "hello",
		"'hello'": "hello",
		`"a"b"`:   `a"b`, // only a fully wrapped pair counts: first+last
		`plain`:   `plain`,
		`"`:       `"`, // too short to be wrapped
		`"ab'cd"`: `ab'cd`,
	}
	for in, want := range cases {
		if got := stripQuotes(in); got != want {
			t.Errorf("stripQuotes(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestLoadDotEnvParsesFile: a temp .env (found via the cwd walk) supplies
// KEY=VALUE pairs, handles export prefixes/comments/blank lines, and NEVER
// overwrites an already-set OS variable.
func TestLoadDotEnvParsesFile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir) // auto-restored after the test

	content := `# comment line
export JEJAK_T_EXPORTED=from-file
JEJAK_T_QUOTED="quoted value"
JEJAK_T_PRESET=from-file

JEJAK_T_MALFORMED_NOEQ
JEJAK_T_EMPTY=
`
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("JEJAK_T_PRESET", "from-os")
	os.Unsetenv("JEJAK_T_EXPORTED")
	os.Unsetenv("JEJAK_T_QUOTED")
	os.Unsetenv("JEJAK_T_EMPTY")

	LoadDotEnv()

	if got := os.Getenv("JEJAK_T_EXPORTED"); got != "from-file" {
		t.Errorf("export prefix: got %q, want from-file", got)
	}
	if got := os.Getenv("JEJAK_T_QUOTED"); got != "quoted value" {
		t.Errorf("quoted value: got %q, want %q", got, "quoted value")
	}
	if got := os.Getenv("JEJAK_T_PRESET"); got != "from-os" {
		t.Errorf("OS env must win: got %q, want from-os", got)
	}
	if got := os.Getenv("JEJAK_T_EMPTY"); got != "" {
		t.Errorf("empty assignment: got %q, want %q", got, "")
	}
}

// TestLoadDotEnvMissingFileIsNoop: no .env anywhere up the tree means plain
// env vars still work and the call does not panic or error.
func TestLoadDotEnvMissingFileIsNoop(t *testing.T) {
	t.Chdir(t.TempDir())
	// The walk may find a real .env in ancestors of the temp dir on some
	// systems; the contract under test is simply "does not crash".
	LoadDotEnv()
}

// TestFindDotEnvWalksUp: a .env in a PARENT directory is discovered from a
// nested working directory (monorepo layout: backend/ subdir, .env at root).
func TestFindDotEnvWalksUp(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("K=V\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(nested)

	got := findDotEnv()
	want := filepath.Join(root, ".env")
	if got != want {
		t.Fatalf("findDotEnv = %q, want %q", got, want)
	}
}
