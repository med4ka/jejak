package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"jejak/internal/env"
)

// roundRobin is the simplest round-robin load balancer for Phase 3: each
// request is forwarded to the next backend in turn (8081 → 8082 → 8081 → ...)
// so load is shared and no single instance is flooded. The counter is atomic,
// making it safe for the many goroutines handling incoming requests (one
// goroutine per request).
// Trade-off: round-robin is blind to actual load: if one instance is slow or
// dead, requests are still sent there (timeouts/502), unlike least-connection.
// There is no health check: a down backend still receives its share of turns.
// Alternative: Nginx/HAProxy (more features: health checks, retry,
// least-conn), but this 60-line proxy suffices to experience the concept of
// load distribution.
type roundRobin struct {
	backends []*httputil.ReverseProxy
	addrs    []string
	n        uint64
}

// next picks the next backend in rotation. Atomic counter, safe for
// concurrent use; uint64 wrap-around stays safe under unsigned modulo.
func (rr *roundRobin) next() (*httputil.ReverseProxy, string) {
	i := atomic.AddUint64(&rr.n, 1)
	// #nosec G115 -- the modulo result is always < len(rr.backends) (a small
	// slice length), so the uint64→int conversion cannot overflow.
	idx := int((i - 1) % uint64(len(rr.backends)))
	return rr.backends[idx], rr.addrs[idx]
}

// ServeHTTP forwards the request to the next backend in round-robin order
// (rr.next advances a shared counter, so consecutive requests spread evenly
// across BACKENDS) and logs the chosen instance before proxying: a wrongly
// routed request must be traceable from the proxy log alone. The path is
// logged with %q so a decoded "%0A" cannot inject forged lines into the log
// (log-injection, G706).
func (rr *roundRobin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p, addr := rr.next()
	// #nosec G706 -- the path is written with %q (newline/quote escaped) and
	// r.Method is a server-validated token: neither can break the log line.
	log.Printf("proxy -> %s %s %q", addr, r.Method, r.URL.Path)
	p.ServeHTTP(w, r)
}

func main() {
	env.LoadDotEnv()

	raw := env.Get("BACKENDS", "http://localhost:8081,http://localhost:8082")
	port := strings.TrimSpace(env.Get("PORT", "8090"))

	var proxies []*httputil.ReverseProxy
	var addrs []string
	for _, s := range strings.Split(raw, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		u, err := url.Parse(s)
		if err != nil {
			log.Fatal("Invalid backend URL:", err)
		}
		proxies = append(proxies, httputil.NewSingleHostReverseProxy(u))
		addrs = append(addrs, s)
	}
	if len(proxies) == 0 {
		log.Fatal("BACKENDS is empty")
	}

	rr := &roundRobin{backends: proxies, addrs: addrs}
	// Timeouts (G114): same rationale as the API server: a zero-value
	// http.Server lets a slow client hold connections open forever.
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           rr,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("LB listening on :%s, backends=%v", port, addrs)
	log.Fatal(srv.ListenAndServe())
}
