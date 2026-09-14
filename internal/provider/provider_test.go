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
