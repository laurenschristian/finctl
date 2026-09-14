package provider

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/laurenschristian/finctl/internal/httpx"
	"github.com/laurenschristian/finctl/internal/model"
)

const cboeQuoteTTL = 60 * time.Second

var cboeBase = "https://cdn.cboe.com/api/global/delayed_quotes/quotes"

// CboeQuote returns a delayed quote from Cboe's free JSON. Index symbols use an
// underscore prefix (VIX -> _VIX); a caret prefix is normalized to that.
func CboeQuote(ctx context.Context, h *httpx.Client, symbol string) (*model.Quote, error) {
	sym := strings.ToUpper(symbol)
	if strings.HasPrefix(sym, "^") {
		sym = "_" + sym[1:]
	}
	u := cboeBase + "/" + sym + ".json"
	b, err := h.Get(ctx, "cboe", u, cboeQuoteTTL)
	if err != nil {
		return nil, err
	}
	var raw struct {
		Timestamp string `json:"timestamp"`
		Data      struct {
			Symbol       string  `json:"symbol"`
			CurrentPrice float64 `json:"current_price"`
			Open         float64 `json:"open"`
			High         float64 `json:"high"`
			Low          float64 `json:"low"`
			Close        float64 `json:"close"`
			PrevDayClose float64 `json:"prev_day_close"`
			Volume       float64 `json:"volume"`
			PriceChange  float64 `json:"price_change"`
			PriceChgPct  float64 `json:"price_change_percent"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	d := raw.Data
	prev := d.PrevDayClose
	if prev == 0 {
		prev = d.Close
	}
	return &model.Quote{
		Symbol: symbol, Last: d.CurrentPrice, Change: d.PriceChange,
		ChangePct: d.PriceChgPct, Open: d.Open, DayHigh: d.High, DayLow: d.Low,
		PrevClose: prev, Volume: d.Volume, AsOf: raw.Timestamp,
	}, nil
}
