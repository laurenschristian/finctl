package vault

import "testing"

func TestParseTargets(t *testing.T) {
	md := "# Dashboard\n\n| Ticker | Target | Notes |\n| --- | --- | --- |\n| NVDA | 25% | core |\n| AAPL | 15% | |\n| notaticker | 5% | |\n"
	got := ParseTargets(md)
	if len(got) != 2 {
		t.Fatalf("want 2 targets, got %d: %+v", len(got), got)
	}
	if got[0].Ticker != "NVDA" || got[0].Weight != 0.25 {
		t.Fatalf("nvda %+v", got[0])
	}
	if got[1].Ticker != "AAPL" || got[1].Weight != 0.15 {
		t.Fatalf("aapl %+v", got[1])
	}
}

func TestParseTargetsReordered(t *testing.T) {
	md := "| Notes | Weight | Symbol |\n|---|---|---|\n| x | 30% | MSFT |\n"
	got := ParseTargets(md)
	if len(got) != 1 || got[0].Ticker != "MSFT" || got[0].Weight != 0.30 {
		t.Fatalf("reordered %+v", got)
	}
}

func TestParseWatchlist(t *testing.T) {
	md := "| Ticker | Buy Zone | Thesis |\n|---|---|---|\n| NVDA | 150 - 170 | AI |\n| AAPL | $200-$220 | services |\n"
	got := ParseWatchlist(md)
	if len(got) != 2 {
		t.Fatalf("want 2, got %+v", got)
	}
	if got[0].Ticker != "NVDA" || got[0].BuyLow != 150 || got[0].BuyHigh != 170 || got[0].Thesis != "AI" {
		t.Fatalf("nvda %+v", got[0])
	}
	if got[1].BuyLow != 200 || got[1].BuyHigh != 220 {
		t.Fatalf("aapl zone %+v", got[1])
	}
}

func TestParseEmpty(t *testing.T) {
	if got := ParseTargets("no table here"); got != nil {
		t.Fatalf("want nil, got %+v", got)
	}
}

func TestParseWatchlistMultiTable(t *testing.T) {
	// A real note has several tables: a buy-zone table, a verdict log (no zone
	// column), and a trade plan (Qty/Limit). Only the zone table is a source,
	// and header words like "Ticker"/"Date" must never leak in as rows.
	md := "" +
		"| Ticker | Live | Recorded zone | Status |\n|---|---|---|---|\n" +
		"| **[[FLNC - Fluence\\|FLNC]]** | $14 | $20-24 | below |\n" +
		"| CAMT | $146 | 130-140 | above |\n\n" +
		"Some prose.\n\n" +
		"| Ticker | Live | Verdict | Why |\n|---|---|---|---|\n" +
		"| ETN | $398 | REJECTED | crowded |\n\n" +
		"| Ticker | Qty | Limit | Thesis |\n|---|---|---|---|\n" +
		"| LITE | 1 | $865 | pair trade |\n"
	got := ParseWatchlist(md)
	if len(got) != 2 {
		t.Fatalf("want 2 zone rows, got %d: %+v", len(got), got)
	}
	if got[0].Ticker != "FLNC" || got[0].BuyLow != 20 || got[0].BuyHigh != 24 {
		t.Fatalf("flnc %+v", got[0])
	}
	if got[1].Ticker != "CAMT" || got[1].BuyLow != 130 || got[1].BuyHigh != 140 {
		t.Fatalf("camt %+v", got[1])
	}
	for _, w := range got {
		switch w.Ticker {
		case "TICKER", "DATE", "ETN", "LITE":
			t.Fatalf("leaked non-zone row: %s", w.Ticker)
		}
	}
}
