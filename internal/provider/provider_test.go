package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/laurenschristian/finctl/internal/cache"
	"github.com/laurenschristian/finctl/internal/httpx"
)

func register(mux *http.ServeMux) {
	mux.HandleFunc("/simple/price", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"bitcoin":{"usd":65000,"usd_24h_change":1.5},"ethereum":{"usd":3200,"usd_24h_change":-0.4}}`))
	})
	mux.HandleFunc("/latest", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"amount":1,"base":"USD","date":"2026-09-14","rates":{"EUR":0.92,"GBP":0.79}}`))
	})
	mux.HandleFunc("/chart/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"chart":{"result":[{"meta":{"symbol":"NVDA","currency":"USD","regularMarketPrice":170.0,"chartPreviousClose":160.0,"regularMarketDayHigh":172.0,"regularMarketDayLow":168.0,"regularMarketVolume":1000,"fiftyTwoWeekHigh":190,"fiftyTwoWeekLow":80},"timestamp":[1,2,3],"indicators":{"quote":[{"open":[100,101,102],"high":[103,104,105],"low":[99,100,101],"close":[101,103,102],"volume":[10,11,12]}]}}],"error":null}}`))
	})
	mux.HandleFunc("/NVDA.json", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"timestamp":"2026-09-14","data":{"symbol":"NVDA","current_price":170.5,"open":168,"high":172,"low":167,"close":169,"prev_day_close":160,"volume":123456,"price_change":10.5,"price_change_percent":6.5}}`))
	})
	mux.HandleFunc("/files/company_tickers.json", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"0":{"cik_str":1045810,"ticker":"NVDA","title":"NVIDIA CORP"},"1":{"cik_str":789019,"ticker":"MSFT","title":"MICROSOFT CORP"}}`))
	})
	mux.HandleFunc("/api/xbrl/companyconcept/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "PaymentsToAcquirePropertyPlantAndEquipment"):
			_, _ = w.Write([]byte(`{"units":{"USD":[{"start":"2024-04-01","end":"2024-06-30","val":1000,"frame":"CY2024Q2","form":"10-Q"},{"start":"2025-04-01","end":"2025-06-30","val":1500,"frame":"CY2025Q2","form":"10-Q"}]}}`))
		case strings.Contains(r.URL.Path, "RevenueFromContractWithCustomerExcludingAssessedTax"):
			_, _ = w.Write([]byte(`{"units":{"USD":[{"start":"2025-04-01","end":"2025-06-30","val":30000,"frame":"CY2025Q2","form":"10-Q"}]}}`))
		case strings.Contains(r.URL.Path, "CommonStockSharesOutstanding"):
			_, _ = w.Write([]byte(`{"units":{"shares":[{"end":"2025-06-30","val":2400,"frame":"CY2025Q2I","form":"10-Q"}]}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	// companyfacts (all facts, one call) backs Fundamentals.
	mux.HandleFunc("/api/xbrl/companyfacts/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(companyFactsFixture))
	})
	// Treasury par yield curve (Atom XML).
	mux.HandleFunc("/resource-center/data-chart-center/interest-rates/pages/xml", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<?xml version="1.0"?><feed><entry><content><properties>` +
			`<NEW_DATE>2026-09-14T00:00:00</NEW_DATE><BC_3MONTH>4.11</BC_3MONTH><BC_2YEAR>4.65</BC_2YEAR>` +
			`<BC_10YEAR>4.97</BC_10YEAR><BC_30YEAR>5.34</BC_30YEAR></properties></content></entry></feed>`))
	})
	mux.HandleFunc("/services/api/fiscal_service/v2/accounting/od/debt_to_penny", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"record_date":"2026-09-11","debt_held_public_amt":"32357573654300.31","intragov_hold_amt":"7688604668492.47","tot_pub_debt_out_amt":"40046178322792.78"}]}`))
	})
	// CFTC COT: two weekly rows for the main contract plus a MICRO decoy.
	mux.HandleFunc("/cftc.json", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[` +
			`{"contract_market_name":"E-MINI S&P 500 STOCK INDEX","report_date_as_yyyy_mm_dd":"2026-09-08T00:00:00.000","noncomm_positions_long_all":"268972","noncomm_positions_short_all":"332505"},` +
			`{"contract_market_name":"E-MINI S&P 500 STOCK INDEX","report_date_as_yyyy_mm_dd":"2026-09-01T00:00:00.000","noncomm_positions_long_all":"260000","noncomm_positions_short_all":"324651"},` +
			`{"contract_market_name":"MICRO E-MINI S&P 500 INDEX","report_date_as_yyyy_mm_dd":"2026-09-08T00:00:00.000","noncomm_positions_long_all":"5","noncomm_positions_short_all":"5"}]`))
	})
	// Kalshi KXFED ladder (nearest meeting), _dollars string fields.
	mux.HandleFunc("/markets", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"markets":[` +
			`{"ticker":"KXFED-26SEP-T3.50","event_ticker":"KXFED-26SEP","yes_sub_title":"Above 3.50%","close_time":"2026-09-16T18:00:00Z","floor_strike":3.5,"yes_bid_dollars":"0.98","yes_ask_dollars":"0.99","last_price_dollars":"0.99"},` +
			`{"ticker":"KXFED-26SEP-T3.75","event_ticker":"KXFED-26SEP","yes_sub_title":"Above 3.75%","close_time":"2026-09-16T18:00:00Z","floor_strike":3.75,"yes_bid_dollars":"0.85","yes_ask_dollars":"0.87","last_price_dollars":"0.86"},` +
			`{"ticker":"KXFED-26SEP-T4.00","event_ticker":"KXFED-26SEP","yes_sub_title":"Above 4.00%","close_time":"2026-09-16T18:00:00Z","floor_strike":4.0,"yes_bid_dollars":"0.01","yes_ask_dollars":"0.02","last_price_dollars":"0.01"}]}`))
	})
	mux.HandleFunc("/series/observations", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"observations":[{"date":"2026-09-01","value":"4.10"},{"date":"2026-08-01","value":"."},{"date":"2026-07-01","value":"4.05"}]}`))
	})
}

