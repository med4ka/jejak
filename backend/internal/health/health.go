// Package health checks link destinations for the link health monitor
// (migration 17, 2026-10-04). The worker runs a batch every 6 hours; the
// dashboard's "check now" trigger runs a single check through the same
// Checker, so both paths share one classification rule, one persistence
// write, and one notification policy.
//
// Politeness budget (spec): <= 50 requests/minute in total. The batch times
// 50 links with 1.2s spacing (50 x 1.2s = 60s per run), and the manual
// trigger has its own 10/min per-IP limiter in the handler - the two
// together stay at or just above the budget, which is acceptable for a
// single-instance learning deployment (documented deviation: no global
// distributed limiter).
package health

import (
	"context"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"jejak/internal/cache"
	"jejak/internal/db"
)

// Statuses stored in urls.health_status (spec: unknown/healthy/broken/
// timeout; "unknown" is also the column DEFAULT for never-checked links).
const (
	StatusUnknown = "unknown"
	StatusHealthy = "healthy"
	StatusBroken  = "broken"
	StatusTimeout = "timeout"
)

const (
	// UserAgent identifies us to destination owners reading their logs -
	// never a browser UA: a health checker must be recognizable as one.
	UserAgent = "Jejak-HealthCheck/1.0"
	// Timeout bounds one request (spec: 5 seconds). DNS hangs, slow TLS and
	// black-hole firewalls all become "timeout" instead of a stuck batch.
	Timeout = 5 * time.Second
	// MaxRedirects caps followed redirects (spec: 3). Beyond that the last
	// response is used as-is (ErrUseLastResponse): a redirect loop target is
	// still REACHABLE, which is what health means here.
	MaxRedirects = 3
	// BatchLimit is the per-run size (spec: 50 links) and BatchInterval the
	// worker cadence (spec: every 6 hours).
	BatchLimit    = 50
	BatchInterval = 6 * time.Hour
	// defaultSpacing keeps the batch within 50 requests/minute
	// (60s / 50 = 1.2s). Tests shrink it.
	defaultSpacing = 1200 * time.Millisecond
)

// Result is one classification: Status plus the HTTP code (0 when the
// request never completed) and a short error text for logs/tests.
type Result struct {
	Status string
	Code   int
	Err    string
}

// CheckURL probes rawURL with HEAD (5s timeout, at most 3 redirects, our
// User-Agent) and classifies:
//
//	2xx/3xx -> healthy, 4xx/5xx -> broken, transport error -> timeout.
//
// HEAD is the spec's method: the body never matters and a tiny request is
// polite to the destination. Servers that answer HEAD with 405/501 (many
// static hosts and some frameworks only implement GET) are retried with GET
// once - classifying those as broken would be a false alarm. The GET body is
// closed unread: only the status line matters.
//
// ctx cancels the whole probe (worker shutdown); the per-request timeout
// comes from the client, not from the caller.
func CheckURL(ctx context.Context, rawURL string) Result {
	client := &http.Client{
		Timeout: Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= MaxRedirects {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	do := func(method string) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, method, rawURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", UserAgent)
		return client.Do(req)
	}

	resp, err := do(http.MethodHead)
	if err != nil {
		return Result{Status: StatusTimeout, Err: err.Error()}
	}
	if resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusNotImplemented {
		resp.Body.Close()
		resp, err = do(http.MethodGet)
		if err != nil {
			return Result{Status: StatusTimeout, Err: err.Error()}
		}
	}
	defer resp.Body.Close()
	// Drain a little so the connection can be reused; nothing is kept.
	_, _ = io.CopyN(io.Discard, resp.Body, 4096)

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 400:
		return Result{Status: StatusHealthy, Code: resp.StatusCode}
	default:
		return Result{Status: StatusBroken, Code: resp.StatusCode}
	}
}

// Store is the subset of db.ShardStore the checker needs. A small interface
// keeps the package testable without a full ShardStore fake; the real
// db.SingleStore / shardStore satisfy it structurally.
type Store interface {
	GetLink(shortCode string) (db.Link, error)
	UpdateLinkHealth(shortCode, status string, checkedAt, notifiedAt time.Time) error
	CreateNotification(creatorID int64, typ, shortCode, message string) (int64, error)
	ListLinksHealthDue(limit int) ([]db.Link, error)
}

