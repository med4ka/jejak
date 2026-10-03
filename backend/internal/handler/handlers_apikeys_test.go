package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"jejak/internal/auth"
	"jejak/internal/db"
	"jejak/internal/ratelimit"
)

func TestHandleGenerateAPIKey(t *testing.T) {
	t.Run("anonymous-401", func(t *testing.T) {
		s := &fakeStore{}
		h := newTestHandler(s)
		h.Auth = auth.NewMemoryStore()
		rr := httptest.NewRecorder()
		h.HandleGenerateAPIKey(rr, postJSON("/api/keys", `{}`))
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rr.Code)
		}
	})

	t.Run("authed-returns-plaintext-once", func(t *testing.T) {
		s := &fakeStore{}
		h := newTestHandler(s)
		h.Auth = auth.NewMemoryStore()
		req := authedReq(s, h, 42, "/api/keys", `{"label":"ci token"}`)
		rr := httptest.NewRecorder()
		h.HandleGenerateAPIKey(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("status = %d (body %q), want 201", rr.Code, rr.Body.String())
		}
		var resp struct {
			ID  int64  `json:"id"`
			Key string `json:"key"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("bad json: %v", err)
		}
		if !strings.HasPrefix(resp.Key, "jjk_") || len(resp.Key) != len("jjk_")+32 {
			t.Errorf("key format wrong: %q (want jjk_ + 32 hex)", resp.Key)
		}
		if resp.ID != 101 {
			t.Errorf("id = %d, want 101 (first fake store key)", resp.ID)
		}
		if len(s.storedKeys) != 1 {
			t.Fatalf("StoreAPIKey called %d times, want 1", len(s.storedKeys))
		}
		got := s.storedKeys[0]
		// Only the hash is stored, never the plaintext.
		if got.keyHash == resp.Key {
			t.Error("plaintext key reached the store layer")
		}
		if got.keyHash != hashAPIKey(resp.Key) {
			t.Errorf("store hash %q != expected hash of returned key", got.keyHash)
		}
		if got.label != "ci token" {
			t.Errorf("label stored = %q, want ci token", got.label)
		}
	})

	t.Run("bad-json-400", func(t *testing.T) {
		s := &fakeStore{}
		h := newTestHandler(s)
		h.Auth = auth.NewMemoryStore()
		req := authedReq(s, h, 42, "/api/keys", `{nope`)
		rr := httptest.NewRecorder()
		h.HandleGenerateAPIKey(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
		if len(s.storedKeys) != 0 {
			t.Fatalf("key stored on bad body: %v", s.storedKeys)
		}
	})

	t.Run("label-too-long-400", func(t *testing.T) {
		s := &fakeStore{}
		h := newTestHandler(s)
		h.Auth = auth.NewMemoryStore()
		long := strings.Repeat("x", 101)
		req := authedReq(s, h, 42, "/api/keys", `{"label":"`+long+`"}`)
		rr := httptest.NewRecorder()
		h.HandleGenerateAPIKey(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rr.Code)
		}
	})
}

func TestHandleListAPIKeys(t *testing.T) {
	t.Run("anonymous-401", func(t *testing.T) {
		h := newTestHandler(&fakeStore{})
		rr := httptest.NewRecorder()
		h.HandleListAPIKeys(rr, httptest.NewRequest(http.MethodGet, "/api/keys", nil))
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rr.Code)
		}
	})

	t.Run("authed-lists-metadata-only", func(t *testing.T) {
		s := &fakeStore{keyList: []db.APIKey{
			{ID: 3, CreatorID: 42, Label: sql.NullString{String: "ci", Valid: true}, CreatedAt: time.Now(), LastUsedAt: sql.NullTime{Time: time.Now(), Valid: true}},
			{ID: 4, CreatorID: 42, CreatedAt: time.Now()},
		}}
		h := newTestHandler(s)
		h.Auth = auth.NewMemoryStore()
		req := authedReq(s, h, 42, "/api/keys", "")
		req.Method = http.MethodGet
		rr := httptest.NewRecorder()
		h.HandleListAPIKeys(rr, req)

		var resp []map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("response is not a JSON array: %v (%q)", err, rr.Body.String())
		}
		if len(resp) != 2 {
			t.Fatalf("got %d keys, want 2", len(resp))
		}
		raw := rr.Body.String()
		if strings.Contains(raw, "key_hash") || strings.Contains(raw, `"key"`) {
			t.Errorf("response leaked key material: %s", raw)
		}
		if resp[0]["label"] != "ci" || resp[0]["last_used_at"] == nil {
			t.Errorf("key 0 shape wrong: %v", resp[0])
		}
		if resp[1]["label"] != "" || resp[1]["last_used_at"] != nil {
			t.Errorf("key 1 shape wrong (expected nulls): %v", resp[1])
		}
	})
}

func TestHandleDeleteAPIKey(t *testing.T) {
	owned := db.APIKey{ID: 10, CreatorID: 42}
	tests := []struct {
		name     string
		keyList  []db.APIKey
		id       string
		anon     bool
		wantCode int
	}{
		{"anonymous-401", nil, "10", true, http.StatusUnauthorized},
		{"invalid-id-400", []db.APIKey{owned}, "abc", false, http.StatusBadRequest},
		{"owns-key-200", []db.APIKey{owned}, "10", false, http.StatusOK},
		{"not-owner-404", []db.APIKey{}, "999", false, http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &fakeStore{keyList: tt.keyList}
			h := newTestHandler(s)
			h.Auth = auth.NewMemoryStore()
			var req *http.Request
			if tt.anon {
				req = httptest.NewRequest(http.MethodDelete, "/api/keys/"+tt.id, nil)
			} else {
				req = authedReq(s, h, 42, "/api/keys/"+tt.id, "")
				req.Method = http.MethodDelete
			}
			// The handler is invoked directly (no ServeMux), so path values
			// are not populated automatically as they are through a route
			// pattern.
			req.SetPathValue("id", tt.id)
			rr := httptest.NewRecorder()
			h.HandleDeleteAPIKey(rr, req)
			if rr.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d", rr.Code, tt.wantCode)
			}
		})
	}
}

func TestHandleV1Shorten(t *testing.T) {
	t.Run("no-key-401", func(t *testing.T) {
		h := newTestHandler(&fakeStore{})
		rr := httptest.NewRecorder()
		h.HandleV1Shorten(rr, postJSON("/api/v1/shorten", `{"url":"https://x.com/"}`))
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rr.Code)
		}
	})

	t.Run("wrong-prefix-401", func(t *testing.T) {
		h := newTestHandler(&fakeStore{})
		req := postJSON("/api/v1/shorten", `{"url":"https://x.com/"}`)
		req.Header.Set("Authorization", "Bearer 0123456789abcdef0123456789abcdef")
		rr := httptest.NewRecorder()
		h.HandleV1Shorten(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401 (non-jjk_ prefix)", rr.Code)
		}
	})

	t.Run("unknown-key-401", func(t *testing.T) {
		s := &fakeStore{}
		h := newTestHandler(s)
		h.Auth = auth.NewMemoryStore()
		req := postJSON("/api/v1/shorten", `{"url":"https://x.com/"}`)
		req.Header.Set("Authorization", "Bearer jjk_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		rr := httptest.NewRecorder()
		h.HandleV1Shorten(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401 (unknown key)", rr.Code)
		}
	})

	t.Run("valid-key-owns-link-and-touches", func(t *testing.T) {
		s := &fakeStore{keyByHash: db.APIKey{ID: 7, CreatorID: 42}}
		h := newTestHandler(s)
		h.Auth = auth.NewMemoryStore()
		req := postJSON("/api/v1/shorten", `{"url":"https://api-created.example/","tags":["via-api"]}`)
		req.Header.Set("Authorization", "Bearer jjk_deadbeefdeadbeefdeadbeefdeadbeef")
		req.Host = "api.example.com"
		rr := httptest.NewRecorder()
		h.HandleV1Shorten(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("status = %d (body %q), want 201", rr.Code, rr.Body.String())
		}
		if len(s.created) != 1 {
			t.Fatalf("CreateURL called %d times, want 1", len(s.created))
		}
		created := s.created[0]
		if created.creatorID == nil || *created.creatorID != 42 {
			t.Errorf("link not owned by API key creator: %v", created.creatorID)
		}
		if created.originalURL != "https://api-created.example/" {
			t.Errorf("original_url = %q", created.originalURL)
		}
		if !strings.Contains(rr.Body.String(), "http://api.example.com/r/") {
			t.Errorf("body should be a short URL on r.Host: %q", rr.Body.String())
		}
		if len(s.touchedKeyIDs) != 1 || s.touchedKeyIDs[0] != 7 {
			t.Errorf("last_used not touched for key 7: %v", s.touchedKeyIDs)
		}
	})

	t.Run("rate-limited-429", func(t *testing.T) {
		s := &fakeStore{keyByHash: db.APIKey{ID: 7, CreatorID: 42}}
		h := newTestHandler(s)
		h.Auth = auth.NewMemoryStore()
		h.APILimiter = ratelimit.NewLimiterWithMax(1)

		// First request consumes the single quota.
		r1 := postJSON("/api/v1/shorten", `{"url":"https://x.com/"}`)
		r1.Header.Set("Authorization", "Bearer jjk_deadbeefdeadbeefdeadbeefdeadbeef")
		h.HandleV1Shorten(httptest.NewRecorder(), r1)

		// Second within the window must be throttled.
		r2 := postJSON("/api/v1/shorten", `{"url":"https://y.com/"}`)
		r2.Header.Set("Authorization", "Bearer jjk_deadbeefdeadbeefdeadbeefdeadbeef")
		rr := httptest.NewRecorder()
		h.HandleV1Shorten(rr, r2)
		if rr.Code != http.StatusTooManyRequests {
			t.Fatalf("status = %d, want 429", rr.Code)
		}
		if rr.Header().Get("Retry-After") == "" {
			t.Error("429 missing Retry-After header")
		}
	})
}
