package research

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/laurenschristian/finctl/pkg/httpx"
	"github.com/laurenschristian/finctl/pkg/model"
	"github.com/laurenschristian/finctl/pkg/provider"
)

func serve(t *testing.T) *httpx.Client {
	t.Helper()
	files := map[string]string{
		"/files/company_tickers.json":                                 "company_tickers.json",
		"/submissions/CIK0000002488.json":                             "submissions.json",
		"/Archives/edgar/data/2488/000000248826000101/form4_buy.xml":  "form4_buy.xml",
		"/Archives/edgar/data/2488/000000248826000100/form4_sell.xml": "form4_sell.xml",
		"/api/xbrl/companyfacts/CIK0000002488.json":                   "companyfacts.json",
		"/data/group/otcMarket/name/consolidatedShortInterest":        "short.json",
		"/options/AMD.json":                                           "options.json",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name, ok := files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		b, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Error(err)
		}
		_, _ = w.Write(b)
	}))
	t.Cleanup(srv.Close)
	provider.SetBases(srv.URL)
	return httpx.New(nil, "test")
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func TestFillAllSections(t *testing.T) {
	h := serve(t)
	d := &Dossier{Symbol: "AMD", Price: 150, Earnings: "2026-12-01"}
	Fill(context.Background(), h, d, Sections...)
	if len(d.Errors) != 0 {
		t.Fatalf("errors: %v", d.Errors)
	}
	if len(d.Filings) != 2 || d.Filings[0].Form != "8-K" || d.Filings[1].Form != "10-Q" {
		t.Fatalf("filings %+v", d.Filings)
	}
	if len(d.Insider) != 3 {
		t.Fatalf("insider %+v", d.Insider)
	}
	buy := d.Insider[0]
	if !buy.Buy() || buy.Shares != 2000 || buy.Price != 155 || buy.Date != "2026-11-07" || buy.Role != "Director" || buy.Owner != "Doe Jane" {
		t.Fatalf("buy %+v", buy)
	}
	if d.Insider[1].Code != "M" || d.Insider[2].Code != "S" || d.Insider[2].Role != "CFO" || d.Insider[2].Buy() {
		t.Fatalf("sale rows %+v", d.Insider[1:])
	}
	if d.Short == nil || d.Short.DaysToCover != 5 {
		t.Fatalf("short %+v", d.Short)
	}
	if d.Options == nil || d.Options.FrontIV != 55 || d.Options.MaxPain != 150 {
		t.Fatalf("options %+v", d.Options)
	}
	f := d.Financials
	if f == nil || len(f.Quarters) != 4 {
		t.Fatalf("financials %+v", f)
	}
	q4 := f.Quarters[3]
	if q4.End != "2025-12-31" || q4.Revenue != 1300 || q4.GrossProfit != 650 || q4.OCF != 230 || q4.Capex != 60 || q4.FCF != 170 || q4.Shares != 1060 || !near(q4.EPS, 0.65) {
		t.Fatalf("derived Q4 %+v", q4)
	}
	if f.TTM == nil || f.TTM.Revenue != 4600 || f.TTM.FCF != 640 || !near(f.TTM.OpMargin(), 520.0/4600) {
		t.Fatalf("ttm %+v", f.TTM)
	}
}

func TestFillRecordsErrors(t *testing.T) {
	h := serve(t)
	d := &Dossier{Symbol: "ZZZZ"}
	Fill(context.Background(), h, d, Options, Filings)
	if d.Errors["options"] == "" || d.Errors["filings"] == "" {
		t.Fatalf("want per-section errors, got %v", d.Errors)
	}
	if Fetch(context.Background(), h, d, "nope") == nil {
		t.Fatal("unknown section should fail")
	}
}

func TestLens(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	byKey := func(items []Item) map[string]Item {
		m := map[string]Item{}
		for _, it := range items {
			m[it.Key] = it
		}
		return m
	}
	empty := byKey(Lens(&Dossier{Symbol: "X"}, now))
	if len(empty) != 13 {
		t.Fatalf("want 13 items, got %d", len(empty))
	}
	for k, it := range empty {
		if it.Status != Unknown {
			t.Errorf("%s: want unknown without data, got %s", k, it.Status)
		}
	}

	pts := []model.Obs{{Value: 100}, {Value: 100}, {Value: 100}, {Value: 100}, {Value: 110}}
	d := &Dossier{
		Symbol: "X", Price: 10, Earnings: "2026-12-01",
		Dilution:   &model.DilutionReport{Points: pts, ChangePctYr: 10, Flag: "diluting"},
		Options:    &model.OptionsSummary{FrontIV: 90},
		Short:      &model.ShortInterest{DaysToCover: 2},
		Insider:    []model.InsiderTrade{{Code: "S"}, {Code: "S"}, {Code: "A"}},
		Financials: &model.Financials{TTM: &model.FinPeriod{Revenue: 100, OperatingIncome: 20, NetIncome: 10}},
	}
	got := byKey(Lens(d, now))
	want := map[string]Status{"financing": No, "catalyst": Yes, "headroom": Yes, "risk": No, "insiders": No, "margins": Yes, "bottleneck": Unknown}
	for k, s := range want {
		if got[k].Status != s {
			t.Errorf("%s: want %s, got %s (%s)", k, s, got[k].Status, got[k].Evidence)
		}
	}
	d.Earnings = "2027-06-01"
	d.Insider = append(d.Insider, model.InsiderTrade{Code: "P"})
	d.Price = 1e9
	got = byKey(Lens(d, now))
	if got["catalyst"].Status != No || got["insiders"].Status != Mixed || got["headroom"].Status != No {
		t.Fatalf("second pass %+v", got)
	}
}

func TestField(t *testing.T) {
	d := &Dossier{}
	for _, s := range Sections {
		if d.Field(s) == nil {
			t.Errorf("no field for %s", s)
		}
	}
	if d.Field("nope") != nil {
		t.Error("unknown section should have no field")
	}
}
