package shortener

import (
	"strings"
	"testing"
)

// TestShardKeyDeterminism + bounds: the routing hash must place the SAME
// code on the SAME shard every time (reads find what writes created) and
// always stay inside [0, numShards).
func TestShardKeyDeterminism(t *testing.T) {
	const shards = 4
	first := ShardKey("abc123", shards)
	for i := 0; i < 20; i++ {
		got := ShardKey("abc123", shards)
		if got != first {
			t.Fatalf("run %d: ShardKey = %d, want stable %d", i, got, first)
		}
	}
	for _, code := range []string{"abc123", "zzzzzz", "000000", "link-1", strings.Repeat("x", 30)} {
		for n := 1; n <= 8; n++ {
			idx := ShardKey(code, n)
			if idx < 0 || idx >= n {
				t.Fatalf("ShardKey(%q, %d) = %d, want 0..%d", code, n, idx, n-1)
			}
		}
	}
}

// TestShardKeySpread: with enough codes the shards should not all collapse
// to one bucket (a broken hash would serialize every write onto shard 0).
func TestShardKeySpread(t *testing.T) {
	const shards = 4
	seen := map[int]bool{}
	for i := 0; i < 200; i++ {
		seen[ShardKey(genCode(i), shards)] = true
	}
	if len(seen) < 3 {
		t.Fatalf("only %d/%d shards hit, hash looks broken", len(seen), shards)
	}
}

// TestShardName: naming follows the key ("urls_shard_a", ...) and always
// produces a single-letter suffix, never a control/rune accident.
func TestShardName(t *testing.T) {
	for _, code := range []string{"abc", "xyz", "123", "long-code-here"} {
		name := ShardName(code, 3)
		if !strings.HasPrefix(name, "urls_shard_") {
			t.Fatalf("name = %q, want urls_shard_ prefix", name)
		}
		suffix := strings.TrimPrefix(name, "urls_shard_")
		if len(suffix) != 1 || suffix[0] < 'a' || suffix[0] >= 'a'+3 {
			t.Fatalf("suffix = %q, want one letter in a..c", suffix)
		}
	}
}

// genCode builds distinct deterministic codes for the spread test.
func genCode(i int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	var b [6]byte
	n := i
	for j := 5; j >= 0; j-- {
		b[j] = alphabet[n%len(alphabet)]
		n /= len(alphabet)
	}
	return string(b[:])
}
