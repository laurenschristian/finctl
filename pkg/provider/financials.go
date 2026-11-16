package provider

import (
	"context"
	"sort"
	"strings"

	"github.com/laurenschristian/finctl/pkg/httpx"
	"github.com/laurenschristian/finctl/pkg/model"
)

var (
	grossTags  = []string{"GrossProfit"}
	costTags   = []string{"CostOfGoodsAndServicesSold", "CostOfRevenue"}
	opIncTags  = []string{"OperatingIncomeLoss"}
	netIncTags = []string{"NetIncomeLoss"}
	epsTags    = []string{"EarningsPerShareDiluted", "EarningsPerShareBasic"}
	ocfTags    = []string{"NetCashProvidedByUsedInOperatingActivities"}
)

// financialsKeep caps the history at three years of quarters.
const financialsKeep = 12

func isQuarterSpan(start, end string) bool {
	n := spanDays(start, end)
	return n >= 80 && n <= 100
}

// discrete returns one value per quarter end for a duration concept. Filers report
// some quarters directly and others only year to date (Q4 sits inside the 10-K
// annual figure), so YTD chains sharing a start date are differenced. For EPS this
// is an approximation, since share counts move inside the year.
func discrete(facts map[string][]factRow, tags []string) map[string]float64 {
	var best map[string]float64
	var bestEnd string
	for _, tag := range tags {
		rows, ok := facts[tag]
		if !ok {
			continue
		}
		byStart := map[string]map[string]factRow{}
		for _, r := range rows {
			if n := spanDays(r.Start, r.End); n < 80 || n > 380 {
				continue
			}
			if byStart[r.Start] == nil {
				byStart[r.Start] = map[string]factRow{}
			}
			if prev, ok := byStart[r.Start][r.End]; !ok || r.Filed > prev.Filed {
				byStart[r.Start][r.End] = r
			}
		}
		direct, derived := map[string]float64{}, map[string]float64{}
		var annual []factRow
		for _, m := range byStart {
			seq := make([]factRow, 0, len(m))
			for _, r := range m {
				seq = append(seq, r)
			}
			sort.Slice(seq, func(i, j int) bool { return seq[i].End < seq[j].End })
			for i, r := range seq {
				if n := spanDays(r.Start, r.End); n >= 350 {
					annual = append(annual, r)
				}
				switch {
				case isQuarterSpan(r.Start, r.End):
					direct[r.End] = r.Val
				case i > 0 && isQuarterSpan(seq[i-1].End, r.End):
					derived[r.End] = r.Val - seq[i-1].Val
				}
			}
		}
		for end, v := range derived {
			if _, ok := direct[end]; !ok {
				direct[end] = v
			}
		}
		for _, a := range annual {
			if _, ok := direct[a.End]; ok {
				continue
			}
			var sum float64
			var n int
			for end, v := range direct {
				if end > a.Start && end < a.End {
					sum += v
					n++
				}
			}
			if n == 3 {
				direct[a.End] = a.Val - sum
			}
		}
		var last string
		for end := range direct {
			last = max(last, end)
		}
		if last > bestEnd {
			best, bestEnd = direct, last
		}
	}
	return best
}

// Financials returns quarterly revenue, margins, EPS, cash flow and share count from
// SEC XBRL companyfacts, plus a trailing-twelve-month roll-up when four quarters line up.
func Financials(ctx context.Context, h *httpx.Client, ticker string) (*model.Financials, error) {
	cik, err := CIKFor(ctx, h, ticker)
	if err != nil {
		return nil, err
	}
	facts, err := companyFacts(ctx, h, cik)
	if err != nil {
		return nil, err
	}
	rev := discrete(facts, append(revenueTags, "RevenueFromContractWithCustomerIncludingAssessedTax"))
	gross := discrete(facts, grossTags)
	cost := discrete(facts, costTags)
	opInc := discrete(facts, opIncTags)
	net := discrete(facts, netIncTags)
	eps := discrete(facts, epsTags)
	ocf := discrete(facts, ocfTags)
	capex := discrete(facts, capexTags)

	var shares []factRow
	for _, tag := range append(sharesTags, "EntityCommonStockSharesOutstanding") {
		shares = append(shares, facts[tag]...)
	}
	ends := make([]string, 0, len(rev))
	for end := range rev {
		ends = append(ends, end)
	}
	sort.Strings(ends)
	if len(ends) > financialsKeep {
		ends = ends[len(ends)-financialsKeep:]
	}
	out := &model.Financials{Symbol: strings.ToUpper(ticker), CIK: cik}
	for _, end := range ends {
		p := model.FinPeriod{Label: cyLabel(end), End: end, Revenue: rev[end], OperatingIncome: opInc[end],
			NetIncome: net[end], EPS: eps[end], OCF: ocf[end], Capex: capex[end]}
		if g, ok := gross[end]; ok {
			p.GrossProfit = g
		} else if c, ok := cost[end]; ok {
			p.GrossProfit = p.Revenue - c
		}
		if _, ok := ocf[end]; ok {
			p.FCF = p.OCF - p.Capex
		}
		var best factRow
		for _, s := range shares {
			if s.End != "" && (best.End == "" || absDayDiff(s.End, end) < absDayDiff(best.End, end)) {
				best = s
			}
		}
		if best.End != "" && absDayDiff(best.End, end) <= 45 {
			p.Shares = best.Val
		}
		out.Quarters = append(out.Quarters, p)
	}
	out.TTM = ttm(out.Quarters)
	return out, nil
}

// ttm sums the last four quarters when they are consecutive; shares are the latest count.
func ttm(qs []model.FinPeriod) *model.FinPeriod {
	if len(qs) < 4 {
		return nil
	}
	last := qs[len(qs)-4:]
	for i := 1; i < 4; i++ {
		if n := absDayDiff(last[i-1].End, last[i].End); n < 80 || n > 100 {
			return nil
		}
	}
	t := &model.FinPeriod{Label: "TTM", End: last[3].End, Shares: last[3].Shares}
	for _, q := range last {
		t.Revenue += q.Revenue
		t.GrossProfit += q.GrossProfit
		t.OperatingIncome += q.OperatingIncome
		t.NetIncome += q.NetIncome
		t.EPS += q.EPS
		t.OCF += q.OCF
		t.Capex += q.Capex
		t.FCF += q.FCF
	}
	return t
}
