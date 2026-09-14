package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/laurenschristian/finctl/internal/httpx"
	"github.com/laurenschristian/finctl/internal/model"
)

const cftcTTL = 24 * time.Hour

// cftcBase is the Socrata legacy combined (futures + options) COT resource.
var cftcBase = "https://publicreporting.cftc.gov/resource/jun7-fc8e.json"

// cotMarkets maps a short market code to a name-search pattern.
var cotMarkets = map[string]string{
	"ES": "E-MINI S&P 500", "SP": "S&P 500",
	"NQ": "NASDAQ", "YM": "DJIA", "RTY": "RUSSELL",
	"GC": "GOLD", "SI": "SILVER", "HG": "COPPER",
	"CL": "CRUDE OIL", "NG": "NATURAL GAS", "RB": "GASOLINE",
	"ZB": "U.S. TREASURY BONDS", "ZN": "10 YEAR", "ZF": "5 YEAR", "ZT": "2 YEAR",
	"DX": "DOLLAR INDEX", "BTC": "BITCOIN", "VX": "VIX",
}

type cotRow struct {
	Name  string `json:"contract_market_name"`
	Date  string `json:"report_date_as_yyyy_mm_dd"`
	Long  string `json:"noncomm_positions_long_all"`
	Short string `json:"noncomm_positions_short_all"`
}

// Cot returns CFTC Commitments of Traders positioning for a market: the latest
// non-commercial net, and the net from the prior weekly report.
func Cot(ctx context.Context, h *httpx.Client, market string) (*model.CotReport, error) {
	market = strings.ToUpper(market)
	pat := cotMarkets[market]
	if pat == "" {
		pat = market // let the caller pass a raw name fragment
	}
	where := fmt.Sprintf("upper(contract_market_name) like '%%%s%%'", strings.ToUpper(pat))
	q := url.Values{}
	q.Set("$where", where)
	q.Set("$order", "report_date_as_yyyy_mm_dd DESC")
	q.Set("$limit", "150")
	u := cftcBase + "?" + q.Encode()
	b, err := h.Get(ctx, "cftc", u, cftcTTL)
	if err != nil {
		return nil, err
	}
	var rows []cotRow
	if err := json.Unmarshal(b, &rows); err != nil {
		return nil, err
	}
	// Prefer the primary contract: drop MICRO/E-MICRO, then pick the name with
	// the largest open positioning (the deepest contract for this market).
	byName := map[string][]cotRow{}
	for _, r := range rows {
		if strings.Contains(r.Name, "MICRO") {
			continue
		}
		byName[r.Name] = append(byName[r.Name], r)
	}
	if len(byName) == 0 {
		return nil, fmt.Errorf("cot: no contract matched %q", market)
	}
	var bestName string
	var bestSize float64
	for name, rs := range byName {
		sz := atof(rs[0].Long) + atof(rs[0].Short)
		if sz > bestSize {
			bestName, bestSize = name, sz
		}
	}
	series := byName[bestName]
	sort.Slice(series, func(i, j int) bool { return series[i].Date > series[j].Date })
	latest := series[0]
	rep := &model.CotReport{
		Market: bestName,
		Date:   trimDate(latest.Date),
		Long:   atof(latest.Long),
		Short:  atof(latest.Short),
		Net:    atof(latest.Long) - atof(latest.Short),
		AsOf:   trimDate(latest.Date),
	}
	if len(series) > 1 {
		rep.NetPrior = atof(series[1].Long) - atof(series[1].Short)
	}
	return rep, nil
}

// trimDate keeps the YYYY-MM-DD prefix of a Socrata timestamp.
func trimDate(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}
