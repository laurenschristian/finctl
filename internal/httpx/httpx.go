// Package httpx is the one outbound HTTP path for every provider: it sets the
// User-Agent, enforces a per-provider rate limit, and routes reads through the
// cache. Providers never touch net/http directly.
package httpx

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/laurenschristian/finctl/internal/cache"
)

// Client wraps net/http with a cache and per-provider rate limiters.
type Client struct {
	HTTP  *http.Client
	Cache *cache.Store
	UA    string

	mu       sync.Mutex
	limiters map[string]*rate.Limiter
}

// providerRate is the requests-per-second ceiling per provider. SEC EDGAR is a
// hard 10/s total cap; the rest are courtesy limits well under any documented
// quota. Unlisted providers get defaultRate.
var providerRate = map[string]rate.Limit{
	"edgar": 8, // stay under the SEC 10/s hard cap with headroom
	"finra": 4,
	"cftc":  4,
	"fred":  8,
}

const defaultRate rate.Limit = 5

func New(store *cache.Store, ua string) *Client {
	return &Client{
		HTTP:     &http.Client{Timeout: 30 * time.Second},
		Cache:    store,
		UA:       ua,
		limiters: map[string]*rate.Limiter{},
	}
}

func (c *Client) limiter(provider string) *rate.Limiter {
	c.mu.Lock()
	defer c.mu.Unlock()
	l, ok := c.limiters[provider]
	if !ok {
		r := defaultRate
		if pr, has := providerRate[provider]; has {
			r = pr
		}
		l = rate.NewLimiter(r, 1)
		c.limiters[provider] = l
	}
	return l
}

// Get fetches url for provider, serving a fresh cache entry when one exists.
func (c *Client) Get(ctx context.Context, provider, url string, ttl time.Duration) ([]byte, error) {
	return c.do(ctx, provider, http.MethodGet, url, nil, ttl)
}

// PostJSON POSTs body (JSON bytes) and caches by url+body hash.
func (c *Client) PostJSON(ctx context.Context, provider, url string, body []byte, ttl time.Duration) ([]byte, error) {
	return c.do(ctx, provider, http.MethodPost, url, body, ttl)
}

func (c *Client) do(ctx context.Context, provider, method, url string, body []byte, ttl time.Duration) ([]byte, error) {
	key := url
	if len(body) > 0 {
		sum := sha256.Sum256(body)
		key = url + "#" + hex.EncodeToString(sum[:8])
	}
	if c.Cache != nil {
		if b, ok := c.Cache.Get(ctx, provider, key, ttl); ok {
			return b, nil
		}
	}
	if err := c.limiter(provider).Wait(ctx); err != nil {
		return nil, err
	}
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, rdr)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.UA)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s %s: HTTP %d", provider, method, resp.StatusCode)
	}
	if c.Cache != nil {
		_ = c.Cache.Put(ctx, provider, key, b)
	}
	return b, nil
}
