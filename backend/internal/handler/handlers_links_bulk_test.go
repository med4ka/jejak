package handler

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"jejak/internal/db"
	"jejak/internal/middleware"
)

// bulkURLs builds an import body with n unique valid URLs.
func bulkURLs(n int) string {
	items := make([]string, n)
	for i := range n {
		items[i] = fmt.Sprintf("https://example.com/p%d", i)
	}
	b, _ := json.Marshal(map[string]any{"urls": items})
	return string(b)
}

// TestHandleBulkShorten401: no session → 401, store untouched (in main the
// rate limit runs after auth; this unit pins down the handler's defensive
// check).
func TestHandleBulkShorten401(t *testing.T) {
	s := &fakeStore{}
	h := newTestHandler(s)

	req := postJSON("/api/links/bulk", bulkURLs(10))
	rr := httptest.NewRecorder()
	h.HandleBulkShorten(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
	if len(s.bulkCalls) != 0 || len(s.created) != 0 {
		t.Fatal("anonymous bulk must not touch the store")
	}
}

// TestHandleBulkShortenCountBounds: 9 → 400, 101 → 400 (spec is 10-100;
// out-of-range input is REJECTED, never silently truncated), while 10 and 100
// pass the count validation (the 100 case is covered by the success test
// below through counting).
func TestHandleBulkShortenCountBounds(t *testing.T) {
	for _, tt := range []struct {
		n    int
		want int
	}{{9, 400}, {101, 400}} {
		s := &fakeStore{}
		h := newTestHandler(s)
		req := middleware.WithCreator(postJSON("/api/links/bulk", bulkURLs(tt.n)), 7)
		rr := httptest.NewRecorder()
		h.HandleBulkShorten(rr, req)
		if rr.Code != tt.want {
			t.Errorf("n=%d status = %d, want %d (body %q)", tt.n, rr.Code, tt.want, rr.Body.String())
		}
		if len(s.bulkCalls) != 0 {
			t.Errorf("n=%d store touched despite 400", tt.n)
		}
	}
}

// TestHandleBulkShortenBodyTooLarge: a body > 100KB is rejected with 400 by
// the decoder (MaxBytesReader) before validation: the decodeJSON pattern,
// with the tighter cap.
func TestHandleBulkShortenBodyTooLarge(t *testing.T) {
	s := &fakeStore{}
	h := newTestHandler(s)

	huge := strings.Repeat("a", 40000)
	body := fmt.Sprintf(`{"urls":["https://example.com/%s","https://example.com/%s","https://example.com/%s"]}`,
		huge, huge, huge)
	req := middleware.WithCreator(postJSON("/api/links/bulk", body), 7)
	rr := httptest.NewRecorder()
	h.HandleBulkShorten(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("oversized body status = %d, want 400", rr.Code)
	}
	if len(s.bulkCalls) != 0 {
		t.Fatal("store touched despite body rejection")
	}
}

// TestHandleBulkShortenAllInvalid400: every row invalid → 400 plus the reason
// per row (not a 201 containing zero links).
func TestHandleBulkShortenAllInvalid400(t *testing.T) {
	s := &fakeStore{}
	h := newTestHandler(s)

	items := make([]string, 10)
	for i := range items {
		items[i] = "javascript:alert(1)"
	}
	b, _ := json.Marshal(map[string]any{"urls": items})
	req := middleware.WithCreator(postJSON("/api/links/bulk", string(b)), 7)
	rr := httptest.NewRecorder()
	h.HandleBulkShorten(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body %q)", rr.Code, rr.Body.String())
	}
	var resp struct {
		Errors []bulkError `json:"errors"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Errors) != 10 {
		t.Fatalf("errors = %d, want 10", len(resp.Errors))
	}
	for i, e := range resp.Errors {
		if e.Line != i+1 || e.Error != "invalid_url" {
			t.Errorf("errors[%d] = %+v, want line %d invalid_url", i, e, i+1)
		}
	}
	if len(s.bulkCalls) != 0 {
		t.Fatal("store touched despite no valid urls")
	}
}

// TestHandleBulkShortenMixedBatch: the core of the feature: 10 mixed rows
// (valid, in-request duplicate, invalid) → 201 partial success, failures do
// NOT cancel the rest, and the {created, errors, summary} shape is correct.
func TestHandleBulkShortenMixedBatch(t *testing.T) {
	s := &fakeStore{}
	h := newTestHandler(s)

	body := `{"urls":[
		"https://example.com/a",
		"https://example.com/b",
		"https://example.com/a",
		"not-a-url",
		"https://example.com/c",
		"https://example.com/d",
		"https://example.com/e",
		"https://example.com/f",
		"https://example.com/g",
		"https://example.com/h"
	]}`
	req := middleware.WithCreator(postJSON("/api/links/bulk", body), 7)
	req.Host = "example.com"
	rr := httptest.NewRecorder()
	h.HandleBulkShorten(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body %q)", rr.Code, rr.Body.String())
	}
	var resp struct {
		Created []struct {
			ShortCode   string `json:"short_code"`
			ShortURL    string `json:"short_url"`
			OriginalURL string `json:"original_url"`
		} `json:"created"`
		Errors  []bulkError `json:"errors"`
		Summary struct {
			Total   int `json:"total"`
			Created int `json:"created"`
			Failed  int `json:"failed"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Summary.Total != 10 || resp.Summary.Created != 8 || resp.Summary.Failed != 2 {
		t.Fatalf("summary = %+v, want total 10 created 8 failed 2", resp.Summary)
	}
	if len(resp.Created) != 8 || len(resp.Errors) != 2 {
		t.Fatalf("created=%d errors=%d, want 8/2", len(resp.Created), len(resp.Errors))
	}
	// Failures: row 3 (in-request duplicate), row 4 (invalid).
	if resp.Errors[0].Line != 3 || resp.Errors[0].Error != "duplicate_in_request" {
		t.Errorf("errors[0] = %+v, want line 3 duplicate_in_request", resp.Errors[0])
	}
	if resp.Errors[1].Line != 4 || resp.Errors[1].Error != "invalid_url" {
		t.Errorf("errors[1] = %+v, want line 4 invalid_url", resp.Errors[1])
	}
	// Shape of created: 6-char code + absolute short_url + correct original.
	for _, c := range resp.Created {
		if len(c.ShortCode) != 6 {
			t.Errorf("short_code %q, want 6 chars", c.ShortCode)
		}
		if c.ShortURL != "http://example.com/r/"+c.ShortCode {
			t.Errorf("short_url = %q, want http://example.com/r/%s", c.ShortURL, c.ShortCode)
		}
	}
	// Store: 8 inserts recorded, all owned by creator 7.
	if len(s.created) != 8 {
		t.Fatalf("store created = %d, want 8", len(s.created))
	}
	for _, c := range s.created {
		if c.creatorID == nil || *c.creatorID != 7 {
			t.Fatalf("created %q owner = %v, want 7", c.code, c.creatorID)
		}
	}
}

// TestHandleBulkShortenDBConflict: ON CONFLICT DO NOTHING in the store returns
// code "" which maps to a per-row "duplicate" error without aborting the batch
// (simulating a collision with a short_code already present in the DB).
func TestHandleBulkShortenDBConflict(t *testing.T) {
	s := &fakeStore{bulkConflictAll: true}
	h := newTestHandler(s)

	req := middleware.WithCreator(postJSON("/api/links/bulk", bulkURLs(10)), 7)
	rr := httptest.NewRecorder()
	h.HandleBulkShorten(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (conflict bukan error HTTP)", rr.Code)
	}
	var resp struct {
		Created []map[string]string `json:"created"`
		Errors  []bulkError         `json:"errors"`
		Summary struct {
			Failed int `json:"failed"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Created) != 0 {
		t.Fatalf("created = %d, want 0", len(resp.Created))
	}
	if len(resp.Errors) != 10 || resp.Summary.Failed != 10 {
		t.Fatalf("errors=%d failed=%d, want 10/10", len(resp.Errors), resp.Summary.Failed)
	}
	for _, e := range resp.Errors {
		if e.Error != "duplicate" {
			t.Errorf("error = %q, want duplicate", e.Error)
		}
	}
}

// TestHandleBulkShortenTag: the optional tag is normalized (trim+lowercase)
// and applied to every imported link.
func TestHandleBulkShortenTag(t *testing.T) {
	s := &fakeStore{}
	h := newTestHandler(s)

	req := middleware.WithCreator(postJSON("/api/links/bulk", `{"urls":[`+urlsList(10)+`],"tag":"  Kerja "}`), 7)
	rr := httptest.NewRecorder()
	h.HandleBulkShorten(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body %q)", rr.Code, rr.Body.String())
	}
	if len(s.created) != 10 {
		t.Fatalf("created = %d, want 10", len(s.created))
	}
	for _, c := range s.created {
		if c.tagsJSON != `["kerja"]` {
			t.Fatalf("tagsJSON = %s, want [\"kerja\"]", c.tagsJSON)
		}
	}
}

// TestHandleBulkShortenInvalidTag400: a tag > 20 chars is rejected (server-side
// validation: client input is never trusted).
func TestHandleBulkShortenInvalidTag400(t *testing.T) {
	s := &fakeStore{}
	h := newTestHandler(s)

	req := middleware.WithCreator(postJSON("/api/links/bulk",
		`{"urls":[`+urlsList(10)+`],"tag":"`+strings.Repeat("x", 21)+`"}`), 7)
	rr := httptest.NewRecorder()
	h.HandleBulkShorten(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	if len(s.bulkCalls) != 0 {
		t.Fatal("store touched despite invalid tag")
	}
}

// urlsList builds a JSON string list "url1","url2",... of length n.
func urlsList(n int) string {
	parts := make([]string, n)
	for i := range n {
		parts[i] = fmt.Sprintf(`"https://example.com/q%d"`, i)
	}
	return strings.Join(parts, ",")
}

// TestHandleBulkQR401: no session → 401.
func TestHandleBulkQR401(t *testing.T) {
	s := &fakeStore{creatorLinks: []db.Link{{ShortCode: "abc123", OriginalURL: "https://example.com"}}}
	h := newTestHandler(s)
	req := httptest.NewRequest(http.MethodGet, "/api/links/qr-bulk.zip", nil)
	rr := httptest.NewRecorder()
	h.HandleBulkQR(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

// TestHandleBulkQRSuccess: a valid ZIP (magic PK) containing one PNG per link,
// attachment header, and a tag-safe file name.
func TestHandleBulkQRSuccess(t *testing.T) {
	s := &fakeStore{creatorLinks: []db.Link{
		{ShortCode: "aaa111", OriginalURL: "https://a.example.com", Tags: []string{"kerja"}},
		{ShortCode: "bbb222", OriginalURL: "https://b.example.com", Tags: []string{"liburan"}},
	}}
	h := newTestHandler(s)

	req := middleware.WithCreator(httptest.NewRequest(http.MethodGet, "/api/links/qr-bulk.zip?tag=kerja", nil), 7)
	req.Host = "example.com"
	rr := httptest.NewRecorder()
	h.HandleBulkQR(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/zip" {
		t.Errorf("Content-Type = %q, want application/zip", ct)
	}
	if cd := rr.Header().Get("Content-Disposition"); !strings.Contains(cd, "jejak-qr-kerja.zip") {
		t.Errorf("Content-Disposition = %q, want jejak-qr-kerja.zip", cd)
	}
	if !bytes.HasPrefix(rr.Body.Bytes(), []byte("PK")) {
		t.Fatalf("body missing ZIP magic: % x", rr.Body.Bytes()[:4])
	}
	zr, err := zip.NewReader(bytes.NewReader(rr.Body.Bytes()), int64(rr.Body.Len()))
	if err != nil {
		t.Fatalf("zip open: %v", err)
	}
	if len(zr.File) != 1 || zr.File[0].Name != "aaa111.png" {
		t.Fatalf("zip entries = %v, want [aaa111.png] (filter tag bekerja)", zr.File)
	}
}

// TestHandleBulkQRWithoutTag: no filter → every link is included (and the
// file name carries no tag).
func TestHandleBulkQRWithoutTag(t *testing.T) {
	s := &fakeStore{creatorLinks: []db.Link{
		{ShortCode: "ccc333", OriginalURL: "https://c.example.com"},
		{ShortCode: "ddd444", OriginalURL: "https://d.example.com"},
	}}
	h := newTestHandler(s)

	req := middleware.WithCreator(httptest.NewRequest(http.MethodGet, "/api/links/qr-bulk.zip", nil), 7)
	req.Host = "example.com"
	rr := httptest.NewRecorder()
	h.HandleBulkQR(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	zr, err := zip.NewReader(bytes.NewReader(rr.Body.Bytes()), int64(rr.Body.Len()))
	if err != nil {
		t.Fatalf("zip open: %v", err)
	}
	if len(zr.File) != 2 {
		t.Fatalf("zip entries = %d, want 2", len(zr.File))
	}
	if cd := rr.Header().Get("Content-Disposition"); !strings.Contains(cd, `filename="jejak-qr.zip"`) {
		t.Errorf("Content-Disposition = %q, want jejak-qr.zip", cd)
	}
}

// TestHandleBulkQREmpty400: 0 matches → 400 (not an empty ZIP).
func TestHandleBulkQREmpty400(t *testing.T) {
	s := &fakeStore{creatorLinks: []db.Link{
		{ShortCode: "eee555", OriginalURL: "https://e.example.com", Tags: []string{"kerja"}},
	}}
	h := newTestHandler(s)
	req := middleware.WithCreator(httptest.NewRequest(http.MethodGet, "/api/links/qr-bulk.zip?tag=kantor", nil), 7)
	rr := httptest.NewRecorder()
	h.HandleBulkQR(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

// TestHandleBulkQRTooMany400: > 200 links → 400 with a hint to use ?tag=.
func TestHandleBulkQRTooMany400(t *testing.T) {
	links := make([]db.Link, bulkQRMaxLinks+1)
	for i := range links {
		links[i] = db.Link{ShortCode: fmt.Sprintf("f%05d", i), OriginalURL: "https://example.com"}
	}
	s := &fakeStore{creatorLinks: links}
	h := newTestHandler(s)
	req := middleware.WithCreator(httptest.NewRequest(http.MethodGet, "/api/links/qr-bulk.zip", nil), 7)
	rr := httptest.NewRecorder()
	h.HandleBulkQR(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body %q)", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "tag") {
		t.Errorf("body %q should mention ?tag= filter", rr.Body.String())
	}
}

// TestHandleBulkQRInvalidTag400: an out-of-limit tag is rejected before the
// store is queried.
func TestHandleBulkQRInvalidTag400(t *testing.T) {
	s := &fakeStore{}
	h := newTestHandler(s)
	req := middleware.WithCreator(httptest.NewRequest(http.MethodGet,
		"/api/links/qr-bulk.zip?tag="+strings.Repeat("z", 21), nil), 7)
	rr := httptest.NewRecorder()
	h.HandleBulkQR(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

// TestRedirectTTLClamp is a pure unit test of redirectTTL: default 300,
// clamped to the remaining expiry when that is < 300, 1 second once already
// past (defensive), and never above 300.
func TestRedirectTTLClamp(t *testing.T) {
	if got := redirectTTL(nil); got != 300 {
		t.Errorf("redirectTTL(nil) = %d, want 300", got)
	}
	far := time.Now().UTC().Add(48 * time.Hour)
	if got := redirectTTL(&far); got != 300 {
		t.Errorf("redirectTTL(48h) = %d, want 300", got)
	}
	near := time.Now().UTC().Add(120 * time.Second)
	if got := redirectTTL(&near); got < 100 || got > 121 {
		t.Errorf("redirectTTL(+120s) = %d, want ~120", got)
	}
	past := time.Now().UTC().Add(-time.Minute)
	if got := redirectTTL(&past); got != 1 {
		t.Errorf("redirectTTL(past) = %d, want 1", got)
	}
}
