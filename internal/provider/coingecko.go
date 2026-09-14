// Package provider wraps each finctl data source. One file per source; every
// call routes through httpx (cache + rate limit + UA).
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/laurenschristian/finctl/internal/httpx"
	"github.com/laurenschristian/finctl/internal/model"
)

const coingeckoTTL = 60 * time.Second

var coingeckoBase = "https://api.coingecko.com/api/v3"

// Crypto returns spot prices for coin ids (e.g. bitcoin, ethereum) vs a fiat.
func Crypto(ctx context.Context, h *httpx.Client, coins []string, vs string) ([]model.CryptoPrice, error) {
	if vs == "" {
		vs = "usd"
	}
	vs = strings.ToLower(vs)
	q := url.Values{}
	q.Set("ids", strings.Join(coins, ","))
	q.Set("vs_currencies", vs)
	q.Set("include_24hr_change", "true")
	u := coingeckoBase + "/simple/price?" + q.Encode()
	b, err := h.Get(ctx, "coingecko", u, coingeckoTTL)
	if err != nil {
		return nil, err
	}
	var raw map[string]map[string]float64
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	out := make([]model.CryptoPrice, 0, len(coins))
	for _, coin := range coins {
		m, ok := raw[coin]
		if !ok {
			return nil, fmt.Errorf("unknown coin %q", coin)
		}
		out = append(out, model.CryptoPrice{
			Coin:     coin,
			VS:       vs,
			Price:    m[vs],
			Change24: m[vs+"_24h_change"],
		})
	}
	return out, nil
}
