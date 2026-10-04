package health

// Tests for the link health monitor (migration 17): classification rules
// (CheckURL) and the persistence/notification policy (CheckOne). CheckURL
// talks to local httptest servers only - no external network.

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"jejak/internal/db"
)

func TestCheckURLClassifiesStatuses(t *testing.T) {
	cases := []struct {
		name    string
		handler http.HandlerFunc
		want    string
	}{
		{"200 is healthy", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}, StatusHealthy},
		{"302 redirect target is healthy", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/elsewhere", http.StatusFound)
		}, StatusHealthy},
		{"404 is broken", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}, StatusBroken},
		{"500 is broken", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}, StatusBroken},
		// HEAD unsupported (405) must fall back to GET instead of a false
		// "broken": many static hosts only implement GET.
		{"405 HEAD falls back to GET 200", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodHead {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			w.WriteHeader(http.StatusOK)
		}, StatusHealthy},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(tc.handler)
			defer srv.Close()
			got := CheckURL(context.Background(), srv.URL)
			if got.Status != tc.want {
				t.Fatalf("CheckURL = %s (%d), want %s", got.Status, got.Code, tc.want)
			}
		})
	}
}

func TestCheckURLUnreachableIsTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close() // nothing listens anymore: connection refused -> timeout class
	got := CheckURL(context.Background(), url)
	if got.Status != StatusTimeout {
		t.Fatalf("CheckURL = %s, want %s (err=%q)", got.Status, StatusTimeout, got.Err)
	}
}

// fakeStore implements the small Store interface with recording.
type fakeStore struct {
	mu      sync.Mutex
	links   map[string]db.Link
	updates []updateRec
	notifs  []notifRec
	due     []db.Link
	// updateErr, when set, is returned by UpdateLinkHealth.
	updateErr error
}

type updateRec struct {
	code       string
	status     string
	checkedAt  time.Time
	notifiedAt time.Time
}

type notifRec struct {
	creatorID int64
	typ       string
	code      string
	message   string
}

func newFakeStore(links ...db.Link) *fakeStore {
	m := map[string]db.Link{}
	for _, l := range links {
		m[l.ShortCode] = l
	}
	return &fakeStore{links: m}
}

func (f *fakeStore) GetLink(shortCode string) (db.Link, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	l, ok := f.links[shortCode]
	if !ok {
		return db.Link{}, errors.New("no rows")
	}
	return l, nil
}

func (f *fakeStore) UpdateLinkHealth(shortCode, status string, checkedAt, notifiedAt time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updateErr != nil {
		return f.updateErr
	}
	f.updates = append(f.updates, updateRec{code: shortCode, status: status, checkedAt: checkedAt, notifiedAt: notifiedAt})
	return nil
}

func (f *fakeStore) CreateNotification(creatorID int64, typ, shortCode, message string) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.notifs = append(f.notifs, notifRec{creatorID: creatorID, typ: typ, code: shortCode, message: message})
	return int64(len(f.notifs)), nil
}

