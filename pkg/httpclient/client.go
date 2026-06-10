package httpclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
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
				ResponseHeaderTimeout:  30 * time.Second,
				DisableKeepAlives:      true,
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

// Do executes an HTTP request with retries on 5xx and network errors.
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	// Set auth header if token is provided
	if c.Token != "" {
		req.Header.Set(c.AuthHeader, c.Token)
	}

	var lastErr error
	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		// Add backoff delay for retries
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			if c.Verbose {
				fmt.Printf("Retry %d/%d after %v backoff\n", attempt, c.MaxRetries, backoff)
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		resp, err := c.HTTP.Do(req.WithContext(ctx))
		if err != nil {
			lastErr = err
			if isNetworkError(err) && attempt < c.MaxRetries {
				if c.Verbose {
					fmt.Printf("Network error (attempt %d/%d): %v\n", attempt+1, c.MaxRetries, err)
				}
				continue
			}
			return nil, err
		}

		// Success or client error - return immediately
		if resp.StatusCode < 500 {
			return resp, nil
		}

		// Server error - retry
		lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
		resp.Body.Close()
		if c.Verbose {
			fmt.Printf("Server error %d (attempt %d/%d)\n", resp.StatusCode, attempt+1, c.MaxRetries)
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