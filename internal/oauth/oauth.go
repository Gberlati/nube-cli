package oauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gberlati/nube-cli/internal/config"
)

const (
	// AuthBaseURL is the Tienda Nube authorization URL.
	AuthBaseURL = "https://www.tiendanube.com/apps"
	// TokenURL is the Tienda Nube token exchange endpoint.
	TokenURL = "https://www.tiendanube.com/apps/authorize/token" //nolint:gosec // URL, not a credential
)

// AuthorizeOptions configures the OAuth flow.
type AuthorizeOptions struct {
	Manual  bool
	Remote  bool
	Step    int
	AuthURL string
	Timeout time.Duration
	Client  string
}

// TokenResponse holds the response from the Tienda Nube token endpoint.
type TokenResponse struct {
	AccessToken string      `json:"access_token"` //nolint:gosec // JSON field name
	TokenType   string      `json:"token_type"`
	Scope       string      `json:"scope"`
	UserID      json.Number `json:"user_id"`
}

var (
	errMissingCode    = errors.New("missing authorization code")
	errMissingAuthURL = errors.New("missing auth URL for remote step 2")
	errStateMismatch  = errors.New("state mismatch (possible CSRF attack)")
	errAuthorization  = errors.New("authorization error")
	errTokenExchange  = errors.New("token exchange failed")
	errNoAccessToken  = errors.New("no access token in response")

	readClientCredentials = config.ReadClientCredentialsFor
	openBrowserFn         = openBrowser
)

// Authorize runs the Tienda Nube OAuth2 flow and returns a TokenResponse.
func Authorize(ctx context.Context, opts AuthorizeOptions) (TokenResponse, error) {
	if opts.Timeout <= 0 {
		opts.Timeout = 2 * time.Minute
	}

	creds, err := readClientCredentials(opts.Client)
	if err != nil {
		return TokenResponse{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	if opts.Manual || opts.Remote || opts.AuthURL != "" {
		return authorizeManual(ctx, opts, creds)
	}

	return authorizeServer(ctx, opts, creds)
}

// ManualAuthURL returns the authorization URL for the manual flow
// (used with --remote --step 1).
func ManualAuthURL(clientName string) (string, error) {
	creds, err := readClientCredentials(clientName)
	if err != nil {
		return "", err
	}

	return authURL(creds.ClientID), nil
}

func authURL(clientID string) string {
	return fmt.Sprintf("%s/%s/authorize", AuthBaseURL, clientID)
}

func authorizeServer(ctx context.Context, _ AuthorizeOptions, creds config.ClientCredentials) (TokenResponse, error) {
	state, err := randomState()
	if err != nil {
		return TokenResponse{}, err
	}

	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return TokenResponse{}, fmt.Errorf("listen for callback: %w", err)
	}

	defer func() { _ = ln.Close() }()

	port := ln.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	srv := &http.Server{
		ReadHeaderTimeout: 5 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/callback" {
				http.NotFound(w, r)
				return
			}

			q := r.URL.Query()

			if q.Get("error") != "" {
				select {
				case errCh <- fmt.Errorf("%w: %s", errAuthorization, q.Get("error")):
				default:
				}

				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, "Authorization failed. You can close this window.")

				return
			}

			if q.Get("state") != state {
				select {
				case errCh <- errStateMismatch:
				default:
				}

				w.WriteHeader(http.StatusBadRequest)
				_, _ = fmt.Fprint(w, "State mismatch. Please try again.")

				return
			}

			code := q.Get("code")
			if code == "" {
				select {
				case errCh <- errMissingCode:
				default:
				}

				w.WriteHeader(http.StatusBadRequest)
				_, _ = fmt.Fprint(w, "Missing authorization code.")

				return
			}

			select {
			case codeCh <- code:
			default:
			}

			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, "Authorization successful! You can close this window.")
		}),
	}

	go func() {
		<-ctx.Done()
		_ = srv.Close()
	}()

	go func() {
		if serveErr := srv.Serve(ln); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			select {
			case errCh <- serveErr:
			default:
			}
		}
	}()

	fullAuthURL := fmt.Sprintf("%s?redirect_uri=%s&state=%s",
		authURL(creds.ClientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(state))

	fmt.Fprintln(os.Stderr, "Opening browser for authorization...")
	fmt.Fprintln(os.Stderr, "If the browser doesn't open, visit this URL:")
	fmt.Fprintln(os.Stderr, fullAuthURL)
	_ = openBrowserFn(fullAuthURL)

	select {
	case code := <-codeCh:
		fmt.Fprintln(os.Stderr, "Authorization received. Exchanging code...")

		tok, exchangeErr := exchangeCode(ctx, creds, code)
		if exchangeErr != nil {
			_ = srv.Close()
			return TokenResponse{}, exchangeErr
		}

		shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 2*time.Second)
		defer shutdownCancel()
		_ = srv.Shutdown(shutdownCtx)

		return tok, nil
	case err := <-errCh:
		_ = srv.Close()
		return TokenResponse{}, err
	case <-ctx.Done():
		_ = srv.Close()
		return TokenResponse{}, fmt.Errorf("authorization timed out: %w", ctx.Err())
	}
}

