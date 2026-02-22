package api

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestEnsureReplayableBody(t *testing.T) {
	t.Parallel()

	t.Run("nil request", func(t *testing.T) {
		t.Parallel()

		if err := ensureReplayableBody(nil); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("nil body", func(t *testing.T) {
		t.Parallel()

		req, _ := http.NewRequestWithContext(context.Background(), "GET", "http://example.com", nil)
		if err := ensureReplayableBody(req); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("already replayable", func(t *testing.T) {
		t.Parallel()

		body := io.NopCloser(strings.NewReader("hello"))
		req, _ := http.NewRequestWithContext(context.Background(), "POST", "http://example.com", body)
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader("hello")), nil
		}

		if err := ensureReplayableBody(req); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("sets GetBody", func(t *testing.T) {
		t.Parallel()

		body := io.NopCloser(strings.NewReader("payload"))
		req, _ := http.NewRequestWithContext(context.Background(), "POST", "http://example.com", body)

		if err := ensureReplayableBody(req); err != nil {
			t.Fatalf("error: %v", err)
		}

		if req.GetBody == nil {
			t.Fatal("GetBody should be set")
		}

		// Verify body can be replayed
		newBody, err := req.GetBody()
		if err != nil {
			t.Fatalf("GetBody() error: %v", err)
		}

		data, _ := io.ReadAll(newBody)
		if string(data) != "payload" {
			t.Errorf("body = %q, want %q", data, "payload")
		}
	})
}

func TestCalculateBackoff(t *testing.T) {
	t.Parallel()

	rt := &RetryTransport{BaseDelay: time.Millisecond}

	t.Run("X-Rate-Limit-Reset priority (milliseconds)", func(t *testing.T) {
		t.Parallel()

		resp := &http.Response{Header: http.Header{}}
		resp.Header.Set(headerRateLimitReset, "3000")
		resp.Header.Set("Retry-After", "10")

		got := rt.calculateBackoff(0, resp)
		if got != 3*time.Second {
			t.Errorf("got %v, want 3s", got)
		}
	})

	t.Run("Retry-After fallback", func(t *testing.T) {
		t.Parallel()

		resp := &http.Response{Header: http.Header{}}
		resp.Header.Set("Retry-After", "5")

		got := rt.calculateBackoff(0, resp)
		if got != 5*time.Second {
			t.Errorf("got %v, want 5s", got)
		}
	})

	t.Run("exponential with jitter", func(t *testing.T) {
		t.Parallel()

		resp := &http.Response{Header: http.Header{}}

		got := rt.calculateBackoff(2, resp) // 2^2 * 1ms = 4ms base, jitter up to 2ms
		if got < time.Millisecond*4 || got > time.Millisecond*6 {
			t.Errorf("got %v, expected 4ms-6ms range", got)
		}
	})
}

func TestRoundTrip_200NoRetry(t *testing.T) {
	t.Parallel()

	var count atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		count.Add(1)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	rt := &RetryTransport{Base: srv.Client().Transport, BaseDelay: time.Millisecond}
	req, _ := http.NewRequestWithContext(context.Background(), "GET", srv.URL, nil)

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	resp.Body.Close()

	if count.Load() != 1 {
		t.Errorf("expected 1 request, got %d", count.Load())
	}
}

func TestRoundTrip_400NoRetry(t *testing.T) {
	t.Parallel()

	var count atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		count.Add(1)
		w.WriteHeader(400)
	}))
	defer srv.Close()

	rt := &RetryTransport{Base: srv.Client().Transport, BaseDelay: time.Millisecond}
	req, _ := http.NewRequestWithContext(context.Background(), "GET", srv.URL, nil)

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	resp.Body.Close()

	if count.Load() != 1 {
		t.Errorf("expected 1 request, got %d", count.Load())
	}
}

func TestRoundTrip_429RetrySuccess(t *testing.T) {
	t.Parallel()

	var count atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := count.Add(1)
		if n == 1 {
			w.WriteHeader(429)
			return
		}

		w.WriteHeader(200)
	}))
	defer srv.Close()

	rt := &RetryTransport{
		Base:          srv.Client().Transport,
		MaxRetries429: 3,
		BaseDelay:     time.Millisecond,
	}
	req, _ := http.NewRequestWithContext(context.Background(), "GET", srv.URL, nil)

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	if count.Load() != 2 {
		t.Errorf("expected 2 requests, got %d", count.Load())
	}
}

