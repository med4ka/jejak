package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"sort"
	"sync"
	"time"
)

// LEARN:
//   Kenapa: Fungsi ini menjalankan N request dengan C koneksi concurrent ke satu
//   endpoint, lalu melaporkan RPS + latency. Konsep design: performance measurement
//   dengan parameter SAMA persis tiap fase, supaya angka baseline vs optimasi bisa
//   dibandingkan secara adil (apel vs apel, bukan apel vs jeruk).
//   Trade-off: Harness sederhana ini tidak memodelkan pola traffic nyata (burst,
//   think time, variasi endpoint) — angkanya hanya untuk perbandingan relatif,
//   bukan prediksi kapasitas production.
//   Alternatif: k6/hey untuk skenario kompleks, tapi tool sendiri cukup untuk
//   baseline lokal dan tidak menambah dependensi.
func runLoadTest(target string, total, concurrency int) {
	// Redirect TIDAK di-follow: untuk endpoint /r/{code}, 302 ADALAH sukses.
	// Kalau di-follow, yang terukur malah example.com (di luar sistem kita).
	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	latencies := make([]time.Duration, 0, total)
	statusCount := make(map[int]int)
	var errors int
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)

	start := time.Now()
	for i := 0; i < total; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			t0 := time.Now()
			resp, err := client.Get(target)
			elapsed := time.Since(t0)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errors++
				return
			}
			resp.Body.Close()
			latencies = append(latencies, elapsed)
			statusCount[resp.StatusCode]++
		}()
	}
	wg.Wait()
	wall := time.Since(start)

	if len(latencies) == 0 {
		fmt.Println("RESULT no successful responses (all requests errored)")
		fmt.Printf("RESULT errors=%d total=%d\n", errors, total)
		os.Exit(1)
	}

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	var sum time.Duration
	for _, d := range latencies {
		sum += d
	}
	avg := sum / time.Duration(len(latencies))
	p50 := latencies[len(latencies)/2]
	p95 := latencies[(len(latencies)*95)/100]
	rps := float64(total) / wall.Seconds()

	ok := 0
	for code, n := range statusCount {
		if code >= 200 && code < 400 {
			ok += n
		}
	}

	fmt.Printf("RESULT url=%s n=%d c=%d\n", target, total, concurrency)
	fmt.Printf("RESULT wall=%s rps=%.2f ok=%d errors=%d\n", wall.Round(time.Millisecond), rps, ok, errors)
	fmt.Printf("RESULT latency avg=%s p50=%s p95=%s\n", avg.Round(time.Microsecond), p50.Round(time.Microsecond), p95.Round(time.Microsecond))
	fmt.Printf("RESULT status=%v\n", statusCount)
}

// LEARN:
//   Kenapa: Fungsi ini mendapatkan informasi sistem (Goroutines, CPU) pada awal baseline.
//   Konsep design yang terkait: resource awareness - tahu resource apa yang
//   sudah digunakan sebelum load test, agar bisa dibandingkan setelah.
//   Trade-off: Information collection memiliki overhead kecil, tetapi memberikan context
//   berharga tentang state sistem sebelum traffic datang.
//   Alternatif: Bisa pakai prometheus metrics atau runtime.MemProfileRate,
//   tapi untuk baseline sederhana, runtime info cukup.
func CaptureSystemState() {
	log.Printf("System Info:")
	log.Printf("  GOMAXPROCS: %d", runtime.GOMAXPROCS(0))
	log.Printf("  NumCPU: %d", runtime.NumCPU())
	log.Printf("  Goroutines: %d", CountGoroutines())
}

// LEARN:
//   Kenapa: Fungsi bantuan menghitung jumlah goroutine aktif.
//   Konsep design yang terkait: goroutine monitoring - dalam server Go, setiap
//   request bisa membuat goroutine baru. Mengetahui baseline membantu mendeteksi
//   memory leaks atau goroutine leaks pada masa depan.
//   Trade-off: Memerlukan iterate through runtime.Gosched, overhead negligible.
//   Alternatif: Bisa pakai third-party library seperti pprof, tapi untuk baseline
//   sederhana, fungsi ini cukup.
func CountGoroutines() int {
	return runtime.NumGoroutine()
}

func main() {
	target := flag.String("url", "http://localhost:8081/r/abc123", "target endpoint (pakai short code yang ada di KEDUA database)")
	n := flag.Int("n", 200, "total requests")
	c := flag.Int("c", 10, "concurrent connections")
	flag.Parse()

	if *n <= 0 || *c <= 0 {
		log.Fatal("n and c must be > 0")
	}
	if *c > *n {
		*c = *n
	}

	CaptureSystemState()
	runLoadTest(*target, *n, *c)
}
