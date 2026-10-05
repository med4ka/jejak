package handler

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"jejak/internal/apierror"
	"jejak/internal/db"
	"jejak/internal/middleware"
)

// ExportRowCap limits the number of rows in a single CSV export. The cap is
// a safeguard: clicks mode reads raw rows rather than aggregates, so a range
// containing millions of clicks could otherwise produce a file of hundreds
// of megabytes and hold the connection open for a long time. 10,000 rows is
// approximately < 1 MB: still useful in a spreadsheet and still cheap to
// request repeatedly (the exportRL rate limit covers the rest). Truncated
// rows are never dropped silently: a note in the final CSV column reports
// the truncation (deviation D), never a "#" comment that would break CSV
// parsing.
const ExportRowCap = 10000

// analyticsCacheTTL is the analytics cache lifetime in seconds. Five minutes
// keeps analytics tab refreshes cheap without making the figures feel stale;
// the same numbers are recomputed anyway whenever the caller changes range.
const analyticsCacheTTL = 300

// HandleClicksByDay serves GET /api/analytics/clicks-by-day for the logged-in
// creator: 30-day window [{date, count}] with zero-filled gaps.
// Auth required (a creator sees only their own numbers); reads go through
// the store's read path (replica when enabled, same staleness caveats).
func (h *Handler) HandleClicksByDay(w http.ResponseWriter, r *http.Request) {
	creatorID := middleware.CreatorID(r)
	if creatorID == nil {
		apierror.WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Login required")
		return
	}

	days, err := h.Store.ClicksByDay(*creatorID)
	if err != nil {
		h.Logger.Printf("ClicksByDay failed: %v", err)
		apierror.WriteError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Database error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(days)
}

// HandleAnalyticsSummary serves GET /api/analytics/summary: the four figures
// of the Summary stats card (total active links, total clicks, 30-day unique
// clicks, growth of the last 30 days versus the previous 30). Auth accepts
// either the session cookie (dashboard) or a Bearer API key (automation).
//
// Two identity paths yield one creator context: the optional session
// middleware stores the id in the request context, and a nil value (a request
// without a cookie) falls back to the API key. The business handler does not
// care where the id came from: the same pattern as HandleV1Shorten, except
// that the session takes precedence here because the dashboard (cookie) calls
// this endpoint more often than automation does.
func (h *Handler) HandleAnalyticsSummary(w http.ResponseWriter, r *http.Request) {
	creatorID := middleware.CreatorID(r)
	if creatorID == nil {
		id, err := h.creatorFromAPIKey(r)
		if err != nil {
			apierror.WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Login required")
			return
		}
		creatorID = id
	}

	summary, err := h.Store.AnalyticsSummary(*creatorID)
	if err != nil {
		h.Logger.Printf("AnalyticsSummary failed: %v", err)
		apierror.WriteError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Database error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// cachedJSON fills dst from the analytics cache when a valid entry exists.
// No cache configured, a miss, or an unparseable payload all return false so
// the caller hits the database as usual. Reads are kept separate from writes
// so the hit and the miss paths emit the identical payload shape: the cache
// must never become a second schema that can drift.
func (h *Handler) cachedJSON(key string, dst any) bool {
	if h.Cache == nil {
		return false
	}
	raw, ok := h.Cache.Get(key)
	if !ok {
		return false
	}
	return json.Unmarshal([]byte(raw), dst) == nil
}

// storeJSON stores an analytics payload in the cache; it is a no-op when no
// cache is configured or when marshaling fails.
func (h *Handler) storeJSON(key string, v any) {
	if h.Cache == nil {
		return
	}
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	if err := h.Cache.Set(key, string(b), analyticsCacheTTL); err != nil {
		h.Logger.Printf("Warning: analytics cache set failed: %v", err)
	}
}

// dataPer reports the analytics read source for the response data_per field:
// "primary" (fresh / no replica) or "replica". Consumers need this value
// because replica figures may lag the newest clicks by a few seconds.
func (h *Handler) dataPer() string {
	if s, err := h.Store.AnalyticsFreshness(); err == nil && s != "" {
		return s
	}
	return "primary"
}

// normalizedRange maps a raw ?range= query value to its canonical label
// (empty → "30d"). It feeds both the cache key and the export filename so
// that "?range=" and "?range=30d" do not become two separate buckets.
func normalizedRange(q string) string {
	if strings.TrimSpace(q) == "" {
		return "30d"
	}
	return strings.TrimSpace(q)
}

// HandleBreakdown serves GET /api/analytics/breakdown?kind=device|referrer&range=7d|30d|90d,
// the "Device" and "Source" cards of the Analytics tab. Buckets are
// classified when the click is written (migration 16 plus
// classifyDevice/classifyReferrer), so the query only needs GROUP BY and
// ordering; no user-agent regex runs on the read path. Session auth is
// required: the figures are per creator, not public.
func (h *Handler) HandleBreakdown(w http.ResponseWriter, r *http.Request) {
	creatorID := middleware.CreatorID(r)
	if creatorID == nil {
		apierror.WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Login required")
		return
	}
	kind := r.URL.Query().Get("kind")
	if kind != "device" && kind != "referrer" {
		apierror.WriteError(w, http.StatusBadRequest, "ANALYTICS_INVALID_KIND", "kind harus device atau referrer")
		return
	}
	rawRange := r.URL.Query().Get("range")
	rp, ok := parseRange(rawRange)
	if !ok {
		apierror.WriteError(w, http.StatusBadRequest, "ANALYTICS_INVALID_RANGE", "range harus 7d, 30d, atau 90d")
		return
	}
	rangeName := normalizedRange(rawRange)

	// Cache ONLY the items payload: data_per is computed fresh on every
	// request, so the freshness label must not be frozen for five minutes.
	key := fmt.Sprintf("analytics:breakdown:%d:%s:%s", *creatorID, kind, rangeName)
	var items []db.BreakdownItem
	if h.cachedJSON(key, &items) {
		h.writeBreakdown(w, kind, rangeName, items)
		return
	}

	var err error
	if kind == "device" {
		items, err = h.Store.DeviceBreakdown(*creatorID, rp.From, rp.To)
	} else {
		items, err = h.Store.ReferrerBreakdown(*creatorID, rp.From, rp.To)
	}
	if err != nil {
		h.Logger.Printf("Breakdown(%s) failed: %v", kind, err)
		apierror.WriteError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Database error")
		return
	}
	h.storeJSON(key, items)
	h.writeBreakdown(w, kind, rangeName, items)
}

// writeBreakdown writes the {kind, range, data_per, items, total} response.
// items is forced to [] rather than null so the frontend can always call
// .map without an extra guard.
func (h *Handler) writeBreakdown(w http.ResponseWriter, kind, rangeName string, items []db.BreakdownItem) {
	if items == nil {
		items = []db.BreakdownItem{}
	}
	var total int64
	for _, it := range items {
		total += it.Count
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"kind":     kind,
		"range":    rangeName,
		"data_per": h.dataPer(),
		"items":    items,
		"total":    total,
	})
}

// HandleTimeseries serves GET /api/analytics/timeseries?range=7d|30d|90d:
// a zero-filled [{date, count}] series for the Analytics tab charts. It
// differs from /api/analytics/clicks-by-day (the older endpoint whose fixed
// 30-day window still backs the Summary MiniTrend): here the window is
// chosen by the caller, and the response always carries range and data_per.
func (h *Handler) HandleTimeseries(w http.ResponseWriter, r *http.Request) {
	creatorID := middleware.CreatorID(r)
	if creatorID == nil {
		apierror.WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Login required")
		return
	}
	rawRange := r.URL.Query().Get("range")
	rp, ok := parseRange(rawRange)
	if !ok {
		apierror.WriteError(w, http.StatusBadRequest, "ANALYTICS_INVALID_RANGE", "range harus 7d, 30d, atau 90d")
		return
	}
	rangeName := normalizedRange(rawRange)

	key := fmt.Sprintf("analytics:timeseries:%d:%s", *creatorID, rangeName)
	var payload struct {
		Days []db.DayCount `json:"days"`
	}
	if h.cachedJSON(key, &payload) {
		h.writeTimeseries(w, rangeName, payload.Days)
		return
	}

	counts, err := h.Store.ClicksDaily(*creatorID, rp.From, rp.To)
	if err != nil {
		h.Logger.Printf("ClicksDaily failed: %v", err)
		apierror.WriteError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Database error")
		return
	}
	days := db.FillDays(counts, rp.From, rp.To)
	h.storeJSON(key, map[string]any{"days": days})
	h.writeTimeseries(w, rangeName, days)
}

func (h *Handler) writeTimeseries(w http.ResponseWriter, rangeName string, days []db.DayCount) {
	if days == nil {
		days = []db.DayCount{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"range":    rangeName,
		"data_per": h.dataPer(),
		"days":     days,
	})
}

// exportFilename builds the download name from mode, range and export date
// (UTC) so several downloads in the same folder do not overwrite each other
// and the file contents stay identifiable without opening the file.
func exportFilename(mode, rangeName string) string {
	return fmt.Sprintf("jejak-analitik-%s-%s-%s.csv", mode, rangeName, time.Now().UTC().Format("20060102"))
}

// HandleExportCSV serves GET /api/analytics/export.csv?mode=daily|links|clicks&range=...,
// a CSV export for spreadsheet use. It is wrapped in a rate limit in
// main.go (5/minute per creator) because this is the only path that pulls
// large volumes of raw rows from the database.
//
//	CSV rules (deviation D): every mode puts the `catatan` column LAST, and
//	the only content of that column is the truncation marker row (when the
//	export exceeds ExportRowCap): never a "#" comment, because "#" makes
//	CSV parsers (and Excel) read the following line as raw text instead of
//	data. Process: all database reads complete BEFORE any header or row is
//	written; once a single byte has been sent a 500 can no longer be
//	returned, so reading first guarantees that errors still arrive as proper
//	responses.
func (h *Handler) HandleExportCSV(w http.ResponseWriter, r *http.Request) {
	creatorID := middleware.CreatorID(r)
	if creatorID == nil {
		apierror.WriteError(w, http.StatusUnauthorized, "AUTH_REQUIRED", "Login required")
		return
	}
	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = "daily"
	}
	if mode != "daily" && mode != "links" && mode != "clicks" {
		apierror.WriteError(w, http.StatusBadRequest, "ANALYTICS_INVALID_MODE", "mode harus daily, links, atau clicks")
		return
	}
	rawRange := r.URL.Query().Get("range")
	rp, ok := parseRange(rawRange)
	if !ok {
		apierror.WriteError(w, http.StatusBadRequest, "ANALYTICS_INVALID_RANGE", "range harus 7d, 30d, atau 90d")
		return
	}
	rangeName := normalizedRange(rawRange)

	var header []string
	var rows [][]string
	var truncated bool
	var err error

	switch mode {
	case "daily":
		header, rows, truncated, err = h.exportDaily(*creatorID, rp)
	case "links":
		header, rows, truncated, err = h.exportLinks(*creatorID, rp)
	case "clicks":
		header, rows, truncated, err = h.exportClicks(*creatorID, rp)
	}
	if err != nil {
		h.Logger.Printf("Export CSV (%s) failed: %v", mode, err)
		apierror.WriteError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Database error")
		return
	}
	if truncated {
		// Marker row with the same width as the header (never a "#"
		// comment: see the HandleExportCSV notes): the last column carries
		// the note and the data columns stay empty so the row still parses
		// as a valid CSV line.
		note := make([]string, len(header))
		note[len(header)-1] = fmt.Sprintf("terpotong: ekspor dibatasi %d baris; masih ada data lain di rentang ini", ExportRowCap)
		rows = append(rows, note)
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", exportFilename(mode, rangeName)))
	// UTF-8 BOM: Excel on Windows opens a CSV without a BOM as CP1252 and
	// mangles non-ASCII characters (clicks and Indonesian column names stay
	// safe, but URLs and tags do not). The BOM makes Excel decode UTF-8
	// correctly.
	w.Write([]byte("\xEF\xBB\xBF"))

	cw := csv.NewWriter(w)
	cw.Write(header)
	for _, row := range rows {
		cw.Write(row)
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		h.Logger.Printf("Export CSV write failed: %v", err)
	}
}

// exportDaily emits clicks aggregated per day (the smallest export: at most
// 90 rows, so it can never be truncated).
func (h *Handler) exportDaily(creatorID int64, rp RangeParams) ([]string, [][]string, bool, error) {
	counts, err := h.Store.ClicksDaily(creatorID, rp.From, rp.To)
	if err != nil {
		return nil, nil, false, err
	}
	days := db.FillDays(counts, rp.From, rp.To)
	rows := make([][]string, 0, len(days))
	for _, d := range days {
		rows = append(rows, []string{d.Date, fmt.Sprintf("%d", d.Count), ""})
	}
	return []string{"tanggal", "klik", "catatan"}, rows, false, nil
}

// exportLinks emits per-link click and unique-click totals for the selected
// range. The query asks for one extra row: if more than ExportRowCap rows
// arrive, the surplus row is dropped and the export is flagged truncated.
func (h *Handler) exportLinks(creatorID int64, rp RangeParams) ([]string, [][]string, bool, error) {
	links, err := h.Store.LinkExportStats(creatorID, rp.From, rp.To, ExportRowCap+1)
	if err != nil {
		return nil, nil, false, err
	}
	truncated := len(links) > ExportRowCap
	if truncated {
		links = links[:ExportRowCap]
	}
	rows := make([][]string, 0, len(links))
	for _, l := range links {
		rows = append(rows, []string{
			l.ShortCode,
			l.OriginalURL,
			strings.Join(l.Tags, ","),
			fmt.Sprintf("%d", l.Clicks),
			fmt.Sprintf("%d", l.UniqueClicks),
			"",
		})
	}
	return []string{"kode", "url", "tags", "klik", "klik_unik", "catatan"}, rows, truncated, nil
}

// exportClicks emits one raw row per click event (the largest export). The
// SQL LIMIT keeps the database from ever handing more than Cap+1 rows to
// application memory.
func (h *Handler) exportClicks(creatorID int64, rp RangeParams) ([]string, [][]string, bool, error) {
	clicks, err := h.Store.ListClicks(creatorID, rp.From, rp.To, ExportRowCap+1)
	if err != nil {
		return nil, nil, false, err
	}
	truncated := len(clicks) > ExportRowCap
	if truncated {
		clicks = clicks[:ExportRowCap]
	}
	rows := make([][]string, 0, len(clicks))
	for _, c := range clicks {
		unique := "0"
		if c.IsUnique {
			unique = "1"
		}
		rows = append(rows, []string{
			c.ClickedAt.UTC().Format(time.RFC3339),
			c.ShortCode,
			c.DeviceType,
			c.ReferrerType,
			c.ReferrerDomain,
			unique,
			"",
		})
	}
	return []string{"waktu", "kode", "perangkat", "sumber", "referrer", "klik_unik", "catatan"}, rows, truncated, nil
}
