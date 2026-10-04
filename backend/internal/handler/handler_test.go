package handler

import (
	"bytes"
	"database/sql"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"jejak/internal/auth"
	"jejak/internal/db"
	"jejak/internal/middleware"
)

// fakeStore is a minimal in-memory ShardStore for handler tests. Only the
// methods exercised by the handlers under test behave meaningfully; the rest
// return zero values so the struct satisfies the interface.
type fakeStore struct {
	// urls maps short_code -> original_url. A code present here counts as an
	// existing (taken) row.
	urls map[string]string
	// created records every successful CreateURL call for assertions.
	created []createCall
	// creator is returned by GetCreatorByUsername/ByID; err (if set) wins.
	creator    db.Creator
	creatorErr error
	// claimCodes records every ClaimLinks call; claimResult is returned.
	claimCodes  []string
	claimResult int64
	// link shadows GetLink result for Smart Link redirect tests.
	link db.Link
	// linkErr returned by GetLink when set (default sql.ErrNoRows if link zero).
	linkErr error
	// updateCalls records every UpdateLink call.
	updateCalls []updateCall
	// updateErr forces an UpdateLink failure when set.
	updateErr error
	// featureCalls records every SetFeaturedLink call.
	featureCalls []featureCall
	// featureErr returns ErrNoRows-shaped failures like the real store when set.
	featureErr error
	// activeCalls records every SetLinkActive call.
	activeCalls []activeCall
	// keyList is returned by ListAPIKeys.
	keyList []db.APIKey
	// keyByHash is returned by GetAPIKeyByHash.
	keyByHash db.APIKey
	// keyByHashErr forces GetAPIKeyByHash failure when set (default sql.ErrNoRows).
	keyByHashErr error
	// storedKeys records (hash,label) for every StoreAPIKey.
	storedKeys []storedKeyCall
	// deletedKeyIDs records every DeleteAPIKey key id.
	deletedKeyIDs []int64
	// touchedKeyIDs records every TouchAPIKeyLastUsed key id.
	touchedKeyIDs []int64
	// profileCalls records every UpdateCreatorProfile call for assertions.
	profileCalls []profileCall
	// creatorLinks is returned by ListLinksByCreator(Primary): lets tests
	// prove GET /api/links returns the caller's own links (scoping), not all.
	creatorLinks []db.Link
	// summary / summaryErr returned by AnalyticsSummary.
	summary    db.AnalyticsSummary
	summaryErr error
	// Account settings (handlers_account_test.go): emailCalls records every
	// UpdateCreatorEmail email; emailErr forces ErrEmailTaken-shaped failure;
	// passwordHashes records UpdateCreatorPassword hashes; deletedAccounts
	// records DeleteCreatorAccount ids.
	emailCalls      []string
	emailErr        error
	passwordHashes  []string
	deletedAccounts []int64
	// Link management (2026-09-30): expiryCalls records SetLinkExpiry
	// (code + nil=clear); bulkCalls records each CreateURLsBatch input;
	// bulkConflicts forces a given code to return "" (ON CONFLICT DO
	// NOTHING); bulkConflictAll = all codes conflict (mass "duplicate" test).
	expiryCalls     []expiryCall
	bulkCalls       [][]db.BulkURL
	bulkConflicts   map[string]bool
	bulkConflictAll bool
	// Analytics depth (2026-09-30): clickEvents records EVERY LogClick: used
	// to prove device/referrer classification happens AT WRITE TIME (deviation
	// A), not at read time. deviceItems/referrerItems/daily/linkStats/clickRows
	// are the read results returned by the 3 new handlers; analyticsErr forces
	// a DB failure and dataPer is the read-source label.
	clickEvents   []db.ClickEvent
	deviceItems   []db.BreakdownItem
	referrerItems []db.BreakdownItem
	daily         map[string]int64
	linkStats     []db.LinkExportRow
	clickRows     []db.ClickRow
	analyticsErr  error
	dataPer       string
	// Health monitor + password (2026-10-04): passwordCalls/fallbackCalls
	// record SetLinkPassword/SetLinkFallback (scoped like the real store via
	// hasCode); healthCalls records UpdateLinkHealth; dueList is returned by
	// ListLinksHealthDue; notifList/notifUnread feed the notification
	// handlers; readCalls records MarkNotificationRead ids; notifSeq gives
	// CreateNotification unique ids.
	passwordCalls []passwordCall
	fallbackCalls []fallbackCall
	healthCalls   []healthCall
	dueList       []db.Link
	notifList     []db.Notification
	notifUnread   int64
	readCalls     []int64
	notifSeq      int64
}

