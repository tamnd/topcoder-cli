// Package topcoder is the library behind the tc command: the HTTP client,
// pacing, and the typed data models for the TopCoder v5 REST API.
//
// The public API at api.topcoder.com/v5 is open for reading: no API key is
// required for challenges and member profiles. This package wraps it with a
// sequential, rate-limited client that the kit operations consume.
package topcoder

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// DefaultUserAgent identifies the client to TopCoder.
const DefaultUserAgent = "tc/dev (+https://github.com/tamnd/topcoder-cli)"

// Host is the site this client targets.
const Host = "topcoder.com"

// APIURL is the base URL for the v5 REST API.
const APIURL = "https://api.topcoder.com/v5"

// Config holds constructor parameters for Client.
type Config struct {
	UserAgent string
	Rate      time.Duration
	Retries   int
	Timeout   time.Duration
}

// DefaultConfig returns sensible defaults for the TopCoder API.
func DefaultConfig() Config {
	return Config{
		UserAgent: DefaultUserAgent,
		Rate:      300 * time.Millisecond,
		Retries:   3,
		Timeout:   30 * time.Second,
	}
}

// Client is a rate-limited HTTP client for the TopCoder v5 API.
type Client struct {
	cfg  Config
	http *http.Client
	mu   sync.Mutex
	last time.Time
}

// NewClient returns a Client configured with cfg.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout},
	}
}

// get fetches a URL with retries on transient errors, returning the body.
func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, rawURL)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) (body []byte, retry bool, err error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, false, ErrNotFound
	}
	if resp.StatusCode == http.StatusServiceUnavailable {
		return nil, true, ErrBlocked
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, true, ErrRateLimited
	}
	if resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

// getWithHeader fetches a URL and also returns the response header value for key.
func (c *Client) getWithHeader(ctx context.Context, rawURL, headerKey string) ([]byte, string, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, "", ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, hdrVal, retry, err := c.doWithHeader(ctx, rawURL, headerKey)
		if err == nil {
			return body, hdrVal, nil
		}
		lastErr = err
		if !retry {
			return nil, "", err
		}
	}
	return nil, "", fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) doWithHeader(ctx context.Context, rawURL, headerKey string) (body []byte, hdrVal string, retry bool, err error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", true, err
	}
	defer func() { _ = resp.Body.Close() }()

	hdrVal = resp.Header.Get(headerKey)

	if resp.StatusCode == http.StatusNotFound {
		return nil, hdrVal, false, ErrNotFound
	}
	if resp.StatusCode == http.StatusServiceUnavailable {
		return nil, hdrVal, true, ErrBlocked
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, hdrVal, true, ErrRateLimited
	}
	if resp.StatusCode >= 500 {
		return nil, hdrVal, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, hdrVal, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, hdrVal, true, err
	}
	return b, hdrVal, false, nil
}

// pace blocks until at least Rate has passed since the previous request.
func (c *Client) pace() {
	if c.cfg.Rate <= 0 {
		return
	}
	c.mu.Lock()
	wait := c.cfg.Rate - time.Since(c.last)
	c.mu.Unlock()
	if wait > 0 {
		time.Sleep(wait)
	}
	c.mu.Lock()
	c.last = time.Now()
	c.mu.Unlock()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

// getJSON is a convenience that calls get and JSON-decodes into v.
func (c *Client) getJSON(ctx context.Context, rawURL string, v any) error {
	body, err := c.get(ctx, rawURL)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("decode %s: %w", rawURL, err)
	}
	return nil
}
