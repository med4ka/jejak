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

// Rationale: this function runs N requests with C concurrent connections
// against one endpoint, then reports RPS + latency. Design concept:
// performance measurement with IDENTICAL parameters in every phase so that
// baseline versus optimization numbers can be compared fairly (apples to
// apples, not apples to oranges).
// Trade-off: this simple harness does not model real traffic patterns (bursts,
// think time, endpoint variety): the numbers are relative comparisons only,
// not predictions of production capacity.
// Alternative: k6/hey for complex scenarios, but a built-in tool suffices for
// a local baseline and adds no dependency.
func runLoadTest(target string, total, concurrency int) {
	// Redirects are NOT followed: for the /r/{code} endpoint a 302 IS success.
	// If they were followed, what gets measured is example.com (outside this
	// system).
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

// CaptureSystemState records system information (goroutines, CPU) at the start
// of a baseline. Design concept: resource awareness: knowing which resources
// are already in use before the load test makes results comparable afterwards.
// Trade-off: collection has small overhead but provides valuable context about
// system state before traffic arrives. Alternative: Prometheus metrics or
// runtime.MemProfileRate, but runtime info suffices for a simple baseline.
func CaptureSystemState() {
	log.Printf("System Info:")
	log.Printf("  GOMAXPROCS: %d", runtime.GOMAXPROCS(0))
	log.Printf("  NumCPU: %d", runtime.NumCPU())
	log.Printf("  Goroutines: %d", CountGoroutines())
}

// CountGoroutines returns the number of active goroutines. Design concept:
// goroutine monitoring: in a Go server every request can start a new
// goroutine, and knowing the baseline helps detect memory or goroutine leaks
// later. Trade-off: negligible overhead (a single runtime query).
// Alternative: a third-party tool such as pprof, but for a simple baseline
// this function suffices.
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