type storedKeyCall struct {
	keyHash string
	label   string
}

type updateCall struct {
	code            string
	deviceRulesJSON string
	tagsJSON        string
}

type featureCall struct {
	code     string
	featured bool
}

type activeCall struct {
	code   string
	active bool
}

type profileCall struct {
	avatarURL string
}

type createCall struct {
	code         string
	originalURL  string
	creatorID    *int64
	tagsJSON     string
	expiresAt    *time.Time // link expiry (nil = no limit)
	passwordHash string     // bcrypt hash ("" = no gate), migration 17
}

// passwordCall records SetLinkPassword(creatorID=shortCode, hash "" = clear).
type passwordCall struct {
	code string
	hash string
}

// fallbackCall records SetLinkFallback(creatorID=shortCode, url "" = clear).
type fallbackCall struct {
	code string
	url  string
}

// healthCall records UpdateLinkHealth (zero notifiedAt = reset marker).
type healthCall struct {
	code       string
	status     string
	checkedAt  time.Time
	notifiedAt time.Time
}

// expiryCall records SetLinkExpiry(creatorID=shortCode, expiresAt nil=clear).
type expiryCall struct {
	code      string
	expiresAt *time.Time
}

func (f *fakeStore) GetURL(shortCode string) (string, error) {
	if u, ok := f.urls[shortCode]; ok {
		return u, nil
	}
	return "", sql.ErrNoRows
}

func (f *fakeStore) CreateURL(shortCode, originalURL string, creatorID *int64, tagsJSON string, expiresAt *time.Time, passwordHash string) error {
	f.created = append(f.created, createCall{code: shortCode, originalURL: originalURL, creatorID: creatorID, tagsJSON: tagsJSON, expiresAt: expiresAt, passwordHash: passwordHash})
	if f.urls == nil {
		f.urls = map[string]string{}
	}
	f.urls[shortCode] = originalURL
	return nil
}

// CreateURLsBatch mirrors the real store: codes listed in bulkConflicts return
// "" (ON CONFLICT: the row is skipped and the transaction still proceeds);
// other rows succeed and are recorded as created too, so hasCode/SetLinkActive
// work with imported codes.
func (f *fakeStore) CreateURLsBatch(creatorID *int64, items []db.BulkURL) ([]string, error) {
	f.bulkCalls = append(f.bulkCalls, items)
	codes := make([]string, len(items))
	for i, it := range items {
		if f.bulkConflictAll || f.bulkConflicts[it.ShortCode] {
			continue // "" = unique conflict in the DB
		}
		if f.urls == nil {
			f.urls = map[string]string{}
		}
		f.urls[it.ShortCode] = it.OriginalURL
		f.created = append(f.created, createCall{
			code: it.ShortCode, originalURL: it.OriginalURL,
			creatorID: creatorID, tagsJSON: it.TagsJSON, expiresAt: it.ExpiresAt,
		})
		codes[i] = it.ShortCode
	}
	return codes, nil
}

// SetLinkExpiry scopes to rows previously created via CreateURL (tracked by
// hasCode): a code the caller does not own or an unknown one → sql.ErrNoRows,
// same as the real store (its WHERE short_code AND creator_id would not match).
func (f *fakeStore) SetLinkExpiry(creatorID int64, shortCode string, expiresAt *time.Time) error {
	if !hasCode(f.created, shortCode) {
		return sql.ErrNoRows
	}
	f.expiryCalls = append(f.expiryCalls, expiryCall{code: shortCode, expiresAt: expiresAt})
	return nil
}

