package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/laurenschristian/finctl/internal/httpx"
	"github.com/laurenschristian/finctl/internal/model"
)

const kalshiTTL = 15 * time.Minute

var kalshiBase = "https://api.elections.kalshi.com/trade-api/v2"

type kalshiMarket struct {
	Ticker      string  `json:"ticker"`
	EventTicker string  `json:"event_ticker"`
	Subtitle    string  `json:"yes_sub_title"`
	CloseTime   string  `json:"close_time"`
	FloorStrike float64 `json:"floor_strike"`
	YesBid      float64 `json:"yes_bid_dollars,string"`
	YesAsk      float64 `json:"yes_ask_dollars,string"`
	LastPrice   float64 `json:"last_price_dollars,string"`
}

// FedOdds derives the market-implied target-rate distribution for the nearest
// FOMC meeting from Kalshi's KXFED ladder. Each rung is P(target above a floor),
// so the probability of a 25bp band is P(above lower) minus P(above upper).
func FedOdds(ctx context.Context, h *httpx.Client) (*model.FedOdds, error) {
	u := kalshiBase + "/markets?series_ticker=KXFED&status=open&limit=500"
	b, err := h.Get(ctx, "kalshi", u, kalshiTTL)
	if err != nil {
		return nil, err
	}
	var raw struct {
		Markets []kalshiMarket `json:"markets"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	// Group by meeting; keep the soonest meeting that has priced rungs.
	byEvent := map[string][]kalshiMarket{}
	for _, m := range raw.Markets {
		byEvent[m.EventTicker] = append(byEvent[m.EventTicker], m)
	}
	type meeting struct {
		event string
		close string
		rungs []kalshiMarket
	}
	var meetings []meeting
	for ev, ms := range byEvent {
		var priced []kalshiMarket
		earliest := ""
		for _, m := range ms {
			if m.YesBid > 0 || m.YesAsk > 0 || m.LastPrice > 0 {
				priced = append(priced, m)
			}
			if earliest == "" || m.CloseTime < earliest {
				earliest = m.CloseTime
			}
		}
		if len(priced) > 0 {
			meetings = append(meetings, meeting{ev, earliest, priced})
		}
	}
	if len(meetings) == 0 {
		return nil, fmt.Errorf("kalshi: no priced KXFED markets")
	}
	sort.Slice(meetings, func(i, j int) bool { return meetings[i].close < meetings[j].close })
	next := meetings[0]
	sort.Slice(next.rungs, func(i, j int) bool { return next.rungs[i].FloorStrike < next.rungs[j].FloorStrike })

	// prob(above floor) from the yes bid/ask midpoint (fall back to last price).
	mid := func(m kalshiMarket) float64 {
		if m.YesBid > 0 && m.YesAsk > 0 {
			return (m.YesBid + m.YesAsk) / 2
		}
		return m.LastPrice
	}
	out := &model.FedOdds{
		Meeting: next.event,
		Source:  "kalshi KXFED",
		AsOf:    trimDate(next.close),
	}
	for i := 0; i < len(next.rungs)-1; i++ {
		lo, hi := next.rungs[i], next.rungs[i+1]
		p := mid(lo) - mid(hi)
		if p < 0 {
			p = 0
		}
		band := fmt.Sprintf("%.2f-%.2f%%", lo.FloorStrike, hi.FloorStrike)
		out.Buckets = append(out.Buckets, model.FedBucket{Band: band, Prob: p})
	}
	// The top rung is the residual "above the highest floor".
	if n := len(next.rungs); n > 0 {
		top := next.rungs[n-1]
		out.Buckets = append(out.Buckets, model.FedBucket{
			Band: fmt.Sprintf(">%.2f%%", top.FloorStrike), Prob: mid(top),
		})
	}
	sort.Slice(out.Buckets, func(i, j int) bool { return out.Buckets[i].Prob > out.Buckets[j].Prob })
	return out, nil
}