// Checker runs checks against a Store (+optional Cache eviction +optional
// Logger). Spacing is the delay between two checks of one batch
// (defaultSpacing when zero). Cache is the full cache.Cache interface so the
// same instance the handler uses can be injected (nil = no eviction; the
// 300s redirect TTL then bounds staleness - documented).
type Checker struct {
	Store   Store
	Cache   cache.Cache
	Logger  *log.Logger
	Spacing time.Duration
}

// spacing returns the configured inter-check delay (default 1.2s).
func (c *Checker) spacing() time.Duration {
	if c.Spacing > 0 {
		return c.Spacing
	}
	return defaultSpacing
}

// logf is a nil-safe Logger.Printf.
func (c *Checker) logf(format string, args ...any) {
	if c.Logger != nil {
		c.Logger.Printf(format, args...)
	}
}

// CheckOne probes the destination of one short code and persists the
// outcome. Returns the stored status.
//
// Notification policy: the FIRST time a link with an owner becomes broken
// (health_notified_at IS NULL) one in-app notification is created and the
// marker is stored, so a link that stays broken for days never spams the
// bell; a healthy result clears the marker so the NEXT breakage notifies
// again. Anonymous links keep the marker unset (no recipient yet - after
// ClaimLinks the next check notifies the new owner). Persistence happens
// BEFORE the notification insert: a crash in between loses one
// notification rather than risking duplicates (at-most-once).
//
// The redirect cache is evicted only when the status actually CHANGED - a
// no-op probe must not cost a Redis DEL.
func (c *Checker) CheckOne(ctx context.Context, shortCode string) (string, error) {
	link, err := c.Store.GetLink(shortCode)
	if err != nil {
		return "", err
	}
	res := CheckURL(ctx, link.OriginalURL)
	now := time.Now().UTC()

	notifiedAt := time.Time{}
	if link.HealthNotifiedAt != nil {
		notifiedAt = *link.HealthNotifiedAt
	}
	notify := false
	switch {
	case res.Status == StatusHealthy:
		// Healed: forget the marker so the next breakage notifies again.
		notifiedAt = time.Time{}
	case res.Status == StatusBroken && link.HealthNotifiedAt == nil && link.CreatorID != nil:
		notifiedAt = now
		notify = true
	}
	// Any other combination (broken again with the marker set, timeout after
	// broken, unknown) keeps the previous marker untouched.

	changed := res.Status != link.HealthStatus
	if err := c.Store.UpdateLinkHealth(shortCode, res.Status, now, notifiedAt); err != nil {
		return res.Status, err
	}
	if changed && c.Cache != nil {
		if err := c.Cache.Delete(shortCode); err != nil {
			c.logf("Warning: cache eviction after health check %q failed: %v", shortCode, err)
		}
	}
	if notify && link.CreatorID != nil {
		msg := "Link /" + shortCode + " tidak bisa dijangkau (tujuan: " + truncateURL(link.OriginalURL) + ")"
		if _, err := c.Store.CreateNotification(*link.CreatorID, "link_broken", shortCode, msg); err != nil {
			c.logf("Warning: notification for broken link %q failed: %v", shortCode, err)
		}
	}
	return res.Status, nil
}

// RunBatch checks the up-to-limit links that are least recently checked
// (never-checked first, see ListLinksHealthDue) with spacing between probes.
// A failing row is logged and skipped - one bad destination must never abort
// the run. Returns how many checks completed.
func (c *Checker) RunBatch(ctx context.Context) (int, error) {
	links, err := c.Store.ListLinksHealthDue(BatchLimit)
	if err != nil {
		return 0, err
	}
	done := 0
	for i, l := range links {
		if i > 0 {
			select {
			case <-ctx.Done():
				return done, ctx.Err()
			case <-time.After(c.spacing()):
			}
		}
		if _, err := c.CheckOne(ctx, l.ShortCode); err != nil {
			c.logf("health check %q failed: %v", l.ShortCode, err)
			continue
		}
		done++
	}
	return done, nil
}

// truncateURL shortens a destination for notification text (long tracking
// URLs would wrap the bell dropdown).
func truncateURL(u string) string {
	const max = 80
	u = strings.TrimSpace(u)
	if len(u) <= max {
		return u
	}
	return u[:max-1] + "…"
}