// SetLinkPassword mirrors the real store: scoped to rows created via
// CreateURL (hasCode); unknown/foreign -> sql.ErrNoRows (404).
func (f *fakeStore) SetLinkPassword(creatorID int64, shortCode, passwordHash string) error {
	if !hasCode(f.created, shortCode) {
		return sql.ErrNoRows
	}
	f.passwordCalls = append(f.passwordCalls, passwordCall{code: shortCode, hash: passwordHash})
	return nil
}

// SetLinkFallback mirrors SetLinkPassword (owner-scoped, "" = clear).
func (f *fakeStore) SetLinkFallback(creatorID int64, shortCode, fallbackURL string) error {
	if !hasCode(f.created, shortCode) {
		return sql.ErrNoRows
	}
	f.fallbackCalls = append(f.fallbackCalls, fallbackCall{code: shortCode, url: fallbackURL})
	return nil
}

// UpdateLinkHealth records the worker/manual check (not owner-scoped, like
// the real store: unknown codes are a no-op, not an error).
func (f *fakeStore) UpdateLinkHealth(shortCode, status string, checkedAt, notifiedAt time.Time) error {
	f.healthCalls = append(f.healthCalls, healthCall{code: shortCode, status: status, checkedAt: checkedAt, notifiedAt: notifiedAt})
	return nil
}

