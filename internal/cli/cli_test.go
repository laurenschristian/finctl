package cli

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/laurenschristian/finctl/internal/provider"
)

func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	flagJSON = false
	root := Root()
	root.SetArgs(args)
	err := root.Execute()
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String(), err
}

// fixtures points providers at a local server and isolates config + cache.
func fixtures(t *testing.T) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/simple/price", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"bitcoin":{"usd":65000,"usd_24h_change":1.5}}`))
	})
	mux.HandleFunc("/latest", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"date":"2026-09-14","rates":{"EUR":0.92}}`))
	})
	mux.HandleFunc("/chart/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"chart":{"result":[{"meta":{"symbol":"NVDA","regularMarketPrice":170,"chartPreviousClose":160,"regularMarketDayHigh":172,"regularMarketDayLow":168,"regularMarketVolume":1000,"fiftyTwoWeekHigh":190,"fiftyTwoWeekLow":80},"timestamp":[1,2,3],"indicators":{"quote":[{"open":[100,101,102],"high":[103,104,105],"low":[99,100,101],"close":[101,103,102],"volume":[10,11,12]}]}}],"error":null}}`))
	})
	mux.HandleFunc("/NVDA.json", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"timestamp":"t","data":{"symbol":"NVDA","current_price":170.5,"open":168,"high":172,"low":167,"prev_day_close":160,"volume":123,"price_change":10.5,"price_change_percent":6.5}}`))
	})
	mux.HandleFunc("/files/company_tickers.json", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"0":{"cik_str":1045810,"ticker":"NVDA","title":"NVIDIA"},"1":{"cik_str":789019,"ticker":"MSFT","title":"MSFT"}}`))
	})
	mux.HandleFunc("/api/xbrl/companyconcept/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "PaymentsToAcquireProperty"):
			_, _ = w.Write([]byte(`{"units":{"USD":[{"end":"2024-06-30","val":1000,"frame":"CY2024Q2"},{"end":"2025-06-30","val":1500,"frame":"CY2025Q2"}]}}`))
		case strings.Contains(r.URL.Path, "RevenueFromContract"):
			_, _ = w.Write([]byte(`{"units":{"USD":[{"end":"2025-06-30","val":30000,"frame":"CY2025Q2"}]}}`))
		case strings.Contains(r.URL.Path, "CommonStockSharesOutstanding"):
			_, _ = w.Write([]byte(`{"units":{"shares":[{"end":"2025-06-30","val":2400,"frame":"CY2025Q2I"}]}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	mux.HandleFunc("/api/xbrl/companyfacts/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"facts":{"us-gaap":{"Revenues":{"units":{"USD":[{"start":"2025-04-01","end":"2025-06-30","val":30000,"fy":2025,"fp":"Q2","form":"10-Q","filed":"2025-07-30"}]}},"CommonStockSharesOutstanding":{"units":{"shares":[{"end":"2025-06-30","val":2400,"fy":2025,"fp":"Q2","form":"10-Q","filed":"2025-07-30"}]}}}}}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	provider.SetBases(srv.URL)
	t.Setenv("FINCTL_CONFIG", os.DevNull)
	t.Setenv("FINCTL_CACHE_DIR", t.TempDir())
}

func TestCommands(t *testing.T) {
	fixtures(t)
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"quote", "NVDA"}, "SYMBOL"},
		{[]string{"chart", "NVDA", "--range", "6m"}, "NVDA"},
		{[]string{"crypto", "bitcoin"}, "COIN"},
		{[]string{"fx", "EUR"}, "USD/EUR"},
		{[]string{"fund", "NVDA"}, "PERIOD"},
		{[]string{"capex", "--tickers", "NVDA"}, "COMPANY"},
	}
	for _, c := range cases {
		out, err := run(t, c.args...)
		if err != nil || !strings.Contains(out, c.want) {
			t.Fatalf("%v -> %v\n%s", c.args, err, out)
		}
		j, err := run(t, append(c.args, "--json")...)
		if err != nil || !strings.Contains(j, "{") && !strings.Contains(j, "[") {
			t.Fatalf("%v --json -> %v\n%s", c.args, err, j)
		}
	}
}

func TestCapexYoY(t *testing.T) {
	fixtures(t)
	out, err := run(t, "capex", "--tickers", "NVDA")
	if err != nil || !strings.Contains(out, "+50.0%") {
		t.Fatalf("capex yoy %v\n%s", err, out)
	}
}

func TestDoctorKeysCache(t *testing.T) {
	fixtures(t)
	if out, err := run(t, "doctor"); err != nil || !strings.Contains(out, "cache") {
		t.Fatalf("doctor %v\n%s", err, out)
	}
	if out, err := run(t, "keys"); err != nil || !strings.Contains(out, "fred") {
		t.Fatalf("keys %v\n%s", err, out)
	}
	if out, err := run(t, "cache", "clear"); err != nil || !strings.Contains(out, "cleared") {
		t.Fatalf("cache clear %v\n%s", err, out)
	}
}

func TestMCPTools(t *testing.T) {
	fixtures(t)
	// build hx for the server the way PersistentPreRun would.
	if _, err := run(t, "doctor"); err != nil {
		t.Fatal(err)
	}
	s := mcpServer()
	ct, st := mcp.NewInMemoryTransports()
	go func() { _ = s.Run(context.Background(), st) }()
	sess, err := mcp.NewClient(&mcp.Implementation{Name: "t"}, nil).Connect(context.Background(), ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sess.Close() }()
	tools, err := sess.ListTools(context.Background(), nil)
	if err != nil || len(tools.Tools) != 6 {
		t.Fatalf("tools=%d %v", len(tools.Tools), err)
	}
}