func (f *fakeStore) ListLinksHealthDue(limit int) ([]db.Link, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := f.due
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// fakeCache records Delete calls.
type fakeCache struct {
	mu      sync.Mutex
	deleted []string
}

func (c *fakeCache) Get(shortCode string) (string, bool)      { return "", false }
func (c *fakeCache) Set(shortCode, v string, ttl int64) error { return nil }
func (c *fakeCache) Delete(shortCode string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deleted = append(c.deleted, shortCode)
	return nil
}

// TestCheckOneNotifiesOncePerBreakage: the first broken probe with an owner
// creates exactly ONE notification and stamps health_notified_at; a second
// broken probe (marker already set) does NOT notify again.
func TestCheckOneNotifiesOncePerBreakage(t *testing.T) {
	creator := int64(7)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	store := newFakeStore(db.Link{
		ShortCode: "brk001", OriginalURL: srv.URL, CreatorID: &creator,
		HealthStatus: StatusUnknown,
	})

	cache := &fakeCache{}
	c := &Checker{Store: store, Cache: cache, Spacing: time.Millisecond}

	status, err := c.CheckOne(context.Background(), "brk001")
	if err != nil {
		t.Fatalf("CheckOne: %v", err)
	}
	if status != StatusBroken {
		t.Fatalf("status = %s, want %s", status, StatusBroken)
	}
	if len(store.notifs) != 1 {
		t.Fatalf("notifications = %d, want 1 (first breakage notifies)", len(store.notifs))
	}
	if store.notifs[0].creatorID != 7 || store.notifs[0].typ != "link_broken" || store.notifs[0].code != "brk001" {
		t.Fatalf("notification = %+v, want creator 7 / link_broken / brk001", store.notifs[0])
	}
	upd := store.updates[len(store.updates)-1]
	if upd.notifiedAt.IsZero() {
		t.Fatal("health_notified_at must be stamped on the first broken run")
	}
	if len(cache.deleted) == 0 {
		t.Fatal("status change must evict the redirect cache")
	}

	// Simulate the stored marker (the real DB write did it) and run again:
	// still broken -> marker already present -> NO second notification.
	notified := upd.notifiedAt
	l := store.links["brk001"]
	l.HealthNotifiedAt = &notified
	l.HealthStatus = StatusBroken
	store.links["brk001"] = l
	if _, err := c.CheckOne(context.Background(), "brk001"); err != nil {
		t.Fatalf("second CheckOne: %v", err)
	}
	if len(store.notifs) != 1 {
		t.Fatalf("notifications = %d, want 1 (no spam while it stays broken)", len(store.notifs))
	}
}

// TestCheckOneHealthyResetsMarker: a healthy result clears health_notified_at
// (zero time -> NULL) so the NEXT breakage notifies again.
func TestCheckOneHealthyResetsMarker(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	creator := int64(3)
	past := time.Now().UTC().Add(-time.Hour)
	store := newFakeStore(db.Link{
		ShortCode: "ok001", OriginalURL: srv.URL, CreatorID: &creator,
		HealthStatus: StatusBroken, HealthNotifiedAt: &past,
	})
	c := &Checker{Store: store, Spacing: time.Millisecond}
	if _, err := c.CheckOne(context.Background(), "ok001"); err != nil {
		t.Fatalf("CheckOne: %v", err)
	}
	upd := store.updates[len(store.updates)-1]
	if upd.status != StatusHealthy {
		t.Fatalf("status = %s, want healthy", upd.status)
	}
	if !upd.notifiedAt.IsZero() {
		t.Fatalf("notifiedAt = %v, want zero (reset after healing)", upd.notifiedAt)
	}
	if len(store.notifs) != 0 {
		t.Fatalf("healthy run must not notify, got %d", len(store.notifs))
	}
}

// TestCheckOneAnonymousBrokenNoNotification: an ownerless link has no
// recipient - no notification, and the marker stays unset so a future
// ClaimLinks owner gets notified on the next check.
func TestCheckOneAnonymousBrokenNoNotification(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	store := newFakeStore(db.Link{
		ShortCode: "anon01", OriginalURL: srv.URL, // CreatorID nil
		HealthStatus: StatusUnknown,
	})
	c := &Checker{Store: store, Spacing: time.Millisecond}
	if _, err := c.CheckOne(context.Background(), "anon01"); err != nil {
		t.Fatalf("CheckOne: %v", err)
	}
	if len(store.notifs) != 0 {
		t.Fatalf("anonymous link must not notify, got %d", len(store.notifs))
	}
	upd := store.updates[len(store.updates)-1]
	if !upd.notifiedAt.IsZero() {
		t.Fatal("marker must stay unset without an owner (notify after claim)")
	}
}

// TestRunBatchChecksDueLinks: every due link is probed, one bad row is
// skipped without aborting the run, and spacing keeps them sequential.
func TestRunBatchChecksDueLinks(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ok.Close()
	creator := int64(1)
	store := newFakeStore(
		db.Link{ShortCode: "due001", OriginalURL: ok.URL, CreatorID: &creator, HealthStatus: StatusUnknown},
		db.Link{ShortCode: "due002", OriginalURL: "http://127.0.0.1:1", CreatorID: &creator, HealthStatus: StatusUnknown}, // dead port -> timeout result, still a completed check
	)
	store.due = []db.Link{{ShortCode: "due001"}, {ShortCode: "due002"}, {ShortCode: "ghost"}}
	// "ghost" has no row: CheckOne errors and must be skipped, not fatal.
	c := &Checker{Store: store, Spacing: time.Millisecond}
	done, err := c.RunBatch(context.Background())
	if err != nil {
		t.Fatalf("RunBatch: %v", err)
	}
	if done != 2 {
		t.Fatalf("done = %d, want 2 (ghost row skipped)", done)
	}
	if len(store.updates) != 2 {
		t.Fatalf("updates = %d, want 2", len(store.updates))
	}
}
