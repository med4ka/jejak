package handler

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"jejak/internal/db"
	"jejak/internal/middleware"
)

// TestHandleAnalyticsSummary: GET /api/analytics/summary: 401 without an
// identity; 200 with the JSON shape when a session is present; the creatorID
// scope is passed to the store (scoping lives in the store layer).
func TestHandleAnalyticsSummary(t *testing.T) {
	t.Run("401 without session or API key", func(t *testing.T) {
		s := &fakeStore{}
		h := newTestHandler(s)
		rr := httptest.NewRecorder()
		h.HandleAnalyticsSummary(rr, httptest.NewRequest(http.MethodGet, "/api/analytics/summary", nil))
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("want 401, got %d", rr.Code)
		}
	})

	t.Run("200 with session returns summary JSON", func(t *testing.T) {
		s := &fakeStore{
			summary: db.AnalyticsSummary{
				TotalLinks:      12,
				TotalClicks:     1247,
				UniqueClicks30d: 892,
				GrowthPct:       18,
			},
		}
		h := newTestHandler(s)
		req := httptest.NewRequest(http.MethodGet, "/api/analytics/summary", nil)
		req = middleware.WithCreator(req, 42)
		rr := httptest.NewRecorder()
		h.HandleAnalyticsSummary(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("want 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var got db.AnalyticsSummary
		if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.TotalLinks != 12 || got.TotalClicks != 1247 || got.UniqueClicks30d != 892 || got.GrowthPct != 18 {
			t.Fatalf("unexpected payload: %+v", got)
		}
		if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
			t.Fatalf("Content-Type: %q", ct)
		}
	})

	t.Run("500 when store fails", func(t *testing.T) {
		s := &fakeStore{summaryErr: errors.New("boom")}
		h := newTestHandler(s)
		req := middleware.WithCreator(httptest.NewRequest(http.MethodGet, "/api/analytics/summary", nil), 1)
		rr := httptest.NewRecorder()
		h.HandleAnalyticsSummary(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("want 500, got %d", rr.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// Analytics depth (2026-09-30): classification at write time, range parsing,
// and three new handlers (breakdown / timeseries / export CSV).
// ---------------------------------------------------------------------------

// TestClassifyDevice locks the bucket order: bot -> tablet -> mobile ->
// desktop -> unknown. Critical case: an iPad user agent contains the word
// "Mobile" yet MUST fall into tablet (deviation B): the ordering inside
// classifyDevice guarantees that, and this test keeps it from being reversed.
func TestClassifyDevice(t *testing.T) {
	cases := []struct {
		name string
		ua   string
		want string
	}{
		{"empty", "", "unknown"},
		{"ipad with Mobile token", "Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1", "tablet"},
		{"android tablet without Mobile", "Mozilla/5.0 (Linux; Android 13; SM-T870) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36", "tablet"},
		{"android phone", "Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36", "mobile"},
		{"iphone", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1", "mobile"},
		{"windows desktop", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36", "desktop"},
		{"mac desktop", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36", "desktop"},
		{"linux desktop", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36", "desktop"},
		{"crawler", "Googlebot/2.1 (+http://www.google.com/bot.html)", "bot"},
		{"link preview", "facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)", "bot"},
	}
	for _, c := range cases {
		if got := classifyDevice(c.ua); got != c.want {
			t.Errorf("%s: classifyDevice(%q) = %q, want %q", c.name, c.ua, got, c.want)
		}
	}
}

// TestClassifyReferrer locks the 5 buckets plus two traps: mail.google.com
// must be "other" (not search, despite the .google.com suffix), and "x.com"
// must match exactly (the host matrix.com must not be swept in by a
// substring match).
func TestClassifyReferrer(t *testing.T) {
	cases := []struct {
		host string
		want string
	}{
		{"", "direct"},
		{"google.com", "search"},
		{"www.google.com", "search"},
		{"search.brave.com", "search"},
		{"duckduckgo.com", "search"},
		{"mail.google.com", "other"},
		{"wa.me", "chat"},
		{"chat.whatsapp.com", "chat"},
		{"t.me", "chat"},
		{"m.me", "chat"},
		{"fb.me", "social"},
		{"l.facebook.com", "social"},
		{"x.com", "social"},
		{"t.co", "social"},
		{"tiktok.com", "social"},
		{"matrix.com", "other"},
		{"example.org", "other"},
	}
	for _, c := range cases {
		if got := classifyReferrer(c.host); got != c.want {
			t.Errorf("classifyReferrer(%q) = %q, want %q", c.host, got, c.want)
		}
	}
}

// TestParseRange accepts only 7d/30d/90d (empty defaults to 30d); anything
// else is rejected so the handler answers 400 instead of silently using a
// window the caller never requested.
func TestParseRange(t *testing.T) {
	today := time.Now().UTC()
	midnight := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)

	for _, c := range []struct {
		in   string
		days int
	}{
		{"", 30},
		{"30d", 30},
		{"7d", 7},
		{"90d", 90},
	} {
		rp, ok := parseRange(c.in)
		if !ok {
			t.Fatalf("parseRange(%q) rejected, want ok", c.in)
		}
		if rp.Days != c.days {
			t.Errorf("parseRange(%q).Days = %d, want %d", c.in, rp.Days, c.days)
		}
		if wantFrom := midnight.AddDate(0, 0, -(c.days - 1)); !rp.From.Equal(wantFrom) {
			t.Errorf("parseRange(%q).From = %v, want %v", c.in, rp.From, wantFrom)
		}
		if wantTo := midnight.AddDate(0, 0, 1); !rp.To.Equal(wantTo) {
			t.Errorf("parseRange(%q).To = %v, want %v", c.in, rp.To, wantTo)
		}
		if got := int(rp.To.Sub(rp.From).Hours() / 24); got != c.days {
			t.Errorf("parseRange(%q) window = %d days, want %d", c.in, got, c.days)
		}
	}

	for _, bad := range []string{"7", "all", "bogus", "1y", "-7d"} {
		if _, ok := parseRange(bad); ok {
			t.Errorf("parseRange(%q) accepted, want rejected", bad)
		}
	}
}

// TestHandleBreakdown: 401 without a session, 400 for an invalid kind or
// range, and 200 with the payload shape {kind, range, data_per, items, total}.
func TestHandleBreakdown(t *testing.T) {
	t.Run("401 without session", func(t *testing.T) {
		h := newTestHandler(&fakeStore{})
		rr := httptest.NewRecorder()
		h.HandleBreakdown(rr, httptest.NewRequest(http.MethodGet, "/api/analytics/breakdown?kind=device", nil))
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("want 401, got %d", rr.Code)
		}
	})

	t.Run("400 unknown kind", func(t *testing.T) {
		h := newTestHandler(&fakeStore{})
		req := middleware.WithCreator(httptest.NewRequest(http.MethodGet, "/api/analytics/breakdown?kind=browser", nil), 7)
		rr := httptest.NewRecorder()
		h.HandleBreakdown(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("want 400, got %d", rr.Code)
		}
	})

	t.Run("400 bad range", func(t *testing.T) {
		h := newTestHandler(&fakeStore{})
		req := middleware.WithCreator(httptest.NewRequest(http.MethodGet, "/api/analytics/breakdown?kind=device&range=bogus", nil), 7)
		rr := httptest.NewRecorder()
		h.HandleBreakdown(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("want 400, got %d", rr.Code)
		}
	})

	t.Run("200 shape with data_per and total", func(t *testing.T) {
		s := &fakeStore{
			deviceItems: []db.BreakdownItem{
				{Key: "mobile", Count: 30},
				{Key: "desktop", Count: 12},
			},
			dataPer: "replica",
		}
		h := newTestHandler(s)
		req := middleware.WithCreator(httptest.NewRequest(http.MethodGet, "/api/analytics/breakdown?kind=device&range=7d", nil), 7)
		rr := httptest.NewRecorder()
		h.HandleBreakdown(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("want 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var got struct {
			Kind    string             `json:"kind"`
			Range   string             `json:"range"`
			DataPer string             `json:"data_per"`
			Items   []db.BreakdownItem `json:"items"`
			Total   int64              `json:"total"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.Kind != "device" || got.Range != "7d" {
			t.Errorf("kind/range = %q/%q, want device/7d", got.Kind, got.Range)
		}
		if got.DataPer != "replica" {
			t.Errorf("data_per = %q, want replica", got.DataPer)
		}
		if got.Total != 42 {
			t.Errorf("total = %d, want 42", got.Total)
		}
		if len(got.Items) != 2 || got.Items[0].Key != "mobile" {
			t.Errorf("items = %+v", got.Items)
		}
	})

	t.Run("500 when store fails", func(t *testing.T) {
		h := newTestHandler(&fakeStore{analyticsErr: errors.New("boom")})
		req := middleware.WithCreator(httptest.NewRequest(http.MethodGet, "/api/analytics/breakdown?kind=device", nil), 7)
		rr := httptest.NewRecorder()
		h.HandleBreakdown(rr, req)
		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("want 500, got %d", rr.Code)
		}
	})
}

// TestHandleTimeseries: a zero-filled series of exactly N days (days without
// clicks still appear as 0: the difference from raw aggregation).
func TestHandleTimeseries(t *testing.T) {
	t.Run("401 without session", func(t *testing.T) {
		h := newTestHandler(&fakeStore{})
		rr := httptest.NewRecorder()
		h.HandleTimeseries(rr, httptest.NewRequest(http.MethodGet, "/api/analytics/timeseries", nil))
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("want 401, got %d", rr.Code)
		}
	})

	t.Run("400 bad range", func(t *testing.T) {
		h := newTestHandler(&fakeStore{})
		req := middleware.WithCreator(httptest.NewRequest(http.MethodGet, "/api/analytics/timeseries?range=1y", nil), 3)
		rr := httptest.NewRecorder()
		h.HandleTimeseries(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("want 400, got %d", rr.Code)
		}
	})

	t.Run("200 zero-filled 7d window", func(t *testing.T) {
		today := time.Now().UTC().Format("2006-01-02")
		s := &fakeStore{daily: map[string]int64{today: 5}}
		h := newTestHandler(s)
		req := middleware.WithCreator(httptest.NewRequest(http.MethodGet, "/api/analytics/timeseries?range=7d", nil), 3)
		rr := httptest.NewRecorder()
		h.HandleTimeseries(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("want 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var got struct {
			Range   string        `json:"range"`
			DataPer string        `json:"data_per"`
			Days    []db.DayCount `json:"days"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.Range != "7d" || got.DataPer != "primary" {
			t.Errorf("range/data_per = %q/%q", got.Range, got.DataPer)
		}
		if len(got.Days) != 7 {
			t.Fatalf("days = %d, want 7", len(got.Days))
		}
		if got.Days[6].Date != today || got.Days[6].Count != 5 {
			t.Errorf("hari terakhir = %+v, want %s/5", got.Days[6], today)
		}
		if got.Days[0].Count != 0 {
			t.Errorf("hari pertama = %d, want 0 (zero-fill)", got.Days[0].Count)
		}
	})
}

// readCSV parses an export response into CSV rows after stripping the UTF-8
// BOM, which would otherwise end up inside the first field.
func readCSV(t *testing.T, rr *httptest.ResponseRecorder) [][]string {
	t.Helper()
	r := csv.NewReader(strings.NewReader(strings.TrimPrefix(rr.Body.String(), "\uFEFF")))
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	return rows
}

// TestHandleExportCSV covers the 3 modes plus deviation D: the last column
// is always "catatan", truncation is reported through a note row (never
// "#"), and the response starts with a UTF-8 BOM so Excel decodes UTF-8.
func TestHandleExportCSV(t *testing.T) {
	exportReq := func(query string) *http.Request {
		return middleware.WithCreator(httptest.NewRequest(http.MethodGet, "/api/analytics/export.csv"+query, nil), 5)
	}

	t.Run("401 without session", func(t *testing.T) {
		h := newTestHandler(&fakeStore{})
		rr := httptest.NewRecorder()
		h.HandleExportCSV(rr, httptest.NewRequest(http.MethodGet, "/api/analytics/export.csv", nil))
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("want 401, got %d", rr.Code)
		}
	})

	t.Run("400 bad mode", func(t *testing.T) {
		h := newTestHandler(&fakeStore{})
		rr := httptest.NewRecorder()
		h.HandleExportCSV(rr, exportReq("?mode=xml"))
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("want 400, got %d", rr.Code)
		}
	})

	t.Run("400 bad range", func(t *testing.T) {
		h := newTestHandler(&fakeStore{})
		rr := httptest.NewRecorder()
		h.HandleExportCSV(rr, exportReq("?mode=daily&range=bogus"))
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("want 400, got %d", rr.Code)
		}
	})

	t.Run("daily: header, BOM, isi rentang default", func(t *testing.T) {
		h := newTestHandler(&fakeStore{})
		rr := httptest.NewRecorder()
		h.HandleExportCSV(rr, exportReq("?mode=daily"))
		if rr.Code != http.StatusOK {
			t.Fatalf("want 200, got %d", rr.Code)
		}
		if !strings.HasPrefix(rr.Body.String(), "\uFEFF") {
			t.Error("CSV harus diawali BOM UTF-8 (Excel)")
		}
		if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/csv") {
			t.Errorf("Content-Type = %q", ct)
		}
		if cd := rr.Header().Get("Content-Disposition"); !strings.Contains(cd, "jejak-analitik-daily-30d-") {
			t.Errorf("Content-Disposition = %q", cd)
		}
		rows := readCSV(t, rr)
		if len(rows) != 31 {
			t.Fatalf("rows = %d, want 1 header + 30 hari", len(rows))
		}
		if got := strings.Join(rows[0], ","); got != "tanggal,klik,catatan" {
			t.Errorf("header = %q", got)
		}
		if rows[1][0] == "" || rows[30][0] == "" {
			t.Error("tanggal tidak boleh kosong")
		}
	})

	t.Run("links: link tanpa klik tetap ikut dengan 0", func(t *testing.T) {
		s := &fakeStore{linkStats: []db.LinkExportRow{
			{ShortCode: "abc123", OriginalURL: "https://example.org", Tags: []string{"news", "hot"}, Clicks: 9, UniqueClicks: 4},
		}}
		h := newTestHandler(s)
		rr := httptest.NewRecorder()
		h.HandleExportCSV(rr, exportReq("?mode=links&range=7d"))
		if rr.Code != http.StatusOK {
			t.Fatalf("want 200, got %d", rr.Code)
		}
		rows := readCSV(t, rr)
		if len(rows) != 2 {
			t.Fatalf("rows = %d, want 2", len(rows))
		}
		if got := strings.Join(rows[0], ","); got != "kode,url,tags,klik,klik_unik,catatan" {
			t.Errorf("header = %q", got)
		}
		if rows[1][0] != "abc123" || rows[1][3] != "9" || rows[1][4] != "4" {
			t.Errorf("baris = %q", rows[1])
		}
		if !strings.Contains(rows[1][2], "news") || !strings.Contains(rows[1][2], "hot") {
			t.Errorf("tags = %q, want berisi news & hot", rows[1][2])
		}
		if rows[1][5] != "" {
			t.Errorf("catatan baris data harus kosong, got %q", rows[1][5])
		}
	})

	t.Run("clicks: pemotongan dilaporkan di kolom catatan, bukan komentar #", func(t *testing.T) {
		clicks := make([]db.ClickRow, 0, ExportRowCap+1)
		for i := 0; i <= ExportRowCap; i++ {
			clicks = append(clicks, db.ClickRow{
				ClickedAt: time.Unix(int64(i), 0).UTC(), ShortCode: "k",
				DeviceType: "desktop", ReferrerType: "direct",
			})
		}
		h := newTestHandler(&fakeStore{clickRows: clicks})
		rr := httptest.NewRecorder()
		h.HandleExportCSV(rr, exportReq("?mode=clicks&range=90d"))
		if rr.Code != http.StatusOK {
			t.Fatalf("want 200, got %d", rr.Code)
		}
		rows := readCSV(t, rr)
		// 1 header + ExportRowCap data rows + 1 note row.
		if len(rows) != ExportRowCap+2 {
			t.Fatalf("rows = %d, want %d", len(rows), ExportRowCap+2)
		}
		last := rows[len(rows)-1]
		if len(last) != len(rows[0]) {
			t.Fatalf("baris catatan lebarnya %d, header %d", len(last), len(rows[0]))
		}
		if last[len(last)-1] == "" || !strings.Contains(last[len(last)-1], "terpotong") {
			t.Errorf("catatan akhir = %q, want berisi keterangan terpotong", last[len(last)-1])
		}
		if strings.HasPrefix(last[len(last)-1], "#") {
			t.Error("catatan tidak boleh berupa komentar #")
		}
	})
}

// TestClickClassificationAtWriteTime: the redirect writes an event that is
// already classified (deviation A): the raw user agent is stored, an iPad
// yields device_type "tablet" (not "mobile" even though the UA contains the
// word Mobile), and referrer_type is derived from the host rather than from
// the raw Referer text.
func TestClickClassificationAtWriteTime(t *testing.T) {
	s := &fakeStore{}
	h := newTestHandler(s)
	s.link = db.Link{ShortCode: "dev001", OriginalURL: "https://example.org/target", IsActive: true}
	h.Cache = &mapCache{}

	req := httptest.NewRequest(http.MethodGet, "/r/dev001", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1")
	req.Header.Set("Referer", "https://wa.me/15551234567")
	rr := httptest.NewRecorder()
	h.HandleRedirect("dev001", rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("redirect = %d, want 302", rr.Code)
	}
	if len(s.clickEvents) != 1 {
		t.Fatalf("logClick dipanggil %d kali, want 1", len(s.clickEvents))
	}
	ev := s.clickEvents[0]
	if ev.DeviceType != "tablet" {
		t.Errorf("DeviceType = %q, want tablet (iPad harus mendahului aturan mobile)", ev.DeviceType)
	}
	if ev.ReferrerType != "chat" {
		t.Errorf("ReferrerType = %q, want chat (wa.me)", ev.ReferrerType)
	}
	if ev.ReferrerDomain != "wa.me" {
		t.Errorf("ReferrerDomain = %q, want wa.me", ev.ReferrerDomain)
	}
	if ev.UserAgent == "" || len(ev.UserAgent) > 512 {
		t.Errorf("UserAgent tersimpan %d karakter, want 1..512", len(ev.UserAgent))
	}
}

// TestTruncateUA: an oversized UA is cut at exactly 512 CHARACTERS, not
// bytes: the VARCHAR(512) column in Postgres counts characters, and a
// byte-boundary cut could split a rune and fail at encoding time.
func TestTruncateUA(t *testing.T) {
	if got := truncateUA("pendek"); got != "pendek" {
		t.Errorf("truncateUA pendek = %q", got)
	}
	long := strings.Repeat("é", 700)
	got := truncateUA(long)
	if n := len([]rune(got)); n != 512 {
		t.Errorf("panjang = %d rune, want 512", n)
	}
	if !strings.HasPrefix(long, got) {
		t.Error("potongan harus berupa prefix dari UA asli")
	}
}
