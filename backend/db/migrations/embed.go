// Package migrations bundles every SQL migration file into the binary through
// go:embed, so the auto-migrate in cmd/server does not depend on the working
// directory when the server runs (safe to invoke from cmd/* and when the
// binary is moved elsewhere). This folder holds only *.sql plus embed.go as
// its "classpath" - embedded data in the style of goose (-- +goose Up marker).
//
// The server reads the SQL files at boot for auto-migrate (see
// internal/migrate). If they were not embedded but read from a path relative
// to the working directory (e.g. "db/migrations"), the server would silently
// fail (or fall back to the wrong location) when started from another
// directory - the same working-directory trap that once made load tests
// inconsistent. Embedding removes that entire class of bugs: the migration
// files travel with the binary and no runtime path lookup happens.
// Trade-off: changing a migration file means rebuilding the binary (normal,
// because migrations are part of a release), not a hot-swap.
package migrations

import "embed"

// FS embeds every *.sql file in this folder; the migration runner reads from
// it instead of the working directory (see the package comment for why that
// matters).
//
//go:embed *.sql
var FS embed.FS

// Names returns the migration file names (including .down.sql; callers filter
// the list themselves, because owning the filesystem must not imply anything
// about its contents). Returns the FS.ReadDir error unchanged when the
// embedded directory cannot be read (a build-time guarantee, but the
// signature keeps embed.FS's contract honest).
func Names() ([]string, error) {
	entries, err := FS.ReadDir(".")
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names, nil
}
