package provider

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"time"

	"github.com/laurenschristian/finctl/internal/httpx"
	"github.com/laurenschristian/finctl/internal/model"
)

const frankfurterTTL = 6 * time.Hour

var frankfurterBase = "https://api.frankfurter.app"

// FX returns spot exchange rates from a base currency to one or more targets
// (ECB reference rates via Frankfurter).
func FX(ctx context.Context, h *httpx.Client, from string, to []string) ([]model.FXRate, error) {
	from = strings.ToUpper(from)
	q := url.Values{}
	q.Set("from", from)
	if len(to) > 0 {
		q.Set("to", strings.ToUpper(strings.Join(to, ",")))
	}
	u := frankfurterBase + "/latest?" + q.Encode()
	b, err := h.Get(ctx, "frankfurter", u, frankfurterTTL)
	if err != nil {
		return nil, err
	}
	var raw struct {
		Date  string             `json:"date"`
		Rates map[string]float64 `json:"rates"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	out := make([]model.FXRate, 0, len(raw.Rates))
	for cur, rate := range raw.Rates {
		out = append(out, model.FXRate{From: from, To: cur, Rate: rate, Date: raw.Date})
	}
	return out, nil
}
