package provider

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/laurenschristian/finctl/pkg/httpx"
)

// tickerMap is SEC's company_tickers.json recorded 2026-11-16 (10,404 entries).
func tickerMap(tb testing.TB) []byte {
	tb.Helper()
	return gunzip(tb, filepath.Join("testdata", "company_tickers.json.gz"))
}

func gunzip(tb testing.TB, path string) []byte {
	tb.Helper()
	f, err := os.Open(path)
	if err != nil {
		tb.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	zr, err := gzip.NewReader(f)
	if err != nil {
		tb.Fatal(err)
	}
	b, err := io.ReadAll(zr)
	if err != nil {
		tb.Fatal(err)
	}
	return b
}

// oldCIK is the pre-streaming lookup: full unmarshal into a map.
func oldCIK(b []byte, ticker string) int {
	var raw map[string]struct {
		CIK    int    `json:"cik_str"`
		Ticker string `json:"ticker"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return 0
	}
	for _, e := range raw {
		if strings.EqualFold(e.Ticker, ticker) {
			return e.CIK
		}
	}
	return 0
}

var benchTickers = []struct{ name, ticker string }{
	{"first", "AAPL"},
	{"middle", "FTEK"},
	{"last", "BLHWF"},
	{"missing", "ZZZZZZ"},
}

func TestCIKFromTickerMap(t *testing.T) {
	b := tickerMap(t)
	for _, tc := range benchTickers[:3] {
		got, err := cikFromTickerMap(b, strings.ToLower(tc.ticker))
		if err != nil || got == 0 || got != oldCIK(b, tc.ticker) {
			t.Fatalf("%s: got %d err %v, want %d", tc.ticker, got, err, oldCIK(b, tc.ticker))
		}
	}
	if _, err := cikFromTickerMap(b, "ZZZZZZ"); err == nil || !strings.Contains(err.Error(), "unknown ticker") {
		t.Fatalf("missing ticker: %v", err)
	}
	if _, err := cikFromTickerMap([]byte(`[]`), "AAPL"); err == nil {
		t.Fatal("want shape error")
	}
	// Pretty-printed or lowercase maps miss the byte search and must still resolve via streaming.
	pretty := []byte(`{"0": {"cik_str": 320193, "ticker": "aapl"}, "1": {"cik_str": 1045810, "ticker": "NVDA"}}`)
	if got, err := cikFromTickerMap(pretty, "nvda"); err != nil || got != 1045810 {
		t.Fatalf("pretty nvda: %d %v", got, err)
	}
	if got, err := cikFromTickerMap(pretty, "AAPL"); err != nil || got != 320193 {
		t.Fatalf("pretty aapl: %d %v", got, err)
	}
	for _, tc := range benchTickers[:3] {
		if got, _ := streamTickerMap(b, tc.ticker, tc.ticker); got != oldCIK(b, tc.ticker) {
			t.Fatalf("stream %s: %d", tc.ticker, got)
		}
	}
}

func TestNormalizeCIK(t *testing.T) {
	for _, in := range []string{"320193", "0000320193", "CIK0000320193", " cik320193 "} {
		if got, err := NormalizeCIK(in); err != nil || got != "CIK0000320193" {
			t.Fatalf("%q: %q %v", in, got, err)
		}
	}
	for _, in := range []string{"", "CIK", "abc", "-1", "0", "12345678901"} {
		if _, err := NormalizeCIK(in); err == nil {
			t.Fatalf("%q: want error", in)
		}
	}
}

func TestFilingsByCIKSkipsTickerMap(t *testing.T) {
	var tickerHits int
	facts := companyFactsMSFT(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/files/company_tickers.json":
			tickerHits++
			_, _ = w.Write([]byte(`{"0":{"cik_str":320193,"ticker":"AAPL","title":"Apple Inc."}}`))
		case "/submissions/CIK0000320193.json":
			_, _ = w.Write([]byte(`{"filings":{"recent":{"accessionNumber":["0001-26-1","0001-26-2"],"form":["8-K","4"],
				"filingDate":["2026-11-10","2026-11-09"],"reportDate":["",""],"primaryDocument":["a.htm","xslF345X05/b.xml"],"primaryDocDescription":["8-K",""]}}}`))
		case "/api/xbrl/companyfacts/CIK0000789019.json":
			_, _ = w.Write(facts)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	SetBases(srv.URL)
	h := httpx.New(nil, "test")
	ctx := context.Background()

	fs, err := FilingsByCIK(ctx, h, "320193", "", 10)
	if err != nil || len(fs) != 2 || fs[0].URL != srv.URL+"/Archives/edgar/data/320193/0001261/a.htm" {
		t.Fatalf("FilingsByCIK %v %+v", err, fs)
	}
	ins, err := InsiderByCIK(ctx, h, "CIK0000320193", 5)
	if err != nil || len(ins) != 1 || ins[0].Form != "4" {
		t.Fatalf("InsiderByCIK %v %+v", err, ins)
	}
	fin, err := FinancialsByCIK(ctx, h, "789019", "msft")
	if err != nil || fin.Symbol != "MSFT" || len(fin.Quarters) != financialsKeep || fin.TTM == nil || fin.TTM.Revenue <= 0 {
		t.Fatalf("FinancialsByCIK %v %+v", err, fin)
	}
	if tickerHits != 0 {
		t.Fatalf("ByCIK fetched the ticker map %d times", tickerHits)
	}
	if fs, err := Filings(ctx, h, "aapl", "8-K", 10); err != nil || len(fs) != 1 || tickerHits != 1 {
		t.Fatalf("Filings %v %+v hits %d", err, fs, tickerHits)
	}
	if _, err := FilingsByCIK(ctx, h, "nope", "", 10); err == nil {
		t.Fatal("want invalid CIK error")
	}
}

func BenchmarkCIKLookup(b *testing.B) {
	data := tickerMap(b)
	for _, tc := range benchTickers {
		b.Run("old/"+tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = oldCIK(data, tc.ticker)
			}
		})
		b.Run("stream/"+tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_, _ = streamTickerMap(data, tc.ticker, tc.ticker)
			}
		})
		b.Run("lookup/"+tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_, _ = cikFromTickerMap(data, tc.ticker)
			}
		})
	}
}

func companyFactsMSFT(tb testing.TB) []byte {
	tb.Helper()
	return gunzip(tb, filepath.Join("testdata", "companyfacts_msft.json.gz"))
}

// oldFacts is the pre-selective parse: every concept unmarshalled and indexed.
func oldFacts(b []byte) map[string][]factRow {
	var raw struct {
		Facts map[string]map[string]factUnits `json:"facts"`
	}
	if json.Unmarshal(b, &raw) != nil {
		return nil
	}
	out := map[string][]factRow{}
	for _, tags := range raw.Facts {
		for tag, f := range tags {
			for _, arr := range f.Units {
				for _, e := range arr {
					out[tag] = append(out[tag], factRow{Start: e.Start, End: e.End, FP: e.FP, Form: e.Form, Filed: e.Filed, FY: e.FY, Val: e.Val})
				}
			}
		}
	}
	return out
}

func TestParseFactsMatchesFullParse(t *testing.T) {
	b := companyFactsMSFT(t)
	full := oldFacts(b)
	tags := financialsTags()
	got, err := parseFacts(b, tags)
	if err != nil {
		t.Fatal(err)
	}
	slow, err := parseAllFacts(b, map[string]bool{"Revenues": true})
	if err != nil || len(slow["Revenues"]) != len(full["Revenues"]) {
		t.Fatalf("fallback %v %d vs %d", err, len(slow["Revenues"]), len(full["Revenues"]))
	}
	var n int
	for _, tag := range tags {
		if len(got[tag]) != len(full[tag]) {
			t.Fatalf("%s: %d rows, full parse %d", tag, len(got[tag]), len(full[tag]))
		}
		n += len(got[tag])
	}
	if n == 0 || len(got) >= len(full) {
		t.Fatalf("selective parse kept %d tags (%d rows) of %d", len(got), n, len(full))
	}
}

func BenchmarkCompanyFacts(b *testing.B) {
	data := companyFactsMSFT(b)
	tags := financialsTags()
	b.Run("old/full", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = oldFacts(data)
		}
	})
	b.Run("new/financials", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, _ = parseFacts(data, tags)
		}
	})
}
