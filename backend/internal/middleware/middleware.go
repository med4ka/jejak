// Package middleware wires cross-cutting HTTP concerns (session auth, rate
// limiting) OUT of the handlers so each handler only does business logic.
// Auth middleware resolves the session cookie into a creator id stored on the
// request context; handlers read it via CreatorID instead of parsing cookies
// themselves.
package middleware

import (
	"context"
	"net/http"

	"jejak/internal/auth"
	"jejak/internal/ratelimit"
)

// ctxKey is the private context key for the resolved creator id.
type ctxKey struct{}

// CreatorID returns the authenticated creator's id from the request context,
// or nil when the request is anonymous (no session / optional-auth miss).
// Handlers that require login treat nil as "401 Login required" (their own
// check keeps direct-call semantics safe even if a route is not wrapped).
func CreatorID(r *http.Request) *int64 {
	if id, ok := r.Context().Value(ctxKey{}).(*int64); ok {
		return id
	}
	return nil
}

// WithCreator attaches a creator id to the request context. Exported so unit
// tests can call handlers directly (bypassing the middleware layer) and for
// any layered handler that already resolved identity elsewhere.
func WithCreator(r *http.Request, creatorID int64) *http.Request {
	v := creatorID
	return r.WithContext(context.WithValue(r.Context(), ctxKey{}, &v))
}

// authFromRequest validates the session cookie against the store and, on a
// successful hit, re-issues the cookie with a fresh expiry so the browser
// cookie and the server-side TTL (already slid inside Store.Get) never drift
// apart. Nil store behaves as anonymous (fail safe: auth disabled).
func authFromRequest(st auth.Store, w http.ResponseWriter, r *http.Request) *int64 {
	if st == nil {
		return nil
	}
	token := auth.TokenFromRequest(r)
	if token == "" {
		return nil
	}
	id, ok := st.Get(token)
	if !ok {
		return nil
	}
	auth.SetCookie(w, token)
	return &id
}

// authAdapter builds the auth middleware. required=true turns anonymous
// requests into an immediate 401; required=false lets them through as
// anonymous (optional auth, e.g. POST /api/shorten).
func authAdapter(st auth.Store, required bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := authFromRequest(st, w, r)
			if required && id == nil {
				http.Error(w, "Login required", http.StatusUnauthorized)
				return
			}
			if id != nil {
				r = WithCreator(r, *id)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAuth wraps next so only logged-in requests reach it. Auth is
// resolved from the session cookie; the creator id lands on the request
// context for the handler.
func RequireAuth(st auth.Store) func(http.Handler) http.Handler {
	return authAdapter(st, true)
}

// OptionalAuth wraps next: valid sessions populate the creator id on the
// context, anonymous requests pass through untouched (nil id).
func OptionalAuth(st auth.Store) func(http.Handler) http.Handler {
	return authAdapter(st, false)
}

// RateLimit wraps next with a fixed-window limiter. key extracts the bucket
// from the request (e.g. the API key). Allow+Record consumes one quota per
// request, mirroring the public API's per-key rule. l == nil (or key == nil)
// disables throttling so baseline mode keeps the same code path.
func RateLimit(l *ratelimit.Limiter, key func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if l == nil || key == nil {
				next.ServeHTTP(w, r)
				return
			}
			bucket := key(r)
			if !l.Allow(bucket) {
				w.Header().Set("Retry-After", "60")
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			l.Record(bucket)
			next.ServeHTTP(w, r)
		})
	}
}
