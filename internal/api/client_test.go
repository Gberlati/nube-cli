package api_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gberlati/nube-cli/internal/api"
)

func newTestClient(t *testing.T, handler http.Handler) *api.Client {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	return api.New("12345", "test-token",
		api.WithBaseURL(srv.URL),
		api.WithHTTPClient(srv.Client()),
	)
}

func TestClient_Get(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotQuery string

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery

		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}))

	resp, err := c.Get(context.Background(), "products", url.Values{"page": {"2"}})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	resp.Body.Close()

	if gotMethod != "GET" {
		t.Errorf("method = %q, want GET", gotMethod)
	}

	if gotPath != "/12345/products" {
		t.Errorf("path = %q, want /12345/products", gotPath)
	}

	if gotQuery != "page=2" {
		t.Errorf("query = %q, want page=2", gotQuery)
	}
}

func TestClient_Post(t *testing.T) {
	t.Parallel()

	var gotMethod string

	var gotBody string

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)

		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{}`))
	}))

	resp, err := c.Post(context.Background(), "products", strings.NewReader(`{"name":"test"}`))
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}

	resp.Body.Close()

	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}

	if gotBody != `{"name":"test"}` {
		t.Errorf("body = %q", gotBody)
	}
}

func TestClient_Put(t *testing.T) {
	t.Parallel()

	var gotMethod string

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method

		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}))

	resp, err := c.Put(context.Background(), "products/1", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	resp.Body.Close()

	if gotMethod != "PUT" {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
}

func TestClient_Delete(t *testing.T) {
	t.Parallel()

	var gotMethod string

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method

		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}))

	resp, err := c.Delete(context.Background(), "products/1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	resp.Body.Close()

	if gotMethod != "DELETE" {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
}

func TestClient_AuthenticationHeader(t *testing.T) {
	t.Parallel()

	var gotAuth string

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authentication")

		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}))

	resp, err := c.Get(context.Background(), "products", nil)
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	resp.Body.Close()

	// Regression guard: Tienda Nube uses "Authentication", NOT "Authorization"
	if gotAuth != "bearer test-token" {
		t.Errorf("Authentication header = %q, want %q", gotAuth, "bearer test-token")
	}
}

func TestClient_UserAgent(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		t.Parallel()

		var gotUA string

		c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotUA = r.Header.Get("User-Agent")

			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{}`))
		}))

		resp, _ := c.Get(context.Background(), "x", nil)
		resp.Body.Close()

		if gotUA != api.DefaultUserAgent {
			t.Errorf("User-Agent = %q, want %q", gotUA, api.DefaultUserAgent)
		}
	})

	t.Run("override", func(t *testing.T) {
		t.Parallel()

		var gotUA string

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotUA = r.Header.Get("User-Agent")

			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{}`))
		}))
		defer srv.Close()

		c := api.New("1", "tok",
			api.WithBaseURL(srv.URL),
			api.WithHTTPClient(srv.Client()),
			api.WithUserAgent("custom-agent/1.0"),
		)

		resp, _ := c.Get(context.Background(), "x", nil)
		resp.Body.Close()

		if gotUA != "custom-agent/1.0" {
			t.Errorf("User-Agent = %q, want %q", gotUA, "custom-agent/1.0")
		}
	})
}

func TestClient_URLConstruction(t *testing.T) {
	t.Parallel()

	var gotURL string

	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.Path

		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}))

	resp, err := c.Get(context.Background(), "products/categories", nil)
	if err != nil {
		t.Fatalf("error = %v", err)
	}

	resp.Body.Close()

	if gotURL != "/12345/products/categories" {
		t.Errorf("URL path = %q, want /12345/products/categories", gotURL)
	}
}

func TestDecodeResponse(t *testing.T) {
	t.Parallel()

	type product struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"id":1,"name":"Widget"}`))
		}))

		resp, err := c.Get(context.Background(), "products/1", nil) //nolint:bodyclose // DecodeResponse closes body
		if err != nil {
			t.Fatalf("error = %v", err)
		}

		p, err := api.DecodeResponse[product](resp)
		if err != nil {
			t.Fatalf("DecodeResponse error = %v", err)
		}

		if p.ID != 1 || p.Name != "Widget" {
			t.Errorf("decoded = %+v", p)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		t.Parallel()

		c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{invalid}`))
		}))

		resp, err := c.Get(context.Background(), "x", nil) //nolint:bodyclose // DecodeResponse closes body
		if err != nil {
			t.Fatalf("error = %v", err)
		}

		_, err = api.DecodeResponse[product](resp)
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
	})
}

func TestClient_ErrorResponses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		status     int
		body       string
		checkError func(t *testing.T, err error)
	}{
		{
			name:   "401 auth error",
			status: 401,
			body:   `{"message":"unauthorized"}`,
			checkError: func(t *testing.T, err error) {
				t.Helper()

				if !api.IsAuthError(err) {
					t.Errorf("expected AuthError, got %T: %v", err, err)
				}
			},
		},
		{
			name:   "404 not found",
			status: 404,
			body:   `{"message":"not found"}`,
			checkError: func(t *testing.T, err error) {
				t.Helper()

				if !api.IsNotFoundError(err) {
					t.Errorf("expected NotFoundError, got %T: %v", err, err)
				}
			},
		},
		{
			name:   "402 payment required",
			status: 402,
			body:   `{"message":"subscription suspended"}`,
			checkError: func(t *testing.T, err error) {
				t.Helper()

				if !api.IsPaymentRequiredError(err) {
					t.Errorf("expected PaymentRequiredError, got %T: %v", err, err)
				}
			},
		},
		{
			name:   "403 permission denied",
			status: 403,
			body:   `{"message":"insufficient scope"}`,
			checkError: func(t *testing.T, err error) {
				t.Helper()

				if !api.IsPermissionDeniedError(err) {
					t.Errorf("expected PermissionDeniedError, got %T: %v", err, err)
				}
			},
		},
		{
			name:   "400 simple error format",
			status: 400,
			body:   `{"error":"Problems parsing JSON"}`,
			checkError: func(t *testing.T, err error) {
				t.Helper()

				if !api.IsAPIError(err) {
					t.Errorf("expected APIError, got %T: %v", err, err)
				}

				if !strings.Contains(err.Error(), "Problems parsing JSON") {
					t.Errorf("error = %q, want containing 'Problems parsing JSON'", err.Error())
				}
			},
		},
		{
			name:   "422 field validation",
			status: 422,
			body:   `{"src":["can't be blank"],"name":["is too long"]}`,
			checkError: func(t *testing.T, err error) {
				t.Helper()

				if !api.IsValidationError(err) {
					t.Errorf("expected ValidationError, got %T: %v", err, err)
				}
			},
		},
		{
			name:   "422 business error",
			status: 422,
			body:   `{"code":422,"message":"Unprocessable Entity","description":"some reason"}`,
			checkError: func(t *testing.T, err error) {
				t.Helper()

				// Business errors with code+message should NOT be ValidationError.
				if api.IsValidationError(err) {
					t.Errorf("expected APIError not ValidationError, got %T: %v", err, err)
				}

				if !api.IsAPIError(err) {
					t.Errorf("expected APIError, got %T: %v", err, err)
				}
			},
		},
		{
			name:   "500 api error",
			status: 500,
			body:   `{"message":"internal error"}`,
			checkError: func(t *testing.T, err error) {
				t.Helper()

				if !api.IsAPIError(err) {
					t.Errorf("expected APIError, got %T: %v", err, err)
				}
			},
		},
		{
			name:   "500 non-json body",
			status: 500,
			body:   `Internal Server Error`,
			checkError: func(t *testing.T, err error) {
				t.Helper()

				if !api.IsAPIError(err) {
					t.Errorf("expected APIError, got %T: %v", err, err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			// Use plain http.Client (no retry transport) for error tests
			c := api.New("1", "tok",
				api.WithBaseURL(srv.URL),
				api.WithHTTPClient(&http.Client{}),
			)

			_, err := c.Get(context.Background(), "x", nil) //nolint:bodyclose // parseErrorResponse closes body
			if err == nil {
				t.Fatal("expected error")
			}

			tt.checkError(t, err)
		})
	}
}

func TestClient_ErrorResponseBodyParsing(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(422)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"code":    "validation_error",
			"message": "name is required",
		})
	}))
	defer srv.Close()

	c := api.New("1", "tok",
		api.WithBaseURL(srv.URL),
		api.WithHTTPClient(&http.Client{}),
	)

	_, err := c.Get(context.Background(), "x", nil) //nolint:bodyclose // parseErrorResponse closes body
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "name is required") {
		t.Errorf("error = %q, want containing 'name is required'", err.Error())
	}
}
