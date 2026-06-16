// Package youku is the library behind the youku command line:
// the HTTP client, request shaping, and the typed data models for Youku
// (优酷, youku.com) — Alibaba's flagship video streaming platform.
//
// Youku exposes a public JSON search API at search.youku.com/api/search
// that returns structured metadata for videos and shows without authentication.
// Individual show detail pages at v.youku.com/v_show/id_<id>.html contain
// rich OGP metadata accessible via SSR HTML.
package youku

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultUserAgent mimics a browser to get SSR HTML from Youku pages.
	DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) " +
		"AppleWebKit/537.36 (KHTML, like Gecko) " +
		"Chrome/120.0.0.0 Safari/537.36"

	// Host is the main site hostname.
	Host = "youku.com"

	// SearchBase is the base URL for the JSON search API.
	SearchBase = "https://search.youku.com/api/search"

	// ShowBase is the base URL for individual show detail pages.
	ShowBase = "https://v.youku.com/v_show/id_"
)

// ErrNotFound signals that the requested show does not exist.
var ErrNotFound = fmt.Errorf("youku: not found")

// ErrRateLimited signals that the server is rate-limiting or blocking the client.
var ErrRateLimited = fmt.Errorf("youku: rate limited or blocked")

// Config holds constructor parameters for Client.
type Config struct {
	SearchBase string
	ShowBase   string
	UserAgent  string
	Rate       time.Duration
	Retries    int
	Timeout    time.Duration
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		SearchBase: SearchBase,
		ShowBase:   ShowBase,
		UserAgent:  DefaultUserAgent,
		Rate:       500 * time.Millisecond,
		Retries:    3,
		Timeout:    30 * time.Second,
	}
}

// Client is a rate-limited HTTP client for Youku.
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

// pace enforces the minimum gap between requests.
func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	elapsed := time.Since(c.last)
	if elapsed < c.cfg.Rate {
		time.Sleep(c.cfg.Rate - elapsed)
	}
	c.last = time.Now()
}

// getJSON fetches a URL and JSON-decodes into v.
func (c *Client) getJSON(ctx context.Context, url string, v any) error {
	body, err := c.get(ctx, url)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("youku: decode %s: %w", url, err)
	}
	return nil
}

// get fetches a URL and returns the body bytes.
func (c *Client) get(ctx context.Context, url string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.doGet(ctx, url)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, lastErr
}

func (c *Client) doGet(ctx context.Context, url string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/json,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Referer", "https://www.youku.com/")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, true, ErrRateLimited
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, false, ErrNotFound
	}
	if resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("youku: http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("youku: http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

var metaRE = regexp.MustCompile(
	`(?i)<meta[^>]+(?:property|name|itemprop)=["']([^"']+)["'][^>]+content=["']([^"']*)["'][^>]*>|` +
		`<meta[^>]+content=["']([^"']*)["'][^>]+(?:property|name|itemprop)=["']([^"']+)["'][^>]*>`)

// parseOGP extracts Open Graph Protocol meta tag values from HTML.
func parseOGP(htmlBody string) map[string]string {
	out := map[string]string{}
	for _, m := range metaRE.FindAllStringSubmatch(htmlBody, -1) {
		if m[1] != "" && m[2] != "" {
			out[m[1]] = m[2]
		} else if m[4] != "" && m[3] != "" {
			out[m[4]] = m[3]
		}
	}
	return out
}

// cleanTitle strips common Youku title suffixes.
func cleanTitle(s string) string {
	suffixes := []string{
		"—优酷",
		"_优酷",
		" - 优酷",
		"_高清在线观看_优酷",
		"_在线观看_优酷",
	}
	for _, suf := range suffixes {
		if strings.HasSuffix(s, suf) {
			return strings.TrimSuffix(s, suf)
		}
	}
	return s
}

// showPageURL builds the canonical show page URL from a show ID.
func showPageURL(id string) string {
	return ShowBase + id + ".html"
}
