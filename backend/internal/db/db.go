package db

import (
	"database/sql"
	"encoding/json"
	"log"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"jejak/internal/shortener"
)

// LEARN:
//
//	Kenapa: Interface ini mendefinisikan operasi basis data terstruktur (sharded).
//	Konsep design yang terkait: Sharding - membagi data ke beberapa database berbeda
//	supaya 1 database tidak menanggung semua beban (write bottleneck). Setiap operasi
//	(Create, Get, Increment) akan di-route ke shard yang tepat berdasarkan hash short_code.
//	Trade-off: Query menjadi lebih kompleks karena butuh logika routing ke shard.
//	Setiap short_code selalu ke shard yang sama (deterministic), tapi jika ingin menambah
//	shard ke-3, perlu rebalancing data yang ada. Jika tidak ada handling, data bisa jadi
//	hilang di shard yang salah.
//	Alternatif: Consistent hashing fleksibel untuk menambah shard, tapi implementasi lebih rumit.
//
// Link is one row for the dashboard list (Fase 8 reads from replica).
// Position drives creator-page order (reorder feature); 0 = never reordered.
// LEARN:
//
//	Kenapa tags JADI SATU kolom JSONB, bukan tabel tags terpisah: tag di sini
//	kecil (maks 5/link), selalu dibaca bersama link-nya (tidak pernah query
//	mandiri), dan tidak pernah di-query lintas creator ("semua link tag X di
//	seluruh platform" bukan use case). Tabel terpisah = JOIN + migration +
//	CRUD ekstra tanpa manfaat — normalisasi demi normalisasi.
//	Trade-off: Tidak ada UNIQUE per tag di DB; validasi (maks 5, ≤20 char,
//	lowercase) di application layer. Kalau nanti butuh "top tags global",
//	full-scan + agregat akan mahal — saat itu baru migrasi ke tabel.
//	Alternatif: Tabel link_tags(link, tag) + tabel tags — benar untuk sistem
//	tagging skala besar, over-engineering untuk JC ini.
type Link struct {
	ShortCode   string            `json:"short_code"`
	OriginalURL string            `json:"original_url"`
	ClickCount  int64             `json:"click_count"`
	Position    int               `json:"position"`
	IsFeatured  bool              `json:"is_featured"`
	Tags        []string          `json:"tags"`
	DeviceRules map[string]string `json:"device_rules,omitempty"`
}

// parseRules decodes the device_rules JSONB document. Corrupt/empty/'{}' -> nil
// (nil map lookup returns "", so handlers treat missing key = fallback).
func parseRules(raw sql.NullString) map[string]string {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" || strings.TrimSpace(raw.String) == "{}" {
		return nil
	}
	var m map[string]string
	if json.Unmarshal([]byte(raw.String), &m) == nil && m != nil {
		return m
	}
	return nil
}

// scanLinks reads link rows (SELECT must include is_featured AND
// COALESCE(tags,'[]') AS tags AND COALESCE(device_rules,'{}') AS device_rules).
// Corrupt JSON fails soft to empty (1 baris rusak tidak boleh merobohkan
// seluruh list).
func scanLinks(rows *sql.Rows) ([]Link, error) {
	var out []Link
	for rows.Next() {
		var l Link
		var tags sql.NullString
		var rules sql.NullString
		if err := rows.Scan(&l.ShortCode, &l.OriginalURL, &l.ClickCount, &l.Position, &l.IsFeatured, &tags, &rules); err != nil {
			rows.Close()
			return nil, err
		}
		l.Tags = []string{}
		if tags.Valid && strings.TrimSpace(tags.String) != "" {
			var parsed []string
			if err := json.Unmarshal([]byte(tags.String), &parsed); err == nil && parsed != nil {
				l.Tags = parsed
			}
		}
		l.DeviceRules = parseRules(rules)
		out = append(out, l)
	}
	rows.Close()
	return out, nil
}

type ShardStore interface {
	GetURL(shortCode string) (string, error)
	CreateURL(shortCode, originalURL string, creatorID *int64, tagsJSON string) error
	IncrementClickCount(shortCode string) error
	GetShard(shortCode string) *sql.DB
	ListLinks() ([]Link, error)
	CreateCreator(username, displayName, bio, passwordHash string) (int64, error)
	GetCreatorByUsername(username string) (Creator, error)
	GetCreatorByID(id int64) (Creator, error)
	// GetCreatorByIDPrimary: baca langsung dari PRIMARY — untuk data milik
	// user sendiri (dashboard) supaya read-your-own-writes, lihat SingleStore
	// implementasi di bawah.
	GetCreatorByIDPrimary(id int64) (Creator, error)
	UpdateCreatorProfile(id int64, displayName, bio, avatarURL, socialsJSON, theme string) error
	ListLinksByCreator(creatorID int64) ([]Link, error)
	// ListLinksByCreatorPrimary: baca daftar link langsung dari PRIMARY (paket
	// read-your-own-writes yang sama dengan GetCreatorByIDPrimary).
	ListLinksByCreatorPrimary(creatorID int64) ([]Link, error)
	ReorderLinks(creatorID int64, order []string) error
	LogClick(shortCode, referrer string) error
	ClaimLinks(creatorID int64, codes []string) (int64, error)
	ClicksByDay(creatorID int64) ([]DayCount, error)
	GetLink(shortCode string) (Link, error)
	UpdateLink(creatorID int64, shortCode, deviceRulesJSON, tagsJSON string) error
	SetFeaturedLink(creatorID int64, shortCode string, featured bool) error
	StoreAPIKey(creatorID int64, keyHash, label string) (int64, error)
	ListAPIKeys(creatorID int64) ([]APIKey, error)
	DeleteAPIKey(creatorID int64, keyID int64) error
	GetAPIKeyByHash(keyHash string) (APIKey, error)
	TouchAPIKeyLastUsed(keyID int64) error
}

