package api_test

import (
	"net/http"
	"sync"
	"testing"
)

// middleware is an HTTP handler wrapper.
type middleware func(http.Handler) http.Handler //nolint:unused // test infrastructure for future commands

// requestTracker records HTTP requests for test assertions.
type requestTracker struct { //nolint:unused // test infrastructure for future commands
	mu       sync.Mutex
	requests []*http.Request
}

// withRequestTracking returns middleware that records all requests.
// The returned function retrieves the recorded requests.
func withRequestTracking(t *testing.T) (middleware, func() []*http.Request) { //nolint:unused // test infrastructure for future commands
	t.Helper()

	tracker := &requestTracker{}

	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tracker.mu.Lock()
			tracker.requests = append(tracker.requests, r)
			tracker.mu.Unlock()

			next.ServeHTTP(w, r)
		})
	}

	get := func() []*http.Request {
		tracker.mu.Lock()
		defer tracker.mu.Unlock()

		out := make([]*http.Request, len(tracker.requests))
		copy(out, tracker.requests)

		return out
	}

	return mw, get
}

// chainMiddleware composes multiple middleware around a handler.
// Middleware is applied in order: chainMiddleware(h, m1, m2) => m1(m2(h)).
func chainMiddleware(handler http.Handler, mws ...middleware) http.Handler { //nolint:unused // test infrastructure for future commands
	for i := len(mws) - 1; i >= 0; i-- {
		handler = mws[i](handler)
	}

	return handler
}
