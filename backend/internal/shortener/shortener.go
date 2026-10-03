package shortener

import (
	// #nosec G501 -- MD5 is used below only as a shard-routing hash
	// (ShardKey), never as a security control; see the note there.
	"crypto/md5"
	"crypto/rand"
	"math/big"
	"regexp"
	"strings"
)

var validSlug = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,30}$`)

// ReservedSlugs lists the level-1 paths the application itself uses: a custom
// slug must never match one exactly, otherwise public links such as /r/api or
// /u/login could never be reached. The list is explicit so words are easy to
// add (one per line).
// Rationale: the short-code namespace is shared with the application's routing
// (/r, /u, /api, static pages). Without this list a user could claim "login"
// and deface an important route: a classic open-redirect-adjacent footgun.
// Trade-off: exact match only (not a prefix): "apiku" stays allowed and only
// exactly "api" is rejected. A static list means a restart to add a word.
// Alternative: a separate namespace (e.g. all links under /s/{code}) so there
// is no collision at all: but URLs become longer.
var ReservedSlugs = []string{
	"api", "u", "r",
	"login", "register", "dashboard", "links",
	"admin", "static", "favicon.ico",
}

// IsReserved reports whether slug collides with app routes (case-insensitive).
func IsReserved(slug string) bool {
	lowered := strings.ToLower(slug)
	for _, r := range ReservedSlugs {
		if lowered == r {
			return true
		}
	}
	return false
}

// ValidSlug enforces 3-30 chars of [a-zA-Z0-9_-] (same alphabet as the
// random generator output, so custom and random codes share one namespace).
func ValidSlug(slug string) bool {
	return validSlug.MatchString(slug)
}

// GenerateShortCode returns a random short code, unique per URL inserted.
// Rationale: collision handling: the code must be unique so database
// conflicts are avoided. Randomness provides a large namespace (62^6
// combinations for a 6-character code) while the collision risk stays very
// small; a scheme that is too structured would produce guessable patterns.
// Alternative: a full UUID, but it is too long for a URL; or an incrementing
// id encoded base62, which requires a central counter.
// Returns the crypto/rand error when the OS entropy source fails; the
// partially built code is discarded ("" is returned) so a failed draw can
// never become a guessable short code.
func GenerateShortCode(length int) (string, error) {
	letters := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(62))
		if err != nil {
			return "", err
		}
		b[i] = letters[n.Int64()]
	}
	return string(b), nil
}

// ShardKey maps a short code to a shard index (0, 1, ...) using an MD5 hash:
// sharding distributes data across several databases so a single one does not
// carry every load. Trade-off: a deterministic hash gives consistent routing
// (same short_code → same shard every time) but requires application-layer
// logic to know which shard to read or write, and changing the number of
// shards requires rebalancing. Alternative: consistent hashing (instead of
// plain modulo) is more flexible when adding shards but more complex to
// implement.
func ShardKey(shortCode string, numShards int) int {
	// #nosec G401,G501 -- MD5 here is a routing hash for shard placement,
	// never a security control: no signature, password, or integrity check
	// depends on it, so hash strength is irrelevant (fast + deterministic is
	// exactly what the modulo needs).
	hash := md5.Sum([]byte(shortCode))
	hashInt := int64(hash[0])<<24 | int64(hash[1])<<16 | int64(hash[2])<<8 | int64(hash[3])
	return int(hashInt) % numShards
}

// ShardName returns the physical table name (urls_shard_a, urls_shard_b, ...)
// for a short code: sharding nomenclature so split tables are easy to
// identify and maintain, with data physically separated instead of all in one
// urls table. Trade-off: descriptive names ease debugging and monitoring (the
// load on shard_a is visible) but add code complexity to derive the name from
// the index, and adding a third shard requires changing the naming logic.
// Alternative: sequential numbering (shard_0, shard_1, shard_2) or region-based
// names, but that reads worse for a simple case like this.
func ShardName(shortCode string, numShards int) string {
	idx := ShardKey(shortCode, numShards)
	// #nosec G115 -- idx is already modulo numShards (a small positive int),
	// so 'a'+idx stays in single-letter range: no truncation is possible.
	return "urls_shard_" + string(rune('a'+idx))
}