// Creator is one row of Fase 9's creators table. Bio is sql.NullString
// because the column is nullable (NULL must scan, empty string must not
// be confused with "no bio" at the DB layer; handler converts for JSON).
// Same for AvatarURL; Socials holds the raw JSONB document.
type Creator struct {
	ID           int64
	Username     string
	DisplayName  string
	Bio          sql.NullString
	AvatarURL    sql.NullString
	Socials      sql.NullString
	Theme        string
	PasswordHash string
}

// APIKey is one row of the api_keys table. key_hash is NEVER exposed to
// handlers/JSON — only id/label/created_at/last_used_at leave the DB layer
// (plaintext only ever exists in the response the moment a key is generated).
// LastUsedAt is NULL until the key is first used (spec: nullable). No JSON
// tags: sql.Null* doesn't marshal to the shape frontends expect, so the
// handler shapes the payload (same as Creator/creatorProfileJSON).
type APIKey struct {
	ID         int64
	CreatorID  int64
	Label      sql.NullString
	CreatedAt  time.Time
	LastUsedAt sql.NullTime
}

// LEARN:
//
//	Kenapa: Link sosial disimpan sebagai SATU kolom JSONB [{platform,url}],
//	bukan tabel socials terpisah. Data ini kecil (≤10 item), selalu dibaca
//	bersama profil (tidak pernah di-query/join mandiri), dan tidak butuh
//	constraint relasional — tabel terpisah hanya menambah JOIN + migration
//	tanpa manfaat. JSONB (bukan TEXT) supaya tetap tervalidasi sebagai JSON
//	dan bisa di-query/di-index kalau nanti dibutuhkan.
//	Trade-off: Tidak ada FK/UNIQUE per platform di level DB; validasi bentuk
//	(maks item, URL valid) pindah ke application layer. Query "semua kreator
//	yang punya link Instagram" jadi mahal — tapi use case itu tidak ada.
//	Alternatif: Tabel socials(creator_id, platform, url) — benar secara
//	normalisasi, tapi over-engineering untuk list kecil yang read-atomic.
type SocialLink struct {
	Platform string `json:"platform"`
	URL      string `json:"url"`
}

// CreateCreator inserts a creator on the PRIMARY (writes always go primary).
// Returns the new id. Caller hashes the password first (see auth package).
func (s *SingleStore) CreateCreator(username, displayName, bio, passwordHash string) (int64, error) {
	var id int64
	err := s.primary.QueryRow(
		"INSERT INTO creators (username, display_name, bio, password_hash) VALUES ($1, $2, NULLIF($3,''), $4) RETURNING id",
		username, displayName, bio, passwordHash,
	).Scan(&id)
	return id, err
}

// GetCreatorByUsername reads via readDB (replica when enabled).
func (s *SingleStore) GetCreatorByUsername(username string) (Creator, error) {
	var c Creator
	err := s.readDB().QueryRow(
		"SELECT id, username, display_name, bio, avatar_url, socials, theme, password_hash FROM creators WHERE username = $1",
		username,
	).Scan(&c.ID, &c.Username, &c.DisplayName, &c.Bio, &c.AvatarURL, &c.Socials, &c.Theme, &c.PasswordHash)
	return c, err
}

// GetCreatorByID reads one creator by id via readDB (replica when enabled).
func (s *SingleStore) GetCreatorByID(id int64) (Creator, error) {
	var c Creator
	err := s.readDB().QueryRow(
		"SELECT id, username, display_name, bio, avatar_url, socials, theme, password_hash FROM creators WHERE id = $1",
		id,
	).Scan(&c.ID, &c.Username, &c.DisplayName, &c.Bio, &c.AvatarURL, &c.Socials, &c.Theme, &c.PasswordHash)
	return c, err
}

// UpdateCreatorProfile writes display_name, bio, avatar_url, socials, theme to the
// PRIMARY by session owner id. Username is immutable (it is the public URL).
func (s *SingleStore) UpdateCreatorProfile(id int64, displayName, bio, avatarURL, socialsJSON, theme string) error {
	_, err := s.primary.Exec(
		"UPDATE creators SET display_name = $1, bio = NULLIF($2,''), avatar_url = NULLIF($3,''), socials = $4, theme = $5 WHERE id = $6",
		displayName, bio, avatarURL, socialsJSON, theme, id,
	)
	return err
}

// GetCreatorByIDPrimary reads one creator by id from the PRIMARY. Hanya untuk
// data milik user sendiri (GET /api/profile dkk): user harus langsung melihat
// perubahannya sendiri tanpa menunggu sinkronisasi replica manual
// (read-your-own-writes). Endpoint publik tetap lewat readDB (replica) supaya
// replication lag tetap terbaca sebagai pelajaran di situ.
func (s *SingleStore) GetCreatorByIDPrimary(id int64) (Creator, error) {
	var c Creator
	err := s.primary.QueryRow(
		"SELECT id, username, display_name, bio, avatar_url, socials, theme, password_hash FROM creators WHERE id = $1",
		id,
	).Scan(&c.ID, &c.Username, &c.DisplayName, &c.Bio, &c.AvatarURL, &c.Socials, &c.Theme, &c.PasswordHash)
	return c, err
}

// ListLinksByCreator returns ALL links of one creator in ONE query, denormalized
// click_count included — this is the deliberate N+1 avoidance (PRD.md §2):
// one WHERE creator_id query instead of 1 + N per-row count queries.
func (s *SingleStore) ListLinksByCreator(creatorID int64) ([]Link, error) {
	rows, err := s.readDB().Query(
		"SELECT short_code, original_url, click_count, position, is_featured, COALESCE(tags,'[]'), COALESCE(device_rules,'{}') FROM urls WHERE creator_id = $1 ORDER BY position ASC, id DESC",
		creatorID,
	)
	if err != nil {
		return nil, err
	}
	return scanLinks(rows)
}

