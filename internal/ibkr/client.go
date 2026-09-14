// Package ibkr is a minimal read-only client for the IBKR Client Portal gateway
// that ibkrctl runs on localhost. finctl reuses the daemon: it never logs in.
// Account ids and holder names are redacted to aliases before anything leaves.
package ibkr

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/laurenschristian/finctl/internal/model"
)

// Client talks to the local Client Portal gateway (self-signed TLS on loopback).
type Client struct {
	Base string
	HTTP *http.Client
}

// New returns a client for the gateway base URL (e.g. https://localhost:5001).
func New(base string) *Client {
	return &Client{
		Base: strings.TrimRight(base, "/"),
		HTTP: &http.Client{
			Timeout: 15 * time.Second,
			// The gateway serves a self-signed cert on loopback only.
			Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}, //nolint:gosec
		},
	}
}

func (c *Client) get(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.Base+path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("IBKR gateway unreachable at %s: is the ibkrctl gateway running?", c.Base)
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("IBKR gateway HTTP %d on %s", resp.StatusCode, path)
	}
	return b, nil
}

// Authenticated reports whether the brokerage session is live. A false result
// means the user must run `ibkrctl login` (2FA); finctl cannot do it.
func (c *Client) Authenticated(ctx context.Context) (bool, error) {
	b, err := c.get(ctx, "/v1/api/iserver/auth/status")
	if err != nil {
		// A gateway that is up but has no brokerage session answers 401 here;
		// that is "not authenticated", not a hard error.
		if strings.Contains(err.Error(), "HTTP 401") {
			return false, nil
		}
		return false, err
	}
	var st struct {
		Authenticated bool `json:"authenticated"`
	}
	if err := json.Unmarshal(b, &st); err != nil {
		return false, err
	}
	return st.Authenticated, nil
}

// aliasFor builds a stable real-id -> alias map (account-1, account-2, ...).
func aliasFor(ids []string) map[string]string {
	sorted := append([]string(nil), ids...)
	sort.Strings(sorted)
	m := map[string]string{}
	for i, id := range sorted {
		m[id] = fmt.Sprintf("account-%d", i+1)
	}
	return m
}

// Positions returns every position across accounts, with account ids replaced
// by aliases. Weights are the fraction of total market value.
func (c *Client) Positions(ctx context.Context) ([]model.Position, error) {
	b, err := c.get(ctx, "/v1/api/portfolio/accounts")
	if err != nil {
		return nil, err
	}
	var accts []struct {
		ID string `json:"accountId"`
	}
	if err := json.Unmarshal(b, &accts); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(accts))
	for _, a := range accts {
		ids = append(ids, a.ID)
	}
	alias := aliasFor(ids)

	var out []model.Position
	var total float64
	for _, id := range ids {
		pb, err := c.get(ctx, "/v1/api/portfolio/"+id+"/positions/0")
		if err != nil {
			return nil, err
		}
		var rows []struct {
			Ticker   string  `json:"ticker"`
			Contract string  `json:"contractDesc"`
			Position float64 `json:"position"`
			MktPrice float64 `json:"mktPrice"`
			MktValue float64 `json:"mktValue"`
			AvgCost  float64 `json:"avgCost"`
			Unreal   float64 `json:"unrealizedPnl"`
		}
		if err := json.Unmarshal(pb, &rows); err != nil {
			return nil, err
		}
		for _, r := range rows {
			sym := r.Ticker
			if sym == "" {
				sym = r.Contract
			}
			out = append(out, model.Position{
				Symbol:    sym,
				Qty:       r.Position,
				Price:     r.MktPrice,
				Value:     r.MktValue,
				CostBasis: r.AvgCost * r.Position,
				UnrealPnL: r.Unreal,
				Account:   alias[id],
			})
			total += r.MktValue
		}
	}
	if total > 0 {
		for i := range out {
			out[i].Weight = out[i].Value / total
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Value > out[j].Value })
	return out, nil
}