// companyFactsFixture: Q2 discrete revenue (30000), a de-cumulable capex YTD
// chain (Q1 500, Q2 2000 -> discrete 1500), and shares outstanding (2400).
const companyFactsFixture = `{"facts":{"us-gaap":{
  "Revenues":{"units":{"USD":[
    {"start":"2025-04-01","end":"2025-06-30","val":30000,"fy":2025,"fp":"Q2","form":"10-Q","filed":"2025-07-30"}
  ]}},
  "PaymentsToAcquirePropertyPlantAndEquipment":{"units":{"USD":[
    {"start":"2025-01-01","end":"2025-03-31","val":500,"fy":2025,"fp":"Q1","form":"10-Q","filed":"2025-04-30"},
    {"start":"2025-01-01","end":"2025-06-30","val":2000,"fy":2025,"fp":"Q2","form":"10-Q","filed":"2025-07-30"}
  ]}},
  "CommonStockSharesOutstanding":{"units":{"shares":[
    {"end":"2025-06-30","val":2400,"fy":2025,"fp":"Q2","form":"10-Q","filed":"2025-07-30"}
  ]}}
}}}`

func TestProviders(t *testing.T) {
	mux := http.NewServeMux()
	var base string
	register(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()
	base = srv.URL
	SetBases(base)
	store, _ := cache.Open(filepath.Join(t.TempDir(), "c.db"))
	defer func() { _ = store.Close() }()
	h := httpx.New(store, "test")
	ctx := context.Background()

	cp, err := Crypto(ctx, h, []string{"bitcoin", "ethereum"}, "usd")
	if err != nil || len(cp) != 2 || cp[0].Price != 65000 {
		t.Fatalf("crypto %v %+v", err, cp)
	}
	fx, err := FX(ctx, h, "usd", []string{"EUR", "GBP"})
	if err != nil || len(fx) != 2 {
		t.Fatalf("fx %v %+v", err, fx)
	}
	q, err := CboeQuote(ctx, h, "NVDA")
	if err != nil || q.Last != 170.5 || q.ChangePct != 6.5 {
		t.Fatalf("cboe %v %+v", err, q)
	}
	yq, err := YahooQuote(ctx, h, "NVDA")
	if err != nil || yq.Last != 170.0 || yq.Change != 10.0 {
		t.Fatalf("yahoo quote %v %+v", err, yq)
	}
	ch, err := YahooChart(ctx, h, "NVDA", "6m", "1d")
	if err != nil || len(ch.Bars) != 3 {
		t.Fatalf("yahoo chart %v %+v", err, ch)
	}
	cik, err := CIKFor(ctx, h, "NVDA")
	if err != nil || cik != "CIK0001045810" {
		t.Fatalf("cik %v %q", err, cik)
	}
	f, err := Fundamentals(ctx, h, "NVDA")
	if err != nil || len(f.Periods) == 0 {
		t.Fatalf("fund %v %+v", err, f)
	}
	var haveRev bool
	for _, p := range f.Periods {
		if p.Fiscal == "CY2025Q2" && p.Revenue == 30000 && p.Capex == 1500 {
			haveRev = true
		}
	}
	if !haveRev {
		t.Fatalf("fund period missing: %+v", f.Periods)
	}
	cx, err := HyperscalerCapex(ctx, h, []string{"NVDA"}, 6)
	if err != nil || len(cx) == 0 {
		t.Fatalf("capex %v %+v", err, cx)
	}
	// YoY on CY2025Q2 vs CY2024Q2 = (1500-1000)/1000 = 50%.
	var yoyOK bool
	for _, p := range cx {
		if p.Fiscal == "CY2025Q2" && p.YoY == 50 {
			yoyOK = true
		}
	}
	if !yoyOK {
		t.Fatalf("capex yoy wrong: %+v", cx)
	}
	if _, err := CIKFor(ctx, h, "ZZZZ"); err == nil {
		t.Fatal("want unknown ticker error")
	}
	curve, err := TreasuryCurve(ctx, h)
	if err != nil || len(curve) != 4 || curve[0].Tenor != "3M" || curve[0].Yield != 4.11 {
		t.Fatalf("curve %v %+v", err, curve)
	}
	debt, err := TreasuryDebt(ctx, h)
	if err != nil || debt.Date != "2026-09-11" || debt.TotalDebt == 0 {
		t.Fatalf("debt %v %+v", err, debt)
	}
	cot, err := Cot(ctx, h, "ES")
	if err != nil || cot.Market != "E-MINI S&P 500 STOCK INDEX" || cot.Net != 268972-332505 {
		t.Fatalf("cot %v %+v", err, cot)
	}
	if cot.NetPrior != 260000-324651 {
		t.Fatalf("cot prior %+v", cot)
	}
	odds, err := FedOdds(ctx, h)
	if err != nil || odds.Meeting != "KXFED-26SEP" || len(odds.Buckets) == 0 {
		t.Fatalf("fedodds %v %+v", err, odds)
	}
	// Top bucket by prob should be 3.75-4.00% at ~0.85 (mid 3.75 rung - mid 4.00 rung).
	if odds.Buckets[0].Band != "3.75-4.00%" || odds.Buckets[0].Prob < 0.8 {
		t.Fatalf("fedodds bucket %+v", odds.Buckets)
	}
	s, err := FredSeries(ctx, h, "testkey", "DGS10", 12)
	if err != nil || len(s.Points) != 2 { // the "." observation is dropped
		t.Fatalf("fred %v %+v", err, s)
	}
	if _, err := FredSeries(ctx, h, "", "DGS10", 12); err == nil {
		t.Fatal("want FRED missing-key error")
	}
}

func TestFXDropsBaseFromSymbols(t *testing.T) {
	var gotSymbols, gotBase string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBase = r.URL.Query().Get("base")
		gotSymbols = r.URL.Query().Get("symbols")
		_, _ = w.Write([]byte(`{"base":"USD","date":"2026-09-14","rates":{"EUR":0.9}}`))
	}))
	defer srv.Close()
	old := frankfurterBase
	frankfurterBase = srv.URL
	defer func() { frankfurterBase = old }()

	// "fx USD" passes USD as base and as the only target; the target must be dropped.
	if _, err := FX(context.Background(), httpx.New(nil, "test"), "USD", []string{"USD"}); err != nil {
		t.Fatalf("fx: %v", err)
	}
	if gotBase != "USD" || gotSymbols != "" {
		t.Fatalf("want base USD and empty symbols, got base=%q symbols=%q", gotBase, gotSymbols)
	}
	if _, err := FX(context.Background(), httpx.New(nil, "test"), "USD", []string{"EUR", "USD"}); err != nil {
		t.Fatalf("fx: %v", err)
	}
	if gotSymbols != "EUR" {
		t.Fatalf("want symbols EUR, got %q", gotSymbols)
	}
}