// ListLinksByCreatorPrimary returns ALL links of one creator directly from the
// PRIMARY (read-your-own-writes untuk dashboard; lihat GetCreatorByIDPrimary).
func (s *SingleStore) ListLinksByCreatorPrimary(creatorID int64) ([]Link, error) {
	rows, err := s.primary.Query(
		"SELECT short_code, original_url, click_count, position, is_featured, COALESCE(tags,'[]'), COALESCE(device_rules,'{}') FROM urls WHERE creator_id = $1 ORDER BY position ASC, id DESC",
		creatorID,
	)
	if err != nil {
		return nil, err
	}
	return scanLinks(rows)
}

// shardStore implements ShardStore using sharded databases
type shardStore struct {
	shards    map[int]*sql.DB
	numShards int
}

// NewShardStore creates a new sharded store with the given number of shards
func NewShardStore(numShards int) *shardStore {
	ss := &shardStore{
		numShards: numShards,
		shards:    make(map[int]*sql.DB),
	}

	for i := 0; i < numShards; i++ {
		dsn := "postgres://jejak:password@postgres_shard" + string(rune('a'+i%26)) + ":5432/jejak?sslmode=disable"
		db, err := sql.Open("postgres", dsn)
		if err != nil {
			log.Printf("Warning: could not connect to shard %d: %v", i, err)
			continue
		}
		ss.shards[i] = db
	}

	return ss
}

// CreateURL implements ShardStore - routes to the correct shard based on short_code hash.
// creatorID nil = anonymous link (Fase 0-8 stay valid); non-nil = owned by creator.
func (s *shardStore) CreateURL(shortCode, originalURL string, creatorID *int64, tagsJSON string) error {
	shardIdx := shortener.ShardKey(shortCode, s.numShards)
	db, ok := s.shards[shardIdx]
	if !ok {
		return sql.ErrConnDone // shard not available
	}
	if tagsJSON == "" {
		tagsJSON = "[]"
	}
	_, err := db.Exec("INSERT INTO urls (short_code, original_url, creator_id, tags) VALUES ($1, $2, $3, $4)", shortCode, originalURL, nullableInt(creatorID), tagsJSON)
	return err
}

// NOTE (shard mode + Fase 9): the creators table has no sharding design, so all
// creator reads/writes go to shard 0. Documented simplification: Fase 9 is meant
// to run on SingleStore (baseline/full); shard mode keeps working for urls.
func (s *shardStore) shardZero() (*sql.DB, bool) {
	db, ok := s.shards[0]
	return db, ok
}

// CreateCreator inserts a creator on shard 0 (see NOTE above).
func (s *shardStore) CreateCreator(username, displayName, bio, passwordHash string) (int64, error) {
	db, ok := s.shardZero()
	if !ok {
		return 0, sql.ErrConnDone
	}
	var id int64
	err := db.QueryRow(
		"INSERT INTO creators (username, display_name, bio, password_hash) VALUES ($1, $2, NULLIF($3,''), $4) RETURNING id",
		username, displayName, bio, passwordHash,
	).Scan(&id)
	return id, err
}

// GetCreatorByUsername reads from shard 0 (see NOTE above).
func (s *shardStore) GetCreatorByUsername(username string) (Creator, error) {
	var c Creator
	db, ok := s.shardZero()
	if !ok {
		return c, sql.ErrConnDone
	}
	err := db.QueryRow(
		"SELECT id, username, display_name, bio, avatar_url, socials, theme, password_hash FROM creators WHERE username = $1",
		username,
	).Scan(&c.ID, &c.Username, &c.DisplayName, &c.Bio, &c.AvatarURL, &c.Socials, &c.Theme, &c.PasswordHash)
	return c, err
}

// GetCreatorByID reads one creator by id from shard 0 (see NOTE above).
func (s *shardStore) GetCreatorByID(id int64) (Creator, error) {
	var c Creator
	db, ok := s.shardZero()
	if !ok {
		return c, sql.ErrConnDone
	}
	err := db.QueryRow(
		"SELECT id, username, display_name, bio, avatar_url, socials, theme, password_hash FROM creators WHERE id = $1",
		id,
	).Scan(&c.ID, &c.Username, &c.DisplayName, &c.Bio, &c.AvatarURL, &c.Socials, &c.Theme, &c.PasswordHash)
	return c, err
}

// UpdateCreatorProfile writes to shard 0 (see NOTE above).
func (s *shardStore) UpdateCreatorProfile(id int64, displayName, bio, avatarURL, socialsJSON, theme string) error {
	db, ok := s.shardZero()
	if !ok {
		return sql.ErrConnDone
	}
	_, err := db.Exec(
		"UPDATE creators SET display_name = $1, bio = NULLIF($2,''), avatar_url = NULLIF($3,''), socials = $4, theme = $5 WHERE id = $6",
		displayName, bio, avatarURL, socialsJSON, theme, id,
	)
	return err
}

// GetCreatorByIDPrimary reads from shard 0 — mode sharded tidak punya
// primary/replica terpisah, jadi hasilnya identik dengan GetCreatorByID.
func (s *shardStore) GetCreatorByIDPrimary(id int64) (Creator, error) {
	return s.GetCreatorByID(id)
}