func authorizeManual(ctx context.Context, opts AuthorizeOptions, creds config.ClientCredentials) (TokenResponse, error) {
	// Remote step 1: print auth URL.
	if opts.Remote && opts.Step == 1 {
		fmt.Fprintln(os.Stderr, "Visit this URL to authorize:")
		fmt.Fprintln(os.Stdout, authURL(creds.ClientID))
		fmt.Fprintln(os.Stderr, "\nAfter authorizing, run again with --remote --step 2 --auth-url <redirect-url>")

		return TokenResponse{}, &StepOneComplete{}
	}

	// Remote step 2 or manual flow with --auth-url.
	authURLInput := strings.TrimSpace(opts.AuthURL)
	if opts.Remote && opts.Step == 2 && authURLInput == "" {
		return TokenResponse{}, errMissingAuthURL
	}

	if authURLInput != "" {
		code, parseErr := extractCodeFromURL(authURLInput)
		if parseErr != nil {
			return TokenResponse{}, parseErr
		}

		return exchangeCode(ctx, creds, code)
	}

	// Interactive manual: print URL and ask user to paste code.
	fmt.Fprintln(os.Stderr, "Visit this URL to authorize:")
	fmt.Fprintln(os.Stderr, authURL(creds.ClientID))
	fmt.Fprintln(os.Stderr)
	fmt.Fprint(os.Stderr, "Paste the authorization code: ")

	var code string
	if _, err := fmt.Fscan(os.Stdin, &code); err != nil {
		if errors.Is(err, io.EOF) {
			return TokenResponse{}, fmt.Errorf("authorization cancelled: %w", context.Canceled)
		}

		return TokenResponse{}, fmt.Errorf("read code: %w", err)
	}

	code = strings.TrimSpace(code)
	if code == "" {
		return TokenResponse{}, errMissingCode
	}

	return exchangeCode(ctx, creds, code)
}

func exchangeCode(ctx context.Context, creds config.ClientCredentials, code string) (TokenResponse, error) {
	data := url.Values{
		"client_id":     {creds.ClientID},
		"client_secret": {creds.ClientSecret},
		"grant_type":    {"authorization_code"},
		"code":          {code},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return TokenResponse{}, fmt.Errorf("create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req) //nolint:gosec // token endpoint URL is constant
	if err != nil {
		return TokenResponse{}, fmt.Errorf("token exchange: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return TokenResponse{}, fmt.Errorf("%w (HTTP %d): %s", errTokenExchange, resp.StatusCode, string(body))
	}

	var tok TokenResponse
	if decodeErr := json.NewDecoder(resp.Body).Decode(&tok); decodeErr != nil {
		return TokenResponse{}, fmt.Errorf("decode token response: %w", decodeErr)
	}

	if tok.AccessToken == "" {
		return TokenResponse{}, errNoAccessToken
	}

	return tok, nil
}

func extractCodeFromURL(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse URL: %w", err)
	}

	code := parsed.Query().Get("code")
	if code == "" {
		// Maybe the user pasted just the code.
		code = strings.TrimSpace(rawURL)
	}

	if code == "" {
		return "", errMissingCode
	}

	return code, nil
}

func randomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate state: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}

// StepOneComplete indicates --remote --step 1 completed (URL was printed).
type StepOneComplete struct{}

func (e *StepOneComplete) Error() string { return "step 1 complete" }
