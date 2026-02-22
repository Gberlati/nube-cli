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

	"github.com/gberlati/nube-cli/internal/credstore"
)

const (
	// AuthBaseURL is the Tienda Nube authorization URL.
	AuthBaseURL = "https://www.tiendanube.com/apps"
	// TokenURL is the Tienda Nube token exchange endpoint.
	TokenURL = "https://www.tiendanube.com/apps/authorize/token" //nolint:gosec // URL, not a credential
	// CallbackPort is the fixed port for the local OAuth callback server.
	CallbackPort = 8910
	// DefaultBrokerURL is the default OAuth broker URL.
	// Set to the deployed Cloudflare Worker URL. See broker/ for the worker source.
	DefaultBrokerURL = "nube-cli-auth-broker.gonzaloberlati.workers.dev"
)

// AuthorizeOptions configures the OAuth flow.
type AuthorizeOptions struct {
	Timeout   time.Duration
	OAuthApp  string
	BrokerURL string
}

// TokenResponse holds the response from the Tienda Nube token endpoint.
type TokenResponse struct {
	AccessToken string      `json:"access_token"` //nolint:gosec // JSON field name
	TokenType   string      `json:"token_type"`
	Scope       string      `json:"scope"`
	UserID      json.Number `json:"user_id"`
}

// authResult is the result received from the local callback server.
// For the broker flow, token and userID are set directly.
// For the native flow, code is set and must be exchanged for a token.
type authResult struct {
	code   string // native flow: authorization code
	token  string // broker flow: access token
	userID string // broker flow: user ID
}

// clientCredentials holds OAuth client ID and secret for the native flow.
type clientCredentials struct {
	clientID     string
	clientSecret string
}

var (
	errMissingCode   = errors.New("missing authorization code")
	errStateMismatch = errors.New("state mismatch (possible CSRF attack)")
	errAuthorization = errors.New("authorization error")
	errTokenExchange = errors.New("token exchange failed")
	errNoAccessToken = errors.New("no access token in response")

	readOAuthClient = defaultReadOAuthClient
	openBrowserFn   = openBrowser
)

func defaultReadOAuthClient(name string) (clientCredentials, error) {
	c, err := credstore.GetOAuthClient(name)
	if err != nil {
		return clientCredentials{}, err
	}

	return clientCredentials{clientID: c.ClientID, clientSecret: c.ClientSecret}, nil
}

// Authorize runs the Tienda Nube OAuth2 flow and returns a TokenResponse.
func Authorize(ctx context.Context, opts AuthorizeOptions) (TokenResponse, error) {
	if opts.Timeout <= 0 {
		opts.Timeout = 2 * time.Minute
	}

	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	brokerURL := opts.BrokerURL
	if brokerURL != "" {
		// Broker flow — no local credentials needed.
		return authorizeServer(ctx, opts, clientCredentials{}, brokerURL)
	}

	// Try native flow — requires local credentials.
	appName := opts.OAuthApp
	if appName == "" {
		appName = "default"
	}

	creds, err := readOAuthClient(appName)
	if err != nil {
		// No credentials and no broker URL explicitly set — fall back to default broker.
		if DefaultBrokerURL != "" {
			var credErr *credstore.OAuthClientMissingError
			if errors.As(err, &credErr) {
				return authorizeServer(ctx, opts, clientCredentials{}, DefaultBrokerURL)
			}
		}

		return TokenResponse{}, err
	}

	return authorizeServer(ctx, opts, creds, "")
}

func authURL(clientID string) string {
	return fmt.Sprintf("%s/%s/authorize", AuthBaseURL, clientID)
}

func authorizeServer(ctx context.Context, _ AuthorizeOptions, creds clientCredentials, brokerURL string) (TokenResponse, error) {
	isBroker := brokerURL != ""

	var state string

	if !isBroker {
		var err error

		state, err = randomState()
		if err != nil {
			return TokenResponse{}, err
		}
	}

	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", fmt.Sprintf("127.0.0.1:%d", CallbackPort))
	if err != nil {
		return TokenResponse{}, fmt.Errorf("listen for callback: %w", err)
	}

	defer func() { _ = ln.Close() }()

	resultCh := make(chan authResult, 1)
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

			// Broker flow: token + user_id come directly.
			if tok := q.Get("token"); tok != "" {
				select {
				case resultCh <- authResult{token: tok, userID: q.Get("user_id")}:
				default:
				}

				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, "Authorization successful! You can close this window.")

				return
			}

			// Native flow: code + state validation.
			if !isBroker {
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
				case resultCh <- authResult{code: code}:
				default:
				}

				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, "Authorization successful! You can close this window.")

				return
			}

			// Broker flow but no token — unexpected.
			select {
			case errCh <- errNoAccessToken:
			default:
			}

			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprint(w, "Missing token in callback.")
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

	// Build the authorization URL.
	var fullAuthURL string
	if isBroker {
		fullAuthURL = fmt.Sprintf("%s/start?port=%d", brokerURL, CallbackPort)
	} else {
		redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", CallbackPort)
		fullAuthURL = fmt.Sprintf("%s?redirect_uri=%s&state=%s",
			authURL(creds.clientID),
			url.QueryEscape(redirectURI),
			url.QueryEscape(state))
	}

	fmt.Fprintln(os.Stderr, "Opening browser for authorization...")
	fmt.Fprintln(os.Stderr, "If the browser doesn't open, visit this URL:")
	fmt.Fprintln(os.Stderr, fullAuthURL)
	_ = openBrowserFn(fullAuthURL)

	select {
	case res := <-resultCh:
		if res.token != "" {
			// Broker flow — token received directly.
			shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 2*time.Second)
			defer shutdownCancel()
			_ = srv.Shutdown(shutdownCtx)

			return TokenResponse{
				AccessToken: res.token,
				UserID:      json.Number(res.userID),
			}, nil
		}

		// Native flow — exchange authorization code.
		fmt.Fprintln(os.Stderr, "Authorization received. Exchanging code...")

		tok, exchangeErr := exchangeCode(ctx, creds, res.code)
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

func exchangeCode(ctx context.Context, creds clientCredentials, code string) (TokenResponse, error) {
	data := url.Values{
		"client_id":     {creds.clientID},
		"client_secret": {creds.clientSecret},
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

func randomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate state: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}