// ListLinksByCreator queries every shard with the same single-query shape
// (denormalized click_count, no N+1) and merges. Cross-shard ordering is
// by shard index, not global time — acceptable for learning scale.
func (s *shardStore) ListLinksByCreator(creatorID int64) ([]Link, error) {
	var out []Link
	for i := 0; i < s.numShards; i++ {
		db, ok := s.shards[i]
		if !ok {
			continue
		}
		rows, err := db.Query(
			"SELECT short_code, original_url, click_count, position, is_featured, COALESCE(tags,'[]'), COALESCE(device_rules,'{}') FROM urls WHERE creator_id = $1 ORDER BY position ASC, id DESC",
			creatorID,
		)
		if err != nil {
			return nil, err
		}
		part, err := scanLinks(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, part...)
	}
	return out, nil
}

// ListLinksByCreatorPrimary — mode sharded idem ListLinksByCreator (tidak ada
// replica terpisah di shardStore).
func (s *shardStore) ListLinksByCreatorPrimary(creatorID int64) ([]Link, error) {
	return s.ListLinksByCreator(creatorID)
}

// ReorderLinks sets position per short_code in caller order. No cross-shard
// transaction exists (shards are separate connections), so each shard updates
// independently — documented limitation; Fase 9 runs on SingleStore anyway.
func (s *shardStore) ReorderLinks(creatorID int64, order []string) error {
	for pos, code := range order {
		shardIdx := shortener.ShardKey(code, s.numShards)
		db, ok := s.shards[shardIdx]
		if !ok {
			return sql.ErrConnDone
		}
		res, err := db.Exec("UPDATE urls SET position = $1 WHERE short_code = $2 AND creator_id = $3", pos, code, creatorID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n != 1 {
			return sql.ErrNoRows
		}
	}
	return nil
}

// LogClick routes the event write to the short_code's shard (log + counter
// in one transaction, same semantics as SingleStore — see LEARN there).
func (s *shardStore) LogClick(shortCode, referrer string) error {
	shardIdx := shortener.ShardKey(shortCode, s.numShards)
	db, ok := s.shards[shardIdx]
	if !ok {
		return sql.ErrConnDone
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		"INSERT INTO click_events (short_code, referrer) VALUES ($1, NULLIF($2,''))",
		shortCode, referrer,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(
		"UPDATE urls SET click_count = click_count + 1 WHERE short_code = $1",
		shortCode,
	); err != nil {
		return err
	}
	return tx.Commit()
}

// ClicksByDay merges per-day counts from all shards, then zero-fills the
// 30-day window (same helper as SingleStore — one fill logic everywhere).
func (s *shardStore) ClicksByDay(creatorID int64) ([]DayCount, error) {
	merged := make(map[string]int64)
	for i := 0; i < s.numShards; i++ {
		db, ok := s.shards[i]
		if !ok {
			continue
		}
		rows, err := db.Query(
			`SELECT TO_CHAR(ce.clicked_at, 'YYYY-MM-DD') AS day, COUNT(*) AS count
			 FROM click_events ce JOIN urls u ON u.short_code = ce.short_code
			 WHERE u.creator_id = $1 AND ce.clicked_at >= CURRENT_DATE - INTERVAL '29 days'
			 GROUP BY day`,
			creatorID,
		)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var day string
			var n int64
			if err := rows.Scan(&day, &n); err != nil {
				rows.Close()
				return nil, err
			}
			merged[day] += n
		}
		rows.Close()
	}
	return FillLast30Days(merged, time.Now()), nil
}

// ClaimLinks routes each code to its shard (same IS NULL guard per row;
// no cross-shard transaction — documented limitation, see ReorderLinks).
func (s *shardStore) ClaimLinks(creatorID int64, codes []string) (int64, error) {
	var claimed int64
	for _, code := range codes {
		shardIdx := shortener.ShardKey(code, s.numShards)
		db, ok := s.shards[shardIdx]
		if !ok {
			return 0, sql.ErrConnDone
		}
		res, err := db.Exec("UPDATE urls SET creator_id = $1 WHERE short_code = $2 AND creator_id IS NULL", creatorID, code)
		if err != nil {
			return 0, err
		}
		if n, _ := res.RowsAffected(); n == 1 {
			claimed++
		}
	}
	return claimed, nil
}

// UpdateLink implements ShardStore - routes to the shard, owner-scoped.
func (s *shardStore) UpdateLink(creatorID int64, shortCode, deviceRulesJSON, tagsJSON string) error {
	shardIdx := shortener.ShardKey(shortCode, s.numShards)
	db, ok := s.shards[shardIdx]
	if !ok {
		return sql.ErrConnDone
	}
	if strings.TrimSpace(deviceRulesJSON) == "" {
		deviceRulesJSON = "{}"
	}
	if strings.TrimSpace(tagsJSON) == "" {
		tagsJSON = "[]"
	}
	res, err := db.Exec(
		"UPDATE urls SET device_rules = $1, tags = $2 WHERE short_code = $3 AND creator_id = $4",
		deviceRulesJSON, tagsJSON, shortCode, creatorID,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return sql.ErrNoRows
	}
	return nil
}

// SetFeaturedLink implements ShardStore. Semantik sama dengan SingleStore
// (radio 1-per-creator), TAPI tidak ada transaksi lintas-shard (shards = koneksi
// terpisah): "unfeature semua" diiterasi per-shard baru target di-set — bila
// proses mati di tengah, mungkin tersisa 2 featured. Ini dokumentasi-limitation,
// Fase 9 berjalan di SingleStore yang transaksional (lihat LEARN di SingleStore).
func (s *shardStore) SetFeaturedLink(creatorID int64, shortCode string, featured bool) error {
	if featured {
		for i := 0; i < s.numShards; i++ {
			db, ok := s.shards[i]
			if !ok {
				continue
			}
			if _, err := db.Exec("UPDATE urls SET is_featured = FALSE WHERE creator_id = $1", creatorID); err != nil {
				return err
			}
		}
	}
	shardIdx := shortener.ShardKey(shortCode, s.numShards)
	db, ok := s.shards[shardIdx]
	if !ok {
		return sql.ErrConnDone
	}
	res, err := db.Exec(
		"UPDATE urls SET is_featured = $1 WHERE short_code = $2 AND creator_id = $3",
		featured, shortCode, creatorID,
	)
	if err != nil {
		return err
	}
	if featured {
		if n, _ := res.RowsAffected(); n != 1 {
			return sql.ErrNoRows
		}
	}
	return nil
}

// API key rows live on shard 0 together with creators (see NOTE at shardZero):
// keys are per-creator metadata, not per-URL data, so they follow the same
// no-sharding simplification as Fase 9's creators table.
func (s *shardStore) StoreAPIKey(creatorID int64, keyHash, label string) (int64, error) {
	db, ok := s.shardZero()
	if !ok {
		return 0, sql.ErrConnDone
	}
	var id int64
	err := db.QueryRow(
		"INSERT INTO api_keys (creator_id, key_hash, label) VALUES ($1, $2, NULLIF($3,'')) RETURNING id",
		creatorID, keyHash, label,
	).Scan(&id)
	return id, err
}

func (s *shardStore) ListAPIKeys(creatorID int64) ([]APIKey, error) {
	db, ok := s.shardZero()
	if !ok {
		return nil, sql.ErrConnDone
	}
	rows, err := db.Query(
		"SELECT id, creator_id, label, created_at, last_used_at FROM api_keys WHERE creator_id = $1 ORDER BY created_at DESC",
		creatorID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.CreatorID, &k.Label, &k.CreatedAt, &k.LastUsedAt); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (s *shardStore) DeleteAPIKey(creatorID int64, keyID int64) error {
	db, ok := s.shardZero()
	if !ok {
		return sql.ErrConnDone
	}
	res, err := db.Exec("DELETE FROM api_keys WHERE id = $1 AND creator_id = $2", keyID, creatorID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *shardStore) GetAPIKeyByHash(keyHash string) (APIKey, error) {
	db, ok := s.shardZero()
	if !ok {
		return APIKey{}, sql.ErrConnDone
	}
	var k APIKey
	err := db.QueryRow(
		"SELECT id, creator_id, label, created_at, last_used_at FROM api_keys WHERE key_hash = $1",
		keyHash,
	).Scan(&k.ID, &k.CreatorID, &k.Label, &k.CreatedAt, &k.LastUsedAt)
	return k, err
}

func (s *shardStore) TouchAPIKeyLastUsed(keyID int64) error {
	db, ok := s.shardZero()
	if !ok {
		return sql.ErrConnDone
	}
	_, err := db.Exec("UPDATE api_keys SET last_used_at = NOW() WHERE id = $1", keyID)
	return err
}

// nullableInt converts *int64 to driver value: nil -> NULL.
func nullableInt(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}

// GetURL implements ShardStore - routes to the correct shard based on short_code hash
func (s *shardStore) GetURL(shortCode string) (string, error) {
	shardIdx := shortener.ShardKey(shortCode, s.numShards)
	db, ok := s.shards[shardIdx]
	if !ok {
		return "", sql.ErrConnDone
	}
	var url string
	err := db.QueryRow("SELECT original_url FROM urls WHERE short_code = $1", shortCode).Scan(&url)
	if err != nil {
		return "", err
	}
	return url, nil
}

// GetLink implements ShardStore - routes to the shard and returns the row
// needed for Smart Link device routing (original_url + device_rules).
func (s *shardStore) GetLink(shortCode string) (Link, error) {
	shardIdx := shortener.ShardKey(shortCode, s.numShards)
	db, ok := s.shards[shardIdx]
	if !ok {
		return Link{}, sql.ErrConnDone
	}
	var l Link
	var rules sql.NullString
	err := db.QueryRow(
		"SELECT short_code, original_url, COALESCE(device_rules,'{}') FROM urls WHERE short_code = $1",
		shortCode,
	).Scan(&l.ShortCode, &l.OriginalURL, &rules)
	if err != nil {
		return Link{}, err
	}
	l.DeviceRules = parseRules(rules)
	return l, nil
}

// IncrementClickCount implements ShardStore - routes to the correct shard
func (s *shardStore) IncrementClickCount(shortCode string) error {
	shardIdx := shortener.ShardKey(shortCode, s.numShards)
	db, ok := s.shards[shardIdx]
	if !ok {
		return sql.ErrConnDone
	}
	_, err := db.Exec("UPDATE urls SET click_count = click_count + 1 WHERE short_code = $1", shortCode)
	return err
}

// GetShard returns the underlying DB for a specific short_code
func (s *shardStore) GetShard(shortCode string) *sql.DB {
	shardIdx := shortener.ShardKey(shortCode, s.numShards)
	db, ok := s.shards[shardIdx]
	if !ok {
		return nil
	}
	return db
}

// ListLinks returns all links across shards (merged in shard index order).
func (s *shardStore) ListLinks() ([]Link, error) {
	var out []Link
	for i := 0; i < s.numShards; i++ {
		db, ok := s.shards[i]
		if !ok {
			continue
		}
		rows, err := db.Query("SELECT short_code, original_url, click_count, position, is_featured, COALESCE(tags,'[]'), COALESCE(device_rules,'{}') FROM urls ORDER BY id DESC")
		if err != nil {
			return nil, err
		}
		part, err := scanLinks(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, part...)
	}
	return out, nil
}

// DB returns the first available shard's connection (for backward compatibility)
func (s *shardStore) DB() *sql.DB {
	for _, db := range s.shards {
		return db
	}
	return nil
}

// NewShardStoreFromDSNs creates a sharded store from explicit DSNs (one per shard).
// DSNs come from env (SHARD_DSNS, comma-separated), never hardcoded hosts.
// LEARN:
//
//	Kenapa: Supaya daftar shard bisa diubah tanpa ubah kode (tambah shard = ubah env +
//	restart), dan tiap environment (lokal/compose) bisa pakai host berbeda.
//	Trade-off: Format env comma-separated rapuh (spasi/typo bikin shard hilang);
//	jumlah DSN menentukan numShards sehingga salah config = routing salah.
//	Alternatif: Service discovery (Consul/etcd), tapi overkill untuk belajar lokal.
func NewShardStoreFromDSNs(dsns []string) (*shardStore, error) {
	ss := &shardStore{
		numShards: len(dsns),
		shards:    make(map[int]*sql.DB),
	}
	for i, dsn := range dsns {
		db, err := sql.Open("pgx", dsn)
		if err != nil {
			return nil, err
		}
		ss.shards[i] = db
	}
	return ss, nil
}

// LEARN:
//
//	Kenapa: Store 1-database untuk mode baseline (Fase 0): tanpa sharding, tanpa
//	replica wajib. Tulis selalu ke primary; baca ke replica HANYA kalau replica
//	DSN diisi dan reachable (read/write splitting ala Fase 4). Mengimplementasikan
//	ShardStore supaya handler tidak perlu tahu backend apa yang dipakai.
//	Trade-off: Kalau replica mati saat start, baca fallback ke primary + warning
//	(availability diutamakan, konsistensi baca tetap dari primary). Ping saat start
//	bikin API crash-loop kalau primary belum ready — ditangani via restart policy.
//	Alternatif: Health-check + circuit breaker per query, tapi kompleks untuk baseline.
type SingleStore struct {
	primary    *sql.DB
	replica    *sql.DB
	useReplica bool
}

// NewSingleStore opens the primary DB and, optionally, a read replica.
// Empty replicaDSN = reads fall back to primary (pure Fase 0 baseline).
func NewSingleStore(primaryDSN, replicaDSN string) (*SingleStore, error) {
	primary, err := sql.Open("pgx", primaryDSN)
	if err != nil {
		return nil, err
	}
	s := &SingleStore{primary: primary}
	if replicaDSN != "" {
		rep, err := sql.Open("pgx", replicaDSN)
		if err != nil {
			primary.Close()
			return nil, err
		}
		if err := rep.Ping(); err != nil {
			log.Printf("Warning: replica unreachable, reads fall back to primary: %v", err)
			rep.Close()
		} else {
			s.replica = rep
			s.useReplica = true
		}
	}
	return s, nil
}

// readDB returns the replica when enabled, otherwise the primary.
func (s *SingleStore) readDB() *sql.DB {
	if s.useReplica && s.replica != nil {
		return s.replica
	}
	return s.primary
}

// CreateURL writes to the primary.
func (s *SingleStore) CreateURL(shortCode, originalURL string, creatorID *int64, tagsJSON string) error {
	if tagsJSON == "" {
		tagsJSON = "[]"
	}
	_, err := s.primary.Exec("INSERT INTO urls (short_code, original_url, creator_id, tags) VALUES ($1, $2, $3, $4)", shortCode, originalURL, nullableInt(creatorID), tagsJSON)
	return err
}

// GetURL reads from the replica when enabled, otherwise the primary.
func (s *SingleStore) GetURL(shortCode string) (string, error) {
	var url string
	err := s.readDB().QueryRow("SELECT original_url FROM urls WHERE short_code = $1", shortCode).Scan(&url)
	if err != nil {
		return "", err
	}
	return url, nil
}

// GetLink reads the full redirect row (original_url + device_rules) — needed
// by the Smart Link redirect path to route by device. Same read path as GetURL.
func (s *SingleStore) GetLink(shortCode string) (Link, error) {
	var l Link
	var rules sql.NullString
	err := s.readDB().QueryRow(
		"SELECT short_code, original_url, COALESCE(device_rules,'{}') FROM urls WHERE short_code = $1",
		shortCode,
	).Scan(&l.ShortCode, &l.OriginalURL, &rules)
	if err != nil {
		return Link{}, err
	}
	l.DeviceRules = parseRules(rules)
	return l, nil
}

// IncrementClickCount writes to the primary.
func (s *SingleStore) IncrementClickCount(shortCode string) error {
	_, err := s.primary.Exec("UPDATE urls SET click_count = click_count + 1 WHERE short_code = $1", shortCode)
	return err
}

// GetShard returns the DB used for reads (single connection, no sharding).
func (s *SingleStore) GetShard(shortCode string) *sql.DB {
	return s.readDB()
}

// ListLinks returns all links, read from the replica when enabled
// (Fase 8 dashboard reads from the read replica).
func (s *SingleStore) ListLinks() ([]Link, error) {
	rows, err := s.readDB().Query("SELECT short_code, original_url, click_count, position, is_featured, COALESCE(tags,'[]'), COALESCE(device_rules,'{}') FROM urls ORDER BY id DESC")
	if err != nil {
		return nil, err
	}
	return scanLinks(rows)
}

// DB returns the primary connection.
func (s *SingleStore) DB() *sql.DB {
	return s.primary
}

// LEARN:
//
//	Kenapa klaim HARUS conditional write (WHERE creator_id IS NULL), bukan
//	update buta: server tidak tahu link anonim mana dibuat browser mana (HTTP
//	stateless, pre-login tidak ada identitas) — jadi daftar kandidat datang
//	dari client (localStorage browser pembuatnya). Kalau server asal update
//	tanpa syarat, SIAPA PUN yang login bisa merebut link milik orang lain
//	hanya dengan menebak short-code. Syarat IS NULL adalah batas keamanannya:
//	link ber-pemilik tidak tersentuh, klaim ganda aman (affected 0).
//	Trade-off: Seluruh batch dalam 1 transaction (semua-atau-tidak-sama-sekali)
//	supaya klaim parsial tidak membingungkan ("3 dari 5 masuk, mana yang mana?").
//	Alternatif: Klaim otomatis semua link NULL ke user yang login (tanpa daftar
//	client) — terlihat simpel tapi itu celah: pendaftar pertama menyapu bersih
//	SEMUA link anonim siapa pun. Jangan pernah lakukan itu.
func (s *SingleStore) ClaimLinks(creatorID int64, codes []string) (int64, error) {
	tx, err := s.primary.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var claimed int64
	for _, code := range codes {
		res, err := tx.Exec("UPDATE urls SET creator_id = $1 WHERE short_code = $2 AND creator_id IS NULL", creatorID, code)
		if err != nil {
			return 0, err
		}
		if n, _ := res.RowsAffected(); n == 1 {
			claimed++
		}
	}
	return claimed, tx.Commit()
}

// UpdateLink overwrites device_rules + tags for one link, scoped to the owner
// (WHERE creator_id). Rows scoped elsewhere / unknown -> sql.ErrNoRows -> 404.
// Both values are full-replace (matches the profile endpoint philosophy: the
// client sends the complete new state, no PATCH merge ambiguity).
func (s *SingleStore) UpdateLink(creatorID int64, shortCode, deviceRulesJSON, tagsJSON string) error {
	if strings.TrimSpace(deviceRulesJSON) == "" {
		deviceRulesJSON = "{}"
	}
	if strings.TrimSpace(tagsJSON) == "" {
		tagsJSON = "[]"
	}
	res, err := s.primary.Exec(
		"UPDATE urls SET device_rules = $1, tags = $2 WHERE short_code = $3 AND creator_id = $4",
		deviceRulesJSON, tagsJSON, shortCode, creatorID,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return sql.ErrNoRows
	}
	return nil
}

// SetFeaturedLink makes `shortCode` the creator's single featured link
// (radio, bukan checkbox): featured=true meng-unfeature SEMUA link akun itu
// lalu menaikkan target — dua write dalam SATU transaksi supaya pengamat
// tidak pernah melihat 0 ATAU 2 featured (at-atomik seperti ClaimLinks).
// Diblokir scope WITHIN creator_id pada target: link orang lain/kode kosong
// -> sql.ErrNoRows -> 404. featured=false idempotent (unfeature target;
// affected 0 = sudah tidak featured = sukses, bukan error). shardStore
// meniru semantik ini tanpa transaksi lintas-shard (lihat LEARN di sana).
func (s *SingleStore) SetFeaturedLink(creatorID int64, shortCode string, featured bool) error {
	tx, err := s.primary.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if featured {
		if _, err := tx.Exec("UPDATE urls SET is_featured = FALSE WHERE creator_id = $1", creatorID); err != nil {
			return err
		}
	}
	res, err := tx.Exec(
		"UPDATE urls SET is_featured = $1 WHERE short_code = $2 AND creator_id = $3",
		featured, shortCode, creatorID,
	)
	if err != nil {
		return err
	}
	if featured {
		if n, _ := res.RowsAffected(); n != 1 {
			return sql.ErrNoRows
		}
	}
	return tx.Commit()
}

// LEARN:
//
//	Kenapa: Storage API key memakai pola yang sama seperti password — hanya
//	hash yang disimpan, key asli tidak pernah menyentuh database. Kalau
//	database bocor, attacker tidak langsung bisa pakai key tersebut; format
//	"jjk_" + 32 hex (128 bit) membuat brute-force SHA-256 mustahil secara
//	komputasi (beda dari password user yang butuh bcrypt/argon2 karena
//	entropy-nya rendah — key itu random penuh, jadi hash cepat cukup dan
//	harga kecepatan itu wajib karena hash dicek di SETIAP request).
//	Trade-off: SHA-256 bukan fungsi lambat (tidak seperti bcrypt), tapi itu
//	disengaja: key 128-bit random tidak pernah bisa di-brute-force lewat
//	offline hash, dan API key di-autentikasi per-request (bcrypt per-request
//	= +100ms latensi per panggilan).
func (s *SingleStore) StoreAPIKey(creatorID int64, keyHash, label string) (int64, error) {
	var id int64
	err := s.primary.QueryRow(
		"INSERT INTO api_keys (creator_id, key_hash, label) VALUES ($1, $2, NULLIF($3,'')) RETURNING id",
		creatorID, keyHash, label,
	).Scan(&id)
	return id, err
}

// ListAPIKeys returns the caller's keys. Writes (creator_id scope) read from
// primary — keys are low-volume and must be instantly consistent for the
// list/delete flow, so replica staleness is not worth it here.
func (s *SingleStore) ListAPIKeys(creatorID int64) ([]APIKey, error) {
	rows, err := s.primary.Query(
		"SELECT id, creator_id, label, created_at, last_used_at FROM api_keys WHERE creator_id = $1 ORDER BY created_at DESC",
		creatorID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []APIKey
	for rows.Next() {
		var k APIKey
		if err := rows.Scan(&k.ID, &k.CreatorID, &k.Label, &k.CreatedAt, &k.LastUsedAt); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// DeleteAPIKey removes one key, scoped to the owner. Unknown id or another
// creator's key -> sql.ErrNoRows (handler maps this to 404, no key-id leak).
func (s *SingleStore) DeleteAPIKey(creatorID int64, keyID int64) error {
	res, err := s.primary.Exec("DELETE FROM api_keys WHERE id = $1 AND creator_id = $2", keyID, creatorID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return sql.ErrNoRows
	}
	return nil
}

// GetAPIKeyByHash looks a hash up for the API-key auth middleware. Read path
// (replica when enabled) because every /api/v1 request hits this.
func (s *SingleStore) GetAPIKeyByHash(keyHash string) (APIKey, error) {
	var k APIKey
	err := s.readDB().QueryRow(
		"SELECT id, creator_id, label, created_at, last_used_at FROM api_keys WHERE key_hash = $1",
		keyHash,
	).Scan(&k.ID, &k.CreatorID, &k.Label, &k.CreatedAt, &k.LastUsedAt)
	return k, err
}

// TouchAPIKeyLastUsed stamps last_used_at on a key (run after successful API
// auth). Fire-and-forget-friendly: error is logged by the caller, never
// fails the request — the request already succeeded by then.
func (s *SingleStore) TouchAPIKeyLastUsed(keyID int64) error {
	_, err := s.primary.Exec("UPDATE api_keys SET last_used_at = NOW() WHERE id = $1", keyID)
	return err
}

// DayCount is one chart point: clicks on a calendar day (YYYY-MM-DD).
type DayCount struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// LEARN:
//
//	Kenapa: Agregasi (GROUP BY hari) dikerjakan DI query DB, bukan dengan
//	menarik semua baris click_events lalu dihitung di aplikasi. Ini prinsip
//	yang sama dengan hindari N+1, versi "ringkasan": kalau yang dibutuhkan
//	cuma 30 angka, jangan transfer ribuan row mentah lewat network lalu
//	buang 99%nya di memori aplikasi. DB memang dirancang untuk agregat
//	(index + GROUP BY), aplikasi tidak.
//	Trade-off: Logika bisnis (bucket tanggal, zero-fill) terbelah: GROUP BY
//	di SQL, zero-fill tanggal kosong di Go (lihat FillLast30Days) — karena
//	SQL tanpa generate_series akan menghilangkan hari tanpa klik (chart
//	bolong). Alternatif: generate_series di SQL (1 query penuh) atau tarik
//	mentah + agregat di Go (boros bandwidth, ditolak).
func (s *SingleStore) dailyCounts(creatorID int64) (map[string]int64, error) {
	rows, err := s.readDB().Query(
		`SELECT TO_CHAR(ce.clicked_at, 'YYYY-MM-DD') AS day, COUNT(*) AS count
		 FROM click_events ce JOIN urls u ON u.short_code = ce.short_code
		 WHERE u.creator_id = $1 AND ce.clicked_at >= CURRENT_DATE - INTERVAL '29 days'
		 GROUP BY day`,
		creatorID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]int64)
	for rows.Next() {
		var day string
		var n int64
		if err := rows.Scan(&day, &n); err != nil {
			return nil, err
		}
		out[day] = n
	}
	return out, nil
}

// ClicksByDay returns a complete 30-day window (today included) with zeros
// for days without clicks, read from the replica when enabled.
func (s *SingleStore) ClicksByDay(creatorID int64) ([]DayCount, error) {
	counts, err := s.dailyCounts(creatorID)
	if err != nil {
		return nil, err
	}
	return FillLast30Days(counts, time.Now()), nil
}

// FillLast30Days merges per-day counts into a complete window ending today.
// Missing days become 0 so charts never have gaps.
func FillLast30Days(counts map[string]int64, today time.Time) []DayCount {
	out := make([]DayCount, 0, 30)
	for i := 29; i >= 0; i-- {
		d := today.AddDate(0, 0, -i).Format("2006-01-02")
		out = append(out, DayCount{Date: d, Count: counts[d]})
	}
	return out
}

// LEARN:
//
//	Kenapa: Satu event klik ditulis sebagai 2 baris dalam 1 transaction:
//	INSERT ke click_events (log append-only) + UPDATE counter click_count.
//	Keduanya harus berhasil bersama — kalau counter naik tapi log hilang
//	(atau sebaliknya), analytics dan counter berbohong satu sama lain.
//	Trade-off: Transaction per event = 2 writes serial per klik (lebih berat
//	dari fire-and-forget murni); batching N event per transaction akan lebih
//	cepat tapi menunda visibilitas data. Delivery tetap at-most-once: gagal
//	di tengah = event hilang + warning log (tidak ada retry/antrian mati).
//	Alternatif: Pisah jadi 2 operasi tanpa transaction (risiko skew) atau
//	hitung counter dari agregat click_events saat dibaca (mahal, anti-Fase 2).
func (s *SingleStore) LogClick(shortCode, referrer string) error {
	tx, err := s.primary.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		"INSERT INTO click_events (short_code, referrer) VALUES ($1, NULLIF($2,''))",
		shortCode, referrer,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(
		"UPDATE urls SET click_count = click_count + 1 WHERE short_code = $1",
		shortCode,
	); err != nil {
		return err
	}
	return tx.Commit()
}

// LEARN:
//
//	Kenapa: Urutan baru ditulis dalam SATU transaction: semua UPDATE position
//	commit bersama, atau rollback semua kalau 1 gagal. Tanpa transaction (N
//	query terpisah), request yang mati di tengah jalan meninggalkan urutan
//	setengah-jadi = data korup yang tidak terdeteksi. Tiap baris dicek
//	kepemilikan (WHERE creator_id) supaya user tidak bisa mengacak link orang.
//	Trade-off: Transaction menahan lock baris selama update (mili-detik untuk
//	puluhan baris, tidak masalah di skala ini); array raksasa akan menahan
//	lock lama — makanya handler membatasi panjang order.
//	Alternatif: N UPDATE tanpa transaction (simpel tapi tidak atomik), atau
//	fractional indexing (sisip tanpa rewrite semua — hemat write tapi kompleks).
func (s *SingleStore) ReorderLinks(creatorID int64, order []string) error {
	tx, err := s.primary.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for pos, code := range order {
		res, err := tx.Exec("UPDATE urls SET position = $1 WHERE short_code = $2 AND creator_id = $3", pos, code, creatorID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n != 1 {
			return sql.ErrNoRows
		}
	}
	return tx.Commit()
}
