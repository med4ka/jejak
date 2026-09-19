package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync/atomic"

	"jejak/internal/env"
)

// LEARN:
//   Kenapa: Load balancer round-robin paling sederhana untuk Fase 3: tiap request
//   diteruskan bergantian ke backend berikut (8081 → 8082 → 8081 → ...), supaya
//   beban terbagi dan tidak ada 1 instance yang kebanjiran. Counter atomic agar
//   aman diakses banyak goroutine sekaligus (1 goroutine per request masuk).
//   Trade-off: Round-robin buta beban aktual — kalau 1 instance lambat/mati,
//   request tetap dikirim ke sana (timeout/502), tidak seperti least-connection.
//   Tidak ada health check: backend yang down tetap kebagian giliran.
//   Alternatif: Nginx/HAProxy (lebih fitur: health check, retry, least-conn),
//   tapi proxy 60-baris ini cukup untuk merasakan konsep distribusi bebannya.
type roundRobin struct {
	backends []*httputil.ReverseProxy
	addrs    []string
	n        uint64
}

// next picks the next backend in rotation. Atomic counter, safe for
// concurrent use; uint64 wrap-around stays safe under unsigned modulo.
func (rr *roundRobin) next() (*httputil.ReverseProxy, string) {
	i := atomic.AddUint64(&rr.n, 1)
	idx := int((i - 1) % uint64(len(rr.backends)))
	return rr.backends[idx], rr.addrs[idx]
}

func (rr *roundRobin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p, addr := rr.next()
	log.Printf("proxy -> %s %s %s", addr, r.Method, r.URL.Path)
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
	log.Printf("LB listening on :%s, backends=%v", port, addrs)
	log.Fatal(http.ListenAndServe(":"+port, rr))
}
