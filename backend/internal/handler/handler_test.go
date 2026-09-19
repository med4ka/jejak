package handler

import (
	"bytes"
	"database/sql"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"jejak/internal/auth"
	"jejak/internal/middleware"
	"jejak/internal/db"
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
	// creatorLinks is returned by ListLinksByCreator(Primary) — lets tests
	// prove GET /api/links returns the caller's own links (scoping), not all.
	creatorLinks []db.Link
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

type profileCall struct {
	avatarURL string
}

type createCall struct {
	code        string
	originalURL string
	creatorID   *int64
	tagsJSON    string
}

func (f *fakeStore) GetURL(shortCode string) (string, error) {
	if u, ok := f.urls[shortCode]; ok {
		return u, nil
	}
	return "", sql.ErrNoRows
}

func (f *fakeStore) CreateURL(shortCode, originalURL string, creatorID *int64, tagsJSON string) error {
	f.created = append(f.created, createCall{code: shortCode, originalURL: originalURL, creatorID: creatorID, tagsJSON: tagsJSON})
	if f.urls == nil {
		f.urls = map[string]string{}
	}
	f.urls[shortCode] = originalURL
	return nil
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
func (f *fakeStore) LogClick(e	db.ClickEvent) error { return nil }
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
func (f *fakeStore) SetLinkActive(creatorID int64, shortCode string, active bool) error { return nil }
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

// setSessionCookie mints a real session for creatorID and returns a request
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
// bytes; filename is what the "client" claims — the handler must NOT trust
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