func TestRoundTrip_429ExhaustRetries(t *testing.T) {
	t.Parallel()

	var count atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		count.Add(1)
		w.WriteHeader(429)
	}))
	defer srv.Close()

	rt := &RetryTransport{
		Base:          srv.Client().Transport,
		MaxRetries429: 2,
		BaseDelay:     time.Millisecond,
	}
	req, _ := http.NewRequestWithContext(context.Background(), "GET", srv.URL, nil)

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	resp.Body.Close()

	if resp.StatusCode != 429 {
		t.Errorf("status = %d, want 429", resp.StatusCode)
	}

	// Initial + 2 retries = 3
	if count.Load() != 3 {
		t.Errorf("expected 3 requests, got %d", count.Load())
	}
}

func TestRoundTrip_5xxRetrySuccess(t *testing.T) {
	t.Parallel()

	var count atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := count.Add(1)
		if n == 1 {
			w.WriteHeader(500)
			return
		}

		w.WriteHeader(200)
	}))
	defer srv.Close()

	rt := &RetryTransport{
		Base:          srv.Client().Transport,
		MaxRetries5xx: 2,
		BaseDelay:     time.Millisecond,
	}
	req, _ := http.NewRequestWithContext(context.Background(), "GET", srv.URL, nil)

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestRoundTrip_5xxExhaustRetries(t *testing.T) {
	t.Parallel()

	var count atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		count.Add(1)
		w.WriteHeader(502)
	}))
	defer srv.Close()

	rt := &RetryTransport{
		Base:          srv.Client().Transport,
		MaxRetries5xx: 1,
		BaseDelay:     time.Millisecond,
	}
	req, _ := http.NewRequestWithContext(context.Background(), "GET", srv.URL, nil)

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	resp.Body.Close()

	if resp.StatusCode != 502 {
		t.Errorf("status = %d, want 502", resp.StatusCode)
	}

	// Initial + 1 retry = 2
	if count.Load() != 2 {
		t.Errorf("expected 2 requests, got %d", count.Load())
	}
}

func TestRoundTrip_ContextCancellation(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(429)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	rt := &RetryTransport{
		Base:          srv.Client().Transport,
		MaxRetries429: 3,
		BaseDelay:     time.Second, // Long delay to ensure context cancellation triggers
	}
	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL, nil)

	resp, err := rt.RoundTrip(req)
	if err == nil {
		resp.Body.Close()
		t.Fatal("expected error from cancelled context")
	}
}

func TestRoundTrip_BodyReplay(t *testing.T) {
	t.Parallel()

	var bodies []string
	var count atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(b))

		n := count.Add(1)
		if n == 1 {
			w.WriteHeader(429)
			return
		}

		w.WriteHeader(200)
	}))
	defer srv.Close()

	rt := &RetryTransport{
		Base:          srv.Client().Transport,
		MaxRetries429: 3,
		BaseDelay:     time.Millisecond,
	}
	body := bytes.NewReader([]byte(`{"key":"value"}`))
	req, _ := http.NewRequestWithContext(context.Background(), "POST", srv.URL, io.NopCloser(body))

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	resp.Body.Close()

	if len(bodies) != 2 {
		t.Fatalf("expected 2 bodies, got %d", len(bodies))
	}

	if bodies[0] != bodies[1] {
		t.Errorf("body not replayed: %q vs %q", bodies[0], bodies[1])
	}
}

func TestRoundTrip_NilBodyGET(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	rt := &RetryTransport{Base: srv.Client().Transport, BaseDelay: time.Millisecond}
	req, _ := http.NewRequestWithContext(context.Background(), "GET", srv.URL, nil)

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}