// ListLinksHealthDue returns the preset dueList (tests control the batch).
func (f *fakeStore) ListLinksHealthDue(limit int) ([]db.Link, error) {
	out := f.dueList
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// CreateNotification appends to notifList and returns a unique id.
func (f *fakeStore) CreateNotification(creatorID int64, typ, shortCode, message string) (int64, error) {
	f.notifSeq++
	f.notifList = append(f.notifList, db.Notification{ID: f.notifSeq, Type: typ, ShortCode: shortCode, Message: message})
	return f.notifSeq, nil
}

// ListNotifications returns the preset notifList (newest first already).
func (f *fakeStore) ListNotifications(creatorID int64, limit int) ([]db.Notification, error) {
	out := f.notifList
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// CountUnreadNotifications returns the preset badge count.
func (f *fakeStore) CountUnreadNotifications(creatorID int64) (int64, error) {
	return f.notifUnread, nil
}

// MarkNotificationRead records the id and answers sql.ErrNoRows for an
// unknown one (404, same contract as the real store).
func (f *fakeStore) MarkNotificationRead(creatorID, id int64) error {
	for _, n := range f.notifList {
		if n.ID == id {
			f.readCalls = append(f.readCalls, id)
			return nil
		}
	}
	return sql.ErrNoRows
}

func (f *fakeStore) IncrementClickCount(shortCode string) error { return nil }
func (f *fakeStore) GetShard(shortCode string) *sql.DB          { return nil }
func (f *fakeStore) ListLinks() ([]db.Link, error)              { return []db.Link{}, nil }
func (f *fakeStore) CreateCreator(username, displayName, bio, passwordHash string) (int64, error) {
	return 1, nil
}
func (f *fakeStore) GetCreatorByUsername(username string) (db.Creator, error) {
	if f.creatorErr != nil {
		return db.Creator{}, f.creatorErr
	}
	return f.creator, nil
}
func (f *fakeStore) GetCreatorByID(id int64) (db.Creator, error) {
	if f.creatorErr != nil {
		return db.Creator{}, f.creatorErr
	}
	return f.creator, nil
}
func (f *fakeStore) GetCreatorByIDPrimary(id int64) (db.Creator, error) {
	return f.GetCreatorByID(id)
}
func (f *fakeStore) GetCreatorAuth(id int64) (db.Creator, error) {
	return f.GetCreatorByID(id)
}
func (f *fakeStore) UpdateCreatorEmail(id int64, email string) error {
	if f.emailErr != nil {
		return f.emailErr
	}
	f.emailCalls = append(f.emailCalls, email)
	return nil
}
func (f *fakeStore) UpdateCreatorPassword(id int64, passwordHash string) error {
	f.passwordHashes = append(f.passwordHashes, passwordHash)
	return nil
}
func (f *fakeStore) DeleteCreatorAccount(id int64) error {
	f.deletedAccounts = append(f.deletedAccounts, id)
	return nil
}
func (f *fakeStore) UpdateCreatorProfile(id int64, displayName, bio, avatarURL, socialsJSON, theme string) error {
	f.profileCalls = append(f.profileCalls, profileCall{avatarURL: avatarURL})
	return nil
}
func (f *fakeStore) ListLinksByCreator(creatorID int64) ([]db.Link, error) {
	if f.creatorLinks == nil {
		return []db.Link{}, nil
	}
	return f.creatorLinks, nil
}
func (f *fakeStore) ListLinksByCreatorPrimary(creatorID int64) ([]db.Link, error) {
	return f.ListLinksByCreator(creatorID)
}
func (f *fakeStore) ReorderLinks(creatorID int64, order []string) error { return nil }

// LogClick records the event (rather than merely returning nil): analytics
// unit tests use this record to prove buildClickEvent classifies
// device/referrer before the row reaches the DB.
func (f *fakeStore) LogClick(e db.ClickEvent) error {
	f.clickEvents = append(f.clickEvents, e)
	return nil
}
func (f *fakeStore) ClaimLinks(creatorID int64, codes []string) (int64, error) {
	f.claimCodes = append(f.claimCodes, codes...)
	return f.claimResult, nil
}

func (f *fakeStore) GetLink(shortCode string) (db.Link, error) {
	if f.linkErr != nil {
		return db.Link{}, f.linkErr
	}
	if f.link.ShortCode == "" {
		return db.Link{}, sql.ErrNoRows
	}
	return f.link, nil
}

func (f *fakeStore) DeleteLink(creatorID int64, shortCode string) error { return nil }
func (f *fakeStore) SetLinkActive(creatorID int64, shortCode string, active bool) error {
	if !hasCode(f.created, shortCode) {
		return sql.ErrNoRows
	}
	f.activeCalls = append(f.activeCalls, activeCall{code: shortCode, active: active})
	return nil
}
func (f *fakeStore) UpdateLink(creatorID int64, shortCode, deviceRulesJSON, tagsJSON string) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	if !hasCode(f.created, shortCode) {
		return sql.ErrNoRows
	}
	f.updateCalls = append(f.updateCalls, updateCall{code: shortCode, deviceRulesJSON: deviceRulesJSON, tagsJSON: tagsJSON})
	return nil
}

func (f *fakeStore) SetFeaturedLink(creatorID int64, shortCode string, featured bool) error {
	if f.featureErr != nil {
		return f.featureErr
	}
	if !hasCode(f.created, shortCode) {
		return sql.ErrNoRows
	}
	f.featureCalls = append(f.featureCalls, featureCall{code: shortCode, featured: featured})
	return nil
}

// hasCode reports whether shortCode was previously created via CreateURL.
func hasCode(created []createCall, code string) bool {
	for _, c := range created {
		if c.code == code {
			return true
		}
	}
	return false
}

func (f *fakeStore) StoreAPIKey(creatorID int64, keyHash, label string) (int64, error) {
	f.storedKeys = append(f.storedKeys, storedKeyCall{keyHash: keyHash, label: label})
	return int64(len(f.storedKeys)) + 100, nil
}

func (f *fakeStore) ListAPIKeys(creatorID int64) ([]db.APIKey, error) {
	if f.keyList == nil {
		return []db.APIKey{}, nil
	}
	return f.keyList, nil
}

func (f *fakeStore) DeleteAPIKey(creatorID int64, keyID int64) error {
	owned := false
	for _, k := range f.keyList {
		if k.ID == keyID {
			owned = true
		}
	}
	if !owned {
		return sql.ErrNoRows
	}
	f.deletedKeyIDs = append(f.deletedKeyIDs, keyID)
	return nil
}

func (f *fakeStore) GetAPIKeyByHash(keyHash string) (db.APIKey, error) {
	if f.keyByHashErr != nil {
		return db.APIKey{}, f.keyByHashErr
	}
	if f.keyByHash.ID == 0 {
		return db.APIKey{}, sql.ErrNoRows
	}
	return f.keyByHash, nil
}

func (f *fakeStore) TouchAPIKeyLastUsed(keyID int64) error {
	f.touchedKeyIDs = append(f.touchedKeyIDs, keyID)
	return nil
}
func (f *fakeStore) ClicksByDay(creatorID int64) ([]db.DayCount, error) { return []db.DayCount{}, nil }

func (f *fakeStore) AnalyticsSummary(creatorID int64) (db.AnalyticsSummary, error) {
	if f.summaryErr != nil {
		return db.AnalyticsSummary{}, f.summaryErr
	}
	return f.summary, nil
}

// Analytics depth (2026-09-30): 6 new stubs. The results they return are set
// per test through fakeStore fields; the limit is honored exactly like the real
// store (returning cap+1 is the handler's mechanism for detecting truncation).
func (f *fakeStore) DeviceBreakdown(creatorID int64, from, to time.Time) ([]db.BreakdownItem, error) {
	if f.analyticsErr != nil {
		return nil, f.analyticsErr
	}
	if f.deviceItems == nil {
		return []db.BreakdownItem{}, nil
	}
	return f.deviceItems, nil
}

func (f *fakeStore) ReferrerBreakdown(creatorID int64, from, to time.Time) ([]db.BreakdownItem, error) {
	if f.analyticsErr != nil {
		return nil, f.analyticsErr
	}
	if f.referrerItems == nil {
		return []db.BreakdownItem{}, nil
	}
	return f.referrerItems, nil
}

func (f *fakeStore) ClicksDaily(creatorID int64, from, to time.Time) (map[string]int64, error) {
	if f.analyticsErr != nil {
		return nil, f.analyticsErr
	}
	if f.daily == nil {
		return map[string]int64{}, nil
	}
	return f.daily, nil
}

func (f *fakeStore) LinkExportStats(creatorID int64, from, to time.Time, limit int) ([]db.LinkExportRow, error) {
	if f.analyticsErr != nil {
		return nil, f.analyticsErr
	}
	out := f.linkStats
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (f *fakeStore) ListClicks(creatorID int64, from, to time.Time, limit int) ([]db.ClickRow, error) {
	if f.analyticsErr != nil {
		return nil, f.analyticsErr
	}
	out := f.clickRows
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (f *fakeStore) AnalyticsFreshness() (string, error) {
	if f.dataPer == "" {
		return "primary", nil
	}
	return f.dataPer, nil
}

// postJSON builds a POST request with the given JSON body.
func postJSON(path, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// putJSON builds a PUT request with the given JSON body.
func putJSON(path, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPut, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func newTestHandler(s db.ShardStore) *Handler {
	return &Handler{
		Store:  s,
		Logger: log.Default(),
	}
}

// authedReq mints a real session for creatorID and returns a request
// carrying it (so handlers exercising the auth gate pass it).
func authedReq(s *fakeStore, h *Handler, creatorID int64, path, body string) *http.Request {
	token, _ := h.Auth.Create(creatorID)
	req := postJSON(path, body)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: token})
	// Context injection mimics what middleware.RequireAuth does on the real
	// route; unit tests call handlers directly without the middleware layer.
	return middleware.WithCreator(req, creatorID)
}

// authedPut is authedReq but issues a PUT request.
func authedPut(s *fakeStore, h *Handler, creatorID int64, path, body string) *http.Request {
	token, _ := h.Auth.Create(creatorID)
	req := putJSON(path, body)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: token})
	return middleware.WithCreator(req, creatorID)
}

// authedMultipart builds a POST multipart/form-data request with one file
// field "avatar" plus the session cookie. content contains the raw file
// bytes; filename is what the "client" claims: the handler must NOT trust
// it (magic bytes decide), so tests can lie about the name deliberately.
func authedMultipart(t *testing.T, h *Handler, creatorID int64, field, filename string, content []byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile(field, filename)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := fw.Write(content); err != nil {
		t.Fatalf("fwrite: %v", err)
	}
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/profile/avatar", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if creatorID != 0 {
		token, _ := h.Auth.Create(creatorID)
		req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: token})
		req = middleware.WithCreator(req, creatorID)
	}
	return req
}
