package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/laurenschristian/finctl/internal/httpx"
	"github.com/laurenschristian/finctl/internal/model"
)

var cboeOptionsBase = "https://cdn.cboe.com/api/global/delayed_quotes/options"

type occOption struct {
	Kind   byte // 'C' or 'P'
	Strike float64
	Expiry string // YYMMDD
	OI     float64
}

// parseOCC extracts kind, strike, and expiry from an OCC option symbol such as
// NVDA260914C00050000 (expiry 2026-09-14, call, strike 50.000).
func parseOCC(sym string) (occOption, bool) {
	if len(sym) < 15 {
		return occOption{}, false
	}
	tail := sym[len(sym)-15:]
	expiry := tail[:6]
	kind := tail[6]
	if kind != 'C' && kind != 'P' {
		return occOption{}, false
	}
	strike := atof(tail[7:]) / 1000
	return occOption{Kind: kind, Strike: strike, Expiry: expiry}, true
}

// OptionsChain summarizes a delayed Cboe option chain: front IV, put/call ratio
// by open interest, max pain, and the biggest open-interest walls.
func OptionsChain(ctx context.Context, h *httpx.Client, symbol string) (*model.OptionsSummary, error) {
	sym := strings.ToUpper(symbol)
	u := cboeOptionsBase + "/" + sym + ".json"
	b, err := h.Get(ctx, "cboe", u, 60*time.Second)
	if err != nil {
		return nil, err
	}
	var raw struct {
		Data struct {
			Current float64 `json:"current_price"`
			IV30    float64 `json:"iv30"`
			Options []struct {
				Option string  `json:"option"`
				OI     float64 `json:"open_interest"`
			} `json:"options"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	if len(raw.Data.Options) == 0 {
		return nil, fmt.Errorf("cboe: no option chain for %s", sym)
	}
	var opts []occOption
	oiByStrike := map[float64]float64{}
	expiries := map[string]bool{}
	var callOI, putOI float64
	for _, o := range raw.Data.Options {
		p, ok := parseOCC(o.Option)
		if !ok {
			continue
		}
		p.OI = o.OI
		opts = append(opts, p)
		oiByStrike[p.Strike] += o.OI
		expiries[p.Expiry] = true
		if p.Kind == 'P' {
			putOI += o.OI
		} else {
			callOI += o.OI
		}
	}
	sum := &model.OptionsSummary{
		Symbol:     sym,
		Underlying: raw.Data.Current,
		FrontIV:    raw.Data.IV30,
		Expiries:   len(expiries),
	}
	if callOI > 0 {
		sum.PutCallRatio = putOI / callOI
	}
	sum.MaxPain = maxPain(opts)
	// Top OI walls.
	type kv struct {
		k float64
		v float64
	}
	var walls []kv
	for k, v := range oiByStrike {
		walls = append(walls, kv{k, v})
	}
	sort.Slice(walls, func(i, j int) bool { return walls[i].v > walls[j].v })
	for i := 0; i < len(walls) && i < 5; i++ {
		sum.OIWalls = append(sum.OIWalls, model.Strike{Strike: walls[i].k, OI: walls[i].v})
	}
	return sum, nil
}

// maxPain returns the strike that minimizes total option-holder intrinsic value
// (the classic writers' pain point) across all strikes in the chain.
func maxPain(opts []occOption) float64 {
	strikes := map[float64]bool{}
	for _, o := range opts {
		strikes[o.Strike] = true
	}
	var best float64
	bestPain := -1.0
	for k := range strikes {
		var pain float64
		for _, o := range opts {
			if o.Kind == 'C' && k > o.Strike {
				pain += (k - o.Strike) * o.OI
			} else if o.Kind == 'P' && k < o.Strike {
				pain += (o.Strike - k) * o.OI
			}
		}
		if bestPain < 0 || pain < bestPain {
			bestPain, best = pain, k
		}
	}
	return best
}
