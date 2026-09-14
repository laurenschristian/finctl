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

const (
	yahooChartTTL = 10 * time.Minute
	yahooQuoteTTL = 60 * time.Second
)

var yahooBase = "https://query1.finance.yahoo.com/v8/finance/chart"

// yahooRange maps a friendly range to Yahoo's vocabulary.
var yahooRange = map[string]string{
	"1d": "1d", "5d": "5d", "1m": "1mo", "3m": "3mo", "6m": "6mo",
	"1y": "1y", "2y": "2y", "5y": "5y", "ytd": "ytd", "max": "max",
}

type yahooChartResp struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Symbol               string  `json:"symbol"`
				Currency             string  `json:"currency"`
				RegularMarketPrice   float64 `json:"regularMarketPrice"`
				ChartPreviousClose   float64 `json:"chartPreviousClose"`
				PreviousClose        float64 `json:"previousClose"`
				RegularMarketVolume  float64 `json:"regularMarketVolume"`
				RegularMarketDayHigh float64 `json:"regularMarketDayHigh"`
				RegularMarketDayLow  float64 `json:"regularMarketDayLow"`
				FiftyTwoWeekHigh     float64 `json:"fiftyTwoWeekHigh"`
				FiftyTwoWeekLow      float64 `json:"fiftyTwoWeekLow"`
			} `json:"meta"`
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Open   []float64 `json:"open"`
					High   []float64 `json:"high"`
					Low    []float64 `json:"low"`
					Close  []float64 `json:"close"`
					Volume []float64 `json:"volume"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error any `json:"error"`
	} `json:"chart"`
}

func yahooGet(ctx context.Context, h *httpx.Client, symbol, rng, interval string, ttl time.Duration) (*yahooChartResp, error) {
	r, ok := yahooRange[strings.ToLower(rng)]
	if !ok {
		r = "6mo"
	}
	if interval == "" {
		interval = "1d"
	}
	q := url.Values{"range": {r}, "interval": {interval}}
	u := yahooBase + "/" + url.PathEscape(symbol) + "?" + q.Encode()
	b, err := h.Get(ctx, "yahoo", u, ttl)
	if err != nil {
		return nil, err
	}
	var resp yahooChartResp
	if err := json.Unmarshal(b, &resp); err != nil {
		return nil, err
	}
	if len(resp.Chart.Result) == 0 {
		return nil, fmt.Errorf("yahoo: no data for %q", symbol)
	}
	return &resp, nil
}

// YahooChart returns OHLCV bars for a symbol over a range.
func YahooChart(ctx context.Context, h *httpx.Client, symbol, rng, interval string) (*model.Chart, error) {
	resp, err := yahooGet(ctx, h, symbol, rng, interval, yahooChartTTL)
	if err != nil {
		return nil, err
	}
	res := resp.Chart.Result[0]
	out := &model.Chart{Symbol: res.Meta.Symbol, Range: rng}
	if len(res.Indicators.Quote) == 0 {
		return out, nil
	}
	q := res.Indicators.Quote[0]
	for i, ts := range res.Timestamp {
		if i >= len(q.Close) || q.Close[i] == 0 {
			continue
		}
		out.Bars = append(out.Bars, model.Bar{
			Time: ts, Open: at(q.Open, i), High: at(q.High, i),
			Low: at(q.Low, i), Close: q.Close[i], Volume: at(q.Volume, i),
		})
	}
	return out, nil
}

// YahooQuote returns a quote from the chart meta (fallback for cboe).
func YahooQuote(ctx context.Context, h *httpx.Client, symbol string) (*model.Quote, error) {
	resp, err := yahooGet(ctx, h, symbol, "1d", "1d", yahooQuoteTTL)
	if err != nil {
		return nil, err
	}
	m := resp.Chart.Result[0].Meta
	prev := m.ChartPreviousClose
	if prev == 0 {
		prev = m.PreviousClose
	}
	q := &model.Quote{
		Symbol: m.Symbol, Last: m.RegularMarketPrice, PrevClose: prev,
		DayLow: m.RegularMarketDayLow, DayHigh: m.RegularMarketDayHigh,
		Volume: m.RegularMarketVolume, Week52Low: m.FiftyTwoWeekLow,
		Week52High: m.FiftyTwoWeekHigh, Currency: m.Currency,
	}
	if prev != 0 {
		q.Change = q.Last - prev
		q.ChangePct = q.Change / prev * 100
	}
	return q, nil
}

func at(a []float64, i int) float64 {
	if i < len(a) {
		return a[i]
	}
	return 0
}
