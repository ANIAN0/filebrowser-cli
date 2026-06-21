package httpclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"
)

// Default timeout values.
const (
	DefaultConnectTimeout = 10 * time.Second
	DefaultRequestTimeout = 60 * time.Second
	DefaultSSETimeout     = 5 * time.Minute
	DefaultMaxRetries     = 3
)

// Client is an HTTP client with retry logic and token support.
type Client struct {
	// BaseURL is the base URL for all requests.
	BaseURL string

	// HTTP is the underlying HTTP client.
	HTTP *http.Client

	// Token is the authentication token (optional).
	Token string

	// Verbose enables verbose logging to stderr.
	Verbose bool

	// MaxRetries is the maximum number of retries for 5xx and network errors.
	MaxRetries int

	// AuthHeader is the header name for authentication (default: "X-Auth" for FileBrowser).
	AuthHeader string

	// OnAuthFailure, when non-nil, is invoked on 401 Unauthorized responses so
	// the caller can refresh the token and retry. Returning a new token
	// triggers exactly one immediate retry of the request with that token;
	// returning an error surfaces the refresh failure to the caller.
	//
	// Only one refresh happens per Do() call regardless of MaxRetries — auth
	// failure and server overload are independent failure modes, and capping
	// the refresh prevents a misconfigured hook from looping forever.
	OnAuthFailure func() (newToken string, err error)
}

// Option configures the Client.
type Option func(*Client)

// WithTimeout sets the request timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.HTTP.Timeout = d }
}

// WithToken sets the authentication token.
func WithToken(t string) Option {
	return func(c *Client) { c.Token = t }
}

// WithVerbose enables verbose logging.
func WithVerbose(v bool) Option {
	return func(c *Client) { c.Verbose = v }
}

// WithMaxRetries sets the maximum number of retries.
func WithMaxRetries(n int) Option {
	return func(c *Client) { c.MaxRetries = n }
}

// WithAuthHeader sets the authentication header name.
func WithAuthHeader(h string) Option {
	return func(c *Client) { c.AuthHeader = h }
}

// WithAuthFailure registers a callback invoked on 401 Unauthorized responses.
// See Client.OnAuthFailure for the contract; in short, returning a new token
// triggers one immediate retry, returning an error surfaces the failure.
func WithAuthFailure(fn func() (newToken string, err error)) Option {
	return func(c *Client) { c.OnAuthFailure = fn }
}

// New creates a new Client with the given base URL and options.
func New(baseURL string, opts ...Option) *Client {
	c := &Client{
		BaseURL: baseURL,
		HTTP: &http.Client{
			Timeout: DefaultRequestTimeout,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   DefaultConnectTimeout,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 30 * time.Second,
				DisableKeepAlives:     true,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
			},
		},
		MaxRetries: DefaultMaxRetries,
		AuthHeader: "X-Auth", // Default for FileBrowser
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Do executes an HTTP request with retries on 5xx and network errors, and
// with an optional one-shot auth refresh on 401 Unauthorized responses.
//
// Retry semantics:
//   - 5xx and network-class failures: retried up to MaxRetries times with
//     exponential backoff.
//   - 401 (when OnAuthFailure is set): one refresh + immediate retry,
//     decoupled from MaxRetries, with no backoff (token rejection is a fast
//     path; the user already paid for a round trip).
//
// The 401 retry does not consume a MaxRetries slot and cannot loop, because
// the authRetried latch caps it at one refresh per Do() call.
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	// Set auth header once: req.WithContext returns a shallow *Request copy
	// that shares the header map, so mutating req.Header (or c.Token) keeps
	// every retry attempt in sync without re-setting per loop iteration.
	if c.Token != "" {
		req.Header.Set(c.AuthHeader, c.Token)
	}

	var (
		lastErr         error
		authRetried     bool
		authJustRetried bool
	)
	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		// Backoff applies to 5xx / network retries. The 401 refresh path
		// sets authJustRetried and 'continue's, so the next iteration skips
		// this wait — keeping token-rejection-recovery instant.
		if attempt > 0 && !authJustRetried {
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			if c.Verbose {
				fmt.Fprintf(os.Stderr, "Retry %d/%d after %v backoff\n", attempt, c.MaxRetries, backoff)
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}
		authJustRetried = false

		resp, err := c.HTTP.Do(req.WithContext(ctx))
		if err != nil {
			lastErr = err
			if isNetworkError(err) && attempt < c.MaxRetries {
				if c.Verbose {
					fmt.Fprintf(os.Stderr, "Network error (attempt %d/%d): %v\n", attempt+1, c.MaxRetries, err)
				}
				continue
			}
			return nil, err
		}

		// 401 Unauthorized: optional one-shot auth refresh + retry.
		// Capped by authRetried (single refresh per Do) so a misconfigured
		// refresh hook cannot loop forever; decoupled from MaxRetries so a
		// tight MaxRetries cannot strip the chance to recover from expiry.
		if resp.StatusCode == http.StatusUnauthorized && !authRetried && c.OnAuthFailure != nil {
			resp.Body.Close()
			newToken, aerr := c.OnAuthFailure()
			if aerr != nil {
				return nil, fmt.Errorf("auth refresh: %w", aerr)
			}
			c.Token = newToken
			req.Header.Set(c.AuthHeader, newToken)
			authRetried = true
			authJustRetried = true
			continue
		}

		// Success or client error - return immediately
		if resp.StatusCode < 500 {
			return resp, nil
		}

		// Server error - retry
		lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
		resp.Body.Close()
		if c.Verbose {
			fmt.Fprintf(os.Stderr, "Server error %d (attempt %d/%d)\n", resp.StatusCode, attempt+1, c.MaxRetries)
		}
	}

	return nil, lastErr
}

// isNetworkError checks if an error is a network-related error.
func isNetworkError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	// Also check for DNS errors, connection refused, etc.
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}
	return false
}

// Get is a convenience method for GET requests.
func (c *Client) Get(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(ctx, req)
}

// Post is a convenience method for POST requests.
func (c *Client) Post(ctx context.Context, path string, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+path, body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return c.Do(ctx, req)
}

// Put is a convenience method for PUT requests.
func (c *Client) Put(ctx context.Context, path string, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "PUT", c.BaseURL+path, body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return c.Do(ctx, req)
}

// Patch is a convenience method for PATCH requests.
func (c *Client) Patch(ctx context.Context, path string, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "PATCH", c.BaseURL+path, body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return c.Do(ctx, req)
}

// Delete is a convenience method for DELETE requests.
func (c *Client) Delete(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "DELETE", c.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(ctx, req)
}

// Download streams a response body to w.
func (c *Client) Download(ctx context.Context, path string, w io.Writer) error {
	resp, err := c.Get(ctx, path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	_, err = io.Copy(w, resp.Body)
	return err
}
