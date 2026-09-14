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
