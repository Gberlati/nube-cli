package api_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gberlati/nube-cli/internal/api"
)

func TestParseLinkHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		header string
		check  func(t *testing.T, info api.PageInfo)
	}{
		{
			name:   "single next",
			header: `<https://api.example.com/v1/123/products?page=2>; rel="next"`,
			check: func(t *testing.T, info api.PageInfo) {
				t.Helper()

				if info.Next != "https://api.example.com/v1/123/products?page=2" {
					t.Errorf("Next = %q", info.Next)
				}
			},
		},
		{
			name:   "multiple rels",
			header: `<http://a.com?page=2>; rel="next", <http://a.com?page=1>; rel="prev"`,
			check: func(t *testing.T, info api.PageInfo) {
				t.Helper()

				if info.Next != "http://a.com?page=2" {
					t.Errorf("Next = %q", info.Next)
				}

				if info.Prev != "http://a.com?page=1" {
					t.Errorf("Prev = %q", info.Prev)
				}
			},
		},
		{
			name:   "all four rels",
			header: `<http://a.com?page=2>; rel="next", <http://a.com?page=1>; rel="prev", <http://a.com?page=1>; rel="first", <http://a.com?page=5>; rel="last"`,
			check: func(t *testing.T, info api.PageInfo) {
				t.Helper()

				if info.Next == "" || info.Prev == "" || info.First == "" || info.Last == "" {
					t.Errorf("expected all rels, got %+v", info)
				}
			},
		},
		{
			name:   "empty",
			header: "",
			check: func(t *testing.T, info api.PageInfo) {
				t.Helper()

				if info.HasNext() {
					t.Error("empty header should not have next")
				}
			},
		},
		{
			name:   "malformed no semicolon",
			header: "not a link header",
			check: func(t *testing.T, info api.PageInfo) {
				t.Helper()

				if info.HasNext() {
					t.Error("malformed should not have next")
				}
			},
		},
		{
			name:   "whitespace handling",
			header: `  <http://a.com?page=2> ; rel="next"  `,
			check: func(t *testing.T, info api.PageInfo) {
				t.Helper()

				if info.Next != "http://a.com?page=2" {
					t.Errorf("Next = %q", info.Next)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			info := api.ParseLinkHeader(tt.header)
			tt.check(t, info)
		})
	}
}

func TestPageInfo_HasNext(t *testing.T) {
	t.Parallel()

	if (api.PageInfo{}).HasNext() {
		t.Error("empty should not have next")
	}

	if !(api.PageInfo{Next: "http://a.com"}).HasNext() {
		t.Error("with Next should have next")
	}
}

func TestCollectAllPages(t *testing.T) {
	t.Parallel()

	type item struct {
		ID int `json:"id"`
	}

	t.Run("3 pages", func(t *testing.T) {
		t.Parallel()

		page := 0

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			page++
			switch page {
			case 1:
				w.Header().Set("Link", fmt.Sprintf(`<%s/12345/products?page=2>; rel="next"`, "http://"+r.Host))
				_, _ = w.Write([]byte(`[{"id":1},{"id":2}]`))
			case 2:
				w.Header().Set("Link", fmt.Sprintf(`<%s/12345/products?page=3>; rel="next"`, "http://"+r.Host))
				_, _ = w.Write([]byte(`[{"id":3}]`))
			case 3:
				_, _ = w.Write([]byte(`[{"id":4}]`))
			}
		}))
		defer srv.Close()

		c := api.New("12345", "tok",
			api.WithBaseURL(srv.URL),
			api.WithHTTPClient(srv.Client()),
		)

		items, err := api.CollectAllPages(context.Background(), c, "products", nil,
			func(resp *http.Response) ([]item, error) {
				return api.DecodeResponse[[]item](resp)
			},
		)
		if err != nil {
			t.Fatalf("error = %v", err)
		}

		if len(items) != 4 {
			t.Errorf("got %d items, want 4", len(items))
		}
	})

	t.Run("single page", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`[{"id":1}]`))
		}))
		defer srv.Close()

		c := api.New("1", "tok",
			api.WithBaseURL(srv.URL),
			api.WithHTTPClient(srv.Client()),
		)

		items, err := api.CollectAllPages(context.Background(), c, "products", nil,
			func(resp *http.Response) ([]item, error) {
				return api.DecodeResponse[[]item](resp)
			},
		)
		if err != nil {
			t.Fatalf("error = %v", err)
		}

		if len(items) != 1 {
			t.Errorf("got %d items, want 1", len(items))
		}
	})

	t.Run("error on page 2", func(t *testing.T) {
		t.Parallel()

		page := 0

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			page++
			if page == 1 {
				w.Header().Set("Link", fmt.Sprintf(`<%s/1/products?page=2>; rel="next"`, "http://"+r.Host))
				_, _ = w.Write([]byte(`[{"id":1}]`))

				return
			}

			w.WriteHeader(500)
			_, _ = w.Write([]byte(`{"message":"error"}`))
		}))
		defer srv.Close()

		c := api.New("1", "tok",
			api.WithBaseURL(srv.URL),
			api.WithHTTPClient(&http.Client{}),
		)

		_, err := api.CollectAllPages(context.Background(), c, "products", nil,
			func(resp *http.Response) ([]item, error) {
				return api.DecodeResponse[[]item](resp)
			},
		)
		if err == nil {
			t.Fatal("expected error on page 2")
		}
	})
}
